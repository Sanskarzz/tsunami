package platform

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunListIsPagedMetadataAndTenantBound(t *testing.T) {
	s, store := fixture(t)
	for _, tenant := range []string{"a", "a", "b", "a"} {
		r, err := store.Create(tenant, "vulnerable", s.Targets["vulnerable"].URL, "list test")
		if err != nil {
			t.Fatal(err)
		}
		if err = store.Update(r.ID, func(r *Run) error {
			r.Captures = []Capture{{Body: strings.Repeat("capture-body-canary", 10000)}}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	seen := map[string]bool{}
	cursor := ""
	for range 4 {
		w := request(s, "GET", "/v1/runs?limit=1&cursor="+cursor, "operator-a", "")
		if w.Code != 200 || strings.Contains(w.Body.String(), "captures") || w.Body.Len() > 2048 {
			t.Fatalf("not bounded metadata: %d", w.Code)
		}
		var page RunPage
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		for _, item := range page.Items {
			if item.Tenant != "a" || seen[item.ID] {
				t.Fatal("tenant or cursor isolation failed")
			}
			seen[item.ID] = true
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if len(seen) != 3 {
		t.Fatal("pagination lost permitted runs")
	}
	for _, query := range []string{"limit=0", "limit=101", "limit=bad", "cursor=../other", "limit=1&limit=2", "unknown=1"} {
		if w := request(s, "GET", "/v1/runs?"+query, "operator-a", ""); w.Code != 400 {
			t.Fatalf("invalid pagination accepted: %s", query)
		}
	}
	w := request(s, "GET", "/v1/runs?limit=100", "operator-b", "")
	var other RunPage
	_ = json.Unmarshal(w.Body.Bytes(), &other)
	if w.Code != 200 || len(other.Items) != 1 || other.Items[0].Tenant != "b" {
		t.Fatal("other tenant listing failed")
	}
	if w := request(s, "GET", "/v1/runs", "unknown-token", ""); w.Code != 401 {
		t.Fatal("unauthenticated history access")
	}
}
