package api

import (
	"encoding/json"
	"testing"
)

// Real fixture from PostHog (id is a UUID string, not int).
const personsListFixture = `{
  "count": 2,
  "next": null,
  "previous": null,
  "results": [
    {
      "type": "person",
      "id": "9bc4fde1-37a3-5d7e-80ff-231fb1b72ed0",
      "uuid": "9bc4fde1-37a3-5d7e-80ff-231fb1b72ed0",
      "name": "alice@example.com",
      "distinct_ids": ["alice@example.com", "anon-123"],
      "properties": {"email": "alice@example.com", "plan": "pro"},
      "created_at": "2026-05-16T12:44:05.459000Z",
      "last_seen_at": "2026-05-16T12:00:00Z"
    },
    {
      "type": "person",
      "id": "f0e1d2c3-b4a5-6789-0abc-def012345678",
      "uuid": "f0e1d2c3-b4a5-6789-0abc-def012345678",
      "distinct_ids": ["bob@example.com"],
      "properties": {},
      "created_at": "2026-05-15T08:12:34.000000Z"
    }
  ]
}`

func TestPersonsListDecode(t *testing.T) {
	var page Page[Person]
	if err := json.Unmarshal([]byte(personsListFixture), &page); err != nil {
		t.Fatalf("unmarshal persons list fixture: %v", err)
	}
	if got, want := len(page.Results), 2; got != want {
		t.Fatalf("results: got %d want %d", got, want)
	}
	first := page.Results[0]
	if first.ID != "9bc4fde1-37a3-5d7e-80ff-231fb1b72ed0" {
		t.Errorf("first.ID = %q", first.ID)
	}
	if first.Type != "person" {
		t.Errorf("first.Type = %q", first.Type)
	}
	if first.Name != "alice@example.com" {
		t.Errorf("first.Name = %q", first.Name)
	}
	if got, want := len(first.DistinctIDs), 2; got != want {
		t.Errorf("first.DistinctIDs len = %d want %d", got, want)
	}
	if first.LastSeenAt == "" {
		t.Errorf("first.LastSeenAt empty (expected RFC3339)")
	}
	if first.Properties["plan"] != "pro" {
		t.Errorf("first.Properties[plan] = %v", first.Properties["plan"])
	}
}
