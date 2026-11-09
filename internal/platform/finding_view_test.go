package platform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Sanskarzz/tsunami/internal/application"
)

func findingSnapshot(t *testing.T, s *Server, id string) FindingsSnapshot {
	t.Helper()
	w := request(s, "GET", "/v1/engagements/"+id+"/findings", "operator-a", "")
	var snapshot FindingsSnapshot
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &snapshot) != nil {
		t.Fatalf("finding snapshot: %d %s", w.Code, w.Body)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("finding response may be cached")
	}
	return snapshot
}

func TestFindingViewUsesValidatorChecksAndExportArtifactIdentity(t *testing.T) {
	for _, mode := range []string{"object_leak", "object_fixed", "wrong_identity", "expired_peer", "wrong_owner", "wrong_proof", "missing_rechecks", "property_leak", "property_filtered", "property_ambiguous"} {
		t.Run(mode, func(t *testing.T) {
			s, store, config, flags := applicationFixture(t)
			if strings.HasPrefix(mode, "property_") {
				config.Plan.ID = "property-evidence"
				config.Plan.Version = application.PropertyAccessProfile
				config.Plan.ExpectedAccess = "filtered"
				config.Plan.DenialStatuses = nil
				flags.property.Store(true)
				if err := s.RegisterApplication(config); err != nil {
					t.Fatal(err)
				}
			}
			flags.fixed.Store(mode == "object_fixed" || mode == "property_filtered")
			flags.wrongIdentity.Store(mode == "wrong_identity")
			flags.expiredPeer.Store(mode == "expired_peer")
			flags.ambiguousProperty.Store(mode == "property_ambiguous")
			e, run := applicationEngagement(t, s, config)
			reads := run.Application.Reads()
			if mode == "missing_rechecks" {
				reads = reads[:5]
			}
			executeAppReads(t, s, run, reads)
			if mode == "wrong_owner" || mode == "wrong_proof" {
				if err := store.Update(run.ID, func(r *Run) error {
					if mode == "wrong_owner" {
						r.Captures[4].Body = `{"identifier":"a","owner":"wrong","private":"opaque-proof-one"}`
					} else {
						r.Captures[4].Body = `{"identifier":"a","owner":"first","private":"wrong-proof"}`
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			result, err := s.Finish(run.ID)
			if err != nil {
				t.Fatal(err)
			}
			before, _ := store.Get(run.ID)
			scopeBefore, _ := store.GetEngagement("a", e.ID)
			countBefore := flags.requests.Load()
			snapshot := findingSnapshot(t, s, e.ID)
			if snapshot.SchemaVersion != 1 || snapshot.EngagementID != e.ID || snapshot.ScopeDigest != e.ScopeDigest || len(snapshot.Assessments) != 1 || len(snapshot.Captures) != len(reads) {
				t.Fatal("incomplete snapshot metadata")
			}
			a := snapshot.Assessments[0]
			if !reflect.DeepEqual(*a.Result, result) || a.StoredVerdict != result.Verdict || len(a.Checks) != 7 || a.OwnerAccount != "owner" || a.PeerAccount != "peer" || a.ObjectPath != config.Plan.Probe.Path {
				t.Fatalf("validation/procedure explanation differs: %+v", a)
			}
			want := "inconclusive"
			if mode == "object_leak" || mode == "property_leak" {
				want = "confirmed"
			}
			if mode == "object_fixed" || mode == "property_filtered" {
				want = "no_issue_observed"
			}
			if result.Verdict != want {
				t.Fatalf("got %s want %s", result.Verdict, want)
			}
			bundle, err := store.ExportEngagement("a", e.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshot.Candidates) != len(bundle.Candidates) {
				t.Fatal("candidate mismatch")
			}
			if want == "confirmed" {
				c := snapshot.Candidates[0]
				original := bundle.Candidates[0]
				if c.ID != original.ID || c.Digest != original.Digest || !reflect.DeepEqual(c.EvidenceIDs, original.EvidenceIDs) || c.Status != "needs_human_review" || c.Review != nil || c.DisplayTitle == config.Plan.Title {
					t.Fatalf("candidate identity/review changed: %+v", c)
				}
				if strings.HasPrefix(mode, "property_") && !strings.Contains(c.DisplayTitle, "property") {
					t.Fatal("property finding mislabeled")
				}
			}
			verified := 0
			refs := map[string]CaptureMetadata{}
			for _, c := range snapshot.Captures {
				refs[c.ID] = c
			}
			for _, check := range a.Checks {
				if check.Outcome == "verified" {
					verified++
				}
				if check.CaptureID == "" {
					if check.EvidenceID != "" {
						t.Fatal("invented missing evidence reference")
					}
					continue
				}
				ref, ok := refs[check.EvidenceID]
				if !ok || ref.CaptureID != check.CaptureID || !slices.Contains(result.EvidenceIDs, check.CaptureID) {
					t.Fatal("check not linked to selected validation evidence")
				}
			}
			if verified != result.Validation.BaselinesVerified {
				t.Fatal("explanation changed prerequisite count")
			}
			for i, metadata := range snapshot.Captures {
				original := bundle.Artifacts[i]
				if metadata.ID != original.ID || metadata.SHA256 != original.SHA256 || metadata.Path != original.Path || metadata.CaptureID != original.CaptureID {
					t.Fatal("metadata artifact identity differs from export")
				}
				w := request(s, "GET", "/v1/engagements/"+e.ID+"/evidence/"+metadata.ID, "operator-a", "")
				var raw EvidenceArtifact
				if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &raw) != nil || !reflect.DeepEqual(raw, original) {
					t.Fatalf("explicit raw read differs from export: %d", w.Code)
				}
				sum := sha256.Sum256(raw.Data)
				if hex.EncodeToString(sum[:]) != raw.SHA256 {
					t.Fatal("raw bytes fail recorded hash")
				}
			}
			body, _ := json.Marshal(snapshot)
			for _, secret := range []string{config.Credentials["owner"], config.Credentials["peer"], "opaque-proof-one", "opaque-proof-two", `"data_base64"`, `"body"`} {
				if bytes.Contains(body, []byte(secret)) {
					t.Fatalf("ordinary metadata leaked %q", secret)
				}
			}
			after, _ := store.Get(run.ID)
			scopeAfter, _ := store.GetEngagement("a", e.ID)
			x, _ := json.Marshal([]any{before, scopeBefore})
			y, _ := json.Marshal([]any{after, scopeAfter})
			if !bytes.Equal(x, y) || countBefore != flags.requests.Load() {
				t.Fatal("finding/raw read mutated state or contacted target")
			}
		})
	}
}

func TestFindingViewRetainsEarlierExposureSelectionAndRedactsDeniedArguments(t *testing.T) {
	s, store, config, flags := applicationFixture(t)
	e, run := applicationEngagement(t, s, config)
	if _, err := s.Execute(context.Background(), run.ID, Request{URL: "http://not-approved.invalid/secret-url-canary", Account: "secret-account-canary"}); err == nil {
		t.Fatal("escape accepted")
	}
	executeAppReads(t, s, run, run.Application.Reads())
	before, _ := store.Get(run.ID)
	originalProbe := before.Captures[5].ID
	flags.fixed.Store(true)
	executeAppReads(t, s, run, []application.Read{run.Application.Probe})
	if result, err := s.Finish(run.ID); err != nil || result.Verdict != "confirmed" {
		t.Fatal("earlier exposure lost")
	}
	snapshot := findingSnapshot(t, s, e.ID)
	if snapshot.Assessments[0].Checks[4].CaptureID != originalProbe || snapshot.Assessments[0].Checks[4].Outcome != "expected_object_and_proof_returned" {
		t.Fatal("explanation selected the later denied probe")
	}
	raw, _ := json.Marshal(snapshot)
	if strings.Contains(string(raw), "canary") || strings.Contains(string(raw), "not-approved.invalid") {
		t.Fatal("denied arguments leaked")
	}
	if snapshot.Captures[0].Outcome != "blocked" || snapshot.Captures[0].Account != "" || snapshot.Captures[0].RequestPath != "" {
		t.Fatal("unapproved intent acquired scope mapping")
	}
}

func TestFindingReadsKeepStopIncompleteAndEnforceTenantArtifactBindings(t *testing.T) {
	s, store, config, _ := applicationFixture(t)
	e, run := applicationEngagement(t, s, config)
	executeAppReads(t, s, run, run.Application.Reads())
	if err := s.Stop(run.ID); err != nil {
		t.Fatal(err)
	}
	snapshot := findingSnapshot(t, s, e.ID)
	if len(snapshot.Candidates) != 0 || snapshot.Assessments[0].Result != nil || snapshot.Assessments[0].State != Stopped {
		t.Fatal("stopped attempt became a completed finding")
	}
	for _, check := range snapshot.Assessments[0].Checks {
		if check.Outcome != "not_evaluated" {
			t.Fatal("Stop invented completed validation checks")
		}
	}
	artifact := snapshot.Captures[0].ID
	token, _ := store.Credential(run.ID)
	for _, path := range []string{"/v1/engagements/" + e.ID + "/findings", "/v1/engagements/" + e.ID + "/evidence/" + artifact} {
		for _, who := range []string{"", token, "operator-b"} {
			w := request(s, "GET", path, who, "")
			if w.Code != 401 && w.Code != 404 {
				t.Fatal("finding/evidence crossed principals")
			}
		}
		if w := request(s, "POST", path, "operator-a", `{"verdict":"confirmed"}`); w.Code != 405 {
			t.Fatal("worker/model event created a finding")
		}
		if w := request(s, "GET", path+"?body=1", "operator-a", ""); w.Code != 400 {
			t.Fatal("unrecognized projection query accepted")
		}
	}
	other, err := store.CreateEngagement("a", config.Plan.Origin, EngagementInput{Application: config.Plan.ID, Name: "another scope", AuthorizationRef: "owned-test"}, *e.Application)
	if err != nil {
		t.Fatal(err)
	}
	if w := request(s, "GET", "/v1/engagements/"+other.ID+"/evidence/"+artifact, "operator-a", ""); w.Code != 404 {
		t.Fatal("artifact crossed engagement binding")
	}
	if w := request(s, "GET", "/v1/engagements/"+e.ID+"/evidence/E-"+strings.Repeat("0", 64), "operator-a", ""); w.Code != 404 {
		t.Fatal("unknown artifact accepted")
	}
}

func TestFindingMetadataReviewMatchesExportWithoutNarrativesOrNewReview(t *testing.T) {
	s, store, e, c := reviewedCandidate(t)
	if _, err := s.StopEngagement("a", e.ID); err != nil {
		t.Fatal(err)
	}
	in := testReview(c)
	in.Rationale = "secret-review-narrative-canary"
	in.Impact = "secret-impact-canary"
	if _, err := store.ReviewFinding("a", e.ID, in); err != nil {
		t.Fatal(err)
	}
	snapshot := findingSnapshot(t, s, e.ID)
	bundle, err := store.ExportEngagement("a", e.ID)
	if err != nil {
		t.Fatal(err)
	}
	view := snapshot.Candidates[0]
	if view.Status != bundle.Candidates[0].Status || view.Review == nil || view.Review.Purpose != "automated_lab_test" || view.Review.Actor != "automated_lab_verifier" || view.Review.Revision != 1 || view.Review.Severity != bundle.Candidates[0].Review.Severity {
		t.Fatal("review was invented or detached from evidence")
	}
	raw, _ := json.Marshal(snapshot)
	if strings.Contains(string(raw), "canary") {
		t.Fatal("review narrative streamed as ordinary metadata")
	}
}

func TestFindingViewLinksUnresolvedEarlierExposureWithoutConfirmingIt(t *testing.T) {
	s, _, config, flags := applicationFixture(t)
	e, run := applicationEngagement(t, s, config)
	p := run.Application
	executeAppReads(t, s, run, []application.Read{p.Baselines[0], p.Baselines[1], p.Probe})
	flags.fixed.Store(true)
	executeAppReads(t, s, run, []application.Read{p.Identity[0], p.Identity[1], p.Probe, p.Baselines[0], p.Identity[1]})
	result, err := s.Finish(run.ID)
	if err != nil || result.Verdict != "inconclusive" {
		t.Fatal("incomplete earlier exposure acquired a clean/confirmed result", err)
	}
	snapshot := findingSnapshot(t, s, e.ID)
	if len(snapshot.Candidates) != 0 || len(snapshot.Assessments[0].Checks) != 8 {
		t.Fatal("earlier unresolved support was hidden or promoted")
	}
	check := snapshot.Assessments[0].Checks[7]
	if check.Role != "earlier_unresolved_probe" || check.Outcome != "unresolved_prior_exposure" || check.CaptureID != snapshot.Captures[2].CaptureID || check.EvidenceID != snapshot.Captures[2].ID {
		t.Fatal("unresolved exposure link is not the captured earlier probe")
	}
}
