package demo

import (
	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/Sanskarzz/tsunami/internal/platform"
)

// classifyAssessment describes the recorded outcome, never changes a verdict
// or excuses a missed planted case. Generic historical errors stay generic.
func classifyAssessment(r platform.Run, expected string) (category, code string) {
	if r.State == platform.Stopped {
		return "stopped", ""
	}
	if r.State == platform.Failed {
		if r.Model != nil {
			if n := len(r.Model.Calls); n > 0 {
				call := r.Model.Calls[n-1]
				if call.Outcome == "failed" {
					return "model_failure", safeDiagnosticCode(call.Code)
				}
				if call.FinishedAt == nil {
					return "model_call_unfinished", "unknown"
				}
				return "worker_failure", "unknown"
			}
			for _, event := range r.Events {
				if event.Kind == "model_error" {
					code := safeDiagnosticCode(event.Detail)
					if code == "model_error" || code == "model_cancelled_or_deadline" {
						return "model_failure_unclassified", code
					}
					return "model_failure", code
				}
			}
		}
		return "worker_failure", "unknown"
	}
	if r.State != platform.Completed {
		return "unfinished", "unknown"
	}
	if r.Result == nil {
		return "incomplete_assessment", ""
	}
	if v := r.Result.Validation; v != nil && (v.BaselinesVerified != v.BaselinesRequired || !v.ProbeCaptured) {
		return "incomplete_assessment", ""
	}
	if r.Result.Verdict == "confirmed" && expected != "confirmed" {
		return "false_positive", ""
	}
	if r.Result.Verdict != expected {
		return "assessment_mismatch", ""
	}
	if expected == "inconclusive" {
		return "expected_inconclusive", ""
	}
	if expected == "confirmed" {
		return "confirmed", ""
	}
	return "expected_access_observed", ""
}

func safeDiagnosticCode(code string) string {
	// Imported records can contain fields from newer versions. Do not reflect
	// arbitrary strings from JSON into an operator report.
	if code == "unknown" || inference.KnownErrorCode(code) {
		return code
	}
	return "model_error"
}

func operational(category string) bool {
	switch category {
	case "model_failure", "model_failure_unclassified", "model_call_unfinished", "worker_failure", "unfinished":
		return true
	}
	return false
}
