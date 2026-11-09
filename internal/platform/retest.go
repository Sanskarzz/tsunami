package platform

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	bolt "go.etcd.io/bbolt"
)

var retestID = regexp.MustCompile(`^[a-f0-9]{32}$`)

// A source link is history, never execution authority. Old scopes without a
// link retain their original digest/serialization. No credentials are copied.
func retestScopeValid(e Engagement) bool {
	return e.Application != nil && e.Application.Validate() == nil && e.digest() == e.ScopeDigest &&
		len(e.Items) == 1 && e.Items[0].Procedure == e.Application.ProcedureID() && e.Items[0].Digest == e.Application.Digest()
}

type RetestAttempt struct {
	RunID         string   `json:"run_id"`
	State         string   `json:"state"`
	StoredVerdict string   `json:"stored_verdict,omitempty"`
	Result        *Result  `json:"result,omitempty"`
	EvidenceIDs   []string `json:"evidence_ids"`
}

type RetestSide struct {
	Engagement    Engagement        `json:"engagement"`
	Procedure     ProcedureMetadata `json:"procedure"`
	AttemptStatus string            `json:"attempt_status"`
	LatestAttempt *RetestAttempt    `json:"latest_attempt,omitempty"`
}

type RetestDifference struct {
	Field  string `json:"field"`
	Source string `json:"source"`
	Retest string `json:"retest"`
}

type RetestComparison struct {
	SchemaVersion int                `json:"schema_version"`
	ObservedAt    time.Time          `json:"observed_at"`
	Compatibility string             `json:"compatibility"`
	Reason        string             `json:"reason"`
	Source        RetestSide         `json:"source"`
	Retest        RetestSide         `json:"retest"`
	Differences   []RetestDifference `json:"differences"`
}

func protectedPointers(p application.Plan) []string {
	fields := []string{}
	for field := range p.Probe.Hashes {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

func retestSide(tx *bolt.Tx, e Engagement) (RetestSide, error) {
	if !retestScopeValid(e) {
		return RetestSide{}, errors.New("application scope unavailable")
	}
	out := RetestSide{Engagement: e, Procedure: applicationProcedure(*e.Application), AttemptStatus: "not_started"}
	if len(e.RunIDs) == 0 {
		return out, nil
	}
	// Durable reservation order, never the most favorable or last clean result.
	r, err := getBoundRun(tx, e, e.RunIDs[len(e.RunIDs)-1])
	if err != nil {
		return out, err
	}
	a := &RetestAttempt{RunID: r.ID, State: r.State, EvidenceIDs: []string{}}
	if r.Result != nil {
		a.StoredVerdict = r.Result.Verdict
	}
	if r.State == Completed {
		result := Validate(r)
		a.Result = &result
		for _, c := range r.Captures {
			for _, id := range result.EvidenceIDs {
				if c.ID == id {
					artifact, err := evidenceArtifact(e.ID, r, c)
					if err != nil {
						return out, err
					}
					a.EvidenceIDs = append(a.EvidenceIDs, artifact.ID)
					break
				}
			}
		}
	}
	out.AttemptStatus, out.LatestAttempt = "recorded", a
	return out, nil
}

func (s *Store) RetestComparison(tenant, id string) (RetestComparison, error) {
	out := RetestComparison{SchemaVersion: 1, ObservedAt: time.Now().UTC(), Differences: []RetestDifference{}}
	err := s.db.View(func(tx *bolt.Tx) error {
		e, err := getEngagement(tx, tenant, id)
		if err != nil {
			return err
		}
		if e.SourceEngagementID == "" {
			return ErrNotFound
		}
		source, err := getEngagement(tx, tenant, e.SourceEngagementID)
		if err != nil {
			return err
		}
		if out.Source, err = retestSide(tx, source); err != nil {
			return err
		}
		if out.Retest, err = retestSide(tx, e); err != nil {
			return err
		}
		a, b := source.Application, e.Application
		out.Compatibility = "comparable"
		out.Reason = "Same supported profile, intended access rule, ownership pointer, protected field names and denial policy. Bindings and approvals remain separate; this is a comparison of configured checks, not proof of application-wide security or a deployed code fix."
		if a.Version != b.Version || a.ExpectedAccess != b.ExpectedAccess || a.OwnerPointer != b.OwnerPointer || !reflect.DeepEqual(protectedPointers(*a), protectedPointers(*b)) || !reflect.DeepEqual(a.DenialStatuses, b.DenialStatuses) {
			out.Compatibility = "incompatible"
			out.Reason = "Profile, intended access rule, ownership pointer, protected field names or denial policy differ. Show each outcome independently; do not infer that the original issue was fixed."
		}
		for _, pair := range []struct {
			field string
			a, b  any
		}{
			{"Application", a.ID, b.ID}, {"Origin", source.Target, e.Target},
			{"Approved addresses", a.ApprovedAddresses, b.ApprovedAddresses}, {"Profile", a.Version, b.Version},
			{"Expected access", a.ExpectedAccess, b.ExpectedAccess}, {"Ownership pointer", a.OwnerPointer, b.OwnerPointer},
			{"Reviewed reads and assertions", a.Reads(), b.Reads()}, {"Denial statuses", a.DenialStatuses, b.DenialStatuses},
			{"Minimum interval (ms)", a.MinIntervalMillis, b.MinIntervalMillis}, {"Credential binding digest", a.CredentialSetID, b.CredentialSetID},
			{"Definition digest", a.Digest(), b.Digest()}, {"Executor", source.Executor, e.Executor},
			{"Authorization reference", source.AuthorizationRef, e.AuthorizationRef}, {"Expiry", source.ExpiresAt, e.ExpiresAt},
			{"Child allowance", source.MaxRuns, e.MaxRuns}, {"Concurrent children", source.MaxConcurrent, e.MaxConcurrent},
			{"Scope digest", source.ScopeDigest, e.ScopeDigest},
		} {
			left, _ := json.Marshal(pair.a)
			right, _ := json.Marshal(pair.b)
			if string(left) != string(right) {
				out.Differences = append(out.Differences, RetestDifference{Field: pair.field, Source: string(left), Retest: string(right)})
			}
		}
		return nil
	})
	return out, err
}

func (s *Server) serveRetest(w http.ResponseWriter, r *http.Request, tenant, id string) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	if r.URL.RawQuery != "" {
		problem(w, 400, errors.New("retest reads accept no query parameters"))
		return
	}
	data, err := s.Store.RetestComparison(tenant, id)
	if errors.Is(err, ErrNotFound) {
		problem(w, 404, ErrNotFound)
		return
	}
	if err != nil {
		problem(w, 500, errors.New("retest comparison unavailable"))
		return
	}
	writeJSON(w, 200, data)
}
