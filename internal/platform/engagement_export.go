package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

type EvidenceArtifact struct {
	ID        string `json:"id"`
	RunID     string `json:"run_id"`
	CaptureID string `json:"capture_id"`
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Data      []byte `json:"data_base64"`
}

// A validated lab observation still needs impact/severity review. It is not
// exported as a toolkit Finding with an invented CVSS score or remediation state.
type FindingCandidate struct {
	Digest      string         `json:"candidate_digest,omitempty"`
	Review      *FindingReview `json:"review,omitempty"`
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Procedure   string         `json:"procedure"`
	Target      string         `json:"target"`
	Status      string         `json:"status"`
	Reason      string         `json:"reason"`
	Remediation string         `json:"remediation"`
	RunIDs      []string       `json:"run_ids"`
	EvidenceIDs []string       `json:"evidence_ids"`
}

type EngagementBundle struct {
	Findings          []ToolkitFinding   `json:"toolkit_findings"`
	Evidence          []ToolkitEvidence  `json:"toolkit_evidence"`
	Version           string             `json:"version"`
	SyntheticOnly     bool               `json:"synthetic_only"`
	View              EngagementView     `json:"view"`
	ToolkitEngagement map[string]any     `json:"toolkit_engagement"`
	Candidates        []FindingCandidate `json:"finding_candidates"`
	Artifacts         []EvidenceArtifact `json:"artifacts"`
	Report            string             `json:"report_markdown"`
	Limitations       []string           `json:"limitations"`
}

func (s *Store) ExportEngagement(tenant, id string) (EngagementBundle, error) {
	var bundle EngagementBundle
	err := s.db.View(func(tx *bolt.Tx) error {
		var err error
		bundle, err = buildEngagementBundle(tx, tenant, id, true)
		return err
	})
	return bundle, err
}

func buildEngagementBundle(tx *bolt.Tx, tenant, id string, reviews bool) (EngagementBundle, error) {
	bundle := EngagementBundle{Version: "tsunami.engagement-export/v2", SyntheticOnly: true, Candidates: []FindingCandidate{}, Artifacts: []EvidenceArtifact{}, Findings: []ToolkitFinding{}, Evidence: []ToolkitEvidence{}, Limitations: []string{"Owned synthetic lab data only; no general customer readiness claim.", "Checklist progress covers selected Tsunami procedures, not complete OWASP coverage.", "Only matching explicit reviews produce rated findings. Test-generated reviews are labeled.", "Hashes identify exported bytes; they are not an independently signed or immutable evidence store."}}
	err := func() error {
		e, err := getEngagement(tx, tenant, id)
		if err != nil {
			return err
		}
		view, runs, err := engagementView(tx, e)
		if err != nil {
			return err
		}
		bundle.View = view
		byProcedure := map[string]*FindingCandidate{}
		for _, r := range runs {
			refs := map[string]string{}
			for _, capture := range r.Captures {
				data, err := json.Marshal(struct {
					EngagementID string  `json:"engagement_id"`
					RunID        string  `json:"run_id"`
					Procedure    string  `json:"procedure"`
					Capture      Capture `json:"capture"`
				}{e.ID, r.ID, r.ProcedureID(), capture})
				if err != nil {
					return err
				}
				sum := sha256.Sum256(data)
				digest := hex.EncodeToString(sum[:])
				artifactID := "E-" + digest
				refs[capture.ID] = artifactID
				bundle.Artifacts = append(bundle.Artifacts, EvidenceArtifact{ID: artifactID, RunID: r.ID, CaptureID: capture.ID, Path: "evidence/" + digest + ".json", SHA256: digest, Data: data})
			}
			if r.State != Completed {
				continue
			}
			result := Validate(r)
			if result.Verdict != "confirmed" {
				continue
			}
			candidate := byProcedure[r.ProcedureID()]
			if candidate == nil {
				sum := sha256.Sum256([]byte(e.ID + "\n" + r.ProcedureDigest() + "\n" + e.Target))
				candidate = &FindingCandidate{ID: "F-" + hex.EncodeToString(sum[:16]), Title: result.Title, Procedure: r.ProcedureID(), Target: e.Target, Status: "needs_human_review", Reason: result.Reason, Remediation: r.ProcedureRemediation(), RunIDs: []string{}, EvidenceIDs: []string{}}
				byProcedure[r.ProcedureID()] = candidate
			}
			candidate.RunIDs = append(candidate.RunIDs, r.ID)
			for _, captureID := range result.EvidenceIDs {
				if artifactID, ok := refs[captureID]; ok {
					candidate.EvidenceIDs = append(candidate.EvidenceIDs, artifactID)
				}
			}
		}
		for _, candidate := range byProcedure {
			candidate.Digest = candidateDigest(*candidate)
			bundle.Candidates = append(bundle.Candidates, *candidate)
		}
		sort.Slice(bundle.Candidates, func(i, j int) bool { return bundle.Candidates[i].ID < bundle.Candidates[j].ID })
		status := "On-Hold"
		if e.State == EngagementDraft {
			status = "Planning"
		} else if e.State == EngagementActive && time.Now().Before(e.ExpiresAt) {
			status = "Testing"
		}
		g := tx.Bucket([]byte("gates"))
		authorized := e.State == EngagementActive && e.ApprovedDigest == e.ScopeDigest && e.digest() == e.ScopeDigest && time.Now().Before(e.ExpiresAt) && g.Get([]byte("global")) == nil && g.Get([]byte("tenant:"+tenant)) == nil
		if !authorized && status == "Testing" {
			status = "On-Hold"
		}
		bundle.ToolkitEngagement = map[string]any{"id": e.ID, "name": e.Name, "target": e.Target, "target_type": "api", "status": status, "authorized": authorized, "authorization_note": e.AuthorizationRef, "scope_in": []string{e.Target}, "start_date": e.CreatedAt.Format("2006-01-02"), "findings": []any{}, "checklist_progress": map[string]any{"tsunami_authorization_v1": map[string]int{"total": view.Progress.Total, "completed": view.Progress.Completed, "skipped": view.Progress.Skipped, "not_applicable": 0}}, "notes": "Synthetic lab export. Procedure completion is not application-wide security assurance. Unrated candidates are separate from findings.", "metadata": map[string]any{"scope_digest": e.ScopeDigest, "finding_candidates": len(bundle.Candidates), "authorization_enforced_by": "Tsunami durable state, not this exported boolean"}}
		return nil
	}()
	if err != nil {
		return bundle, err
	}
	if reviews {
		if err := applyFindingReviews(tx, &bundle); err != nil {
			return bundle, err
		}
	}
	bundle.Report = engagementReport(bundle)
	return bundle, nil
}

