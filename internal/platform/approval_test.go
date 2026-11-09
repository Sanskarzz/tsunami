package platform

import (
	"encoding/json"
	"strings"
	"testing"

	bolt "go.etcd.io/bbolt"
)

func TestApprovalRequiresExactReviewedDigest(t *testing.T) {
	for _, tc := range []struct {
		name string
		body func(Engagement, Engagement) string
		code int
	}{
		{"missing", func(_, _ Engagement) string { return `{}` }, 409},
		{"null", func(_, _ Engagement) string { return `null` }, 409},
		{"empty", func(_, _ Engagement) string { return `{"scope_digest":""}` }, 409},
		{"wrong", func(_, _ Engagement) string { return `{"scope_digest":"` + strings.Repeat("0", 64) + `"}` }, 409},
		{"other scope", func(_, other Engagement) string { return `{"scope_digest":"` + other.ScopeDigest + `"}` }, 409},
		{"wrong type", func(_, _ Engagement) string { return `{"scope_digest":7}` }, 400},
		{"authority field", func(e, _ Engagement) string {
			return `{"scope_digest":"` + e.ScopeDigest + `","target":"http://unapproved.invalid"}`
		}, 400},
		{"reviewed", func(e, _ Engagement) string { return `{"scope_digest":"` + e.ScopeDigest + `"}` }, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, store := fixture(t)
			e := engagementFixture(t, s, "opaque-ownership/v1")
			other := engagementFixture(t, s, "role-boundary/v1")
			w := request(s, "POST", "/v1/engagements/"+e.ID+"/approve", "operator-a", tc.body(e, other))
			if w.Code != tc.code {
				t.Fatalf("approval status = %d, want %d", w.Code, tc.code)
			}
			stored, err := store.GetEngagement("a", e.ID)
			if err != nil || stored.ScopeDigest != e.ScopeDigest || len(stored.RunIDs) != 0 {
				t.Fatal("approval changed scope or dispatched work")
			}
			if tc.code == 200 {
				if stored.State != EngagementActive || stored.ApprovedDigest != e.ScopeDigest {
					t.Fatal("reviewed scope was not approved")
				}
			} else {
				if stored.State != EngagementDraft || stored.ApprovedDigest != "" || stored.ApprovedAt != nil {
					t.Fatal("invalid approval changed permission")
				}
				if _, err := store.CreateEngagementRun("a", e.ID, e.Items[0].Procedure, nil); err == nil {
					t.Fatal("rejected approval permitted Start")
				}
			}
		})
	}
}

func TestApprovalRejectsPreviouslyReviewedScope(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	previous := e.ScopeDigest
	// Simulate a future scope-revision path between review and approval. The
	// comparison must use the scope read inside the approval transaction.
	if err := store.db.Update(func(tx *bolt.Tx) error {
		e.AuthorizationRef = "owned-lab/revised-permission"
		e.ScopeDigest = e.digest()
		return putEngagement(tx, e)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.ApproveEngagement("a", e.ID, previous); err == nil {
		t.Fatal("stale client review approved revised scope")
	}
	if err := store.ApproveEngagement("a", e.ID, e.ScopeDigest); err != nil {
		t.Fatal(err)
	}
}

func TestReviewedDigestDoesNotAuthorizeAnotherPrincipal(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	r := running(t, s, "a")
	token, err := store.Credential(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"scope_digest": e.ScopeDigest})
	for _, credential := range []string{"operator-b", token} {
		w := request(s, "POST", "/v1/engagements/"+e.ID+"/approve", credential, string(body))
		if w.Code != 401 && w.Code != 404 {
			t.Fatal("knowing the reviewed digest bypassed tenant/operator authorization")
		}
	}
}
