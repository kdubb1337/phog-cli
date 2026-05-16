package api

import (
	"encoding/json"
	"testing"
)

// Real fixture from PostHog. Notable: `types` is an array of [name, type]
// pairs (not []string), and `has_more` is actually returned as `hasMore`.
const hogqlResultFixture = `{
  "columns": ["event", "count()"],
  "types": [["event", "String"], ["count()", "UInt64"]],
  "results": [["$pageleave", 145], ["$identify", 24]],
  "hasMore": false,
  "hogql": "SELECT event, count() FROM events GROUP BY event"
}`

func TestHogQLResultDecode(t *testing.T) {
	var out HogQLResult
	if err := json.Unmarshal([]byte(hogqlResultFixture), &out); err != nil {
		t.Fatalf("unmarshal hogql result fixture: %v", err)
	}
	if got, want := out.Columns, []string{"event", "count()"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Columns = %v, want %v", got, want)
	}
	if got, want := len(out.Types), 2; got != want {
		t.Fatalf("Types: got %d pairs, want %d", got, want)
	}
	if out.Types[0][0] != "event" || out.Types[0][1] != "String" {
		t.Errorf("Types[0] = %v, want [event String]", out.Types[0])
	}
	if out.Types[1][0] != "count()" || out.Types[1][1] != "UInt64" {
		t.Errorf("Types[1] = %v, want [count() UInt64]", out.Types[1])
	}
	if got, want := len(out.Results), 2; got != want {
		t.Fatalf("Results: got %d rows, want %d", got, want)
	}
	if out.HasMore {
		t.Errorf("HasMore = true, want false")
	}
	if out.HogQL == "" {
		t.Errorf("HogQL empty, want a string")
	}
}
