package platform

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type OperationRecord struct {
	Process   string    `json:"process"`
	Kind      string    `json:"kind"`
	StartedAt time.Time `json:"started_at"`
}

// These are observations, never authority to dispatch or resume a run.
type Lifecycle struct {
	Version              int                        `json:"version"`
	FromCreation         bool                       `json:"tracked_from_creation"`
	DispatchIntentAt     *time.Time                 `json:"dispatch_intent_at,omitempty"`
	RevocationReason     string                     `json:"revocation_reason,omitempty"`
	Operations           map[string]OperationRecord `json:"operations,omitempty"`
	LastAdmittedAt       *time.Time                 `json:"last_admitted_at,omitempty"`
	LocalDrainObservedAt *time.Time                 `json:"local_drain_observed_at,omitempty"`
	PodUID               string                     `json:"pod_uid,omitempty"`
	WorkerObservedAt     *time.Time                 `json:"worker_observed_at,omitempty"`
	ExitObservedAt       *time.Time                 `json:"exit_observed_at,omitempty"`
	ObservedAt           *time.Time                 `json:"observed_at,omitempty"`
	ResourcesAbsentAt    *time.Time                 `json:"api_resources_absent_at,omitempty"`
	ObservationCode      string                     `json:"observation_code,omitempty"`
}

func tracked(r *Run) *Lifecycle {
	if r.Lifecycle == nil {
		r.Lifecycle = &Lifecycle{Version: 1}
	}
	if r.Lifecycle.Operations == nil {
		r.Lifecycle.Operations = map[string]OperationRecord{}
	}
	return r.Lifecycle
}

func revoke(r *Run, reason string, now time.Time) {
	l := tracked(r)
	l.RevocationReason = reason
	if len(l.Operations) == 0 && l.FromCreation {
		l.LocalDrainObservedAt = &now
	}
}

type ExecutionObservation struct {
	PodUID                           string
	Running, Exited, ResourcesAbsent bool
}
type ExecutionObserver interface {
	Observe(context.Context, Run) (ExecutionObservation, error)
}

func (s *Server) observeExecution(ctx context.Context, r Run) {
	observer, ok := s.Scheduler.(ExecutionObserver)
	if !ok || r.Lifecycle == nil || r.Lifecycle.ResourcesAbsentAt != nil {
		return
	}
	if at := r.Lifecycle.ObservedAt; at != nil && time.Since(*at) < time.Second {
		return
	}
	observeCtx, done := context.WithTimeout(ctx, 2*time.Second)
	observation, err := observer.Observe(observeCtx, r)
	done()
	_ = s.Store.Update(r.ID, func(current *Run) error {
		l := tracked(current)
		now := time.Now().UTC()
		l.ObservedAt = &now
		if err != nil {
			l.ObservationCode = "executor_observation_unavailable"
			return nil
		}
		l.ObservationCode = "observed"
		if observation.PodUID != "" {
			l.PodUID = observation.PodUID
		}
		if observation.Running && l.WorkerObservedAt == nil {
			l.WorkerObservedAt = &now
		}
		if observation.Exited && l.ExitObservedAt == nil {
			l.ExitObservedAt = &now
		}
		if observation.ResourcesAbsent && current.DeletionRequestedAt != nil {
			l.ResourcesAbsentAt = &now
		}
		return nil
	})
}

type RouteHealth struct {
	State string `json:"state"`
	Code  string `json:"code,omitempty"`
}

func (s *Server) SetRouteHealth(route, state, code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.health == nil {
		s.health = map[string]RouteHealth{}
	}
	s.health[route] = RouteHealth{State: state, Code: code}
}

func (s *Server) admissionLocked(route string, model, curl bool) error {
	if s.shuttingDown {
		return errors.New("service is shutting down")
	}
	checks := []string{"scheduler", route}
	if model {
		checks = append(checks, "model")
	}
	if curl {
		checks = append(checks, "curl")
	}
	for _, key := range checks {
		if h, ok := s.health[key]; ok && h.State != "ready" {
			return errors.New("execution dependency unavailable: " + key)
		}
	}
	return nil
}

func (s *Server) CheckAdmission(route string, model, curl bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.admissionLocked(route, model, curl)
}

func routeFor(r Run) string {
	if r.Application != nil {
		return "applications"
	}
	if r.Connectivity == "private" {
		return "private"
	}
	return "public"
}

func (s *Server) Quiesce() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shuttingDown = true
	for _, cancel := range s.setupActive {
		cancel()
	}
	for _, ops := range s.active {
		for _, cancel := range ops {
			cancel()
		}
	}
}

func (s *Server) finishOperation(id, op string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.active[id], op)
	_ = s.Store.Update(id, func(r *Run) error {
		l := tracked(r)
		if record, ok := l.Operations[op]; ok && record.Process == s.processID {
			delete(l.Operations, op)
		}
		if len(l.Operations) == 0 && r.StopAcceptedAt != nil && l.FromCreation {
			now := time.Now().UTC()
			l.LocalDrainObservedAt = &now
		}
		return nil
	})
}

