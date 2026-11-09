package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
)

type appFlags struct {
	fixed, wrongIdentity, deleted, generic                atomic.Bool
	expiredPeer                                           atomic.Bool
	requests                                              atomic.Int32
	block                                                 atomic.Bool
	property, redacted, deniedProperty, ambiguousProperty atomic.Bool
	started, cancelled                                    chan struct{}
}

func applicationFixture(t *testing.T) (*Server, *Store, ApplicationConfig, *appFlags) {
	t.Helper()
	s, store := fixture(t)
	flags := &appFlags{started: make(chan struct{}), cancelled: make(chan struct{})}
	ownerToken := strings.Repeat("a", 48)
	peerToken := strings.Repeat("b", 48)
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flags.requests.Add(1)
		if flags.block.Swap(false) {
			close(flags.started)
			<-r.Context().Done()
			close(flags.cancelled)
			return
		}
		if flags.expiredPeer.Load() && r.Header.Get("Authorization") == "Bearer "+peerToken {
			w.WriteHeader(401)
			return
		}
		who := ""
		switch r.Header.Get("Authorization") {
		case "Bearer " + ownerToken:
			who = "first"
		case "Bearer " + peerToken:
			who = "second"
		default:
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/me" {
			if flags.wrongIdentity.Load() {
				who = "same-user"
			}
			fmt.Fprintf(w, `{"account":{"name":%q}}`, who)
			return
		}
		if r.URL.Path == "/objects/a" {
			if who == "second" && flags.property.Load() {
				if flags.deniedProperty.Load() {
					w.WriteHeader(403)
					return
				}
				if flags.ambiguousProperty.Load() {
					fmt.Fprint(w, `{"error":"generic success"}`)
					return
				}
				if flags.redacted.Load() {
					fmt.Fprint(w, `{"identifier":"a","owner":"first","private":"[redacted]"}`)
					return
				}
				if flags.fixed.Load() {
					fmt.Fprint(w, `{"identifier":"a","owner":"first"}`)
					return
				}
			}
			if flags.deleted.Load() && who == "first" {
				w.WriteHeader(404)
				return
			}
			if who == "second" && flags.fixed.Load() {
				w.WriteHeader(404)
				return
			}
			if who == "second" && flags.generic.Load() {
				fmt.Fprint(w, `{"error":"generic 200"}`)
				return
			}
			fmt.Fprint(w, `{"identifier":"a","owner":"first","private":"opaque-proof-one"}`)
			return
		}
		if r.URL.Path == "/objects/b" {
			fmt.Fprint(w, `{"identifier":"b","owner":"second","private":"opaque-proof-two"}`)
			return
		}
		w.WriteHeader(404)
	}))
	t.Cleanup(app.Close)
	hash := func(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }
	owner := application.Read{Account: "owner", Path: "/objects/a", Equals: map[string]string{"/identifier": "a", "/owner": "first"}, Hashes: map[string]string{"/private": hash("opaque-proof-one")}}
	peer := application.Read{Account: "peer", Path: "/objects/b", Equals: map[string]string{"/identifier": "b", "/owner": "second"}, Hashes: map[string]string{"/private": hash("opaque-proof-two")}}
	probe := owner
	probe.Account = "peer"
	c := ApplicationConfig{Tenant: "a", Credentials: map[string]string{"owner": ownerToken, "peer": peerToken}, Plan: application.Plan{Version: "http-object-access/v2", ID: "independent-api", Title: "Configured cross-user object read", Origin: app.URL, ApprovedAddresses: []string{"127.0.0.1"}, SyntheticDataOnly: true, OwnerPointer: "/owner", Identity: []application.Read{{Account: "owner", Path: "/me", Equals: map[string]string{"/account/name": "first"}}, {Account: "peer", Path: "/me", Equals: map[string]string{"/account/name": "second"}}}, Baselines: []application.Read{owner, peer}, Probe: probe, ExpectedAccess: "deny", DenialStatuses: []int{401, 403, 404}}}
	if err := s.RegisterApplication(c); err != nil {
		t.Fatal(err)
	}
	return s, store, c, flags
}

