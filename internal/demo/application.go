package demo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/toolcontract"
	"github.com/Sanskarzz/tsunami/internal/worker"
)

type ApplicationCheck struct {
	Application  string `json:"application"`
	EngagementID string `json:"engagement_id"`
	RunID        string `json:"run_id"`
	Expected     string `json:"expected"`
	Actual       string `json:"actual"`
	Requests     int    `json:"target_requests"`
	Candidates   int    `json:"candidates"`
	Findings     int    `json:"reviewed_findings"`
}
type ApplicationSummary struct {
	Executor    string             `json:"executor,omitempty"`
	CheckedAt   time.Time          `json:"checked_at"`
	Directory   string             `json:"artifact_directory"`
	Mode        string             `json:"mode"`
	Checks      []ApplicationCheck `json:"checks"`
	Limitations []string           `json:"limitations"`
}

func ApplicationVerify(ctx context.Context, o Operator, out string, cases map[string]string, exerciseReview bool, log io.Writer, executors ...string) (ApplicationSummary, error) {
	executor := ""
	if len(executors) > 1 {
		return ApplicationSummary{}, errors.New("one executor required")
	}
	if len(executors) == 1 {
		executor = executors[0]
	}
	if executor != "" && executor != toolcontract.CurlHTTP {
		return ApplicationSummary{}, errors.New("unknown executor")
	}
	report := ApplicationSummary{Executor: executor, CheckedAt: time.Now().UTC(), Mode: "scripted Mecatl / configured owned application / no external model", Checks: []ApplicationCheck{}, Limitations: []string{"Known accounts, object inventory and access rule; not blind discovery or customer readiness.", "Setup creates synthetic data outside the agent. The agent receives only read capabilities.", "Any review performed here is explicitly automated_lab_test, not human signoff.", "No private application connector, arbitrary shell/scanners, real customer data or scale qualification."}}
	if len(cases) == 0 || len(cases) > 4 {
		return report, errors.New("one to four explicit application evaluation cases required")
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		return report, err
	}
	dir, err := os.MkdirTemp(out, "campaign-")
	if err != nil {
		return report, err
	}
	report.Directory = dir
	var registered []application.Plan
	if err := o.Call(ctx, "GET", "/v1/applications", nil, &registered); err != nil {
		return report, err
	}
	plans := map[string]application.Plan{}
	for _, p := range registered {
		plans[p.ID] = p
	}
	ids := []string{}
	for id, want := range cases {
		if _, ok := plans[id]; !ok || (want != "confirmed" && want != "no_issue_observed" && want != "inconclusive") {
			return report, errors.New("unknown application or expected outcome")
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	writeSummary := func() error {
		data, _ := json.MarshalIndent(report, "", "  ")
		return os.WriteFile(filepath.Join(dir, "summary.json"), append(data, '\n'), 0600)
	}
	defer func() { _ = writeSummary() }()
	for _, id := range ids {
		var e platform.Engagement
		if err := o.Call(ctx, "POST", "/v1/engagements", platform.EngagementInput{Name: "Owned application access-control test", AuthorizationRef: "owned-lab/application-verifier", Application: id, Executor: executor}, &e); err != nil {
			return report, err
		}
		base := "/v1/engagements/" + e.ID
		if err := o.Call(ctx, "POST", base+"/approve", platform.ApprovalInput{ScopeDigest: e.ScopeDigest}, nil); err != nil {
			return report, err
		}
		var r platform.Run
		if err := o.Call(ctx, "POST", base+"/runs", map[string]string{"idempotency_key": platform.NewStartKey(), "assessment": plans[id].ProcedureID()}, &r); err != nil {
			return report, err
		}
		if log != nil {
			fmt.Fprintf(log, "APPLICATION_RUN application=%s engagement=%s run=%s\n", id, e.ID, r.ID)
		}
		waitCtx, cancel := context.WithTimeout(ctx, 100*time.Second)
		finished, err := Wait(waitCtx, o, r.ID, func(r platform.Run) bool {
			return r.State == platform.Completed || r.State == platform.Stopped || r.State == platform.Failed
		})
		cancel()
		if err != nil {
			return report, err
		}
		if finished.Executor != executor || finished.State != platform.Completed || finished.Result == nil || finished.Result.Verdict != cases[id] {
			return report, errors.New("application outcome mismatch; retain run evidence")
		}
		if err := o.Call(ctx, "POST", base+"/stop", struct{}{}, nil); err != nil {
			return report, err
		}
		var b platform.EngagementBundle
		if err := o.Call(ctx, "GET", base+"/export", nil, &b); err != nil {
			return report, err
		}
		if exerciseReview {
			for _, candidate := range b.Candidates {
				in := platform.ReviewInput{CandidateID: candidate.ID, CandidateDigest: candidate.Digest, Decision: "confirm", Purpose: "automated_lab_test", Rationale: "Automated lab review exercises schema and evidence export; it is not human approval.", Impact: "An authenticated synthetic peer retrieved another synthetic test account's seeded private object proof.", CVSSVector: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N"}
				if err := o.Call(ctx, "POST", base+"/reviews", in, nil); err != nil {
					return report, err
				}
			}
			if err := o.Call(ctx, "GET", base+"/export", nil, &b); err != nil {
				return report, err
			}
		}
		if err := saveEngagementBundle(filepath.Join(dir, id), b); err != nil {
			return report, err
		}
		report.Checks = append(report.Checks, ApplicationCheck{id, e.ID, r.ID, cases[id], finished.Result.Verdict, finished.RequestCount, len(b.Candidates), len(b.Findings)})
		if log != nil {
			fmt.Fprintf(log, "APPLICATION_RESULT application=%s engagement=%s verdict=%s requests=%d candidates=%d reviewed_findings=%d\n", id, e.ID, finished.Result.Verdict, finished.RequestCount, len(b.Candidates), len(b.Findings))
		}
	}
	if err := writeSummary(); err != nil {
		return report, err
	}
	return report, nil
}

func ApplicationLocal(ctx context.Context, config, out string, cases map[string]string, exerciseReview bool, log io.Writer) (ApplicationSummary, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	dir, err := os.MkdirTemp("", "tsunami-application-")
	if err != nil {
		return ApplicationSummary{}, err
	}
	defer os.RemoveAll(dir)
	store, err := platform.OpenStore(filepath.Join(dir, "state.db"))
	if err != nil {
		return ApplicationSummary{}, err
	}
	defer store.Close()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return ApplicationSummary{}, err
	}
	token := hex.EncodeToString(key)
	s := platform.NewServer(store, nil, map[string]string{token: "demo-a"})
	if err := s.LoadApplications(config); err != nil {
		return ApplicationSummary{}, err
	}
	api := httptest.NewServer(s)
	defer api.Close()
	s.Scheduler = &worker.LocalScheduler{ClientURL: api.URL, OnError: func(id string, _ error) { _ = store.Fail(id) }}
	go s.Reconcile(ctx)
	return ApplicationVerify(ctx, Operator{BaseURL: api.URL, Token: token}, out, cases, exerciseReview, log)
}
