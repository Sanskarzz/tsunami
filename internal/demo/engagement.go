package demo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/Sanskarzz/tsunami/internal/lab"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/worker"
)

type EngagementCheck struct {
	EngagementID string `json:"engagement_id"`
	Scenario     string `json:"scenario"`
	Completed    int    `json:"procedures_completed"`
	Candidates   int    `json:"finding_candidates"`
	Artifacts    int    `json:"evidence_artifacts"`
}
type EngagementSummary struct {
	CheckedAt             time.Time         `json:"checked_at"`
	Mode                  string            `json:"mode"`
	Connectivity          string            `json:"connectivity,omitempty"`
	Directory             string            `json:"artifact_directory"`
	Checks                []EngagementCheck `json:"checks"`
	DraftDispatchDenied   bool              `json:"draft_dispatch_denied"`
	StoppedDispatchDenied bool              `json:"stopped_dispatch_denied"`
	Limitations           []string          `json:"limitations"`
}

func EngagementVerify(ctx context.Context, o Operator, out string, fixture bool, log io.Writer) (EngagementSummary, error) {
	mode := "scripted"
	if fixture {
		mode = "offline Responses fixture"
	}
	report := EngagementSummary{CheckedAt: time.Now().UTC(), Mode: mode, Connectivity: o.Connectivity, Checks: []EngagementCheck{}, Limitations: []string{"Synthetic lab targets and initial procedures only, not customer readiness or broad toolkit execution.", "This campaign exports unrated candidates; separate explicit review produces finding-schema exports.", "Stop here checks dispatch revocation; active-request cancellation is tested separately.", "No general multi-agent role scheduler, VM/device executor, or scale qualification is demonstrated."}}
	if err := os.MkdirAll(out, 0700); err != nil {
		return report, err
	}
	dir, err := os.MkdirTemp(out, "campaign-")
	if err != nil {
		return report, err
	}
	report.Directory = dir
	writeSummary := func() error {
		data, _ := json.MarshalIndent(report, "", "  ")
		return os.WriteFile(filepath.Join(dir, "summary.json"), append(data, '\n'), 0600)
	}
	defer func() { _ = writeSummary() }()
	ids := []string{}
	for _, d := range assessment.List() {
		ids = append(ids, d.ID)
	}
	for _, scenario := range []string{"vulnerable", "fixed"} {
		var e platform.Engagement
		input := platform.EngagementInput{Name: "Owned lab authorization assessment", AuthorizationRef: "owned-lab/engagement-verifier", Scenario: scenario, Connectivity: o.Connectivity, Assessments: ids}
		if err := o.Call(ctx, "POST", "/v1/engagements", input, &e); err != nil {
			return report, err
		}
		base := "/v1/engagements/" + e.ID
		if o.Call(ctx, "POST", base+"/runs", map[string]string{"idempotency_key": platform.NewStartKey(), "assessment": ids[0]}, nil) == nil {
			return report, errors.New("draft dispatched a run")
		}
		report.DraftDispatchDenied = true
		if err := o.Call(ctx, "POST", base+"/approve", platform.ApprovalInput{ScopeDigest: e.ScopeDigest}, nil); err != nil {
			return report, err
		}
		for _, id := range ids {
			body := map[string]string{"idempotency_key": platform.NewStartKey(), "assessment": id}
			if fixture {
				body["execution"] = "live"
				body["required_provider"] = "fixture"
			}
			var r platform.Run
			if err := o.Call(ctx, "POST", base+"/runs", body, &r); err != nil {
				return report, err
			}
			if log != nil {
				fmt.Fprintf(log, "ENGAGEMENT_RUN engagement=%s run=%s procedure=%s variant=%s\n", e.ID, r.ID, id, scenario)
			}
			waitCtx, cancel := context.WithTimeout(ctx, 100*time.Second)
			finished, err := Wait(waitCtx, o, r.ID, func(r platform.Run) bool {
				return r.State == platform.Completed || r.State == platform.Stopped || r.State == platform.Failed
			})
			cancel()
			if err != nil {
				return report, err
			}
			if finished.State != platform.Completed || finished.Result == nil || finished.Result.Verdict != catalogExpected(id, scenario) {
				return report, errors.New("engagement procedure failed expected fixture outcome")
			}
		}
		var bundle platform.EngagementBundle
		if err := o.Call(ctx, "GET", base+"/export", nil, &bundle); err != nil {
			return report, err
		}
		if err := saveEngagementBundle(filepath.Join(dir, scenario), bundle); err != nil {
			return report, err
		}
		expected := 0
		if scenario == "vulnerable" {
			expected = 3
		}
		if bundle.View.Progress.Completed != len(ids) || len(bundle.Candidates) != expected {
			return report, errors.New("incorrect evidence or checklist result")
		}
		report.Checks = append(report.Checks, EngagementCheck{e.ID, scenario, bundle.View.Progress.Completed, len(bundle.Candidates), len(bundle.Artifacts)})
		if err := o.Call(ctx, "POST", base+"/stop", struct{}{}, nil); err != nil {
			return report, err
		}
		if o.Call(ctx, "POST", base+"/runs", map[string]string{"idempotency_key": platform.NewStartKey(), "assessment": ids[0]}, nil) == nil {
			return report, errors.New("stopped engagement dispatched a run")
		}
		report.StoppedDispatchDenied = true
		if log != nil {
			fmt.Fprintf(log, "ENGAGEMENT_RESULT id=%s variant=%s completed=%d candidates=%d artifacts=%d dispatch_after_stop=denied\n", e.ID, scenario, bundle.View.Progress.Completed, len(bundle.Candidates), len(bundle.Artifacts))
		}
	}
	if err := writeSummary(); err != nil {
		return report, err
	}
	if log != nil {
		fmt.Fprintf(log, "ENGAGEMENT_SUMMARY records=%s no_external_model_calls=true\n", dir)
	}
	return report, nil
}

