package api

import (
	"context"
	"net/url"
	"strconv"
)

// Event mirrors PostHog's /api/projects/{id}/events response shape.
// We keep Properties as raw JSON-decoded so agents see the full structure.
type Event struct {
	ID          string         `json:"id"`
	DistinctID  string         `json:"distinct_id"`
	Event       string         `json:"event"`
	Timestamp   string         `json:"timestamp"`
	Properties  map[string]any `json:"properties,omitempty"`
	ElementsURL string         `json:"elements_chain,omitempty"`
	PersonID    string         `json:"person_id,omitempty"`
}

// EventsListParams are the supported filter knobs for `phog events list`.
type EventsListParams struct {
	Event      string // canonical PostHog event name, e.g. "$pageview"
	DistinctID string // filter to a single person
	After      string // RFC3339 timestamp (lower bound, inclusive)
	Before     string // RFC3339 timestamp (upper bound, inclusive)
	Limit      int
	Cursor     string // opaque cursor from a prior `next`
}

func (c *Client) EventsList(ctx context.Context, p EventsListParams) (Page[Event], error) {
	q := url.Values{}
	if p.Event != "" {
		q.Set("event", p.Event)
	}
	if p.DistinctID != "" {
		q.Set("distinct_id", p.DistinctID)
	}
	if p.After != "" {
		q.Set("after", p.After)
	}
	if p.Before != "" {
		q.Set("before", p.Before)
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	ApplyCursor(q, "after", p.Cursor)

	var page Page[Event]
	err := c.Do(ctx, "GET", "/api/projects/{project_id}/events/", q, nil, &page)
	return page, err
}

func (c *Client) EventGet(ctx context.Context, id string) (Event, error) {
	var ev Event
	err := c.Do(ctx, "GET", "/api/projects/{project_id}/events/"+url.PathEscape(id)+"/", nil, nil, &ev)
	return ev, err
}
