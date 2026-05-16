package api

import (
	"context"
)

// HogQLResult is the response shape from /api/projects/{id}/query for HogQLQuery kind.
//
// Two PostHog quirks worth knowing:
//   - `types` is returned as an array of [columnName, clickhouseType] pairs,
//     not a flat []string. Decoded here as [][]string.
//   - The pagination flag is `hasMore` (camelCase), not `has_more`.
type HogQLResult struct {
	Columns     []string   `json:"columns"`
	Types       [][]string `json:"types,omitempty"`
	Results     [][]any    `json:"results"`
	HogQL       string     `json:"hogql,omitempty"`
	Modifiers   any        `json:"modifiers,omitempty"`
	HasMore     bool       `json:"hasMore,omitempty"`
	Timings     any        `json:"timings,omitempty"`
	QueryStatus any        `json:"query_status,omitempty"`
}

type hogqlEnvelope struct {
	Query map[string]any `json:"query"`
}

// Query executes a HogQL query string and returns the structured result.
// `query` should be a plain SQL-ish string (e.g.
//
//	SELECT event, count() FROM events WHERE timestamp > now() - INTERVAL 1 DAY GROUP BY event
//
// ). The CLI wraps it in the HogQLQuery envelope PostHog expects.
func (c *Client) Query(ctx context.Context, query string) (HogQLResult, error) {
	body := hogqlEnvelope{
		Query: map[string]any{
			"kind":  "HogQLQuery",
			"query": query,
		},
	}
	var out HogQLResult
	err := c.Do(ctx, "POST", "/api/projects/{project_id}/query/", nil, body, &out)
	return out, err
}

// ProjectGet hits /api/projects/{id}/ — used by `doctor` to verify auth + project ID.
func (c *Client) ProjectGet(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	err := c.Do(ctx, "GET", "/api/projects/{project_id}/", nil, nil, &out)
	return out, err
}
