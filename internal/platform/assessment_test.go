package platform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/inference"
)

func assessmentEvidence(t *testing.T, id string) Run {
	t.Helper()
	d, err := assessment.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	r := Run{Target: "http://fixture.invalid", Assessment: &d, AssessmentDigest: d.Digest()}
	for i, read := range d.Reads() {
		body, _ := json.Marshal(read.Object)
		r.Captures = append(r.Captures, Capture{ID: strings.Repeat(string(rune('a'+i)), 32), URL: r.Target + read.Path, Account: read.Account, Status: 200, Body: string(body)})
	}
	return r
}

func TestAssessmentEvidenceDoesNotPromoteAmbiguousResponses(t *testing.T) {
	for _, name := range []string{"confirmed", "error-200", "wrong-object-200", "failed-baseline", "late-interrupted-probe", "modified-digest", "known-object-404"} {
		t.Run(name, func(t *testing.T) {
			r := assessmentEvidence(t, "opaque-ownership/v1")
			want := "inconclusive"
			switch name {
			case "confirmed":
				want = "confirmed"
			case "error-200":
				r.Captures[2].Body = `{"error":"forbidden"}`
			case "wrong-object-200":
				r.Captures[2].Body = r.Captures[1].Body
			case "failed-baseline":
				r.Captures[0].Status = 404
			case "late-interrupted-probe":
				late := r.Captures[2]
				late.Error = "interrupted"
				r.Captures = append(r.Captures, late)
			case "modified-digest":
				r.AssessmentDigest = "modified"
			case "known-object-404":
				r.Captures[2].Status = 404
				r.Captures[2].Body = `{"error":"not found"}`
				want = "no_issue_observed"
			}
			if actual := Validate(r); actual.Verdict != want {
				t.Fatalf("%s: %s", name, actual.Verdict)
			}
		})
	}
	if got := Validate(assessmentEvidence(t, "shared-access/v1")); got.Verdict != "no_issue_observed" {
		t.Fatal("approved sharing became a false positive")
	}
	if got := Validate(assessmentEvidence(t, "missing-object/v1")); got.Verdict != "inconclusive" {
		t.Fatal("unknown object established authorization")
	}
}

func TestAssessmentCannotChangeLegacyOrOtherProcedureScope(t *testing.T) {
	s, _ := fixture(t)
	legacy := running(t, s, "a")
	if caps, err := s.Execute(context.Background(), legacy.ID, Request{URL: legacy.Target + "/api/reviewed/objects/r-7f3c9a", Account: "bob"}); err == nil || len(caps) != 1 || !caps[0].Blocked {
		t.Fatal("catalog expanded legacy run scope")
	}
	r, err := s.Store.CreateAssessment("a", "vulnerable", legacy.Target, "native", "", "opaque-ownership/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Store.Update(r.ID, func(r *Run) error { r.State = Running; return nil }); err != nil {
		t.Fatal(err)
	}
	for _, input := range []Request{{URL: r.Target + "/api/reviewed/admin/summary", Account: "bob"}, {URL: r.Target + "/api/reviewed/objects/r-b82d16", Account: "alice"}, {URL: r.Target + "/api/reviewed/objects/r-7f3c9a", Account: "anonymous"}} {
		caps, err := s.Execute(context.Background(), r.ID, input)
		if err == nil || len(caps) != 1 || !caps[0].Blocked {
			t.Fatal("procedure capability bypass")
		}
	}
	fresh, _ := s.Store.Get(r.ID)
	if fresh.RequestCount != 0 {
		t.Fatal("denied requests reached target reservation")
	}
	for _, body := range []string{`{"scenario":"vulnerable","assessment":"../unknown"}`, `{"scenario":"vulnerable","assessment":"opaque-ownership/v1","assessment_digest":"forged"}`, `{"scenario":"vulnerable","assessment":{"id":"custom"}}`} {
		if w := request(s, "POST", "/v1/runs", "operator-a", body); w.Code != 400 {
			t.Fatal("caller-defined assessment accepted")
		}
	}
}