func TestPropertyAccessProfileUsesVerifiedPrivateBaselinesAndFiltering(t *testing.T) {
	for _, test := range []struct{ mode, want string }{{"leaked", "confirmed"}, {"filtered", "no_issue_observed"}, {"redacted", "inconclusive"}, {"denied", "inconclusive"}, {"generic", "inconclusive"}, {"expired", "inconclusive"}, {"deleted_after_probe", "inconclusive"}} {
		t.Run(test.mode, func(t *testing.T) {
			s, _, c, flags := applicationFixture(t)
			c.Plan.ID = "property-api"
			c.Plan.Version = application.PropertyAccessProfile
			c.Plan.ExpectedAccess = "filtered"
			c.Plan.DenialStatuses = nil
			if err := s.RegisterApplication(c); err != nil {
				t.Fatal(err)
			}
			flags.property.Store(true)
			flags.fixed.Store(test.mode == "filtered")
			flags.redacted.Store(test.mode == "redacted")
			flags.deniedProperty.Store(test.mode == "denied")
			flags.ambiguousProperty.Store(test.mode == "generic")
			flags.expiredPeer.Store(test.mode == "expired")
			_, r := applicationEngagement(t, s, c)
			if r.ProcedureID() != "property-api/property-access/v1" {
				t.Fatal("wrong procedure selected")
			}
			for i, read := range r.Application.Reads() {
				if i == 5 && test.mode == "deleted_after_probe" {
					flags.deleted.Store(true)
				}
				executeAppReads(t, s, r, []application.Read{read})
			}
			result, err := s.Finish(r.ID)
			if err != nil || result.Verdict != test.want {
				t.Fatalf("got %+v, %v", result, err)
			}
			if result.Validation.BaselinesRequired != 6 || !strings.Contains(r.ProcedureRemediation(), "property") {
				t.Fatal("property validation/remediation was not selected")
			}
		})
	}
}

func TestPropertyProfileRejectsUnclearOrUnsupportedPolicies(t *testing.T) {
	for _, variant := range []string{"allow", "deny", "denial_status", "nested", "missing_object_identity"} {
		t.Run(variant, func(t *testing.T) {
			s, _, c, _ := applicationFixture(t)
			c.Plan.ID, c.Plan.Version, c.Plan.ExpectedAccess, c.Plan.DenialStatuses = "property-policy", application.PropertyAccessProfile, "filtered", nil
			switch variant {
			case "allow", "deny":
				c.Plan.ExpectedAccess = variant
			case "denial_status":
				c.Plan.DenialStatuses = []int{403}
			case "nested":
				c.Plan.Baselines[0].Hashes = map[string]string{"/nested/private": c.Plan.Probe.Hashes["/private"]}
				c.Plan.Probe.Hashes = c.Plan.Baselines[0].Hashes
			case "missing_object_identity":
				c.Plan.Baselines[0].Equals = map[string]string{"/owner": "first"}
				c.Plan.Probe.Equals = c.Plan.Baselines[0].Equals
			}
			if err := s.RegisterApplication(c); err == nil {
				t.Fatal("ambiguous property policy accepted")
			}
		})
	}
}

func TestLaterResponsesCannotEraseCapturedPropertyExposure(t *testing.T) {
	for _, tc := range []struct {
		ordered bool
		generic bool
	}{{true, false}, {false, false}, {true, true}, {false, true}} {
		t.Run(fmt.Sprintf("ordered_baselines_%t_later_generic_%t", tc.ordered, tc.generic), func(t *testing.T) {
			s, store, c, flags := applicationFixture(t)
			c.Plan.ID, c.Plan.Version, c.Plan.ExpectedAccess, c.Plan.DenialStatuses = "repeated-property", application.PropertyAccessProfile, "filtered", nil
			flags.property.Store(true)
			if err := s.RegisterApplication(c); err != nil {
				t.Fatal(err)
			}
			_, run := applicationEngagement(t, s, c)
			reads := run.Application.Reads()
			if tc.ordered {
				executeAppReads(t, s, run, reads[:4])
			}
			executeAppReads(t, s, run, reads[4:5]) // First response exposes the value.
			snapshot, _ := store.Get(run.ID)
			exposed := snapshot.Captures[len(snapshot.Captures)-1].ID
			flags.fixed.Store(!tc.generic)
			flags.ambiguousProperty.Store(tc.generic)
			if !tc.ordered {
				executeAppReads(t, s, run, reads[:4])
			}
			executeAppReads(t, s, run, reads[4:]) // Later response filters it or is ambiguous.
			result, err := s.Finish(run.ID)
			want := "inconclusive"
			if tc.ordered {
				want = "confirmed"
			}
			if err != nil || result.Verdict != want {
				t.Fatalf("earlier leak erased: %+v %v", result, err)
			}
			found := false
			for _, id := range result.EvidenceIDs {
				if id == exposed {
					found = true
				}
			}
			if !found {
				t.Fatal("earlier exposure disappeared from referenced evidence")
			}
		})
	}
}

