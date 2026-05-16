// Package api is a thin, typed wrapper around the PostHog REST + HogQL API.
//
// All commands go through Client. The client maps HTTP status codes to the
// CLI's typed exit codes via output.Errorf so agents can self-correct.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kdubb1337/phog-cli/internal/config"
	"github.com/kdubb1337/phog-cli/internal/output"
)

// Client talks to a single PostHog project.
type Client struct {
	host      string
	apiKey    string
	projectID string
	http      *http.Client
}

// New builds a client from the resolved config. Returns a 4 (auth) error if
// the key is missing and a 2 (usage) error if the project ID is missing.
func New() (*Client, error) {
	s := config.Get()
	if s.APIKey == "" {
		return nil, output.ErrorfHint(4, "missing_api_key",
			"export PHOG_API_KEY=phx_... (Settings → Personal API keys in PostHog)",
			"PHOG_API_KEY is not set")
	}
	if s.ProjectID == "" {
		return nil, output.ErrorfHint(2, "missing_project_id",
			"export PHOG_PROJECT_ID=<numeric id from the URL /project/<ID>/...> or pass --project",
			"PHOG_PROJECT_ID is not set")
	}
	if !strings.HasPrefix(s.APIKey, "phx_") {
		// Soft-warn on stderr; PostHog will reject it with a 401 anyway.
		output.Progress("warning: PHOG_API_KEY does not start with 'phx_' — the project ingestion key (phc_*) cannot query the API.")
	}
	return &Client{
		host:      s.Host,
		apiKey:    s.APIKey,
		projectID: s.ProjectID,
		http:      &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// ProjectID exposes the resolved project ID (used by `doctor`).
func (c *Client) ProjectID() string { return c.projectID }

// Host exposes the resolved host (used by `doctor`).
func (c *Client) Host() string { return c.host }

// Do performs an authenticated request against a project-scoped path.
// path must start with "/api/projects/<id>/..." or "/api/...". If it starts
// with "/api/projects/{project_id}", the placeholder is substituted.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	full := c.buildURL(path, query)

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return output.Errorf(1, "marshal_body", "encode request body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, full, reqBody)
	if err != nil {
		return output.Errorf(8, "build_request", "build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	output.Debug("%s %s", method, full)

	resp, err := c.http.Do(req)
	if err != nil {
		return output.Errorf(8, "network", "%s %s: %v", method, full, err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return c.mapError(resp, respBytes)
	}

	if out == nil || len(respBytes) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBytes, out); err != nil {
		return output.Errorf(1, "decode_response", "decode response: %v (body=%s)", err, truncate(string(respBytes), 200))
	}
	return nil
}

// buildURL substitutes {project_id} and appends query params.
func (c *Client) buildURL(path string, query url.Values) string {
	path = strings.ReplaceAll(path, "{project_id}", c.projectID)
	u := c.host + path
	if len(query) > 0 {
		sep := "?"
		if strings.Contains(u, "?") {
			sep = "&"
		}
		u = u + sep + query.Encode()
	}
	return u
}

// mapError converts an HTTP error response to a typed CLIError with the right exit code.
func (c *Client) mapError(resp *http.Response, body []byte) error {
	excerpt := truncate(string(body), 300)
	// Try to pull PostHog's standard {"type", "code", "detail", "attr"} envelope.
	var ph struct {
		Type   string `json:"type"`
		Code   string `json:"code"`
		Detail string `json:"detail"`
		Attr   string `json:"attr"`
	}
	_ = json.Unmarshal(body, &ph)
	msg := ph.Detail
	if msg == "" {
		msg = excerpt
	}
	code := ph.Code
	if code == "" {
		code = "http_" + strconv.Itoa(resp.StatusCode)
	}

	switch {
	case resp.StatusCode == 401 || resp.StatusCode == 403:
		return output.ErrorfHint(4, code,
			"check PHOG_API_KEY scopes (need query:read / insight:read / person:read / project:read) and that the key is a Personal API key (phx_*), not a project key (phc_*)",
			"auth (%d): %s", resp.StatusCode, msg)
	case resp.StatusCode == 404:
		return output.Errorf(3, code, "not found (%d): %s", resp.StatusCode, msg)
	case resp.StatusCode == 409:
		return output.Errorf(6, code, "conflict (%d): %s", resp.StatusCode, msg)
	case resp.StatusCode == 422 || resp.StatusCode == 400:
		return output.ErrorfHint(9, code,
			"see PostHog API docs and verify query/HogQL syntax",
			"validation (%d): %s", resp.StatusCode, msg)
	case resp.StatusCode == 429:
		return output.ErrorfHint(7, code,
			"honor Retry-After header: "+resp.Header.Get("Retry-After"),
			"rate limited (%d): %s", resp.StatusCode, msg)
	case resp.StatusCode >= 500:
		return output.Errorf(5, code, "upstream (%d): %s", resp.StatusCode, msg)
	default:
		return output.Errorf(1, code, "unexpected (%d): %s", resp.StatusCode, msg)
	}
}

// Page is the standard PostHog paginated envelope. `Next` is a full URL.
type Page[T any] struct {
	Results []T    `json:"results"`
	Next    string `json:"next"`
	Prev    string `json:"previous"`
	Count   int    `json:"count,omitempty"`
}

// FollowCursor extracts the cursor portion from a `next` URL so the agent can
// re-issue with --cursor=<value>. Returns "" when there is no next page.
func FollowCursor(next string) string {
	if next == "" {
		return ""
	}
	u, err := url.Parse(next)
	if err != nil {
		return ""
	}
	q := u.Query()
	for _, k := range []string{"after", "cursor", "before"} {
		if v := q.Get(k); v != "" {
			return v
		}
	}
	// Some endpoints page by `offset=` — surface that too.
	if v := q.Get("offset"); v != "" {
		return "offset:" + v
	}
	return next
}

// ApplyCursor sets the appropriate query param from a cursor produced by
// FollowCursor (or one the user provided). PostHog uses different cursor
// param names per endpoint; the caller passes the canonical one.
func ApplyCursor(q url.Values, paramName, cursor string) {
	if cursor == "" {
		return
	}
	if strings.HasPrefix(cursor, "offset:") {
		q.Set("offset", strings.TrimPrefix(cursor, "offset:"))
		return
	}
	q.Set(paramName, cursor)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ParseDuration accepts "24h", "7d", "30m", or a date-string and returns an
// ISO-8601 timestamp suitable for PostHog's `after`/`before` query params.
func ParseDuration(input string) (string, error) {
	if input == "" {
		return "", nil
	}
	// Allow raw ISO-8601 / RFC3339 dates to pass through.
	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	// Allow "7d" as shorthand for 7 days ago.
	if strings.HasSuffix(input, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(input, "d"))
		if err == nil {
			return time.Now().Add(-time.Duration(days) * 24 * time.Hour).UTC().Format(time.RFC3339), nil
		}
	}
	// Fall back to Go's duration parser ("24h", "30m", "1h30m").
	d, err := time.ParseDuration(input)
	if err != nil {
		return "", fmt.Errorf("could not parse %q as duration (try 7d, 24h, 30m) or RFC3339 timestamp", input)
	}
	return time.Now().Add(-d).UTC().Format(time.RFC3339), nil
}
