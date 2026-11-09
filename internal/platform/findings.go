package platform

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/scoring"
	bolt "go.etcd.io/bbolt"
)

type ReviewInput struct {
	CandidateID      string `json:"candidate_id"`
	CandidateDigest  string `json:"candidate_digest"`
	ExpectedRevision int    `json:"expected_revision"`
	Decision         string `json:"decision"`
	Purpose          string `json:"purpose"`
	Rationale        string `json:"rationale"`
	Impact           string `json:"impact,omitempty"`
	CVSSVector       string `json:"cvss_vector,omitempty"`
}
type FindingReview struct {
	ReviewInput
	Revision   int       `json:"revision"`
	ReviewedAt time.Time `json:"reviewed_at"`
	Actor      string    `json:"actor"`
	CVSSScore  *float64  `json:"cvss_score,omitempty"`
	Severity   string    `json:"severity,omitempty"`
}
type ToolkitEvidence struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	FindingID   string    `json:"finding_id"`
	CapturedAt  time.Time `json:"captured_at"`
	Tool        string    `json:"tool"`
}
type ToolkitFinding struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Severity     string            `json:"severity"`
	CVSSScore    float64           `json:"cvss_score"`
	CVSSVector   string            `json:"cvss_vector"`
	Status       string            `json:"status"`
	Category     string            `json:"category"`
	CWEID        int               `json:"cwe_id"`
	Description  string            `json:"description_flaw"`
	Impact       string            `json:"description_impact"`
	Steps        []string          `json:"steps_to_reproduce"`
	Evidence     []ToolkitEvidence `json:"evidence"`
	Remediation  string            `json:"remediation"`
	Target       string            `json:"target"`
	EngagementID string            `json:"engagement_id"`
	FoundAt      time.Time         `json:"found_at"`
	FoundBy      string            `json:"found_by"`
	Tags         []string          `json:"tags"`
}