func applicationEngagement(t *testing.T, s *Server, c ApplicationConfig) (Engagement, Run) {
	t.Helper()
	data, _ := json.Marshal(EngagementInput{Name: "Configured owned application", AuthorizationRef: "owned-lab/application-test", Application: c.Plan.ID})
	w := request(s, "POST", "/v1/engagements", "operator-a", string(data))
	if w.Code != 201 {
		t.Fatalf("engagement: %s", w.Body)
	}
	var e Engagement
	_ = json.Unmarshal(w.Body.Bytes(), &e)
	approveEngagement(t, s, e)
	w = request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", `{"idempotency_key":"`+NewStartKey()+`","assessment":"`+e.Application.ProcedureID()+`"}`)
	if w.Code != 201 {
		t.Fatalf("run: %s", w.Body)
	}
	var r Run
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	if err := s.Store.Update(r.ID, func(r *Run) error { r.State = Running; return nil }); err != nil {
		t.Fatal(err)
	}
	return e, r
}

func executeAppReads(t *testing.T, s *Server, r Run, reads []application.Read) {
	t.Helper()
	for _, read := range reads {
		if _, err := s.Execute(context.Background(), r.ID, Request{r.Target + read.Path, read.Account}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConfiguredApplicationProofAndNegativeCases(t *testing.T) {
	for _, tc := range []struct{ name, want string }{{"vulnerable", "confirmed"}, {"fixed", "no_issue_observed"}, {"wrong_identity", "inconclusive"}, {"deleted_owner", "inconclusive"}, {"generic_body", "inconclusive"}, {"missing_identity", "inconclusive"}, {"expired_peer", "inconclusive"}} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, c, flags := applicationFixture(t)
			_, r := applicationEngagement(t, s, c)
			flags.fixed.Store(tc.name == "fixed" || tc.name == "deleted_owner" || tc.name == "expired_peer")
			flags.wrongIdentity.Store(tc.name == "wrong_identity")
			flags.generic.Store(tc.name == "generic_body")
			reads := r.Application.Reads()
			if tc.name == "missing_identity" {
				reads = reads[2:]
			}
			for i, read := range reads {
				if tc.name == "deleted_owner" && i == 5 {
					flags.deleted.Store(true)
				}
				if tc.name == "expired_peer" && i == 4 {
					flags.expiredPeer.Store(true)
				}
				executeAppReads(t, s, r, []application.Read{read})
			}
			result, err := s.Finish(r.ID)
			if err != nil || result.Verdict != tc.want {
				t.Fatalf("%s: %+v %v", tc.name, result, err)
			}
		})
	}
}

func TestApplicationScopeCredentialsAndConfigurationDrift(t *testing.T) {
	s, store, c, flags := applicationFixture(t)
	e, r := applicationEngagement(t, s, c)
	for _, in := range []Request{{r.Target + "/debug", "owner"}, {r.Target + "/objects/b", "owner"}, {r.Target + "/objects/a", "anonymous"}, {"http://not-approved.invalid/objects/a", "peer"}} {
		if _, err := s.Execute(context.Background(), r.ID, in); err == nil {
			t.Fatal("unapproved request accepted")
		}
	}
	if flags.requests.Load() != 0 {
		t.Fatal("denied request reached application")
	}
	data, _ := json.Marshal(EngagementInput{Name: "other tenant", AuthorizationRef: "lab", Application: c.Plan.ID})
	if w := request(s, "POST", "/v1/engagements", "operator-b", string(data)); w.Code != 400 {
		t.Fatal("other tenant acquired application")
	}
	token, _ := store.Credential(r.ID)
	if w := request(s, "GET", "/v1/applications", token, ""); w.Code != 401 {
		t.Fatal("worker read registry")
	}
	encoded, _ := json.Marshal(r)
	for _, secret := range c.Credentials {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("credential in worker policy")
		}
	}
	if w := request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", `{"idempotency_key":"`+NewStartKey()+`","assessment":"`+r.Application.ProcedureID()+`","execution":"live"}`); w.Code != 409 {
		t.Fatal("unqualified model path enabled")
	}
	// Model a process restart with a changed credential. The old durable run
	// must not silently use it, even under the same public application ID.
	newServer := NewServer(store, s.Targets, s.AdminTokens)
	c.Credentials["peer"] = strings.Repeat("c", 48)
	if err := newServer.RegisterApplication(c); err != nil {
		t.Fatal(err)
	}
	if _, err := newServer.Execute(context.Background(), r.ID, Request{r.Target + "/me", "owner"}); err == nil {
		t.Fatal("changed config reused old run")
	}
	if flags.requests.Load() != 0 {
		t.Fatal("configuration drift sent traffic")
	}
}

