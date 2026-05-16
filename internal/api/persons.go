package api

import (
	"context"
	"net/url"
	"strconv"
)

// Person mirrors PostHog's /api/projects/{id}/persons response shape.
type Person struct {
	ID          int            `json:"id"`
	UUID        string         `json:"uuid"`
	DistinctIDs []string       `json:"distinct_ids"`
	Name        string         `json:"name,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
	CreatedAt   string         `json:"created_at"`
}

type PersonsListParams struct {
	Search     string // matches name / distinct_id / email
	DistinctID string // exact match
	Limit      int
	Cursor     string
}

func (c *Client) PersonsList(ctx context.Context, p PersonsListParams) (Page[Person], error) {
	q := url.Values{}
	if p.Search != "" {
		q.Set("search", p.Search)
	}
	if p.DistinctID != "" {
		q.Set("distinct_id", p.DistinctID)
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	ApplyCursor(q, "offset", p.Cursor)

	var page Page[Person]
	err := c.Do(ctx, "GET", "/api/projects/{project_id}/persons/", q, nil, &page)
	return page, err
}

// PersonGet looks up a person by their distinct_id by listing with the
// `distinct_id` filter (PostHog's REST surface; the `/persons/{id}/` form
// expects the internal numeric ID, which agents rarely have).
func (c *Client) PersonGet(ctx context.Context, distinctID string) (Person, error) {
	page, err := c.PersonsList(ctx, PersonsListParams{DistinctID: distinctID, Limit: 1})
	if err != nil {
		return Person{}, err
	}
	if len(page.Results) == 0 {
		return Person{}, nil
	}
	return page.Results[0], nil
}
