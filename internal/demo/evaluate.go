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
	"time"

	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/Sanskarzz/tsunami/internal/lab"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/worker"
)

type Evaluation struct {
	CheckedAt time.Time          `json:"checked_at"`
	Workflow  string             `json:"workflow"`
	Results   []EvaluationResult `json:"results"`
}
type EvaluationResult struct {
	RunID      string               `json:"run_id"`
	Scenario   string               `json:"scenario"`
	Expected   string               `json:"expected"`
	Actual     string               `json:"actual"`
	State      string               `json:"state"`
	Match      bool                 `json:"match"`
	DurationMS int64                `json:"duration_ms"`
	Mode       string               `json:"mode"`
	Usage      *platform.ModelUsage `json:"model"`
}

// Evaluate is explicitly separate from deterministic verification. It records
// failures/inconclusive results instead of assuming a live model follows a script.
func Evaluate(ctx context.Context, o Operator, out string, repeats int, log io.Writer) (Evaluation, error) {
	evaluation := Evaluation{CheckedAt: time.Now().UTC(), Workflow: inference.WorkflowID, Results: []EvaluationResult{}}
	if repeats < 1 || repeats > 3 {
		return evaluation, errors.New("repeat must be 1-3")
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		return evaluation, err
	}
	passed := true
	for repeat := 0; repeat < repeats; repeat++ {
		for _, scenario := range []string{"vulnerable", "fixed"} {
			start := time.Now()
			var run platform.Run
			if err := o.Call(ctx, "POST", "/v1/runs", map[string]string{"scenario": scenario, "execution": "live", "required_price_policy": o.RequiredPricePolicy}, &run); err != nil {
				return evaluation, err
			}
			if log != nil {
				fmt.Fprintf(log, "EVAL_CREATED run=%s scenario=%s\n", run.ID, scenario)
			}
			waitCtx, cancel := context.WithTimeout(ctx, 100*time.Second)
			run, err := Wait(waitCtx, o, run.ID, func(r platform.Run) bool {
				return r.State == platform.Completed || r.State == platform.Failed || r.State == platform.Stopped
			})
			cancel()
			if err != nil {
				return evaluation, err
			}
			expected := "confirmed"
			if scenario == "fixed" {
				expected = "no_issue_observed"
			}
			actual := "inconclusive"
			if run.Result != nil {
				actual = run.Result.Verdict
			}
			match := run.State == platform.Completed && actual == expected
			passed = passed && match
			result := EvaluationResult{RunID: run.ID, Scenario: scenario, Expected: expected, Actual: actual, State: run.State, Match: match, DurationMS: time.Since(start).Milliseconds(), Mode: run.Mode, Usage: run.Model}
			evaluation.Results = append(evaluation.Results, result)
			stem := fmt.Sprintf("%02d-%s", repeat+1, scenario)
			data, err := json.MarshalIndent(run, "", "  ")
			if err != nil {
				return evaluation, err
			}
			if err = os.WriteFile(filepath.Join(out, stem+".json"), data, 0600); err != nil {
				return evaluation, err
			}
			if err = os.WriteFile(filepath.Join(out, stem+".md"), []byte(platform.Report(run)), 0600); err != nil {
				return evaluation, err
			}
			if log != nil {
				fmt.Fprintf(log, "EVAL run=%s %-10s expected=%-18s actual=%-18s match=%t mode=%s\n", run.ID, scenario, expected, actual, match, run.Mode)
			}
		}
	}
	data, err := json.MarshalIndent(evaluation, "", "  ")
	if err != nil {
		return evaluation, err
	}
	if err = os.WriteFile(filepath.Join(out, "evaluation.json"), data, 0600); err != nil {
		return evaluation, err
	}
	if !passed {
		return evaluation, errors.New("evaluation included mismatches; inspect retained reports")
	}
	return evaluation, nil
}

// ModelFixture exercises the real provider SDK, API transport, broker and
// Mecatl loop locally. All inference responses are deterministic fixture data.
func ModelFixture(ctx context.Context, out string, log io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	model := &inference.Fixture{}
	modelHTTP := httptest.NewServer(model)
	defer modelHTTP.Close()
	v := httptest.NewServer(&lab.Server{})
	defer v.Close()
	f := httptest.NewServer(&lab.Server{Fixed: true})
	defer f.Close()
	vt, err := platform.LabTarget(ctx, v.URL)
	if err != nil {
		return err
	}
	ft, err := platform.LabTarget(ctx, f.URL)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "tsunami-model-fixture-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	store, err := platform.OpenStore(filepath.Join(dir, "state.db"))
	if err != nil {
		return err
	}
	defer store.Close()
	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return err
	}
	token := hex.EncodeToString(key)
	server := platform.NewServer(store, map[string]platform.Target{"vulnerable": vt, "fixed": ft}, map[string]string{token: "demo-a"})
	server.Inference, err = inference.NewConfig("fixture", "tsunami-wire-fixture", modelHTTP.URL+"/v1/", "")
	if err != nil {
		return err
	}
	api := httptest.NewServer(server)
	defer api.Close()
	server.Scheduler = &worker.LocalScheduler{ClientURL: api.URL, Log: log, OnError: func(id string, _ error) { _ = store.Fail(id) }}
	go server.Reconcile(ctx)
	_, err = Evaluate(ctx, Operator{BaseURL: api.URL, Token: token}, out, 1, log)
	if err == nil && log != nil {
		fmt.Fprintf(log, "PASS model wire fixture: %d upstream requests; no live model, key, or bill\n", model.Requests.Load())
	}
	return err
}