func candidateDigest(c FindingCandidate) string {
	c.Digest = ""
	c.Review = nil
	c.Status = "needs_human_review"
	data, _ := json.Marshal(c)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (s *Store) ReviewFinding(tenant, engagementID string, in ReviewInput) (FindingReview, error) {
	var review FindingReview
	if (in.Decision != "confirm" && in.Decision != "false_positive") || (in.Purpose != "operator_review" && in.Purpose != "automated_lab_test") || !shortText(in.Rationale, 1024) || in.ExpectedRevision < 0 {
		return review, errors.New("explicit review purpose, decision and rationale required")
	}
	if in.Decision == "confirm" {
		if !shortText(in.Impact, 2048) {
			return review, errors.New("reviewed impact required")
		}
		score, severity, err := scoring.Base(in.CVSSVector)
		if err != nil {
			return review, err
		}
		review.CVSSScore = &score
		review.Severity = severity
	} else if in.Impact != "" || in.CVSSVector != "" {
		return review, errors.New("false-positive disposition must not invent a score")
	}
	err := s.db.Update(func(tx *bolt.Tx) error {
		bundle, err := buildEngagementBundle(tx, tenant, engagementID, false)
		if err != nil {
			return err
		}
		if bundle.View.Engagement.State != EngagementStopped {
			return errors.New("stop the engagement before final finding review")
		}
		found := false
		for _, candidate := range bundle.Candidates {
			if candidate.ID == in.CandidateID && candidate.Digest == in.CandidateDigest {
				found = true
			}
		}
		if !found {
			return errors.New("candidate or evidence fingerprint unavailable or changed")
		}
		key := engagementID + ":" + in.CandidateID
		bucket := tx.Bucket([]byte("finding_reviews"))
		revision := 0
		if data := bucket.Get([]byte(key)); data != nil {
			var prior FindingReview
			if json.Unmarshal(data, &prior) != nil {
				return errors.New("invalid prior finding review")
			}
			revision = prior.Revision
		}
		if revision != in.ExpectedRevision {
			return errors.New("finding review revision changed")
		}
		review.ReviewInput = in
		review.Revision = revision + 1
		review.ReviewedAt = time.Now().UTC()
		review.Actor = "tenant_operator"
		if in.Purpose == "automated_lab_test" {
			review.Actor = "automated_lab_verifier"
		}
		data, err := json.Marshal(review)
		if err != nil {
			return err
		}
		historyKey := fmt.Sprintf("%s:%08d", key, review.Revision)
		if tx.Bucket([]byte("review_history")).Get([]byte(historyKey)) != nil {
			return errors.New("review history conflict")
		}
		if err := tx.Bucket([]byte("review_history")).Put([]byte(historyKey), data); err != nil {
			return err
		}
		return bucket.Put([]byte(key), data)
	})
	return review, err
}

// Shared by exports and metadata reads. Only an exact current evidence digest
// can acquire stored review provenance/rating; a read never creates a review.
func loadCandidateReview(tx *bolt.Tx, engagementID string, c *FindingCandidate) error {
	data := tx.Bucket([]byte("finding_reviews")).Get([]byte(engagementID + ":" + c.ID))
	if data == nil {
		return nil
	}
	var review FindingReview
	if json.Unmarshal(data, &review) != nil {
		return errors.New("invalid finding review")
	}
	if review.CandidateDigest != c.Digest {
		return nil
	}
	if review.Decision == "false_positive" {
		c.Review = &review
		c.Status = "reviewed_false_positive"
		return nil
	}
	calculated, severity, err := scoring.Base(review.CVSSVector)
	if err != nil || review.CVSSScore == nil || *review.CVSSScore != calculated || review.Severity != severity {
		return errors.New("stored finding rating invalid")
	}
	c.Review = &review
	c.Status = "reviewed_confirmed"
	return nil
}

func applyFindingReviews(tx *bolt.Tx, b *EngagementBundle) error {
	artifacts := map[string]EvidenceArtifact{}
	for _, a := range b.Artifacts {
		artifacts[a.ID] = a
	}
	index := []map[string]any{}
	for i := range b.Candidates {
		c := &b.Candidates[i]
		if err := loadCandidateReview(tx, b.View.Engagement.ID, c); err != nil {
			return err
		}
		if c.Review == nil || c.Review.Decision == "false_positive" {
			continue
		}
		review := *c.Review
		f := ToolkitFinding{ID: c.ID, Title: c.Title, Severity: review.Severity, CVSSScore: *review.CVSSScore, CVSSVector: review.CVSSVector, Status: "Confirmed", Category: "Broken Access Control", CWEID: 639, Description: c.Reason, Impact: review.Impact, Steps: []string{}, Evidence: []ToolkitEvidence{}, Remediation: c.Remediation, Target: c.Target, EngagementID: b.View.Engagement.ID, FoundBy: "trusted_http_validator", Tags: []string{"owned-lab", review.Purpose}}
		if plan := b.View.Engagement.Application; plan != nil && plan.Version == application.PropertyAccessProfile {
			f.Category = "Broken Object Property Level Authorization"
			f.CWEID = 213
		} else if c.Procedure == "role-boundary/v1" {
			f.Category = "Broken Function Level Authorization"
			f.CWEID = 862
		} else if c.Procedure == "anonymous-access/v1" {
			f.CWEID = 862
		} else {
			f.Category = "Broken Object Level Authorization"
		}
		for _, id := range c.EvidenceIDs {
			a, ok := artifacts[id]
			if !ok {
				return errors.New("reviewed evidence unavailable")
			}
			var payload struct {
				Capture Capture `json:"capture"`
			}
			if json.Unmarshal(a.Data, &payload) != nil {
				return errors.New("invalid evidence bytes")
			}
			cap := payload.Capture
			e := ToolkitEvidence{ID: a.ID, Type: "request_response", Path: a.Path, Description: "Broker-captured GET using account alias " + cap.Account, FindingID: c.ID, CapturedAt: cap.StartedAt, Tool: "tsunami-http-broker"}
			f.Evidence = append(f.Evidence, e)
			b.Evidence = append(b.Evidence, e)
			f.Steps = append(f.Steps, "Read "+cap.URL+" using account alias "+cap.Account+"; inspect captured status and proof fields.")
			if f.FoundAt.IsZero() || cap.FinishedAt.After(f.FoundAt) {
				f.FoundAt = cap.FinishedAt
			}
		}
		b.Findings = append(b.Findings, f)
		index = append(index, map[string]any{"id": f.ID, "title": f.Title, "severity": f.Severity, "status": f.Status, "cvss_score": f.CVSSScore})
	}
	b.ToolkitEngagement["findings"] = index
	return nil
}

func csvCell(value string) string {
	trimmed := strings.TrimLeftFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) })
	if (trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0]))) || strings.HasPrefix(value, "\t") || strings.HasPrefix(value, "\r") || strings.HasPrefix(value, "\n") {
		return "'" + value
	}
	return value
}

func FindingsCSV(w io.Writer, findings []ToolkitFinding) error {
	c := csv.NewWriter(w)
	if err := c.Write([]string{"id", "title", "severity", "cvss_score", "cvss_vector", "status", "target", "impact"}); err != nil {
		return err
	}
	for _, f := range findings {
		row := []string{f.ID, f.Title, f.Severity, fmt.Sprint(f.CVSSScore), f.CVSSVector, f.Status, f.Target, f.Impact}
		for i := range row {
			row[i] = csvCell(row[i])
		}
		if err := c.Write(row); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}
