package demo

import (
	"context"
	"io"
	"testing"
)

func TestCatalogThroughMecatlScriptAndRealResponsesAdapter(t *testing.T) {
	for _, fixture := range []bool{false, true} {
		name := "scripted"
		if fixture {
			name = "offline Responses fixture"
		}
		t.Run(name, func(t *testing.T) {
			summary, err := CatalogLocal(context.Background(), t.TempDir(), fixture, 1, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if len(summary.Results) != 10 || summary.Matched != 10 || summary.ConfirmedVulnerable != 3 || summary.MissedVulnerable != 0 || summary.FalsePositives != 0 || summary.Inconclusive != 2 || summary.IncompleteEvidence != 0 {
				t.Fatalf("unexpected catalog metrics: %+v", summary)
			}
			for _, r := range summary.Results {
				if fixture && (r.Model == nil || r.Model.Provider != "fixture" || r.Model.WireAttempts != 5) {
					t.Fatal("offline evaluation bypassed real model adapter")
				}
				if !fixture && r.Model != nil {
					t.Fatal("scripted evaluation requested a model")
				}
			}
		})
	}
}

func TestPrematureModelFinishRecoversThroughTrustedFeedback(t *testing.T) {
	summary, err := CatalogLocal(context.Background(), t.TempDir(), true, 1, io.Discard, true)
	if err != nil {
		t.Fatalf("%v; results: %+v", err, summary.Results)
	}
	if summary.Matched != 10 || summary.IncompleteEvidence != 0 || summary.FalsePositives != 0 {
		t.Fatalf("feedback recovery failed: %+v", summary)
	}
	for _, r := range summary.Results {
		if r.Model == nil || r.Model.WireAttempts != 5 || r.CompletionFeedback != 1 {
			t.Fatal("unbounded or bypassed model recovery")
		}
	}
}
