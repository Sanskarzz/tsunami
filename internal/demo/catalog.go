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
	"strings"
	"time"

	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/Sanskarzz/tsunami/internal/lab"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/worker"
)

type CatalogOptions struct {
	Mode    string // scripted, fixture, or free; paid routing is unsupported here
	Repeats int
	IDs     []string
}
type CatalogResult struct {
	RunID              string               `json:"run_id"`
	Assessment         string               `json:"assessment"`
	Digest             string               `json:"definition_digest"`
	Scenario           string               `json:"fixture_variant"`
	Expected           string               `json:"expected"`
	Actual             string               `json:"actual"`
	State              string               `json:"state"`
	Match              bool                 `json:"match"`
	EvidenceComplete   bool                 `json:"evidence_complete"`
	DurationMS         int64                `json:"duration_ms"`
	Requests           int                  `json:"requests_reserved"`
	CompletionFeedback int                  `json:"completion_feedback_count"`
	Model              *platform.ModelUsage `json:"model,omitempty"`
	OutcomeCategory    string               `json:"outcome_category"`
	FailureCode        string               `json:"failure_code,omitempty"`
}
type CatalogSummary struct {
	CheckedAt             time.Time       `json:"checked_at"`
	Mode                  string          `json:"mode"`
	Connectivity          string          `json:"connectivity"`
	Repeats               int             `json:"repeats"`
	Results               []CatalogResult `json:"results"`
	Matched               int             `json:"matched"`
	ConfirmedVulnerable   int             `json:"confirmed_vulnerable"`
	MissedVulnerable      int             `json:"missed_vulnerable"`
	FalsePositives        int             `json:"false_positives"`
	Inconclusive          int             `json:"inconclusive_results"`
	IncompleteEvidence    int             `json:"incomplete_evidence"`
	OperationalFailures   int             `json:"operational_failures"`
	AssessmentMisses      int             `json:"assessment_misses"`
	IncompleteAssessments int             `json:"incomplete_assessments"`
	StoppedRuns           int             `json:"stopped_runs"`
	Limitations           []string        `json:"limitations"`
}

// Ground truth belongs to the evaluation harness, never worker configuration
// or the definition catalog. These are immutable synthetic application rules.
func catalogExpected(id, variant string) string {
	if id == "missing-object/v1" {
		return "inconclusive"
	}
	if id == "shared-access/v1" || variant == "fixed" {
		return "no_issue_observed"
	}
	return "confirmed"
}