func TestRepeatedObjectProbePreservesCapturedExposure(t *testing.T) {
	for _, tc := range []struct {
		name, later, rechecks, want string
		latePrerequisites           bool
		repeatAfterRechecks         bool
	}{
		{name: "later_denial", later: "denied", want: "confirmed"},
		{name: "later_generic_200", later: "generic", want: "confirmed"},
		{name: "later_expired_probe", later: "expired", want: "confirmed"},
		{name: "denial_after_complete_evidence", later: "denied", repeatAfterRechecks: true, want: "confirmed"},
		{name: "generic_after_complete_evidence", later: "generic", repeatAfterRechecks: true, want: "confirmed"},
		{name: "prerequisites_only_after_leak", later: "denied", latePrerequisites: true, want: "inconclusive"},
		{name: "missing_rechecks", later: "denied", rechecks: "none", want: "inconclusive"},
		{name: "missing_owner_recheck", later: "denied", rechecks: "peer_only", want: "inconclusive"},
		{name: "missing_peer_recheck", later: "denied", rechecks: "owner_only", want: "inconclusive"},
		{name: "expired_peer_recheck", later: "denied", rechecks: "expired_peer", want: "inconclusive"},
		{name: "deleted_owner_recheck", later: "denied", rechecks: "deleted_owner", want: "inconclusive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, store, c, flags := applicationFixture(t)
			e, run := applicationEngagement(t, s, c)
			reads := run.Application.Reads()
			if !tc.latePrerequisites {
				executeAppReads(t, s, run, reads[:4])
			}
			executeAppReads(t, s, run, reads[4:5])
			snapshot, err := store.Get(run.ID)
			if err != nil {
				t.Fatal(err)
			}
			exposed := snapshot.Captures[len(snapshot.Captures)-1].ID
			if tc.latePrerequisites {
				executeAppReads(t, s, run, reads[:4])
			}
			if tc.repeatAfterRechecks {
				executeAppReads(t, s, run, reads[5:])
			}
			flags.fixed.Store(tc.later == "denied")
			flags.generic.Store(tc.later == "generic")
			flags.expiredPeer.Store(tc.later == "expired")
			executeAppReads(t, s, run, reads[4:5])
			// Expiry on the repeated probe cannot substitute for the required
			// successful peer recheck. Restore it only when that recheck passes.
			flags.expiredPeer.Store(tc.rechecks == "expired_peer")
			flags.deleted.Store(tc.rechecks == "deleted_owner")
			if !tc.repeatAfterRechecks {
				if tc.rechecks != "none" && tc.rechecks != "peer_only" {
					executeAppReads(t, s, run, reads[5:6])
				}
				if tc.rechecks != "none" && tc.rechecks != "owner_only" {
					executeAppReads(t, s, run, reads[6:7])
				}
			}
			result, err := s.Finish(run.ID)
			if err != nil || result.Verdict != tc.want {
				t.Fatalf("got %+v, %v; want %s", result, err, tc.want)
			}
			if !slices.Contains(result.EvidenceIDs, exposed) {
				t.Fatal("earlier exposure disappeared from referenced evidence")
			}
			snapshot, err = store.Get(run.ID)
			if err != nil || snapshot.State != Completed || snapshot.Result == nil || snapshot.Result.Verdict != tc.want {
				t.Fatalf("result was not persisted: %+v, %v", snapshot.Result, err)
			}
			if snapshot.RequestCount > snapshot.MaxRequests || snapshot.MaxRequests != 8 {
				t.Fatal("repeated-probe scenario exceeded the approved request budget")
			}
			bundle, err := store.ExportEngagement("a", e.ID)
			if err != nil {
				t.Fatal(err)
			}
			if tc.want != "confirmed" {
				if len(bundle.Candidates) != 0 {
					t.Fatal("incomplete evidence produced a finding candidate")
				}
				return
			}
			if result.Validation.BaselinesVerified != 6 || !result.Validation.ProbeCaptured || len(result.EvidenceIDs) != 7 {
				t.Fatalf("confirmation lacks its ordered prerequisites/probe: %+v", result)
			}
			if len(bundle.Candidates) != 1 || bundle.View.Checklist[0].Verdict != "confirmed" {
				t.Fatal("earlier exposure disappeared from engagement progress/export")
			}
			candidate := bundle.Candidates[0]
			if candidate.Status != "needs_human_review" || candidate.Review != nil || len(bundle.Findings) != 0 {
				t.Fatal("validator result invented a human review or rating")
			}
			if len(candidate.EvidenceIDs) != len(result.EvidenceIDs) {
				t.Fatal("candidate lost supporting evidence")
			}
			foundExposure := false
			for _, artifact := range bundle.Artifacts {
				if slices.Contains(candidate.EvidenceIDs, artifact.ID) && !slices.Contains(result.EvidenceIDs, artifact.CaptureID) {
					t.Fatal("candidate references evidence outside the confirmed result")
				}
				if artifact.CaptureID == exposed && slices.Contains(candidate.EvidenceIDs, artifact.ID) {
					foundExposure = true
				}
			}
			if !foundExposure {
				t.Fatal("candidate lost the original exposure artifact")
			}
		})
	}
}

