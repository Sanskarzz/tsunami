package onboarding

import (
	"encoding/json"
	"testing"

	"github.com/Sanskarzz/tsunami/internal/binding"
)

func TestReadinessAsksOnlyForMissingFactsAndNeverApproves(t *testing.T) {
	s := Session{ID: "abc", Details: Details{APIOrigin: "https://api.example.test", ApprovedAddresses: []string{"1.1.1.1"}, AuthMethod: "bearer", IdentityPath: "/me", OwnerPath: "/objects/a", PeerPath: "/objects/b", ExpectedAccess: "deny", ProtectedFields: []string{"/secret"}, SyntheticOnly: true}}
	s.Advance()
	if s.Question.Field != "access" {
		t.Fatal("asked for an already supplied fact")
	}
	n := len(s.Messages)
	s.Advance()
	if len(s.Messages) != n {
		t.Fatal("repeated pending question")
	}
	s.AccessReady = true
	s.AccessOrigin = s.Details.APIOrigin
	s.Advance()
	if s.State != "ready_for_validation" || s.Plan != nil || s.EngagementID != "" {
		t.Fatal("metadata approved assessment or invented observations")
	}
	s.Details.AuthMethod = "sso"
	s.Advance()
	if s.State != "unsupported" {
		t.Fatal("unsupported auth considered ready")
	}
}
func TestInferRequiresUnambiguousActualBaselines(t *testing.T) {
	doc := func(s string) binding.Document { return binding.Document{Status: 200, Body: json.RawMessage(s)} }
	s := Session{ID: "a123", Details: Details{Name: "Owned API", APIOrigin: "https://api.example.test", ApprovedAddresses: []string{"1.1.1.1"}, IdentityPath: "/me", OwnerPath: "/objects/a", PeerPath: "/objects/b", ExpectedAccess: "deny", ProtectedFields: []string{"/secret"}}}
	docs := [4]binding.Document{doc(`{"id":"user-a"}`), doc(`{"id":"user-b"}`), doc(`{"id":"a","owner":"user-a","secret":"synthetic-private-proof-aaaaaa"}`), doc(`{"id":"b","owner":"user-b","secret":"synthetic-private-proof-bbbbbb"}`)}
	p, e := Infer(s, docs)
	if e != nil || p.Identity[0].Equals["/id"] != "user-a" || p.CredentialSetID != "" {
		t.Fatal("legitimate bindings not derived")
	}
	for _, bad := range []string{`{"id":"user-a","other_id":"user-a"}`, `{"id":123}`, `{"id":"user-a","id":"user-b"}`, `{"status":"ok"}`} {
		changed := docs
		changed[0] = doc(bad)
		if bad == `{"id":"user-a","other_id":"user-a"}` {
			changed[1] = doc(`{"id":"user-b","other_id":"user-b"}`)
		}
		if _, e := Infer(s, changed); e == nil {
			t.Fatal("ambiguous/unsupported identity accepted")
		}
	}
}
