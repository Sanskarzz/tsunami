package demo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/platform"
)

func TestCollectorUsesPagedMetadataAndFiltersMixedHistory(t *testing.T) {
	older := platform.Run{ID: strings.Repeat("a", 32), Tenant: "a", Scenario: "vulnerable", State: platform.Completed, CreatedAt: time.Unix(1, 0)}
	newer := older
	newer.ID = strings.Repeat("b", 32)
	newer.CreatedAt = time.Unix(2, 0)
	fixed := older
	fixed.ID = strings.Repeat("c", 32)
	fixed.Scenario = "fixed"
	app := older
	app.ID = strings.Repeat("d", 32)
	app.Scenario = "application"
	app.Application = &application.Plan{ID: "historical-app"}
	listed := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/runs" {
			listed++
			if r.URL.Query().Get("cursor") == "" {
				_ = json.NewEncoder(w).Encode(platform.RunPage{Items: []platform.RunSummary{{ID: app.ID, Scenario: app.Scenario}, {ID: older.ID, Scenario: older.Scenario, CreatedAt: older.CreatedAt}}, NextCursor: older.ID})
			} else {
				_ = json.NewEncoder(w).Encode(platform.RunPage{Items: []platform.RunSummary{{ID: newer.ID, Scenario: newer.Scenario, CreatedAt: newer.CreatedAt}, {ID: fixed.ID, Scenario: fixed.Scenario}}})
			}
			return
		}
		for _, run := range []platform.Run{newer, fixed} {
			if r.URL.Path == "/v1/runs/"+run.ID {
				_ = json.NewEncoder(w).Encode(run)
				return
			}
		}
		t.Error("collector fetched an unselected run")
		w.WriteHeader(500)
	}))
	defer api.Close()
	out := filepath.Join(t.TempDir(), "reports")
	if err := collectReports(context.Background(), Operator{BaseURL: api.URL}, out, nil, CollectOptions{Execution: "scripted"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "vulnerable.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got platform.Run
	_ = json.Unmarshal(data, &got)
	if got.ID != newer.ID || listed != 2 {
		t.Fatal("latest legacy selection or pagination changed")
	}
	files, _ := os.ReadDir(out)
	if len(files) != 4 {
		t.Fatal("application history contaminated legacy collection")
	}
}

func TestExactApplicationCollectionSkipsHistoryListing(t *testing.T) {
	run := platform.Run{ID: strings.Repeat("a", 32), Scenario: "application", Execution: "scripted", State: platform.Failed, Application: &application.Plan{ID: "historical-app"}}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/runs/"+run.ID {
			t.Error("exact-ID collection attempted a history download")
			w.WriteHeader(500)
			return
		}
		_ = json.NewEncoder(w).Encode(run)
	}))
	defer api.Close()
	out := filepath.Join(t.TempDir(), "reports")
	if err := collectReports(context.Background(), Operator{BaseURL: api.URL}, out, nil, CollectOptions{Execution: "scripted", IDs: []string{run.ID}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, run.ID+".md")); err != nil {
		t.Fatal(err)
	}
}