func saveEngagementBundle(dir string, b platform.EngagementBundle) error {
	// Each campaign has a fresh directory; do not overwrite or follow prior
	// evidence entries. Paths are content-derived, never caller-selected.
	if err := os.Mkdir(dir, 0700); err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(dir, "evidence"), 0700); err != nil {
		return err
	}
	writeJSON := func(name string, value any) error {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, name), append(data, '\n'), 0600)
	}
	if err := writeJSON("bundle.json", b); err != nil {
		return err
	}
	if err := writeJSON("engagement.json", b.ToolkitEngagement); err != nil {
		return err
	}
	if err := writeJSON("finding-candidates.json", b.Candidates); err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(dir, "findings"), 0700); err != nil {
		return err
	}
	for _, finding := range b.Findings {
		if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`).MatchString(finding.ID) {
			return errors.New("invalid finding export ID")
		}
		if err := writeJSON(filepath.Join("findings", "finding_"+finding.ID+".json"), finding); err != nil {
			return err
		}
	}
	csv, err := os.OpenFile(filepath.Join(dir, "findings.csv"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = platform.FindingsCSV(csv, b.Findings)
	closeErr := csv.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	for _, evidence := range b.Evidence {
		if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`).MatchString(evidence.ID) {
			return errors.New("invalid evidence export ID")
		}
		if err := writeJSON(filepath.Join("evidence", evidence.ID+".meta.json"), evidence); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "report-draft.md"), []byte(b.Report), 0600); err != nil {
		return err
	}
	for _, a := range b.Artifacts {
		sum := sha256.Sum256(a.Data)
		digest := hex.EncodeToString(sum[:])
		if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(a.SHA256) || a.SHA256 != digest || a.Path != "evidence/"+digest+".json" {
			return errors.New("untrusted evidence path or content mismatch")
		}
		f, err := os.OpenFile(filepath.Join(dir, a.Path), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		_, err = f.Write(a.Data)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func WriteEngagementBundle(dir string, b platform.EngagementBundle) error {
	return saveEngagementBundle(dir, b)
}

func EngagementLocal(ctx context.Context, out string, fixture bool, log io.Writer) (EngagementSummary, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	v := httptest.NewServer(&lab.Server{})
	defer v.Close()
	f := httptest.NewServer(&lab.Server{Fixed: true})
	defer f.Close()
	vt, err := platform.LabTarget(ctx, v.URL)
	if err != nil {
		return EngagementSummary{}, err
	}
	ft, err := platform.LabTarget(ctx, f.URL)
	if err != nil {
		return EngagementSummary{}, err
	}
	dir, err := os.MkdirTemp("", "tsunami-engagement-")
	if err != nil {
		return EngagementSummary{}, err
	}
	defer os.RemoveAll(dir)
	store, err := platform.OpenStore(filepath.Join(dir, "state.db"))
	if err != nil {
		return EngagementSummary{}, err
	}
	defer store.Close()
	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return EngagementSummary{}, err
	}
	token := hex.EncodeToString(key)
	s := platform.NewServer(store, map[string]platform.Target{"vulnerable": vt, "fixed": ft}, map[string]string{token: "demo-a"})
	if fixture {
		provider := httptest.NewServer(&inference.Fixture{})
		defer provider.Close()
		s.Inference, err = inference.NewConfig("fixture", "tsunami-wire-fixture", provider.URL+"/v1/", "")
		if err != nil {
			return EngagementSummary{}, err
		}
	}
	api := httptest.NewServer(s)
	defer api.Close()
	s.Scheduler = &worker.LocalScheduler{ClientURL: api.URL, OnError: func(id string, _ error) { _ = store.Fail(id) }}
	go s.Reconcile(ctx)
	return EngagementVerify(ctx, Operator{BaseURL: api.URL, Token: token}, out, fixture, log)
}
