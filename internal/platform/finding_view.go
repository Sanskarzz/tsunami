package platform

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	bolt "go.etcd.io/bbolt"
)

type EvidenceCheck struct {
	Role       string `json:"role"`
	Account    string `json:"account"`
	Path       string `json:"path"`
	Outcome    string `json:"outcome"`
	CaptureID  string `json:"capture_id,omitempty"`
	EvidenceID string `json:"evidence_id,omitempty"`
}

func applicationChecks(r Run) []EvidenceCheck {
	checks := []EvidenceCheck{}
	if r.Application == nil || !r.ProcedureValid() {
		return checks
	}
	roles := []string{"owner_identity", "peer_identity", "owner_baseline", "peer_baseline", "peer_probe", "owner_recheck", "peer_identity_recheck"}
	for i, read := range r.Application.Reads() {
		checks = append(checks, EvidenceCheck{Role: roles[i], Account: read.Account, Path: read.Path, Outcome: "not_evaluated"})
	}
	return checks
}

func applicationProbeOutcome(r Run, c Capture) string {
	if c.Blocked {
		return "blocked"
	}
	if c.Error != "" {
		return "failed_or_interrupted"
	}
	p := r.Application
	if c.Status == 200 {
		if p.Version == application.PropertyAccessProfile {
			switch p.Probe.PropertyResult(c.Body) {
			case "leaked":
				return "known_private_value_returned"
			case "filtered":
				return "configured_fields_omitted"
			}
		} else if p.Probe.Matches(c.Body) {
			return "expected_object_and_proof_returned"
		}
	}
	if p.ExpectedAccess == "deny" {
		for _, status := range p.DenialStatuses {
			if status == c.Status {
				return "configured_denial"
			}
		}
	}
	return "response_unproven"
}

