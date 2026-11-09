package platform

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/assessment"
	bolt "go.etcd.io/bbolt"
)

// These read-only projections are not part of approved plans, policy digests,
// or stored exports. Availability is local configuration, never session health.
type AccessMetadata struct {
	Availability             string     `json:"availability"`
	CredentialHealth         string     `json:"credential_health"`
	SetupPermissionExpiresAt *time.Time `json:"setup_permission_expires_at,omitempty"`
	BaselinesValidatedAt     *time.Time `json:"baselines_validated_at,omitempty"`
}

type ProcedureMetadata struct {
	ID                string `json:"id"`
	Version           string `json:"version"`
	Digest            string `json:"digest"`
	Supported         bool   `json:"supported"`
	MaxRequests       int    `json:"max_requests,omitempty"`
	PlannedReads      int    `json:"planned_reads,omitempty"`
	MinIntervalMillis int    `json:"min_interval_ms"`
}

type ApplicationMetadata struct {
	ObservedAt            time.Time         `json:"observed_at"`
	Registration          string            `json:"registration"`
	ExecutionMode         string            `json:"execution_mode"`
	MaxRunDurationSeconds int               `json:"max_run_duration_seconds"`
	Procedure             ProcedureMetadata `json:"procedure"`
	Access                AccessMetadata    `json:"access"`
}

type ApplicationSummary struct {
	application.Plan
	Metadata ApplicationMetadata `json:"metadata"`
}