func TestObjectExposurePreservationRequiresProtectedProof(t *testing.T) {
	s, store, c, flags := applicationFixture(t)
	_, run := applicationEngagement(t, s, c)
	reads := run.Application.Reads()
	executeAppReads(t, s, run, reads[:5])
	flags.fixed.Store(true)
	executeAppReads(t, s, run, reads[4:])
	snapshot, err := store.Get(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*Capture)
	}{
		{"generic_200", func(c *Capture) { c.Body = `{"message":"ok"}` }},
		{"wrong_proof", func(c *Capture) { c.Body = strings.ReplaceAll(c.Body, "opaque-proof-one", "unrelated-proof") }},
		{"wrong_owner", func(c *Capture) { c.Body = strings.ReplaceAll(c.Body, "first", "second") }},
		{"wrong_object", func(c *Capture) { c.Body = strings.ReplaceAll(c.Body, `"a"`, `"b"`) }},
		{"blocked", func(c *Capture) { c.Blocked = true }},
		{"interrupted", func(c *Capture) { c.Error = "response interrupted" }},
		{"unsuccessful_status", func(c *Capture) { c.Status = 401 }},
		{"wrong_account", func(c *Capture) { c.Account = "owner" }},
		{"wrong_destination", func(c *Capture) { c.URL = "http://not-approved.invalid/objects/a" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Change only this copy of captured evidence, never the stored run.
			r := snapshot
			r.Captures = slices.Clone(snapshot.Captures)
			tc.change(&r.Captures[4])
			if result := Validate(r); result.Verdict != "no_issue_observed" {
				t.Fatalf("invalid earlier capture overrode a valid denial: %+v", result)
			}
		})
	}
}

func TestRepeatedIntentionalSharingIsNotExposure(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		generic    bool
	}{
		{name: "shared_object", want: "no_issue_observed"},
		{name: "later_generic_response", generic: true, want: "inconclusive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, c, flags := applicationFixture(t)
			c.Plan.ID, c.Plan.ExpectedAccess = "shared-api", "allow"
			if err := s.RegisterApplication(c); err != nil {
				t.Fatal(err)
			}
			_, run := applicationEngagement(t, s, c)
			reads := run.Application.Reads()
			executeAppReads(t, s, run, reads[:5])
			flags.generic.Store(tc.generic)
			executeAppReads(t, s, run, reads[4:])
			result, err := s.Finish(run.ID)
			if err != nil || result.Verdict != tc.want {
				t.Fatalf("incorrect sharing result: %+v, %v; want %s", result, err, tc.want)
			}
		})
	}
}

func TestApplicationStopCancelsBrokerWithoutKubernetes(t *testing.T) {
	s, _, c, flags := applicationFixture(t)
	e, r := applicationEngagement(t, s, c)
	flags.block.Store(true)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = s.Execute(context.Background(), r.ID, Request{r.Target + r.Application.Identity[0].Path, "owner"})
	}()
	select {
	case <-flags.started:
	case <-time.After(2 * time.Second):
		t.Fatal("application request did not start")
	}
	if _, err := s.StopEngagement("a", e.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-flags.cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("target did not observe cancellation")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("broker did not drain")
	}
	if _, err := s.Execute(context.Background(), r.ID, Request{r.Target + r.Application.Probe.Path, "peer"}); err == nil {
		t.Fatal("application request accepted after Stop")
	}
}