type CaptureMetadata struct {
	ID          string    `json:"id"`
	RunID       string    `json:"run_id"`
	CaptureID   string    `json:"capture_id"`
	SHA256      string    `json:"sha256"`
	Path        string    `json:"artifact_path"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	Status      int       `json:"status,omitempty"`
	Outcome     string    `json:"outcome"`
	Account     string    `json:"account,omitempty"`
	RequestPath string    `json:"request_path,omitempty"`
}

type EvidenceAssessment struct {
	RunID           string          `json:"run_id"`
	State           string          `json:"state"`
	Procedure       string          `json:"procedure"`
	ProcedureDigest string          `json:"procedure_digest"`
	DisplayTitle    string          `json:"display_title"`
	StoredVerdict   string          `json:"stored_verdict,omitempty"`
	Result          *Result         `json:"result,omitempty"`
	Checks          []EvidenceCheck `json:"checks"`
	ExpectedAccess  string          `json:"expected_access,omitempty"`
	OwnerAccount    string          `json:"owner_account,omitempty"`
	PeerAccount     string          `json:"peer_account,omitempty"`
	ObjectPath      string          `json:"object_path,omitempty"`
	ProtectedFields []string        `json:"protected_fields"`
	Remediation     string          `json:"remediation"`
}

type ReviewMetadata struct {
	Decision   string    `json:"decision"`
	Purpose    string    `json:"purpose"`
	Actor      string    `json:"actor"`
	Revision   int       `json:"revision"`
	ReviewedAt time.Time `json:"reviewed_at"`
	Severity   string    `json:"severity,omitempty"`
	CVSSScore  *float64  `json:"cvss_score,omitempty"`
}

type CandidateMetadata struct {
	ID           string          `json:"id"`
	Digest       string          `json:"candidate_digest"`
	Title        string          `json:"title"` // original candidate title/digest stay compatible
	DisplayTitle string          `json:"display_title"`
	Procedure    string          `json:"procedure"`
	Target       string          `json:"target"`
	Status       string          `json:"status"`
	Reason       string          `json:"reason"`
	Remediation  string          `json:"remediation"`
	RunIDs       []string        `json:"run_ids"`
	EvidenceIDs  []string        `json:"evidence_ids"`
	Review       *ReviewMetadata `json:"review,omitempty"`
}

type FindingsSnapshot struct {
	SchemaVersion int                  `json:"schema_version"`
	EngagementID  string               `json:"engagement_id"`
	ObservedAt    time.Time            `json:"observed_at"`
	ScopeDigest   string               `json:"scope_digest"`
	Candidates    []CandidateMetadata  `json:"candidates"`
	Assessments   []EvidenceAssessment `json:"assessments"`
	Captures      []CaptureMetadata    `json:"captures"`
}

func findingDisplayTitle(r Run) string {
	if p := r.Application; p != nil {
		if p.Version == application.PropertyAccessProfile {
			return "Owner-only property exposed to another user"
		}
		return "Cross-user access to a protected object"
	}
	return r.Assessment.Title
}

// Metadata-only response; validation/hashing necessarily reads stored evidence
// internally, but no body, rejected URL/account, error text or review narrative
// is returned. No target/model request or state mutation occurs.
func (s *Store) FindingsSnapshot(tenant, id string) (FindingsSnapshot, error) {
	out := FindingsSnapshot{SchemaVersion: 1, EngagementID: id, ObservedAt: time.Now().UTC(), Candidates: []CandidateMetadata{}, Assessments: []EvidenceAssessment{}, Captures: []CaptureMetadata{}}
	err := s.db.View(func(tx *bolt.Tx) error {
		e, err := getEngagement(tx, tenant, id)
		if err != nil {
			return err
		}
		out.ScopeDigest = e.ScopeDigest
		byProcedure := map[string]*FindingCandidate{}
		titles := map[string]string{}
		for _, runID := range e.RunIDs {
			r, err := getBoundRun(tx, e, runID)
			if err != nil {
				return err
			}
			a := EvidenceAssessment{RunID: r.ID, State: r.State, Procedure: r.ProcedureID(), ProcedureDigest: r.ProcedureDigest(), DisplayTitle: "Configured authorization check", Checks: applicationChecks(r), ProtectedFields: []string{}, Remediation: r.ProcedureRemediation()}
			if r.Result != nil {
				a.StoredVerdict = r.Result.Verdict
			}
			if p := r.Application; p != nil {
				a.ExpectedAccess = p.ExpectedAccess
				a.OwnerAccount = p.Identity[0].Account
				a.PeerAccount = p.Identity[1].Account
				a.ObjectPath = p.Probe.Path
				for field := range p.Probe.Hashes {
					a.ProtectedFields = append(a.ProtectedFields, field)
				}
				sort.Strings(a.ProtectedFields)
			}
			refs := map[string]string{}
			for _, c := range r.Captures {
				artifact, err := evidenceArtifact(e.ID, r, c)
				if err != nil {
					return err
				}
				refs[c.ID] = artifact.ID
				m := CaptureMetadata{ID: artifact.ID, RunID: r.ID, CaptureID: c.ID, SHA256: artifact.SHA256, Path: artifact.Path, StartedAt: c.StartedAt, FinishedAt: c.FinishedAt, Status: c.Status, Outcome: "response_captured"}
				if c.Error != "" {
					m.Outcome = "failed_or_interrupted"
				}
				if c.Blocked {
					m.Outcome = "blocked"
				}
				// Only resolve exact reviewed pairs. Denied arguments can contain
				// secrets, including in a URL or account alias; never echo them.
				if !c.Blocked && r.Application != nil {
					for _, read := range r.Application.Reads() {
						if c.URL == r.Target+read.Path && c.Account == read.Account {
							m.Account = read.Account
							m.RequestPath = read.Path
							break
						}
					}
				}
				out.Captures = append(out.Captures, m)
			}
			if r.State == Completed {
				result := Validate(r)
				if r.Application != nil {
					result = validateApplicationEvidence(r, &a.Checks)
				}
				a.Result = &result
				if result.Verdict == "confirmed" {
					a.DisplayTitle = findingDisplayTitle(r)
				}
				for i := range a.Checks {
					a.Checks[i].EvidenceID = refs[a.Checks[i].CaptureID]
				}
				appendFindingCandidate(byProcedure, e, r, result, refs)
			}
			titles[r.ProcedureID()] = findingDisplayTitle(r)
			out.Assessments = append(out.Assessments, a)
		}
		for _, c := range sortedCandidates(byProcedure) {
			if err := loadCandidateReview(tx, e.ID, &c); err != nil {
				return err
			}
			m := CandidateMetadata{ID: c.ID, Digest: c.Digest, Title: c.Title, DisplayTitle: titles[c.Procedure], Procedure: c.Procedure, Target: c.Target, Status: c.Status, Reason: c.Reason, Remediation: c.Remediation, RunIDs: c.RunIDs, EvidenceIDs: c.EvidenceIDs}
			if v := c.Review; v != nil {
				m.Review = &ReviewMetadata{Decision: v.Decision, Purpose: v.Purpose, Actor: v.Actor, Revision: v.Revision, ReviewedAt: v.ReviewedAt, Severity: v.Severity, CVSSScore: v.CVSSScore}
			}
			out.Candidates = append(out.Candidates, m)
		}
		return nil
	})
	return out, err
}

// Single explicit raw inspection. Bytes and identifiers are exactly those used
// by the existing export; reading evidence never replays a target request.
func (s *Store) ReadEvidence(tenant, id, artifactID string) (EvidenceArtifact, error) {
	var out EvidenceArtifact
	if !strings.HasPrefix(artifactID, "E-") || !activityDigest.MatchString(strings.TrimPrefix(artifactID, "E-")) {
		return out, ErrNotFound
	}
	err := s.db.View(func(tx *bolt.Tx) error {
		e, err := getEngagement(tx, tenant, id)
		if err != nil {
			return err
		}
		for _, runID := range e.RunIDs {
			r, err := getBoundRun(tx, e, runID)
			if err != nil {
				return err
			}
			for _, c := range r.Captures {
				a, err := evidenceArtifact(e.ID, r, c)
				if err != nil {
					return err
				}
				if a.ID == artifactID {
					out = a
					return nil
				}
			}
		}
		return ErrNotFound
	})
	return out, err
}

func (s *Server) serveFindings(w http.ResponseWriter, r *http.Request, tenant, id, artifactID string) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	if r.URL.RawQuery != "" {
		problem(w, 400, errors.New("evidence reads accept no query parameters"))
		return
	}
	var data any
	var err error
	if artifactID == "" {
		data, err = s.Store.FindingsSnapshot(tenant, id)
	} else {
		data, err = s.Store.ReadEvidence(tenant, id, artifactID)
	}
	if errors.Is(err, ErrNotFound) {
		problem(w, 404, ErrNotFound)
		return
	}
	if err != nil {
		problem(w, 500, errors.New("finding or evidence snapshot unavailable"))
		return
	}
	writeJSON(w, 200, data)
}