func TestAnonymousCapabilityOmitsTargetAuthorization(t *testing.T) {
	s, _ := fixture(t)
	received := make(chan string, 1)
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer web.Close()
	target, err := LabTarget(context.Background(), web.URL)
	if err != nil {
		t.Fatal(err)
	}
	s.Targets["vulnerable"] = target
	r, err := s.Store.CreateAssessment("a", "vulnerable", target.URL, "native", "", "anonymous-access/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Store.Update(r.ID, func(r *Run) error { r.State = Running; return nil })
	if _, err = s.Execute(context.Background(), r.ID, Request{URL: r.Target + "/api/reviewed/objects/r-7f3c9a", Account: "anonymous"}); err != nil {
		t.Fatal(err)
	}
	if <-received != "" {
		t.Fatal("anonymous request leaked a target credential")
	}
}

func TestCatalogProviderConstraintIsCheckedBeforeRunCreation(t *testing.T) {
	s, _ := fixture(t)
	var err error
	s.Inference, err = inference.NewConfig("fixture", "synthetic", "http://127.0.0.1:1/v1/", "")
	if err != nil {
		t.Fatal(err)
	}
	if w := request(s, "POST", "/v1/runs", "operator-a", `{"scenario":"vulnerable","assessment":"opaque-ownership/v1","execution":"live","required_provider":"openrouter-free"}`); w.Code != 409 {
		t.Fatal("provider race guard failed")
	}
	runs, _ := s.Store.List()
	if len(runs) != 0 {
		t.Fatal("provider mismatch created a run")
	}
	s.Inference, err = inference.NewConfig("responses", "paid-placeholder", "https://model.invalid/v1/", "synthetic-unused-key")
	if err != nil {
		t.Fatal(err)
	}
	if w := request(s, "POST", "/v1/runs", "operator-a", `{"scenario":"vulnerable","assessment":"opaque-ownership/v1","execution":"live"}`); w.Code != 409 {
		t.Fatal("catalog accepted a paid provider")
	}
}

func TestMissingEvidenceFeedbackIsDurableAndBounded(t *testing.T) {
	s, _ := fixture(t)
	target := s.Targets["vulnerable"]
	r, err := s.Store.CreateAssessment("a", "vulnerable", target.URL, "native", "", "opaque-ownership/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Store.Update(r.ID, func(r *Run) error { r.State = Running; return nil })
	_, err = s.Finish(r.ID)
	var needed *MissingEvidence
	if !errors.As(err, &needed) || len(needed.Reads) != 3 {
		t.Fatal("missing evidence was accepted or lacked trusted feedback")
	}
	for _, read := range needed.Reads {
		if !r.Assessment.Allows(read.Account, read.Path) || read.Object != nil {
			t.Fatal("feedback expanded scope or included object content")
		}
	}
	path := s.Store.db.Path()
	_ = s.Store.Close()
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	s = NewServer(reopened, s.Targets, nil)
	stored, _ := s.Store.Get(r.ID)
	if stored.CompletionFeedback != 1 || stored.State != Running || stored.RequestCount != 0 {
		t.Fatal("feedback state was lost or consumed target permission")
	}
	if _, err = s.Finish(r.ID); !errors.As(err, &needed) {
		t.Fatal("second feedback unavailable")
	}
	result, err := s.Finish(r.ID)
	if err != nil || result.Verdict != "inconclusive" {
		t.Fatal("feedback loop failed to terminate conservatively")
	}
	stored, _ = s.Store.Get(r.ID)
	if stored.CompletionFeedback != 2 || stored.State != Completed {
		t.Fatal("feedback cap not enforced")
	}
}

func TestStopRevokesAnAssessmentWaitingForEvidence(t *testing.T) {
	s, _ := fixture(t)
	target := s.Targets["vulnerable"]
	r, err := s.Store.CreateAssessment("a", "vulnerable", target.URL, "native", "", "opaque-ownership/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Store.Update(r.ID, func(r *Run) error { r.State = Running; return nil })
	if _, err = s.Finish(r.ID); err == nil {
		t.Fatal("missing prerequisites were accepted")
	}
	if err = s.Stop(r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Finish(r.ID); err == nil {
		t.Fatal("feedback reopened a stopped assessment")
	}
	if _, err = s.Execute(context.Background(), r.ID, Request{URL: r.Target + "/api/reviewed/objects/r-7f3c9a", Account: "alice"}); err == nil {
		t.Fatal("feedback allowed a request after Stop")
	}
}