func Catalog(ctx context.Context, o Operator, out string, opts CatalogOptions, log io.Writer) (CatalogSummary, error) {
	summary := CatalogSummary{CheckedAt: time.Now().UTC(), Mode: opts.Mode, Connectivity: o.Connectivity, Repeats: opts.Repeats, Results: []CatalogResult{}, Limitations: []string{"Known synthetic applications and developer-supplied access rules; not a blind discovery benchmark.", "Procedures are initial engineering definitions; security-engineer review is pending.", "Scripted and offline fixture metrics measure integration, not LLM accuracy.", "Live results, when explicitly requested, are one guided sample; no production or application-wide conclusion."}}
	if opts.Repeats < 1 || opts.Repeats > 3 {
		return summary, errors.New("repeat must be 1-3")
	}
	if opts.Mode != "scripted" && opts.Mode != "fixture" && opts.Mode != "free" {
		return summary, errors.New("catalog mode must be scripted, fixture or free")
	}
	if o.Connectivity == "private" && opts.Mode != "scripted" {
		return summary, errors.New("private catalog currently supports scripted execution only")
	}
	defs := assessment.List()
	if len(opts.IDs) > 0 {
		defs = nil
		seen := map[string]bool{}
		for _, id := range opts.IDs {
			d, err := assessment.Get(id)
			if err != nil || seen[id] {
				return summary, errors.New("unknown or duplicate catalog selection")
			}
			seen[id] = true
			defs = append(defs, d)
		}
	}
	if opts.Mode == "free" && (opts.Repeats != 1 || len(defs) > 2) {
		return summary, errors.New("free catalog trial requires one repeat and at most two selected procedures")
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		return summary, err
	}
	writeSummary := func() error {
		data, _ := json.MarshalIndent(summary, "", "  ")
		return os.WriteFile(filepath.Join(out, "evaluation.json"), append(data, '\n'), 0600)
	}
	for repeat := 0; repeat < opts.Repeats; repeat++ {
		for _, d := range defs {
			for _, variant := range []string{"vulnerable", "fixed"} {
				input := map[string]string{"scenario": variant, "assessment": d.ID, "connectivity": o.Connectivity}
				if opts.Mode != "scripted" {
					input["execution"] = "live"
					input["required_provider"] = "fixture"
				}
				if opts.Mode == "free" {
					input["required_provider"] = "openrouter-free"
					input["required_price_policy"] = inference.FreePolicyID
				}
				start := time.Now()
				var run platform.Run
				if err := o.Call(ctx, "POST", "/v1/runs", input, &run); err != nil {
					_ = writeSummary()
					return summary, err
				}
				if log != nil {
					fmt.Fprintf(log, "CATALOG_CREATED run=%s assessment=%s variant=%s\n", run.ID, d.ID, variant)
				}
				waitCtx, cancel := context.WithTimeout(ctx, 100*time.Second)
				r, err := Wait(waitCtx, o, run.ID, func(r platform.Run) bool {
					return r.State == platform.Completed || r.State == platform.Failed || r.State == platform.Stopped
				})
				cancel()
				if err != nil {
					_ = writeSummary()
					return summary, err
				}
				actual := "inconclusive"
				complete := false
				if r.Result != nil {
					actual = r.Result.Verdict
					if v := r.Result.Validation; v != nil {
						complete = v.BaselinesVerified == v.BaselinesRequired && v.ProbeCaptured
						for _, capture := range r.Captures {
							if !capture.Blocked && capture.Error != "" {
								complete = false
							}
						}
					}
				}
				expected := catalogExpected(d.ID, variant)
				match := r.State == platform.Completed && actual == expected
				result := CatalogResult{RunID: r.ID, Assessment: d.ID, Digest: r.AssessmentDigest, Scenario: variant, Expected: expected, Actual: actual, State: r.State, Match: match, EvidenceComplete: complete, DurationMS: time.Since(start).Milliseconds(), Requests: r.RequestCount, CompletionFeedback: r.CompletionFeedback, Model: r.Model}
				result.OutcomeCategory, result.FailureCode = classifyAssessment(r, expected)
				summary.Results = append(summary.Results, result)
				if operational(result.OutcomeCategory) {
					summary.OperationalFailures++
				} else if result.OutcomeCategory != "stopped" && expected == "confirmed" && (!match || !complete) {
					summary.AssessmentMisses++
				}
				if result.OutcomeCategory == "incomplete_assessment" {
					summary.IncompleteAssessments++
				}
				if result.OutcomeCategory == "stopped" {
					summary.StoppedRuns++
				}
				if match {
					summary.Matched++
				}
				if actual == "inconclusive" {
					summary.Inconclusive++
				}
				if !complete {
					summary.IncompleteEvidence++
				}
				if expected == "confirmed" {
					if actual == "confirmed" && r.State == platform.Completed {
						summary.ConfirmedVulnerable++
					} else {
						summary.MissedVulnerable++
					}
				} else if actual == "confirmed" {
					summary.FalsePositives++
				}
				stem := fmt.Sprintf("%02d-%s-%s", repeat+1, strings.ReplaceAll(d.ID, "/", "-"), variant)
				data, _ := json.MarshalIndent(r, "", "  ")
				if err = os.WriteFile(filepath.Join(out, stem+".json"), data, 0600); err != nil {
					return summary, err
				}
				if err = os.WriteFile(filepath.Join(out, stem+".md"), []byte(platform.Report(r)), 0600); err != nil {
					return summary, err
				}
				if err = writeSummary(); err != nil {
					return summary, err
				}
				if log != nil {
					fmt.Fprintf(log, "CATALOG run=%s assessment=%s variant=%s expected=%s actual=%s match=%t evidence_complete=%t\n", r.ID, d.ID, variant, expected, actual, match, complete)
				}
			}
		}
	}
	if log != nil {
		fmt.Fprintf(log, "CATALOG_SUMMARY total=%d matched=%d missed=%d false_positives=%d inconclusive=%d incomplete_evidence=%d mode=%s\n", len(summary.Results), summary.Matched, summary.MissedVulnerable, summary.FalsePositives, summary.Inconclusive, summary.IncompleteEvidence, summary.Mode)
	}
	if summary.Matched != len(summary.Results) || summary.IncompleteEvidence > 0 {
		return summary, errors.New("catalog evaluation has mismatches or incomplete evidence; records retained")
	}
	return summary, nil
}

func CatalogLocal(ctx context.Context, out string, fixture bool, repeats int, log io.Writer, premature ...bool) (CatalogSummary, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	v := httptest.NewServer(&lab.Server{})
	defer v.Close()
	f := httptest.NewServer(&lab.Server{Fixed: true})
	defer f.Close()
	vt, err := platform.LabTarget(ctx, v.URL)
	if err != nil {
		return CatalogSummary{}, err
	}
	ft, err := platform.LabTarget(ctx, f.URL)
	if err != nil {
		return CatalogSummary{}, err
	}
	dir, err := os.MkdirTemp("", "tsunami-catalog-")
	if err != nil {
		return CatalogSummary{}, err
	}
	defer os.RemoveAll(dir)
	store, err := platform.OpenStore(filepath.Join(dir, "state.db"))
	if err != nil {
		return CatalogSummary{}, err
	}
	defer store.Close()
	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return CatalogSummary{}, err
	}
	token := hex.EncodeToString(key)
	s := platform.NewServer(store, map[string]platform.Target{"vulnerable": vt, "fixed": ft}, map[string]string{token: "demo-a"})
	mode := "scripted"
	if fixture {
		providerFixture := &inference.Fixture{}
		if len(premature) > 0 {
			providerFixture.PrematureFinish = premature[0]
		}
		mock := httptest.NewServer(providerFixture)
		defer mock.Close()
		s.Inference, err = inference.NewConfig("fixture", "tsunami-wire-fixture", mock.URL+"/v1/", "")
		if err != nil {
			return CatalogSummary{}, err
		}
		mode = "fixture"
	}
	api := httptest.NewServer(s)
	defer api.Close()
	s.Scheduler = &worker.LocalScheduler{ClientURL: api.URL, OnError: func(id string, _ error) { _ = store.Fail(id) }}
	go s.Reconcile(ctx)
	return Catalog(ctx, Operator{BaseURL: api.URL, Token: token}, out, CatalogOptions{Mode: mode, Repeats: repeats}, log)
}
