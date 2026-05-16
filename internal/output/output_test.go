package output

import (
	"encoding/json"
	"testing"
)

func TestStripNulls(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "drops top-level nulls",
			in:   `{"a": 1, "b": null, "c": "x"}`,
			want: `{"a":1,"c":"x"}`,
		},
		{
			name: "drops nested nulls",
			in:   `{"props": {"email": null, "plan": "pro", "$lib": null}, "id": "abc"}`,
			want: `{"id":"abc","props":{"plan":"pro"}}`,
		},
		{
			name: "preserves empty object left after stripping",
			in:   `{"props": {"a": null, "b": null}, "id": "abc"}`,
			want: `{"id":"abc","props":{}}`,
		},
		{
			name: "preserves array null elements (positional)",
			in:   `{"items": [1, null, 3]}`,
			want: `{"items":[1,null,3]}`,
		},
		{
			name: "recurses into array of objects",
			in:   `{"results": [{"a": null, "b": 1}, {"a": 2, "c": null}]}`,
			want: `{"results":[{"b":1},{"a":2}]}`,
		},
		{
			name: "preserves false / zero / empty string (not null)",
			in:   `{"f": false, "z": 0, "e": "", "n": null}`,
			want: `{"e":"","f":false,"z":0}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var in any
			if err := json.Unmarshal([]byte(tc.in), &in); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			cleaned := stripNulls(in)
			got, err := json.Marshal(cleaned)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s\nwant %s", got, tc.want)
			}
		})
	}
}
