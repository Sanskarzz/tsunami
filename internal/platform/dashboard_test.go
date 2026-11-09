package platform

import (
	"encoding/json"
	"strings"
	"testing"

	bolt "go.etcd.io/bbolt"
)

func TestDashboardSummariesAreTenantBoundStoredObservations(t *testing.T) {
	s, store, config, flags := applicationFixture(t)
	e, run := applicationEngagement(t, s, config)
	if err := store.Update(run.ID, func(r *Run) error {
		r.State = Completed
		r.Result = &Result{Verdict: "inconclusive", Reason: "private-reason-canary", EvidenceIDs: []string{}}
		r.Captures = []Capture{{ID: randomID(), Body: "private-body-canary"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before, _ := store.Get(run.ID)
	scopeBefore, _ := store.GetEngagement("a", e.ID)
	w := request(s, "GET", "/v1/engagements", "operator-a", "")
	var items []EngagementSummary
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &items) != nil || len(items) != 1 {
		t.Fatalf("dashboard: %d %s", w.Code, w.Body)
	}
	i := items[0]
	if i.State != EngagementActive || i.LatestAttemptStatus != "recorded" || i.LatestAttempt == nil || i.LatestAttempt.ID != run.ID || i.LatestAttempt.State != Completed || i.LatestAttempt.Verdict != "inconclusive" || !i.LatestAttempt.Deadline.Equal(run.Deadline) || i.ObservedAt.IsZero() {
		t.Fatalf("incorrect stored observation: %+v", i)
	}
	for _, secret := range []string{"private-reason-canary", "private-body-canary", config.Credentials["owner"], config.Credentials["peer"]} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatal("dashboard exposed evidence/credential")
		}
	}
	after, _ := store.Get(run.ID)
	scopeAfter, _ := store.GetEngagement("a", e.ID)
	a, _ := json.Marshal([]any{before, scopeBefore})
	b, _ := json.Marshal([]any{after, scopeAfter})
	if string(a) != string(b) || flags.requests.Load() != 0 {
		t.Fatal("dashboard read mutated scope/run or contacted target")
	}
	legacy, err := store.ListEngagements("a")
	if err != nil || len(legacy) != 1 || legacy[0].ScopeDigest != scopeBefore.ScopeDigest {
		t.Fatal("existing store list changed")
	}
	for _, token := range []string{"", "operator-b"} {
		w = request(s, "GET", "/v1/engagements", token, "")
		if token == "" && w.Code != 401 {
			t.Fatal("unauthenticated dashboard")
		}
		if token == "operator-b" && (w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]") {
			t.Fatal("other tenant dashboard leaked data")
		}
	}
	runToken, _ := store.Credential(run.ID)
	if w = request(s, "GET", "/v1/engagements", runToken, ""); w.Code != 401 {
		t.Fatal("worker read dashboard")
	}
}

func TestDashboardMissingChildNeverMeansNoAttempt(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	items, err := store.ListEngagementSummaries("a")
	if err != nil || len(items) != 1 || items[0].LatestAttemptStatus != "not_started" || items[0].LatestAttempt != nil {
		t.Fatal("draft incorrectly observed")
	}
	foreign := running(t, s, "b")
	for _, id := range []string{randomID(), foreign.ID} {
		if err := store.db.Update(func(tx *bolt.Tx) error {
			v, err := getEngagement(tx, "a", e.ID)
			if err != nil {
				return err
			}
			v.RunIDs = []string{id}
			return putEngagement(tx, v)
		}); err != nil {
			t.Fatal(err)
		}
		items, err = store.ListEngagementSummaries("a")
		if err != nil || len(items) != 1 || items[0].LatestAttemptStatus != "unavailable" || items[0].LatestAttempt != nil {
			t.Fatal("missing or foreign child was promoted to attempt observation")
		}
	}
}
