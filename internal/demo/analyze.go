package demo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/platform"
)

type AnalysisItem struct {
	RunID       string `json:"run_id"`
	Assessment  string `json:"assessment"`
	Variant     string `json:"fixture_variant"`
	State       string `json:"stored_state"`
	Verdict     string `json:"stored_verdict"`
	Expected    string `json:"expected"`
	Match       bool   `json:"match"`
	Category    string `json:"outcome_category"`
	FailureCode string `json:"failure_code,omitempty"`
}
type Analysis struct {
	GeneratedAt              time.Time      `json:"generated_at"`
	Source                   string         `json:"source_label"`
	Items                    []AnalysisItem `json:"items"`
	Matched                  int            `json:"matched"`
	MissedVulnerable         int            `json:"missed_vulnerable"`
	OperationalFailures      int            `json:"operational_failures"`
	AssessmentMisses         int            `json:"assessment_misses"`
	IncompleteAssessments    int            `json:"incomplete_assessments"`
	StoppedRuns              int            `json:"stopped_runs"`
	SourceRecordsChanged     bool           `json:"source_records_changed"`
	HistoricalCausesInferred bool           `json:"historical_causes_inferred"`
}

// AnalyzeCatalogDirectory reads stored outcomes. It does not rerun validation,
// upgrade a verdict, or assign a missing historical gateway error cause.
func AnalyzeCatalogDirectory(source, out string) (Analysis, error) {
	report := Analysis{GeneratedAt: time.Now().UTC(), Source: filepath.Base(filepath.Clean(source)), Items: []AnalysisItem{}}
	input, err := filepath.Abs(source)
	if err != nil {
		return report, err
	}
	output, err := filepath.Abs(out)
	if err != nil {
		return report, err
	}
	// Resolve symlinks in existing ancestors so a differently spelled output
	// cannot overwrite the archived source directory.
	input, err = filepath.EvalSymlinks(input)
	if err != nil {
		return report, err
	}
	output, err = resolveOutput(output)
	if err != nil {
		return report, err
	}
	rel, err := filepath.Rel(input, output)
	if err != nil {
		return report, err
	}
	if rel == "." || (!strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != "..") {
		return report, errors.New("analysis output must be outside archived source")
	}
	if err = os.MkdirAll(output, 0700); err != nil {
		return report, err
	}
	files, err := os.ReadDir(input)
	if err != nil {
		return report, err
	}
	seen := map[string]bool{}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		if file.Type()&os.ModeSymlink != 0 {
			return report, errors.New("archive record symlinks are not supported")
		}
		info, err := file.Info()
		if err != nil {
			return report, err
		}
		if info.Size() > 4<<20 {
			return report, errors.New("archive record exceeds analysis limit")
		}
		data, err := os.ReadFile(filepath.Join(input, file.Name()))
		if err != nil {
			return report, err
		}
		var r platform.Run
		if json.Unmarshal(data, &r) != nil {
			return report, errors.New("invalid archive JSON")
		}
		if r.ID == "" || r.Assessment == nil {
			continue
		}
		if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(r.ID) || seen[r.ID] {
			return report, errors.New("invalid or duplicate archive run identity")
		}
		seen[r.ID] = true
		d, err := assessment.Get(r.Assessment.ID)
		if err != nil || d.Digest() != r.AssessmentDigest || !r.Assessment.Canonical() {
			return report, errors.New("unsupported archive procedure; refusing reinterpretation")
		}
		if r.Scenario != "vulnerable" && r.Scenario != "fixed" {
			return report, errors.New("unknown fixture variant")
		}
		switch r.State {
		case platform.Completed, platform.Failed, platform.Stopped, platform.Pending, platform.Running:
		default:
			return report, errors.New("unknown stored run state")
		}
		verdict := "inconclusive"
		if r.Result != nil {
			verdict = r.Result.Verdict
		}
		if verdict != "confirmed" && verdict != "no_issue_observed" && verdict != "inconclusive" {
			return report, errors.New("unknown stored verdict")
		}
		expected := catalogExpected(d.ID, r.Scenario)
		category, code := classifyAssessment(r, expected)
		match := r.State == platform.Completed && verdict == expected
		report.Items = append(report.Items, AnalysisItem{r.ID, d.ID, r.Scenario, r.State, verdict, expected, match, category, code})
		if match {
			report.Matched++
		}
		if expected == "confirmed" && !match {
			report.MissedVulnerable++
		}
		if operational(category) {
			report.OperationalFailures++
		} else if category != "stopped" && expected == "confirmed" && !match {
			report.AssessmentMisses++
		}
		if category == "incomplete_assessment" {
			report.IncompleteAssessments++
		}
		if category == "stopped" {
			report.StoppedRuns++
		}
	}
	if len(report.Items) == 0 {
		return report, errors.New("no procedure run records found")
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	// Replace the directory entry rather than following an existing output
	// symlink or truncating a hard-linked archived record.
	tmp, err := os.CreateTemp(output, ".analysis-*")
	if err != nil {
		return report, err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return report, err
	}
	if err = tmp.Close(); err != nil {
		return report, err
	}
	err = os.Rename(tmp.Name(), filepath.Join(output, "analysis.json"))
	return report, err
}

func resolveOutput(path string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(path)
	if parent == path {
		return "", errors.New("output ancestor unavailable")
	}
	resolved, err := resolveOutput(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, filepath.Base(path)), nil
}
