package api

import (
	"context"
	"net/url"
	"strconv"
)

// Insight mirrors PostHog's /api/projects/{id}/insights response shape.
// `Result` is left as raw because shape depends on the insight type
// (Trends, Funnels, Retention, HogQL, etc.).
type Insight struct {
	ID          int            `json:"id"`
	ShortID     string         `json:"short_id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Filters     map[string]any `json:"filters,omitempty"`
	Query       map[string]any `json:"query,omitempty"`
	Result      any            `json:"result,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at,omitempty"`
	CreatedBy   map[string]any `json:"created_by,omitempty"`
}

type InsightsListParams struct {
	Search string
	Limit  int
	Cursor string // PostHog insights endpoint pages by offset
}

func (c *Client) InsightsList(ctx context.Context, p InsightsListParams) (Page[Insight], error) {
	q := url.Values{}
	if p.Search != "" {
		q.Set("search", p.Search)
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	ApplyCursor(q, "offset", p.Cursor)

	var page Page[Insight]
	err := c.Do(ctx, "GET", "/api/projects/{project_id}/insights/", q, nil, &page)
	return page, err
}

// InsightGet supports both numeric ID and short_id.
func (c *Client) InsightGet(ctx context.Context, idOrShort string) (Insight, error) {
	var ins Insight
	err := c.Do(ctx, "GET", "/api/projects/{project_id}/insights/"+url.PathEscape(idOrShort)+"/", nil, nil, &ins)
	return ins, err
}