type RunStatus struct {
	RunID                  string     `json:"run_id"`
	ExecutionMode          string     `json:"execution_mode"`
	Executor               string     `json:"executor"`
	TargetRequestsReserved int        `json:"target_requests_reserved"`
	Permission             string     `json:"permission"`
	RevocationReason       string     `json:"revocation_reason,omitempty"`
	RevokedAt              *time.Time `json:"revoked_at,omitempty"`
	ExpiresAt              time.Time  `json:"expires_at"`
	Execution              string     `json:"execution"`
	DispatchIntentAt       *time.Time `json:"dispatch_intent_at,omitempty"`
	WorkerObservedAt       *time.Time `json:"worker_observed_at,omitempty"`
	ExitObservedAt         *time.Time `json:"exit_observed_at,omitempty"`
	Activity               string     `json:"activity"`
	LocalOperations        int        `json:"local_operations"`
	UncertainOperations    int        `json:"uncertain_operations"`
	LastAdmittedAt         *time.Time `json:"last_admitted_at,omitempty"`
	LocalDrainObservedAt   *time.Time `json:"local_drain_observed_at,omitempty"`
	Cleanup                string     `json:"cleanup"`
	DeleteAcceptedAt       *time.Time `json:"delete_accepted_at,omitempty"`
	ResourcesAbsentAt      *time.Time `json:"api_resources_absent_at,omitempty"`
	ObservationCode        string     `json:"observation_code,omitempty"`
	Assessment             *Result    `json:"assessment,omitempty"`
}

func (s *Server) Status(id string) (RunStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.Store.Get(id)
	if err != nil {
		return RunStatus{}, err
	}
	out := RunStatus{RunID: id, ExecutionMode: r.Execution, Executor: r.Executor, TargetRequestsReserved: r.RequestCount, Permission: "not_active", ExpiresAt: r.Deadline, Execution: "unknown", Activity: "unknown", Cleanup: "not_requested", Assessment: r.Result, LocalOperations: len(s.active[id])}
	if r.State == Running {
		out.Permission = "active"
	}
	if !terminal(r.State) && time.Now().After(r.Deadline) {
		out.Permission = "expired"
	}
	if terminal(r.State) {
		out.Permission = "closed"
	}
	if r.StopAcceptedAt != nil {
		out.Permission = "revoked"
		out.RevokedAt = r.StopAcceptedAt
	}
	if r.State == Pending {
		out.Execution = "queued"
	}
	if r.DeletionRequestedAt != nil {
		out.Cleanup = "delete_accepted"
		out.DeleteAcceptedAt = r.DeletionRequestedAt
	} else if terminal(r.State) {
		out.Cleanup = "pending"
	}
	if l := r.Lifecycle; l != nil {
		out.RevocationReason = l.RevocationReason
		if l.RevocationReason == "deadline" {
			out.Permission = "expired"
		}
		out.DispatchIntentAt = l.DispatchIntentAt
		out.WorkerObservedAt = l.WorkerObservedAt
		out.ExitObservedAt = l.ExitObservedAt
		out.LastAdmittedAt = l.LastAdmittedAt
		out.LocalDrainObservedAt = l.LocalDrainObservedAt
		out.ResourcesAbsentAt = l.ResourcesAbsentAt
		out.ObservationCode = l.ObservationCode
		if l.DispatchIntentAt != nil {
			out.Execution = "dispatch_intent"
		}
		if l.WorkerObservedAt != nil {
			out.Execution = "worker_observed"
		}
		if l.ExitObservedAt != nil {
			out.Execution = "exit_observed"
		} else if l.ResourcesAbsentAt != nil {
			out.Execution = "unknown_after_resource_removal"
		}
		for key, op := range l.Operations {
			if op.Process != s.processID || s.active[id][key] == nil {
				out.UncertainOperations++
			}
		}
		if l.FromCreation {
			out.Activity = "no_local_work"
		}
		if out.LocalOperations > 0 {
			out.Activity = "active_locally"
		}
		if out.UncertainOperations > 0 {
			out.Activity = "uncertain_prior_or_unpersisted_completion"
		}
		if l.LocalDrainObservedAt != nil && out.UncertainOperations == 0 && out.LocalOperations == 0 {
			out.Activity = "drained_locally"
		}
		if l.ResourcesAbsentAt != nil {
			out.Cleanup = "api_resources_absent"
		}
	}
	return out, nil
}

func (s *Server) serveHealth(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/healthz" && r.URL.Path != "/livez" && r.URL.Path != "/readyz" && r.URL.Path != "/v1/status" {
		return false
	}
	if r.Method != "GET" {
		w.WriteHeader(405)
		return true
	}
	if r.URL.Path == "/healthz" || r.URL.Path == "/livez" {
		w.WriteHeader(200)
		return true
	}
	if r.URL.Path == "/v1/status" {
		if _, ok := s.admin(r); !ok {
			problem(w, 401, errors.New("operator authentication required"))
			return true
		}
	}
	err := s.Store.Ping()
	s.mu.Lock()
	stopping := s.shuttingDown
	health := map[string]RouteHealth{}
	for k, v := range s.health {
		health[k] = v
	}
	s.mu.Unlock()
	state := "ready"
	code := 200
	if err != nil || stopping {
		state = "unavailable"
		code = 503
	}
	if r.URL.Path == "/readyz" {
		writeJSON(w, code, map[string]string{"state": state, "scope": "state_and_revocation_api"})
	} else {
		writeJSON(w, code, map[string]any{"state": state, "scope": "startup_configuration_not_live_dependency_probe", "execution_routes": health})
	}
	return true
}
