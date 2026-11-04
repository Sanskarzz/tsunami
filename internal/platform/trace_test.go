package platform

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sanskarzz/tsunami/internal/inference"
)

func TestCallRecordsSurviveRestartAndMissingUsageStaysUncertain(t *testing.T) {
	s, store := fixture(t)
	provider := httptest.NewServer(&inference.FaultFixture{Mode: "missing_usage"})
	defer provider.Close()
	cfg, err := inference.NewConfig("fixture", "fixture", provider.URL+"/v1/", "")
	if err != nil {
		t.Fatal(err)
	}
	r := liveRun(t, s, cfg)
	if _, err = s.Infer(context.Background(), r.ID, modelInput()); err != nil {
		t.Fatal(err)
	}
	before, _ := s.Snapshot(r.ID)
	if len(before.Model.Calls) != 1 || before.Model.Calls[0].Code != "ok" || !before.Model.UsageUncertain || before.Model.Calls[0].UsageAccepted || before.Model.CompletedCalls != 1 {
		t.Fatal("omitted provider usage became verified zero")
	}
	path := store.db.Path()
	_ = store.Close()
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after, err := reopened.Get(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(before.Model)
	b, _ := json.Marshal(after.Model)
	if string(a) != string(b) {
		t.Fatal("durable diagnostic changed after restart")
	}
	if strings.Contains(Report(after), inference.FaultCanary) {
		t.Fatal("private provider text rendered")
	}
}
