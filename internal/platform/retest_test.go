package platform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	bolt "go.etcd.io/bbolt"
)

func retestDraft(t *testing.T, s *Server, source, app string) Engagement {
	t.Helper()
	data, _ := json.Marshal(EngagementInput{SourceEngagementID: source, Application: app, Name: "Fresh owned retest", AuthorizationRef: "owned-lab/new-permission"})
	w := request(s, "POST", "/v1/engagements", "operator-a", string(data))
	var e Engagement
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &e) != nil {
		t.Fatalf("retest draft: %d %s", w.Code, w.Body)
	}
	return e
}

func retestRead(t *testing.T, s *Server, id string) RetestComparison {
	t.Helper()
	w := request(s, "GET", "/v1/engagements/"+id+"/retest", "operator-a", "")
	var c RetestComparison
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || json.Unmarshal(w.Body.Bytes(), &c) != nil {
		t.Fatalf("retest comparison: %d %s", w.Code, w.Body)
	}
	return c
}

func startRetest(t *testing.T, s *Server, e Engagement) Run {
	t.Helper()
	w := request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", `{"assessment":"`+e.Application.ProcedureID()+`","execution":"scripted"}`)
	var r Run
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &r) != nil {
		t.Fatalf("start retest: %d %s", w.Code, w.Body)
	}
	if err := s.Store.Update(r.ID, func(r *Run) error { r.State = Running; return nil }); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRetestFreshOriginUsesOwnCredentialsApprovalAndPreservesOriginal(t *testing.T) {
	s, store, config, flags := applicationFixture(t)
	source, run := applicationEngagement(t, s, config)
	executeAppReads(t, s, run, run.Application.Reads())
	if result, err := s.Finish(run.ID); err != nil || result.Verdict != "confirmed" {
		t.Fatalf("source result: %+v %v", result, err)
	}
	before, _ := store.Get(run.ID)
	sourceBefore, _ := store.GetEngagement("a", source.ID)
	bundleBefore, _ := store.ExportEngagement("a", source.ID)
	var reads, wrongCredential atomic.Int32
	ownerToken, peerToken := strings.Repeat("x", 48), strings.Repeat("y", 48)
	fixed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		who := ""
		switch r.Header.Get("Authorization") {
		case "Bearer " + ownerToken:
			who = "first"
		case "Bearer " + peerToken:
			who = "second"
		default:
			wrongCredential.Add(1)
			w.WriteHeader(401)
			return
		}
		switch r.URL.Path {
		case "/me":
			fmt.Fprintf(w, `{"account":{"name":%q}}`, who)
		case "/objects/a":
			if who == "second" {
				w.WriteHeader(403)
				return
			}
			fmt.Fprint(w, `{"identifier":"a","owner":"first","private":"opaque-proof-one"}`)
		case "/objects/b":
			fmt.Fprint(w, `{"identifier":"b","owner":"second","private":"opaque-proof-two"}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer fixed.Close()
	config.Plan.ID, config.Plan.Origin = "owned-fixed", fixed.URL
	config.Credentials = map[string]string{"owner": ownerToken, "peer": peerToken}
	if err := s.RegisterApplication(config); err != nil {
		t.Fatal(err)
	}
	e := retestDraft(t, s, source.ID, config.Plan.ID)
	if e.State != EngagementDraft || e.ApprovedAt != nil || e.ApprovedDigest != "" || len(e.RunIDs) != 0 || e.SourceEngagementID != source.ID || e.ScopeDigest == source.ScopeDigest || e.Target == source.Target {
		t.Fatal("retest inherited authority or scope")
	}
	if w := request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", `{"assessment":"`+e.Application.ProcedureID()+`"}`); w.Code != 409 {
		t.Fatalf("unapproved retest started: %d", w.Code)
	}
	c := retestRead(t, s, e.ID)
	if c.Compatibility != "comparable" || c.Source.LatestAttempt.Result.Verdict != "confirmed" || c.Retest.AttemptStatus != "not_started" || c.Retest.LatestAttempt != nil {
		t.Fatalf("draft comparison invented a result: %+v", c)
	}
	changed := map[string]bool{}
	for _, d := range c.Differences {
		changed[d.Field] = true
	}
	for _, field := range []string{"Origin", "Application", "Definition digest", "Credential binding digest", "Scope digest"} {
		if !changed[field] {
			t.Fatalf("missing changed binding: %s", field)
		}
	}
	if reads.Load() != 0 || flags.requests.Load() != 7 {
		t.Fatal("draft/read sent traffic")
	}
	approveEngagement(t, s, e)
	r := startRetest(t, s, e)
	if r.Target != fixed.URL || r.Application.CredentialSetID == run.Application.CredentialSetID {
		t.Fatal("source access binding inherited")
	}
	executeAppReads(t, s, r, r.Application.Reads())
	if result, err := s.Finish(r.ID); err != nil || result.Verdict != "no_issue_observed" {
		t.Fatalf("fixed result: %+v %v", result, err)
	}
	c = retestRead(t, s, e.ID)
	if c.Retest.LatestAttempt.Result.Verdict != "no_issue_observed" || c.Source.LatestAttempt.Result.Verdict != "confirmed" || len(c.Retest.LatestAttempt.EvidenceIDs) != 7 || len(c.Source.LatestAttempt.EvidenceIDs) != 7 {
		t.Fatal("comparison lost result or evidence")
	}
	freshBundle, _ := store.ExportEngagement("a", e.ID)
	for _, id := range c.Retest.LatestAttempt.EvidenceIDs {
		found := false
		for _, artifact := range freshBundle.Artifacts {
			found = found || artifact.ID == id
		}
		if !found {
			t.Fatal("comparison artifact absent from export")
		}
	}
	body, _ := json.Marshal(c)
	for _, secret := range []string{ownerToken, peerToken, "opaque-proof-one", "opaque-proof-two", `"data_base64"`, `"body"`} {
		if bytes.Contains(body, []byte(secret)) {
			t.Fatalf("comparison leaked %q", secret)
		}
	}
	after, _ := store.Get(run.ID)
	sourceAfter, _ := store.GetEngagement("a", source.ID)
	bundleAfter, _ := store.ExportEngagement("a", source.ID)
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(sourceBefore, sourceAfter) || !reflect.DeepEqual(bundleBefore.Candidates, bundleAfter.Candidates) || !reflect.DeepEqual(bundleBefore.Artifacts, bundleAfter.Artifacts) || reads.Load() != 7 || flags.requests.Load() != 7 || wrongCredential.Load() != 0 {
		t.Fatal("retest/read altered original history, replayed traffic or sent credentials to the wrong origin")
	}
	path := store.db.Path()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	s.Store = reopened
	if retestRead(t, s, e.ID).Retest.Engagement.SourceEngagementID != source.ID {
		t.Fatal("source link lost after restart")
	}
}

func TestRetestTenantBoundariesTamperingAndLegacyDigest(t *testing.T) {
	s, store, config, flags := applicationFixture(t)
	source, run := applicationEngagement(t, s, config)
	foreign := config
	foreign.Tenant = "b"
	if err := s.RegisterApplication(foreign); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(EngagementInput{Name: "Foreign source", AuthorizationRef: "owned-b", Application: config.Plan.ID})
	w := request(s, "POST", "/v1/engagements", "operator-b", string(data))
	var other Engagement
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &other) != nil {
		t.Fatal("foreign fixture unavailable")
	}
	initial, _ := store.ListEngagements("a")
	for _, id := range []string{other.ID, strings.Repeat("f", 32), "../escape", "INVALID"} {
		data, _ := json.Marshal(EngagementInput{Name: "Denied link", AuthorizationRef: "owned-a", Application: config.Plan.ID, SourceEngagementID: id})
		if w := request(s, "POST", "/v1/engagements", "operator-a", string(data)); w.Code != 400 {
			t.Fatalf("invalid source accepted: %s %d", id, w.Code)
		}
	}
	after, _ := store.ListEngagements("a")
	if len(initial) != len(after) {
		t.Fatal("rejected link left a draft")
	}
	e := retestDraft(t, s, source.ID, config.Plan.ID)
	token, _ := store.Credential(run.ID)
	for _, auth := range []string{"operator-b", token, ""} {
		w := request(s, "GET", "/v1/engagements/"+e.ID+"/retest", auth, "")
		if w.Code != 401 && w.Code != 404 {
			t.Fatalf("comparison tenant bypass: %d", w.Code)
		}
	}
	for _, tc := range []struct {
		method, suffix string
		code           int
	}{{"POST", "/retest", 405}, {"GET", "/retest?source=" + other.ID, 400}} {
		if w := request(s, tc.method, "/v1/engagements/"+e.ID+tc.suffix, "operator-a", `{}`); w.Code != tc.code {
			t.Fatalf("comparison contract: %d want %d", w.Code, tc.code)
		}
	}
	if w := request(s, "GET", "/v1/engagements/"+source.ID+"/retest", "operator-a", ""); w.Code != 404 {
		t.Fatal("invented legacy retest link")
	}
	// Empty additive fields must preserve the exact pre-D8 digest contract.
	legacy := source
	legacy.SourceEngagementID = ""
	if legacy.digest() != source.ScopeDigest {
		t.Fatal("legacy approved scope changed")
	}
	if err := store.db.Update(func(tx *bolt.Tx) error {
		x, _ := getEngagement(tx, "a", e.ID)
		x.SourceEngagementID = other.ID // does not update approved digest
		return putEngagement(tx, x)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.ApproveEngagement("a", e.ID, e.ScopeDigest); err == nil {
		t.Fatal("changed source link approved")
	}
	if w := request(s, "GET", "/v1/engagements/"+e.ID+"/retest", "operator-a", ""); w.Code != 404 {
		t.Fatal("foreign linked history exposed")
	}
	if _, err := s.StopEngagement("a", e.ID); err != nil || flags.requests.Load() != 0 {
		t.Fatal("invalid comparison blocked Stop or sent traffic")
	}
}

func TestRetestStoppedExpiredIncompleteAndIncompatibleSources(t *testing.T) {
	for _, mode := range []string{"stopped", "expired", "incompatible_profile", "changed_rule", "incomplete_retest", "stopped_retest", "failed_latest_retest"} {
		t.Run(mode, func(t *testing.T) {
			s, store, config, flags := applicationFixture(t)
			source, sourceRun := applicationEngagement(t, s, config)
			if mode == "stopped" {
				_, _ = s.StopEngagement("a", source.ID)
			}
			if mode == "expired" {
				if err := store.db.Update(func(tx *bolt.Tx) error {
					e, _ := getEngagement(tx, "a", source.ID)
					e.ExpiresAt = time.Now().UTC().Add(-time.Minute)
					e.ScopeDigest = e.digest()
					e.ApprovedDigest = e.ScopeDigest
					return putEngagement(tx, e)
				}); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "incompatible_profile" || mode == "changed_rule" {
				config.Plan.ID = "changed-check"
				config.Plan.ExpectedAccess = "allow"
				if mode == "incompatible_profile" {
					config.Plan.Version = application.PropertyAccessProfile
					config.Plan.ExpectedAccess = "filtered"
					config.Plan.DenialStatuses = nil
				}
				if err := s.RegisterApplication(config); err != nil {
					t.Fatal(err)
				}
			}
			e := retestDraft(t, s, source.ID, config.Plan.ID)
			c := retestRead(t, s, e.ID)
			if c.Source.LatestAttempt.Result != nil || c.Retest.LatestAttempt != nil {
				t.Fatal("unfinished source invented a result")
			}
			if strings.HasPrefix(mode, "incompatible") || mode == "changed_rule" {
				if c.Compatibility != "incompatible" {
					t.Fatal("incompatible checks called comparable")
				}
			}
			approveEngagement(t, s, e)
			r := startRetest(t, s, e)
			if mode == "incomplete_retest" {
				executeAppReads(t, s, r, r.Application.Reads()[:5])
				if _, err := s.Finish(r.ID); err != nil {
					t.Fatal(err)
				}
			} else if mode == "stopped_retest" {
				_, _ = s.StopEngagement("a", e.ID)
			} else if mode == "failed_latest_retest" {
				flags.fixed.Store(true)
				executeAppReads(t, s, r, r.Application.Reads())
				if _, err := s.Finish(r.ID); err != nil {
					t.Fatal(err)
				}
				last := startRetest(t, s, e)
				if err := store.Update(last.ID, func(r *Run) error { r.State = Failed; return nil }); err != nil {
					t.Fatal(err)
				}
			}
			c = retestRead(t, s, e.ID)
			if mode == "incomplete_retest" {
				if c.Retest.LatestAttempt.Result == nil || c.Retest.LatestAttempt.Result.Verdict != "inconclusive" {
					t.Fatal("incomplete retest called clean")
				}
			} else if c.Retest.LatestAttempt.Result != nil {
				t.Fatal("unfinished/stopped/failed latest attempt called clean")
			}
			storedSource, _ := store.Get(sourceRun.ID)
			if storedSource.Result != nil {
				t.Fatal("retest mutated source")
			}
		})
	}
}
