package platform

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sanskarzz/tsunami/internal/assessment"
)

// MissingEvidence is trusted feedback, not authority to widen the procedure.
// It contains only already-approved account/path pairs, never credentials.
type MissingEvidence struct {
	Reads []assessment.Read `json:"required_reads"`
}

func (e *MissingEvidence) Error() string {
	return "assessment prerequisites incomplete; collect the approved required reads before retrying validation"
}

func missingAssessmentReads(r Run) []assessment.Read {
	missing := []assessment.Read{}
	if r.Assessment == nil {
		return missing
	}
	for i, read := range r.Assessment.Reads() {
		var latest *Capture
		for j := range r.Captures {
			c := &r.Captures[j]
			if c.URL == strings.TrimRight(r.Target, "/")+read.Path && c.Account == read.Account {
				latest = c
			}
		}
		valid := latest != nil && !latest.Blocked && latest.Error == "" && latest.Status > 0
		if i < len(r.Assessment.Baselines) {
			var actual assessment.Object
			valid = valid && latest.Status == 200 && read.Object != nil && json.Unmarshal([]byte(latest.Body), &actual) == nil && actual == *read.Object
		}
		if !valid {
			missing = append(missing, assessment.Read{Account: read.Account, Path: read.Path})
		}
	}
	return missing
}

type ValidationDetails struct {
	BaselinesRequired int    `json:"baselines_required"`
	BaselinesVerified int    `json:"baselines_verified"`
	ProbeCaptured     bool   `json:"probe_captured"`
	ExpectedAccess    string `json:"expected_access"`
}

func validateAssessment(r Run) Result {
	d := r.Assessment
	out := Result{Verdict: "inconclusive", Title: d.Title, Reason: "Complete approved baseline and probe evidence is required.", EvidenceIDs: []string{}, Validation: &ValidationDetails{BaselinesRequired: len(d.Baselines), ExpectedAccess: d.ExpectedAccess}}
	if !d.Canonical() || d.Digest() != r.AssessmentDigest {
		out.Reason = "Stored procedure differs from the current version; do not reinterpret old evidence."
		return out
	}
	last := func(read assessment.Read) *Capture {
		var found *Capture
		for i := range r.Captures {
			c := &r.Captures[i]
			if c.URL == strings.TrimRight(r.Target, "/")+read.Path && c.Account == read.Account {
				found = c
			}
		}
		return found
	}
	match := func(c *Capture, expected *assessment.Object) bool {
		if c == nil || c.Blocked || c.Error != "" || c.Status != 200 || expected == nil {
			return false
		}
		var actual assessment.Object
		return json.Unmarshal([]byte(c.Body), &actual) == nil && actual == *expected
	}
	for _, baseline := range d.Baselines {
		c := last(baseline)
		if c != nil {
			out.EvidenceIDs = append(out.EvidenceIDs, c.ID)
		}
		if match(c, baseline.Object) {
			out.Validation.BaselinesVerified++
		}
	}
	probe := last(d.Probe)
	if probe != nil {
		out.EvidenceIDs = append(out.EvidenceIDs, probe.ID)
		out.Validation.ProbeCaptured = !probe.Blocked && probe.Error == "" && probe.Status > 0
	}
	if out.Validation.BaselinesVerified != len(d.Baselines) {
		out.Reason = "One or more known-owner or legitimate-role baselines failed; no authorization conclusion."
		return out
	}
	if probe == nil || probe.Blocked || probe.Error != "" {
		out.Reason = "The probe is missing, denied by the broker, or interrupted; no target authorization conclusion."
		return out
	}
	if d.ExpectedAccess == "unknown" || d.Probe.Object == nil {
		out.Reason = "The probe uses an object with no confirmed existence. Its rejection cannot prove ownership enforcement."
		return out
	}
	if match(probe, d.Probe.Object) {
		if d.ExpectedAccess == "allow" {
			out.Verdict = "no_issue_observed"
			out.Reason = "The captured shared-object response matches the explicitly approved access rule. This is a positive control, not a finding."
		} else {
			out.Verdict = "confirmed"
			out.Reason = "The probe returned the exact protected fixture object despite the reviewed deny rule; legitimate baselines also succeeded."
		}
		return out
	}
	if d.ExpectedAccess == "deny" && (probe.Status == 401 || probe.Status == 403 || probe.Status == 404) {
		out.Verdict = "no_issue_observed"
		out.Reason = fmt.Sprintf("Known immutable fixture baselines succeeded and the prohibited read returned HTTP %d. This applies only to this procedure, not application-wide security.", probe.Status)
		return out
	}
	out.Reason = "The status or response body does not establish protected-data exposure or the approved successful access. A generic HTTP 200 is insufficient."
	return out
}