func engagementReport(b EngagementBundle) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# Tsunami engagement draft\n\n%s\n\nEngagement: `%s`; state: %s. Synthetic targets only.\n\n", markdownText(b.View.Engagement.Name), b.View.Engagement.ID, b.View.Engagement.State)
	p := b.View.Progress
	fmt.Fprintf(&out, "Selected procedures: %d; completed: %d; skipped: %d; pending: %d; running: %d; failed: %d; stopped: %d; incomplete evidence: %d.\n\nCompletion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.\n\n", p.Total, p.Completed, p.Skipped, p.Pending, p.Running, p.Failed, p.Stopped, p.Inconclusive)
	out.WriteString("| Procedure | Progress | Verdict |\n|---|---|---|\n")
	for _, item := range b.View.Checklist {
		fmt.Fprintf(&out, "| %s | %s | %s |\n", item.Procedure, item.Status, item.Verdict)
	}
	out.WriteString("\n## Finding candidates\n\nCandidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.\n\n")
	for _, f := range b.Candidates {
		rating := "unrated"
		if f.Review != nil && f.Review.Decision == "confirm" {
			rating = f.Review.Severity + " (see review provenance below)"
		}
		fmt.Fprintf(&out, "### %s\n\n%s\n\nStatus: %s; severity: %s.\n\n%s\n\nRemediation: %s\n\nEvidence IDs: %s\n\n", f.ID, markdownText(f.Title), f.Status, rating, markdownText(f.Reason), markdownText(f.Remediation), strings.Join(f.EvidenceIDs, ", "))
	}
	if len(b.Candidates) == 0 {
		out.WriteString("No confirmed candidate in the captured runs. This does not establish absence of vulnerabilities.\n\n")
	}
	if len(b.Findings) > 0 {
		out.WriteString("## Reviewed findings\n\n")
		for _, f := range b.Findings {
			fmt.Fprintf(&out, "### %s: %s\n\nSeverity: %s; CVSS 3.1 base score: %.1f; vector: `%s`.\n\nReview purpose: %s. An automated lab review is a test, not human signoff.\n\nImpact supplied by reviewer: %s\n\n", f.ID, markdownText(f.Title), f.Severity, f.CVSSScore, f.CVSSVector, strings.Join(f.Tags, ", "), markdownText(f.Impact))
		}
	}
	out.WriteString("## Evidence index\n\n| Evidence | Run | Artifact path |\n|---|---|---|\n")
	for _, a := range b.Artifacts {
		fmt.Fprintf(&out, "| %s | %s | %s |\n", a.ID, a.RunID, a.Path)
	}
	out.WriteString("\nThis is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.\n")
	return out.String()
}
