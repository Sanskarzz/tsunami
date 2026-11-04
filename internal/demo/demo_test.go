package demo

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRealMecatlLabWorkflow(t *testing.T) {
	dir := t.TempDir()
	if err := Local(context.Background(), dir, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var summary Summary
	if err = json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Checks) != 5 {
		t.Fatalf("only %d checks completed", len(summary.Checks))
	}
	if summary.Checks[0].Verdict != "confirmed" || summary.Checks[1].Verdict != "no_issue_observed" || summary.Checks[2].Verdict != "inconclusive" {
		t.Fatal("finding validation regressed")
	}
}

func TestRealAdapterBrokerMecatlWorkflow(t *testing.T) {
	dir := t.TempDir()
	if err := ModelFixture(context.Background(), dir, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "evaluation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var evaluation Evaluation
	if err = json.Unmarshal(data, &evaluation); err != nil {
		t.Fatal(err)
	}
	if len(evaluation.Results) != 2 {
		t.Fatal("missing paired evaluation")
	}
	for _, result := range evaluation.Results {
		if !result.Match || result.Usage == nil || result.Usage.Provider != "fixture" || result.Usage.CallsReserved != 5 || result.Usage.WireAttempts != 5 || result.Usage.UsageUncertain {
			t.Fatalf("unexpected result: %+v", result)
		}
	}
}
