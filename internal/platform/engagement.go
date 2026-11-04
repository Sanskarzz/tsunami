package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/toolcontract"
	bolt "go.etcd.io/bbolt"
)

const (
	EngagementDraft         = "draft"
	EngagementActive        = "active"
	EngagementStopped       = "stopped"
	EngagementMaxRuns       = 20
	EngagementMaxConcurrent = 2
)

// These records are operator-owned. A run token cannot create, approve, skip,
// stop, or read an engagement. This first integration still selects lab targets.
type Engagement struct {
	Executor         string            `json:"executor,omitempty"`
	Application      *application.Plan `json:"application,omitempty"`
	ID               string            `json:"id"`
	Tenant           string            `json:"tenant"`
	Name             string            `json:"name"`
	State            string            `json:"state"`
	AuthorizationRef string            `json:"authorization_ref"`
	Target           string            `json:"target"`
	Scenario         string            `json:"scenario"`
	Connectivity     string            `json:"connectivity,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	ExpiresAt        time.Time         `json:"expires_at"`
	ApprovedAt       *time.Time        `json:"approved_at,omitempty"`
	StoppedAt        *time.Time        `json:"stopped_at,omitempty"`
	ScopeDigest      string            `json:"scope_digest"`
	ApprovedDigest   string            `json:"approved_digest,omitempty"`
	MaxRuns          int               `json:"max_runs"`
	MaxConcurrent    int               `json:"max_concurrent"`
	Items            []EngagementItem  `json:"items"`
	RunIDs           []string          `json:"run_ids"`
	Events           []Event           `json:"events"`
}

type EngagementItem struct {
	Procedure  string `json:"procedure"`
	Digest     string `json:"definition_digest"`
	Title      string `json:"title"`
	SkipReason string `json:"skip_reason,omitempty"`
}

type EngagementInput struct {
	Executor         string   `json:"executor,omitempty"`
	Application      string   `json:"application,omitempty"`
	Name             string   `json:"name"`
	AuthorizationRef string   `json:"authorization_ref"`
	Scenario         string   `json:"scenario"`
	Connectivity     string   `json:"connectivity,omitempty"`
	Assessments      []string `json:"assessments"`
}

type ChecklistItem struct {
	Procedure  string `json:"procedure"`
	Status     string `json:"status"`
	Verdict    string `json:"verdict,omitempty"`
	LatestRun  string `json:"latest_run,omitempty"`
	SkipReason string `json:"skip_reason,omitempty"`
}

type ChecklistProgress struct {
	Total        int `json:"total"`
	Completed    int `json:"completed"`
	Skipped      int `json:"skipped"`
	Pending      int `json:"pending"`
	Running      int `json:"running"`
	Failed       int `json:"failed"`
	Stopped      int `json:"stopped"`
	Inconclusive int `json:"inconclusive"`
}

type EngagementView struct {
	Engagement Engagement        `json:"engagement"`
	Checklist  []ChecklistItem   `json:"checklist"`
	Progress   ChecklistProgress `json:"progress"`
}

func shortText(s string, max int) bool {
	if strings.TrimSpace(s) == "" || len(s) > max {
		return false
	}
	for _, c := range s {
		if unicode.IsControl(c) {
			return false
		}
	}
	return true
}

func (e Engagement) digest() string {
	// Runtime state, child IDs and skip decisions do not change approved scope.
	type item struct{ Procedure, Digest string }
	items := []item{}
	for _, i := range e.Items {
		items = append(items, item{i.Procedure, i.Digest})
	}
	data, _ := json.Marshal(struct {
		Tenant, Target, Scenario, Connectivity, AuthorizationRef string
		ExpiresAt                                                time.Time
		MaxRuns, MaxConcurrent                                   int
		Items                                                    []item
		ApplicationDigest                                        string `json:",omitempty"`
		Executor                                                 string `json:",omitempty"`
	}{e.Tenant, e.Target, e.Scenario, e.Connectivity, e.AuthorizationRef, e.ExpiresAt, e.MaxRuns, e.MaxConcurrent, items, func() string {
		if e.Application != nil {
			return e.Application.Digest()
		}
		return ""
	}(), e.Executor})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func getEngagement(tx *bolt.Tx, tenant, id string) (Engagement, error) {
	var e Engagement
	data := tx.Bucket([]byte("engagements")).Get([]byte(id))
	if data == nil {
		return e, ErrNotFound
	}
	if err := json.Unmarshal(data, &e); err != nil {
		return e, errors.New("invalid engagement state")
	}
	if e.Tenant != tenant {
		return Engagement{}, ErrNotFound
	}
	return e, nil
}

func putEngagement(tx *bolt.Tx, e Engagement) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return tx.Bucket([]byte("engagements")).Put([]byte(e.ID), data)
}

func (s *Store) CreateEngagement(tenant, target string, in EngagementInput, plans ...application.Plan) (Engagement, error) {
	var e Engagement
	configured := len(plans) == 1
	if in.Executor != "" && (in.Executor != toolcontract.CurlHTTP || !configured) {
		return e, errors.New("unsupported executor for this procedure")
	}
	if len(plans) > 1 || !shortText(in.Name, 120) || !shortText(in.AuthorizationRef, 256) || tenant == "" || target == "" || (!configured && len(in.Assessments) == 0) || len(in.Assessments) > len(assessment.List()) {
		return e, errors.New("name, authorization reference and a bounded procedure selection are required")
	}
	if !configured && ((in.Scenario != "vulnerable" && in.Scenario != "fixed") || (in.Connectivity != "" && in.Connectivity != "private") || in.Application != "") {
		return e, errors.New("unsupported lab engagement")
	}
	now := time.Now().UTC()
	e = Engagement{Executor: in.Executor, ID: randomID(), Tenant: tenant, Name: in.Name, State: EngagementDraft, AuthorizationRef: in.AuthorizationRef, Target: target, Scenario: in.Scenario, Connectivity: in.Connectivity, CreatedAt: now, ExpiresAt: now.Add(time.Hour), MaxRuns: EngagementMaxRuns, MaxConcurrent: EngagementMaxConcurrent, Items: []EngagementItem{}, RunIDs: []string{}, Events: []Event{{now, "engagement_created", "draft scope; explicit operator approval required"}}}
	if configured {
		p := plans[0]
		if p.Validate() != nil || p.ID != in.Application || p.Origin != target || in.Scenario != "" || in.Connectivity != "" || len(in.Assessments) != 0 {
			return Engagement{}, errors.New("invalid configured application selection")
		}
		e.Application = &p
		e.Scenario = "application"
		e.Items = append(e.Items, EngagementItem{Procedure: p.ProcedureID(), Digest: p.Digest(), Title: p.Title})
	}
	seen := map[string]bool{}
	for _, id := range in.Assessments {
		d, err := assessment.Get(id)
		if err != nil || seen[id] {
			return Engagement{}, errors.New("unknown or duplicate procedure")
		}
		seen[id] = true
		e.Items = append(e.Items, EngagementItem{Procedure: d.ID, Digest: d.Digest(), Title: d.Title})
	}
	sort.Slice(e.Items, func(i, j int) bool { return e.Items[i].Procedure < e.Items[j].Procedure })
	e.ScopeDigest = e.digest()
	err := s.db.Update(func(tx *bolt.Tx) error {
		count := 0
		if err := tx.Bucket([]byte("engagements")).ForEach(func(_, v []byte) error {
			var other Engagement
			if json.Unmarshal(v, &other) != nil {
				return errors.New("invalid engagement state")
			}
			if other.Tenant == tenant {
				count++
			}
			return nil
		}); err != nil {
			return err
		}
		if count >= 100 {
			return errors.New("lab tenant engagement limit reached")
		}
		return putEngagement(tx, e)
	})
	return e, err
}

func (s *Store) ApproveEngagement(tenant, id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		e, err := getEngagement(tx, tenant, id)
		if err != nil {
			return err
		}
		if e.State != EngagementDraft || time.Now().After(e.ExpiresAt) || e.digest() != e.ScopeDigest {
			return errors.New("engagement cannot be approved")
		}
		g := tx.Bucket([]byte("gates"))
		if g.Get([]byte("global")) != nil || g.Get([]byte("tenant:"+tenant)) != nil {
			return errors.New("new runs are disabled")
		}
		now := time.Now().UTC()
		e.State = EngagementActive
		e.ApprovedAt = &now
		e.ApprovedDigest = e.ScopeDigest
		e.Events = append(e.Events, Event{now, "engagement_approved", "operator approved pinned lab scope"})
		return putEngagement(tx, e)
	})
}

func (s *Store) CreateEngagementRun(tenant, id, procedure string, model *ModelUsage) (Run, error) {
	var e Engagement
	err := s.db.View(func(tx *bolt.Tx) error { var err error; e, err = getEngagement(tx, tenant, id); return err })
	if err != nil {
		return Run{}, err
	}
	if e.Application != nil {
		if procedure != e.Application.ProcedureID() || model != nil {
			return Run{}, errors.New("application procedure or model execution unavailable")
		}
		return s.create(tenant, e.Scenario, e.Target, "scripted Mecatl / owned application", nil, "", createOptions{Application: e.Application, EngagementID: id})
	}
	d, err := assessment.Get(procedure)
	if err != nil {
		return Run{}, err
	}
	mode := "scripted Mecatl / synthetic lab engagement"
	if model != nil {
		mode = "Responses model / synthetic lab engagement"
		if model.Provider == "fixture" {
			mode = "Responses wire fixture / no live model"
		}
	}
	return s.create(tenant, e.Scenario, e.Target, mode, model, e.Connectivity, createOptions{Definition: &d, EngagementID: id})
}

// This executes in the SAME transaction as run/credential creation. A concurrent
// Stop or competing reservation cannot leave an untracked permitted child.
func bindEngagementRun(tx *bolt.Tx, r *Run, id string) error {
	e, err := getEngagement(tx, r.Tenant, id)
	if err != nil {
		return err
	}
	if e.State != EngagementActive || time.Now().After(e.ExpiresAt) || e.ApprovedDigest != e.ScopeDigest || e.digest() != e.ScopeDigest || e.Target != r.Target || e.Scenario != r.Scenario || e.Connectivity != r.Connectivity || !r.ProcedureValid() {
		return errors.New("engagement permission unavailable")
	}
	allowed := false
	for _, item := range e.Items {
		if item.Procedure == r.ProcedureID() && item.Digest == r.ProcedureDigest() && item.SkipReason == "" {
			allowed = true
		}
	}
	if !allowed || len(e.RunIDs) >= e.MaxRuns {
		return errors.New("procedure unavailable or engagement run budget exhausted")
	}
	active := 0
	for _, runID := range e.RunIDs {
		child, err := getBoundRun(tx, e, runID)
		if err != nil {
			return err
		}
		if !terminal(child.State) {
			// Expired work still occupies its slot until Stop reconciles it.
			active++
			if child.ProcedureID() == r.ProcedureID() {
				return errors.New("procedure already has an active run")
			}
		}
	}
	if active >= e.MaxConcurrent {
		return errors.New("engagement concurrency limit")
	}
	// Selection is copied from durable approved state in the child transaction.
	r.Executor = e.Executor
	r.EngagementID = e.ID
	if e.ExpiresAt.Before(r.Deadline) {
		r.Deadline = e.ExpiresAt
	}
	e.RunIDs = append(e.RunIDs, r.ID)
	e.Events = append(e.Events, Event{time.Now().UTC(), "engagement_run_reserved", r.ID})
	return putEngagement(tx, e)
}

func getStoredChild(tx *bolt.Tx, e Engagement, id string) (Run, error) {
	var r Run
	data := tx.Bucket([]byte("runs")).Get([]byte(id))
	if data == nil || json.Unmarshal(data, &r) != nil {
		return r, errors.New("engagement child state unavailable")
	}
	if r.Tenant != e.Tenant || r.EngagementID != e.ID {
		return Run{}, errors.New("engagement child identity changed")
	}
	return r, nil
}

func getBoundRun(tx *bolt.Tx, e Engagement, id string) (Run, error) {
	r, err := getStoredChild(tx, e, id)
	if err != nil {
		return Run{}, err
	}
	if r.Executor != e.Executor || r.Target != e.Target || r.Scenario != e.Scenario || r.Connectivity != e.Connectivity || !r.ProcedureValid() || (e.Application != nil && r.Application == nil) || (e.Application == nil && r.Application != nil) {
		return Run{}, errors.New("engagement child binding changed")
	}
	approved := false
	for _, item := range e.Items {
		if item.Procedure == r.ProcedureID() && item.Digest == r.ProcedureDigest() {
			approved = true
		}
	}
	if !approved {
		return Run{}, errors.New("unapproved child procedure")
	}
	return r, nil
}

func (s *Store) StopEngagement(tenant, id string) ([]string, error) {
	ids := []string{}
	err := s.db.Update(func(tx *bolt.Tx) error {
		e, err := getEngagement(tx, tenant, id)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, runID := range e.RunIDs {
			// Revocation must still work when a procedure version or target
			// binding is no longer acceptable for execution or reporting.
			r, err := getStoredChild(tx, e, runID)
			if err != nil {
				return err
			}
			if terminal(r.State) {
				continue
			}
			r.State = Stopped
			r.StopAcceptedAt = &now
			r.Events = append(r.Events, Event{now, "stopped", "engagement permission revoked"})
			data, err := json.Marshal(r)
			if err != nil {
				return err
			}
			if err = tx.Bucket([]byte("runs")).Put([]byte(runID), data); err != nil {
				return err
			}
			ids = append(ids, runID)
		}
		if e.State != EngagementStopped {
			e.State = EngagementStopped
			e.StoppedAt = &now
			e.Events = append(e.Events, Event{now, "engagement_stopped", "all active child permissions revoked atomically"})
		}
		return putEngagement(tx, e)
	})
	return ids, err
}

func (s *Store) SkipEngagementItem(tenant, id, procedure, reason string) error {
	if !shortText(reason, 256) {
		return errors.New("a short skip reason is required")
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		e, err := getEngagement(tx, tenant, id)
		if err != nil {
			return err
		}
		if e.State != EngagementActive || time.Now().After(e.ExpiresAt) {
			return errors.New("engagement permission unavailable")
		}
		for _, runID := range e.RunIDs {
			r, err := getBoundRun(tx, e, runID)
			if err != nil {
				return err
			}
			if r.ProcedureID() == procedure {
				return errors.New("an attempted procedure cannot be relabeled skipped")
			}
		}
		for i := range e.Items {
			if e.Items[i].Procedure == procedure {
				if e.Items[i].SkipReason != "" {
					return errors.New("procedure already skipped")
				}
				e.Items[i].SkipReason = reason
				e.Events = append(e.Events, Event{time.Now().UTC(), "engagement_item_skipped", procedure})
				return putEngagement(tx, e)
			}
		}
		return errors.New("unknown engagement procedure")
	})
}

func engagementView(tx *bolt.Tx, e Engagement) (EngagementView, []Run, error) {
	view := EngagementView{Engagement: e, Checklist: []ChecklistItem{}, Progress: ChecklistProgress{Total: len(e.Items)}}
	runs := []Run{}
	latest := map[string]Run{}
	for _, id := range e.RunIDs {
		r, err := getBoundRun(tx, e, id)
		if err != nil {
			return view, nil, err
		}
		runs = append(runs, r)
		latest[r.ProcedureID()] = r
	}
	for _, item := range e.Items {
		row := ChecklistItem{Procedure: item.Procedure, Status: "pending"}
		if item.SkipReason != "" {
			row.Status = "skipped"
			row.SkipReason = item.SkipReason
		} else if r, ok := latest[item.Procedure]; ok {
			row.LatestRun = r.ID
			switch r.State {
			case Pending, Running:
				row.Status = "running"
			case Failed:
				row.Status = "failed"
			case Stopped:
				row.Status = "stopped"
			case Completed:
				// Recompute from trusted captures, rather than accepting an agent
				// completion signal as proof that a checklist item was tested.
				result := Validate(r)
				row.Verdict = result.Verdict
				row.Status = "inconclusive"
				if result.Validation != nil && result.Validation.BaselinesRequired == result.Validation.BaselinesVerified && result.Validation.ProbeCaptured {
					row.Status = "completed"
				}
			}
		}
		switch row.Status {
		case "pending":
			view.Progress.Pending++
		case "running":
			view.Progress.Running++
		case "failed":
			view.Progress.Failed++
		case "stopped":
			view.Progress.Stopped++
		case "inconclusive":
			view.Progress.Inconclusive++
		case "skipped":
			view.Progress.Skipped++
		case "completed":
			view.Progress.Completed++
		}
		view.Checklist = append(view.Checklist, row)
	}
	return view, runs, nil
}

func (s *Store) EngagementSnapshot(tenant, id string) (EngagementView, error) {
	var view EngagementView
	err := s.db.View(func(tx *bolt.Tx) error {
		e, err := getEngagement(tx, tenant, id)
		if err != nil {
			return err
		}
		view, _, err = engagementView(tx, e)
		return err
	})
	return view, err
}

func (s *Store) GetEngagement(tenant, id string) (Engagement, error) {
	var e Engagement
	err := s.db.View(func(tx *bolt.Tx) error {
		var err error
		e, err = getEngagement(tx, tenant, id)
		return err
	})
	return e, err
}

func (s *Store) ListEngagements(tenant string) ([]Engagement, error) {
	items := []Engagement{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("engagements")).ForEach(func(_, data []byte) error {
			var e Engagement
			if json.Unmarshal(data, &e) != nil {
				return errors.New("invalid engagement state")
			}
			if e.Tenant == tenant {
				items = append(items, e)
			}
			return nil
		})
	})
	return items, err
}
