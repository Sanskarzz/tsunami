package demo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/Sanskarzz/tsunami/internal/platform"
)

func analysisRun(t *testing.T, id string) platform.Run {
	t.Helper()
	d, err := assessment.Get("opaque-ownership/v1")
	if err != nil {
		t.Fatal(err)
	}
	return platform.Run{ID: strings.Repeat(id, 32), Scenario: "vulnerable", State: platform.Completed, Assessment: &d, AssessmentDigest: d.Digest(), Result: &platform.Result{Verdict: "inconclusive", Validation: &platform.ValidationDetails{BaselinesRequired: 2, ProbeCaptured: true}}}
}

func TestArchivedAnalysisPreservesVerdictsAndUnknownCauses(t *testing.T) {
	source := t.TempDir()
	out := t.TempDir()
	incomplete := analysisRun(t, "a")
	failed := analysisRun(t, "b")
	failed.State = platform.Failed
	failed.Model = &platform.ModelUsage{}
	failed.Events = []platform.Event{{Kind: "model_error", Detail: "model_error"}}
	before := map[string][]byte{}
	for _, r := range []platform.Run{incomplete, failed} {
		data, _ := json.Marshal(r)
		path := filepath.Join(source, r.ID+".json")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		before[path] = data
	}
	report, err := AnalyzeCatalogDirectory(source, out)
	if err != nil {
		t.Fatal(err)
	}
	if report.Matched != 0 || report.MissedVulnerable != 2 || report.OperationalFailures != 1 || report.AssessmentMisses != 1 || report.IncompleteAssessments != 1 || report.SourceRecordsChanged || report.HistoricalCausesInferred {
		t.Fatalf("misleading analysis: %+v", report)
	}
	for _, item := range report.Items {
		if item.RunID == failed.ID && (item.Category != "model_failure_unclassified" || item.FailureCode != "model_error") {
			t.Fatal("historical cause invented")
		}
	}
	for path, data := range before {
		after, err := os.ReadFile(path)
		if err != nil || string(after) != string(data) {
			t.Fatal("historical record changed")
		}
	}
}

func TestAnalysisRejectsOutputsInsideArchiveIncludingSymlinks(t *testing.T) {
	source := t.TempDir()
	inside := filepath.Join(source, "new-analysis")
	if _, err := AnalyzeCatalogDirectory(source, inside); err == nil {
		t.Fatal("archive output accepted")
	}
	if _, err := os.Stat(inside); !os.IsNotExist(err) {
		t.Fatal("rejected analysis still modified source directory")
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := AnalyzeCatalogDirectory(source, filepath.Join(alias, "new-analysis")); err == nil {
		t.Fatal("symlink bypass accepted")
	}
}

func TestAnalysisDoesNotFollowExistingOutputFileLink(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			source, out := t.TempDir(), t.TempDir()
			r := analysisRun(t, "a")
			data, _ := json.Marshal(r)
			path := filepath.Join(source, "run.json")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			link := os.Link
			if kind == "symlink" {
				link = os.Symlink
			}
			if err := link(path, filepath.Join(out, "analysis.json")); err != nil {
				t.Fatal(err)
			}
			if _, err := AnalyzeCatalogDirectory(source, out); err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(data) {
				t.Fatal("existing output link modified archive")
			}
		})
	}
}

func TestFailureClassificationDoesNotBlameRecoveredModelCall(t *testing.T) {
	r := analysisRun(t, "a")
	r.State = platform.Failed
	now := time.Now()
	r.Model = &platform.ModelUsage{Calls: []inference.CallRecord{{Outcome: "failed", Code: "upstream_http_503", FinishedAt: &now}, {Outcome: "succeeded", Code: "ok", FinishedAt: &now}}}
	category, _ := classifyAssessment(r, "confirmed")
	if category != "worker_failure" {
		t.Fatal("earlier recovered error became terminal cause")
	}
	r.Model.Calls = r.Model.Calls[:1]
	category, code := classifyAssessment(r, "confirmed")
	if category != "model_failure" || code != "upstream_http_503" {
		t.Fatal("known terminal failure lost")
	}
	r.Model.Calls[0].Code = inference.FaultCanary
	_, code = classifyAssessment(r, "confirmed")
	if code != "model_error" {
		t.Fatal("imported error text reflected")
	}
}
