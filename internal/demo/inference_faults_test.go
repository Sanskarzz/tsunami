package demo

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/Sanskarzz/tsunami/internal/inference"
)

func TestInferenceFaultCampaignPreservesBoundaries(t *testing.T) {
	summary, err := DiagnoseInference(context.Background(), t.TempDir(), io.Discard)
	if err != nil {
		t.Fatalf("%v; results: %+v", err, summary.Results)
	}
	if len(summary.Results) != 14 {
		t.Fatal("fault cases missing")
	}
	data, _ := json.Marshal(summary)
	for _, sensitive := range []string{inference.FixtureKey, inference.FaultCanary, "sensitive-provider-id", "Authorization", "Bearer ", "arguments"} {
		if strings.Contains(string(data), sensitive) {
			t.Fatal("sensitive provider data escaped into diagnostics")
		}
	}
	for _, result := range summary.Results {
		if !result.Passed || result.TargetRequests != 0 || result.ProviderAttempts != 1 {
			t.Fatal("unsafe or inconclusive diagnostic")
		}
	}
}