type AttemptMetadata struct {
	ID        string    `json:"id"`
	State     string    `json:"state"`
	Verdict   string    `json:"verdict,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Deadline  time.Time `json:"deadline"`
}

// Dashboard projection only. It preserves the existing array/engagement shape;
// neither a stored result nor the last reserved attempt closes scope approval.
type EngagementSummary struct {
	Engagement
	ObservedAt          time.Time        `json:"observed_at"`
	LatestAttemptStatus string           `json:"latest_attempt_status"`
	LatestAttempt       *AttemptMetadata `json:"latest_attempt,omitempty"`
}

func (s *Store) ListEngagementSummaries(tenant string) ([]EngagementSummary, error) {
	items := []EngagementSummary{}
	err := s.db.View(func(tx *bolt.Tx) error {
		engagements, err := listEngagements(tx, tenant)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, e := range engagements {
			item := EngagementSummary{Engagement: e, ObservedAt: now, LatestAttemptStatus: "not_started"}
			if len(e.RunIDs) > 0 {
				item.LatestAttemptStatus = "unavailable"
				id := e.RunIDs[len(e.RunIDs)-1] // durable reservation order
				// Decode only metadata; capture bodies, errors, model calls and
				// legacy event text never enter the dashboard response.
				var stored struct {
					AttemptMetadata
					Tenant       string `json:"tenant"`
					EngagementID string `json:"engagement_id"`
					Result       *struct {
						Verdict string `json:"verdict"`
					} `json:"result"`
				}
				raw := tx.Bucket([]byte("runs")).Get([]byte(id))
				if raw != nil && json.Unmarshal(raw, &stored) == nil && stored.ID == id && stored.Tenant == tenant && stored.EngagementID == e.ID {
					if stored.Result != nil {
						stored.AttemptMetadata.Verdict = stored.Result.Verdict
					}
					item.LatestAttempt = &stored.AttemptMetadata
					item.LatestAttemptStatus = "recorded"
				}
			}
			items = append(items, item)
		}
		return nil
	})
	return items, err
}

type EngagementMetadata struct {
	ObservedAt            time.Time           `json:"observed_at"`
	Permission            string              `json:"permission"`
	ExecutionMode         string              `json:"execution_mode"`
	Executor              string              `json:"executor"`
	MaxRunDurationSeconds int                 `json:"max_run_duration_seconds"`
	Procedures            []ProcedureMetadata `json:"procedures"`
	Access                *AccessMetadata     `json:"access,omitempty"`
	LatestAttempt         *AttemptMetadata    `json:"latest_attempt,omitempty"`
}

func applicationProcedure(p application.Plan) ProcedureMetadata {
	m := ProcedureMetadata{ID: p.ProcedureID(), Version: p.Version, Digest: p.Digest(), Supported: p.Validate() == nil}
	if m.Supported {
		m.MaxRequests, m.PlannedReads, m.MinIntervalMillis = p.MaxRequests(), len(p.Reads()), p.MinIntervalMillis
	}
	return m
}

func procedureVersion(id string) string {
	_, version, ok := strings.Cut(id, "/")
	if !ok {
		return ""
	}
	return version
}

func (s *Server) applicationMetadata(tenant string, p application.Plan) ApplicationMetadata {
	m := ApplicationMetadata{ObservedAt: time.Now().UTC(), Registration: "unavailable", ExecutionMode: "scripted", MaxRunDurationSeconds: int(MaxRunDuration / time.Second), Procedure: applicationProcedure(p), Access: AccessMetadata{Availability: "unavailable", CredentialHealth: "unknown"}}
	if _, ok := s.Applications[tenant][p.ID]; ok {
		m.Registration = "startup"
	} else {
		// Expiry here is the persisted setup permission, not bearer-token expiry.
		_ = s.Store.db.View(func(tx *bolt.Tx) error {
			raw := tx.Bucket([]byte("onboarded_apps")).Get(setupKey(tenant, p.ID))
			var app dynamicApp
			if raw == nil || json.Unmarshal(raw, &app) != nil {
				return nil
			}
			m.Registration = "onboarding"
			if setup, err := getSetup(tx, tenant, app.SessionID); err == nil {
				m.Access.SetupPermissionExpiresAt = &setup.ExpiresAt
				m.Access.BaselinesValidatedAt = setup.ValidatedAt
			}
			return nil
		})
	}
	if current, ok := s.application(tenant, p.ID); ok {
		m.Access.Availability = "configured"
		if current.Plan.Digest() != p.Digest() {
			m.Access.Availability = "configuration_changed"
		}
	}
	return m
}

// Attach only on operator reads/approval responses. Store snapshots and exports
// keep their existing shape and do not acquire time-varying health assertions.
func (s *Server) engagementMetadataView(tenant, id string) (EngagementView, error) {
	var view EngagementView
	err := s.Store.db.View(func(tx *bolt.Tx) error {
		e, err := getEngagement(tx, tenant, id)
		if err != nil {
			return err
		}
		var runs []Run
		view, runs, err = engagementView(tx, e)
		if err != nil {
			return err
		}
		m := &EngagementMetadata{ObservedAt: time.Now().UTC(), Permission: "unavailable", ExecutionMode: "selected_per_run", Executor: e.Executor, MaxRunDurationSeconds: int(MaxRunDuration / time.Second), Procedures: []ProcedureMetadata{}}
		if m.Executor == "" {
			m.Executor = "native"
		}
		gates := tx.Bucket([]byte("gates"))
		switch {
		case e.State == EngagementStopped:
			m.Permission = "revoked"
		case !m.ObservedAt.Before(e.ExpiresAt):
			m.Permission = "expired"
		case gates.Get([]byte("global")) != nil || gates.Get([]byte("tenant:"+tenant)) != nil:
			m.Permission = "disabled"
		case e.digest() != e.ScopeDigest:
		case e.State == EngagementDraft:
			m.Permission = "unapproved"
		case e.State == EngagementActive && e.ApprovedDigest == e.ScopeDigest:
			m.Permission = "active"
		}
		for _, item := range e.Items {
			p := ProcedureMetadata{ID: item.Procedure, Version: procedureVersion(item.Procedure), Digest: item.Digest}
			if e.Application != nil {
				p = applicationProcedure(*e.Application)
				p.Supported = p.Supported && p.ID == item.Procedure && p.Digest == item.Digest
				m.ExecutionMode = "scripted"
			} else if definition, err := assessment.Get(item.Procedure); err == nil && definition.Digest() == item.Digest {
				p.Supported, p.MaxRequests, p.PlannedReads = true, definition.MaxRequests, len(definition.Reads())
			}
			p.ID, p.Digest = item.Procedure, item.Digest
			if !p.Supported {
				p.MaxRequests, p.PlannedReads, p.MinIntervalMillis = 0, 0, 0
			}
			m.Procedures = append(m.Procedures, p)
		}
		if len(runs) > 0 {
			r := runs[len(runs)-1] // Durable reservation order, not random run ID order.
			m.LatestAttempt = &AttemptMetadata{ID: r.ID, State: r.State, CreatedAt: r.CreatedAt, Deadline: r.Deadline}
			if r.Result != nil {
				m.LatestAttempt.Verdict = r.Result.Verdict
			}
		}
		view.Metadata = m
		return nil
	})
	if err == nil && view.Engagement.Application != nil {
		access := s.applicationMetadata(tenant, *view.Engagement.Application).Access
		view.Metadata.Access = &access
	}
	return view, err
}
