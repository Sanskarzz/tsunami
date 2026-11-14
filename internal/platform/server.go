package platform

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/inference"
)

type Scheduler interface {
	Ensure(context.Context, Run, string) error
	Stop(context.Context, string) error
}

type Server struct {
	EnableCurlAdapter bool           // trusted startup switch; not a worker capability
	SetupKey          []byte         // optional key mounted only in the trusted control plane
	SetupDenied       []netip.Prefix // trusted infrastructure exclusions, outside customer/agent input
	setupActive       map[string]context.CancelFunc
	Applications      map[string]map[string]RegisteredApplication // tenant -> operator-owned application ID
	Store             *Store
	Targets           map[string]Target
	PrivateTargets    map[string]map[string]Target // tenant -> fixture; operator-owned
	// Token -> tenant. This operator API is for a local lab, not a customer portal.
	AdminTokens   map[string]string
	GlobalToken   string
	Scheduler     Scheduler
	Mode          string
	Inference     *inference.Config
	mu            sync.Mutex
	active        map[string]map[string]context.CancelFunc
	modelActive   map[string]bool
	modelInFlight int
	modelIngress  chan struct{}
	processID     string
	health        map[string]RouteHealth
	shuttingDown  bool
}

func NewServer(store *Store, targets map[string]Target, admins map[string]string) *Server {
	return &Server{Store: store, Targets: targets, AdminTokens: admins, Mode: "scripted Mecatl / synthetic lab only", active: map[string]map[string]context.CancelFunc{}, modelIngress: make(chan struct{}, 2), processID: randomID()}
}

func (s *Server) Snapshot(id string) (Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.Store.Get(id)
	r.ActiveRequests = len(s.active[id])
	if s.modelActive[id] {
		r.ActiveModelRequests = 1
	}
	return r, err
}

func equal(a, b string) bool {
	return a != "" && b != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func bearer(r *http.Request) string {
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
}

func (s *Server) admin(r *http.Request) (string, bool) {
	for token, tenant := range s.AdminTokens {
		if equal(bearer(r), token) {
			return tenant, true
		}
	}
	return "", false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	return decodeLimit(w, r, v, 4096)
}
func decodeLimit(w http.ResponseWriter, r *http.Request, v any, maxBytes int64) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		problem(w, 400, errors.New("invalid request body"))
		return false
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		problem(w, 400, errors.New("unexpected extra JSON"))
		return false
	}
	return true
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.serveHealth(w, r) {
		return
	}
	if r.URL.Path == "/v1/applications" {
		tenant, ok := s.admin(r)
		if !ok {
			problem(w, 401, errors.New("operator authentication required"))
			return
		}
		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}
		items := []any{}
		for _, configured := range s.Applications[tenant] {
			items = append(items, ApplicationSummary{Plan: configured.Plan, Metadata: s.applicationMetadata(tenant, configured.Plan)})
		}
		if len(s.SetupKey) == 32 {
			plans, err := s.Store.onboardedPlans(tenant)
			if err != nil {
				problem(w, 503, errors.New("application state unavailable"))
				return
			}
			for _, plan := range plans {
				items = append(items, ApplicationSummary{Plan: plan, Metadata: s.applicationMetadata(tenant, plan)})
			}
		}
		writeJSON(w, 200, items)
		return
	}
	if r.URL.Path == "/v1/onboarding" || strings.HasPrefix(r.URL.Path, "/v1/onboarding/") {
		s.serveOnboarding(w, r)
		return
	}
	if r.Method == "POST" && r.URL.Path == "/v1/stop-all" {
		if !equal(bearer(r), s.GlobalToken) {
			problem(w, 401, errors.New("operator authentication required"))
			return
		}
		ids, err := s.StopGroup("", true)
		if err != nil {
			problem(w, 500, errors.New("stop was not persisted"))
			return
		}
		writeJSON(w, 200, map[string]any{"stopped": ids})
		return
	}
	if r.Method == "POST" && r.URL.Path == "/v1/stop-tenant" {
		tenant, ok := s.admin(r)
		if !ok {
			problem(w, 401, errors.New("operator authentication required"))
			return
		}
		ids, err := s.StopGroup(tenant, false)
		if err != nil {
			problem(w, 500, errors.New("stop was not persisted"))
			return
		}
		writeJSON(w, 200, map[string]any{"stopped": ids})
		return
	}
	if r.URL.Path == "/v1/engagements" || strings.HasPrefix(r.URL.Path, "/v1/engagements/") {
		s.serveEngagements(w, r)
		return
	}
	if r.URL.Path == "/v1/runs" {
		tenant, ok := s.admin(r)
		if !ok {
			problem(w, 401, errors.New("operator authentication required"))
			return
		}
		switch r.Method {
		case "POST":
			var in struct {
				Scenario            string `json:"scenario"`
				Execution           string `json:"execution"`
				Connectivity        string `json:"connectivity"`
				RequiredPricePolicy string `json:"required_price_policy"`
				Assessment          string `json:"assessment"`
				RequiredProvider    string `json:"required_provider"`
			}
			if !decode(w, r, &in) {
				return
			}
			route := "public"
			if in.Connectivity == "private" {
				route = "private"
			}
			if err := s.CheckAdmission(route, in.Execution == "live", false); err != nil {
				problem(w, 503, err)
				return
			}
			if in.Scenario != "vulnerable" && in.Scenario != "fixed" && in.Scenario != "inconclusive" && in.Scenario != "redirect" && in.Scenario != "stop" {
				problem(w, 400, errors.New("unknown synthetic lab scenario"))
				return
			}
			if in.Execution != "" && in.Execution != "scripted" && in.Execution != "live" {
				problem(w, 400, errors.New("unknown execution mode"))
				return
			}
			if in.Connectivity != "" && in.Connectivity != "private" {
				problem(w, 400, errors.New("unknown connectivity mode"))
				return
			}
			if in.Assessment != "" {
				if _, err := assessment.Get(in.Assessment); err != nil {
					problem(w, 400, err)
					return
				}
				if in.Scenario != "vulnerable" && in.Scenario != "fixed" {
					problem(w, 400, errors.New("assessment requires a configured fixture variant"))
					return
				}
			}
			if in.Connectivity == "private" && in.Execution == "live" {
				problem(w, 409, errors.New("private lab currently supports scripted execution only"))
				return
			}
			if in.Execution == "live" && (s.Inference == nil || (in.Scenario != "vulnerable" && in.Scenario != "fixed")) {
				problem(w, 409, errors.New("live model mode is disabled or scenario is unsupported"))
				return
			}
			if in.Assessment != "" && in.Execution == "live" && s.Inference.Provider != "fixture" && s.Inference.Provider != "openrouter-free" {
				problem(w, 409, errors.New("catalog inference supports only offline fixture or free-only provider"))
				return
			}
			target, targetErr := s.target(tenant, in.Scenario, in.Connectivity)
			if in.RequiredPricePolicy != "" && (in.RequiredPricePolicy != inference.FreePolicyID || in.Execution != "live" || s.Inference == nil || s.Inference.PricePolicy() != in.RequiredPricePolicy) {
				problem(w, 409, errors.New("required price policy is not active"))
				return
			}
			if in.RequiredProvider != "" && (in.Execution != "live" || s.Inference == nil || in.RequiredProvider != s.Inference.Provider || (in.RequiredProvider != "fixture" && in.RequiredProvider != "openrouter-free")) {
				problem(w, 409, errors.New("required provider is not active"))
				return
			}
			if targetErr != nil {
				problem(w, 503, errors.New("fixture is not configured"))
				return
			}
			var run Run
			var err error
			if in.Execution == "live" {
				mode := "live Responses model / synthetic lab only"
				if s.Inference.Provider == "fixture" {
					mode = "Responses wire fixture / no live model"
				}
				if s.Inference.Provider == "openrouter-free" {
					mode = "OpenRouter free model / synthetic lab only"
				}
				model := ModelUsage{Provider: s.Inference.Provider, Name: s.Inference.Model, ConfigID: s.Inference.Identity(), MaxCalls: inference.MaxCalls, MaxOutputTokens: inference.MaxOutputTokens, PricePolicy: s.Inference.PricePolicy()}
				if in.Assessment != "" {
					run, err = s.Store.CreateAssessment(tenant, in.Scenario, target.URL, mode, in.Connectivity, in.Assessment, &model)
				} else {
					run, err = s.Store.CreateLive(tenant, in.Scenario, target.URL, mode, model)
				}
			} else if in.Assessment != "" {
				run, err = s.Store.CreateAssessment(tenant, in.Scenario, target.URL, s.Mode, in.Connectivity, in.Assessment, nil)
			} else if in.Connectivity == "private" {
				run, err = s.Store.CreatePrivate(tenant, in.Scenario, target.URL, s.Mode+" / private connector")
			} else {
				run, err = s.Store.Create(tenant, in.Scenario, target.URL, s.Mode)
			}
			if err != nil {
				problem(w, 409, err)
				return
			}
			writeJSON(w, 201, run)
		case "GET":
			s.listRuns(w, r, tenant)
		default:
			w.WriteHeader(405)
		}
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "v1" || parts[1] != "runs" {
		w.WriteHeader(404)
		return
	}
	id := parts[2]
	run, err := s.Store.Get(id)
	if err != nil {
		problem(w, 404, ErrNotFound)
		return
	}
	if len(parts) == 4 && (parts[3] == "request" || parts[3] == "finish" || parts[3] == "config" || parts[3] == "model") {
		token, e := s.Store.Credential(id)
		if e != nil || !equal(token, bearer(r)) {
			problem(w, 401, errors.New("run authentication required"))
			return
		}
		switch parts[3] {
		case "model":
			if r.Method != "POST" {
				w.WriteHeader(405)
				return
			}
			if s.Inference == nil || run.Execution != "live" || run.Model == nil || terminal(run.State) || time.Now().After(run.Deadline) || run.Model.CallsReserved >= run.Model.MaxCalls {
				problem(w, 409, errors.New("model permission unavailable"))
				return
			}
			// Bound body readers as well as upstream work. Stop uses another route
			// and never waits for these slots or a model request body to arrive.
			select {
			case s.modelIngress <- struct{}{}:
				defer func() { <-s.modelIngress }()
			default:
				problem(w, 429, errors.New("model gateway busy"))
				return
			}
			var in inference.Request
			if !decodeLimit(w, r, &in, inference.MaxRequestBytes) {
				return
			}
			reply, e := s.Infer(r.Context(), id, in)
			if e != nil {
				problem(w, 409, errors.New("model call denied, failed, or interrupted; inspect run state"))
				return
			}
			writeJSON(w, 200, reply)
		case "config":
			if r.Method != "GET" {
				w.WriteHeader(405)
				return
			}
			if terminal(run.State) {
				problem(w, 409, errors.New("run is terminal"))
				return
			}
			execution := run.Execution
			if execution == "" {
				execution = "scripted"
			}
			writeJSON(w, 200, map[string]any{"executor": run.Executor, "scenario": run.Scenario, "target": run.Target, "execution": execution, "workflow": run.Workflow, "deadline": run.Deadline.Format(time.RFC3339Nano), "assessment": run.Assessment, "assessment_digest": run.AssessmentDigest, "application": run.Application, "application_digest": run.ApplicationDigest})
		case "request":
			if r.Method != "POST" {
				w.WriteHeader(405)
				return
			}
			var in Request
			if !decode(w, r, &in) {
				return
			}
			captures, e := s.Execute(r.Context(), id, in)
			if e != nil {
				writeJSON(w, 403, map[string]any{"error": e.Error(), "captures": captures})
				return
			}
			writeJSON(w, 200, map[string]any{"captures": captures})
		case "finish":
			if r.Method != "POST" {
				w.WriteHeader(405)
				return
			}
			var in struct{}
			if !decode(w, r, &in) {
				return
			}
			result, e := s.Finish(id)
			if e != nil {
				var missing *MissingEvidence
				if errors.As(e, &missing) {
					writeJSON(w, 409, map[string]any{"error": missing.Error(), "required_reads": missing.Reads})
					return
				}
				problem(w, 409, e)
				return
			}
			writeJSON(w, 200, result)
		}
		return
	}
	tenant, ok := s.admin(r)
	if !ok {
		problem(w, 401, errors.New("operator authentication required"))
		return
	}
	if tenant != run.Tenant {
		problem(w, 404, ErrNotFound)
		return
	}
	if len(parts) == 4 && parts[3] == "events" {
		s.serveActivity(w, r, tenant, id)
		return
	}
	if len(parts) == 4 && parts[3] == "status" && r.Method == "GET" {
		status, err := s.Status(id)
		if err != nil {
			problem(w, 500, errors.New("run status unavailable"))
		} else {
			writeJSON(w, 200, status)
		}
		return
	}
	if len(parts) == 3 && r.Method == "GET" {
		snap, e := s.Snapshot(id)
		if e != nil {
			problem(w, 500, e)
			return
		}
		writeJSON(w, 200, snap)
		return
	}
	if len(parts) == 4 && parts[3] == "stop" && r.Method == "POST" {
		if e := s.Stop(id); e != nil {
			problem(w, 500, errors.New("stop was not persisted"))
			return
		}
		snap, _ := s.Snapshot(id)
		writeJSON(w, 200, snap)
		return
	}
	if len(parts) == 4 && parts[3] == "report" && r.Method == "GET" {
		snap, e := s.Snapshot(id)
		if e != nil {
			problem(w, 500, e)
			return
		}
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		_, _ = io.WriteString(w, Report(snap))
		return
	}
	w.WriteHeader(404)
}

func (s *Server) Stop(id string) error {
	return s.stop(id, "run_stop")
}

func (s *Server) stop(id, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.Store.Update(id, func(r *Run) error {
		if terminal(r.State) {
			return nil
		}
		now := time.Now().UTC()
		r.State = Stopped
		r.StopAcceptedAt = &now
		revoke(r, reason, now)
		r.Events = append(r.Events, Event{now, "stopped", "durable permission revoked before acknowledgement"})
		return nil
	})
	if err != nil {
		return err
	}
	for _, cancel := range s.active[id] {
		cancel()
	}
	return nil
}

func (s *Server) StopGroup(tenant string, global bool) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids, err := s.Store.StopGroup(tenant, global)
	if err != nil {
		return nil, err
	}
	// Assessment revocation has committed. Cancel admitted operations even if
	// the separate onboarding transaction fails; still report that failure.
	for _, id := range ids {
		for _, cancel := range s.active[id] {
			cancel()
		}
	}
	setupIDs, err := s.Store.stopSetups(tenant, global)
	for _, id := range setupIDs {
		if cancel := s.setupActive[id]; cancel != nil {
			cancel()
		}
	}
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *Server) begin(id string) (context.Context, context.CancelFunc, string, error) {
	return s.beginFor(id, 10*time.Second)
}
func (s *Server) beginFor(id string, timeout time.Duration) (context.Context, context.CancelFunc, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.Store.Get(id)
	if err != nil {
		return nil, nil, "", err
	}
	if err := s.admissionLocked(routeFor(r), r.Execution == "live", r.Executor != ""); err != nil {
		return nil, nil, "", err
	}
	if r.State != Running || time.Now().After(r.Deadline) {
		return nil, nil, "", errors.New("run has no permission to execute")
	}
	if len(r.Captures)+len(s.active[id]) >= 24 || len(s.active[id]) >= 2 {
		return nil, nil, "", errors.New("action budget exceeded")
	}
	deadline := time.Now().Add(timeout)
	if r.Deadline.Before(deadline) {
		deadline = r.Deadline
	}
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	op := randomID()
	err = s.Store.Update(id, func(r *Run) error {
		if r.State != Running || time.Now().After(r.Deadline) {
			return errors.New("run permission revoked")
		}
		kind := "target"
		if timeout > 10*time.Second {
			kind = "model"
		}
		l := tracked(r)
		if len(l.Operations) >= 32 {
			return errors.New("uncertain operation allowance exhausted")
		}
		l.Operations[op] = OperationRecord{Process: s.processID, Kind: kind, StartedAt: time.Now().UTC()}
		return nil
	})
	if err != nil {
		cancel()
		return nil, nil, "", err
	}
	if s.active[id] == nil {
		s.active[id] = map[string]context.CancelFunc{}
	}
	s.active[id][op] = cancel
	return ctx, cancel, op, nil
}

func (s *Server) Execute(parent context.Context, id string, in Request) ([]Capture, error) {
	ctx, cancel, op, err := s.begin(id)
	if err != nil {
		return nil, err
	}
	stopParent := context.AfterFunc(parent, cancel)
	defer func() { stopParent(); cancel(); s.finishOperation(id, op) }()
	r, err := s.Store.Get(id)
	if err != nil {
		return nil, err
	}
	target, err := s.targetForRun(r)
	if err != nil || target.URL != r.Target {
		return nil, errors.New("target route unavailable or changed")
	}
	if r.Assessment != nil {
		if !r.Assessment.Canonical() || r.Assessment.Digest() != r.AssessmentDigest {
			return nil, errors.New("assessment policy changed")
		}
		target.Policy = target.Policy.WithPaths(r.Assessment.Paths())
	}
	if r.Application != nil {
		target.Policy = target.Policy.WithPaths(r.Application.Paths())
	}
	transport := &http.Transport{Proxy: nil, DialContext: target.Policy.dial, DisableKeepAlives: true, ResponseHeaderTimeout: 8 * time.Second, MaxResponseHeaderBytes: 16 << 10}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	captures := []Capture{}
	raw := in.URL
	for hop := 0; hop < 4; hop++ {
		cap := Capture{ID: randomID(), URL: raw, Account: in.Account, StartedAt: time.Now().UTC()}
		u, e := target.Policy.Validate(raw)
		if e == nil && r.Assessment != nil && !r.Assessment.Allows(in.Account, u.Path) {
			e = errors.New("account/path pair outside the approved procedure")
		}
		if e == nil && r.Application != nil && !r.Application.Allows(in.Account, u.Path) {
			e = errors.New("account/path pair outside the approved application plan")
		}
		if e == nil && r.Assessment == nil && r.Application == nil && in.Account != "alice" && in.Account != "bob" {
			e = errors.New("unknown test account")
		}
		if e != nil {
			cap.Blocked = true
			cap.Error = e.Error()
			if err := s.capture(id, op, &cap); err != nil {
				return captures, err
			}
			captures = append(captures, cap)
			return captures, e
		}
		if e = s.reserveTarget(ctx, id, op, cap.ID, in.Account, u.Path); e != nil {
			return captures, e
		}
		req, e := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if e != nil {
			return captures, e
		}
		var responseCredentials map[string]string
		// Synthetic fixture credentials only; no agent-controlled Authorization or Host.
		if r.Application != nil {
			configured, ok := s.application(r.Tenant, r.Application.ID)
			if !ok {
				return captures, errors.New("application permission unavailable")
			}
			credential := configured.credentials[in.Account]
			responseCredentials = configured.credentials
			if credential == "" {
				return captures, errors.New("application credential unavailable")
			}
			req.Header.Set("Authorization", "Bearer "+credential)
		} else if in.Account != "anonymous" {
			req.Header.Set("Authorization", "Bearer lab-"+in.Account)
		}
		req.Header.Set("X-Tsunami-Request-ID", cap.ID)
		captureID := cap.ID
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
			GotConn: func(info httptrace.GotConnInfo) { cap.Peer = info.Conn.RemoteAddr().String() },
			WroteRequest: func(info httptrace.WroteRequestInfo) {
				if info.Err != nil {
					return
				}
				at := time.Now().UTC()
				if err := s.Store.Update(id, func(current *Run) error {
					appendActivity(current, "request.sent", op, captureID, ActivityData{ObservedAt: &at})
					return nil
				}); err != nil {
					slog.Warn("request send observation not persisted", "run", id)
				}
			},
		}))
		var resp *http.Response
		if target.Forward != nil {
			resp, cap.Peer, e = target.Forward(ctx, r, req)
		} else {
			resp, e = client.Do(req)
		}
		if e != nil {
			cap.Error = "request failed or was cancelled"
			if err := s.capture(id, op, &cap); err != nil {
				return captures, err
			}
			captures = append(captures, cap)
			return captures, errors.New(cap.Error)
		}
		cap.Status = resp.StatusCode
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, (64<<10)+1))
		_ = resp.Body.Close()
		if readErr != nil {
			cap.Error = "response interrupted"
		} else if len(body) > 64<<10 {
			cap.Error = "response size limit exceeded"
		} else {
			cap.Body = string(body)
		}
		if reflectsCredential(cap.Body, responseCredentials) {
			cap.Body = ""
			cap.Error = "response reflected an assigned credential"
		}
		if err := s.capture(id, op, &cap); err != nil {
			return captures, err
		}
		captures = append(captures, cap)
		if cap.Error != "" {
			return captures, errors.New(cap.Error)
		}
		if resp.StatusCode >= 300 && resp.StatusCode <= 399 && resp.Header.Get("Location") != "" {
			next, e := url.Parse(resp.Header.Get("Location"))
			if e != nil {
				return captures, errors.New("invalid redirect")
			}
			raw = u.ResolveReference(next).String()
			continue
		}
		return captures, nil
	}
	return captures, errors.New("redirect limit exceeded")
}

func (s *Server) target(tenant, scenario, connectivity string) (Target, error) {
	name := "vulnerable"
	if scenario == "fixed" {
		name = "fixed"
	}
	target := s.Targets[name]
	if connectivity == "private" {
		target = s.PrivateTargets[tenant][name]
		if target.Forward == nil {
			return Target{}, errors.New("private connector unavailable")
		}
	} else if connectivity != "" {
		return Target{}, errors.New("unknown connectivity")
	}
	if target.Policy == nil {
		return Target{}, errors.New("target policy unavailable")
	}
	return target, nil
}

func (s *Server) capture(id, op string, c *Capture) error {
	c.FinishedAt = time.Now().UTC()
	if err := s.Store.Update(id, func(r *Run) error {
		r.Captures = append(r.Captures, *c)
		outcome := "response_captured"
		if c.Error != "" {
			outcome = "failed_or_interrupted"
		}
		if c.Blocked {
			outcome = "blocked"
			appendActivity(r, "request.rejected", op, c.ID, ActivityData{Stage: "scope", Code: "scope_not_approved"})
		}
		appendActivity(r, "request.capture_persisted", op, c.ID, ActivityData{Status: c.Status, DurationMillis: c.FinishedAt.Sub(c.StartedAt).Milliseconds(), Outcome: outcome})
		return nil
	}); err != nil {
		slog.Error("evidence persistence failed", "run", id, "error", err)
		return errors.New("evidence persistence unavailable")
	}
	return nil
}

// Validation uses broker-captured fixture evidence, never a worker's verdict.
func Validate(r Run) Result {
	if r.Application != nil {
		return validateApplication(r)
	}
	if r.Assessment != nil {
		return validateAssessment(r)
	}
	out := Result{Verdict: "inconclusive", Title: "Cross-user document access", Reason: "Both known-owner baselines and a conclusive cross-user response are required.", EvidenceIDs: []string{}}
	var a, b, c *Capture
	for i := range r.Captures {
		x := &r.Captures[i]
		if x.Blocked || x.Error != "" {
			continue
		}
		var doc Document
		_ = json.Unmarshal([]byte(x.Body), &doc)
		u, e := url.Parse(x.URL)
		if e != nil {
			continue
		}
		if x.Account == "alice" && u.Path == "/api/documents/doc-alice" && x.Status == 200 && doc.ID == "doc-alice" && doc.Owner == "alice" && doc.Content != "" {
			a = x
		}
		if x.Account == "bob" && u.Path == "/api/documents/doc-bob" && x.Status == 200 && doc.ID == "doc-bob" && doc.Owner == "bob" && doc.Content != "" {
			b = x
		}
		if x.Account == "bob" && u.Path == "/api/documents/doc-alice" {
			c = x
		}
	}
	if a == nil || b == nil || c == nil {
		return out
	}
	out.EvidenceIDs = []string{a.ID, b.ID, c.ID}
	var doc Document
	_ = json.Unmarshal([]byte(c.Body), &doc)
	if c.Status == 200 && doc.ID == "doc-alice" && doc.Owner == "alice" && doc.Content != "" {
		out.Verdict = "confirmed"
		out.Title = "Broken Object Level Authorization (synthetic invoice API)"
		out.Reason = "Bob received Alice's known private fixture document. Both legitimate ownership baselines succeeded."
	} else if c.Status == 403 {
		out.Verdict = "no_issue_observed"
		out.Reason = "The two known-owner baselines succeeded; the cross-user read was forbidden. This is not proof of application-wide security."
	}
	return out
}

func (s *Server) Finish(id string) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result Result
	var feedback *MissingEvidence
	err := s.Store.Update(id, func(r *Run) error {
		if r.State != Running || time.Now().After(r.Deadline) {
			return errors.New("run is not running")
		}
		if len(s.active[id]) != 0 {
			return errors.New("requests still active")
		}
		result = Validate(*r)
		if r.Assessment != nil && r.Assessment.Canonical() && r.Assessment.Digest() == r.AssessmentDigest && result.Validation != nil && r.RequestCount < r.MaxRequests && r.CompletionFeedback < 2 {
			missing := missingAssessmentReads(*r)
			if len(missing) > 0 {
				r.CompletionFeedback++
				r.Events = append(r.Events, Event{time.Now().UTC(), "assessment_incomplete", "trusted missing-evidence feedback; permission scope unchanged"})
				feedback = &MissingEvidence{Reads: missing}
				return nil
			}
		}
		r.Result = &result
		r.State = Completed
		now := time.Now().UTC()
		r.FinishedAt = &now
		r.Events = append(r.Events, Event{now, "validated", result.Verdict})
		data := ActivityData{Verdict: result.Verdict}
		if v := result.Validation; v != nil {
			data.Validation = &ValidationCounts{Required: v.BaselinesRequired, Verified: v.BaselinesVerified, ProbeCaptured: v.ProbeCaptured}
		}
		for _, id := range result.EvidenceIDs {
			if activityRef(id) != "" && len(data.EvidenceIDs) < 24 {
				data.EvidenceIDs = append(data.EvidenceIDs, id)
			}
		}
		appendActivity(r, "validation.completed", "", "", data)
		return nil
	})
	if err == nil && feedback != nil {
		return Result{}, feedback
	}
	return result, err
}

func (s *Server) Reconcile(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runs, err := s.Store.List()
			if err != nil {
				slog.Error("state unavailable; no work dispatched")
				continue
			}
			for _, r := range runs {
				if !terminal(r.State) && time.Now().After(r.Deadline) {
					_ = s.stop(r.ID, "deadline")
					continue
				}
				if s.Scheduler == nil {
					continue
				}
				s.observeExecution(ctx, r)
				if terminal(r.State) {
					if r.DeletionRequestedAt != nil {
						continue
					}
					cleanupCtx, done := context.WithTimeout(ctx, 2*time.Second)
					err := s.Scheduler.Stop(cleanupCtx, r.ID)
					done()
					if err != nil {
						slog.Warn("workload cleanup pending", "run", r.ID)
					} else {
						_ = s.Store.Update(r.ID, func(run *Run) error {
							if run.DeletionRequestedAt == nil {
								now := time.Now().UTC()
								run.DeletionRequestedAt = &now
								appendActivity(run, "cleanup.delete_accepted", "", "", ActivityData{ObservedAt: &now})
							}
							return nil
						})
					}
					continue
				}
				allowCreate := false
				if r.State == Pending {
					if s.CheckAdmission(routeFor(r), r.Execution == "live", r.Executor != "") != nil {
						continue
					}
					s.mu.Lock()
					if s.admissionLocked(routeFor(r), r.Execution == "live", r.Executor != "") != nil {
						s.mu.Unlock()
						continue
					}
					err = s.Store.Update(r.ID, func(run *Run) error {
						if run.State == Pending {
							run.State = Running
							now := time.Now().UTC()
							tracked(run).DispatchIntentAt = &now
							appendActivity(run, "execution.dispatch_requested", "", "", ActivityData{ObservedAt: &now})
							allowCreate = true
						}
						return nil
					})
					s.mu.Unlock()
					if err != nil {
						continue
					}
				}
				latest, err := s.Store.Get(r.ID)
				if err != nil || terminal(latest.State) {
					continue
				}
				latest.AllowJobCreation = allowCreate
				token, err := s.Store.Credential(r.ID)
				if err != nil {
					continue
				}
				err = s.Scheduler.Ensure(ctx, latest, token)
				if err != nil {
					if errors.Is(err, ErrWorkerFailed) {
						_ = s.Store.Fail(r.ID)
						continue
					}
					slog.Warn("workload scheduling pending", "run", r.ID, "error", err)
				}
			}
		}
	}
}

func Report(r Run) string {
	if r.Application != nil && !r.ProcedureValid() {
		verdict := "unrecorded"
		if r.Result != nil {
			verdict = r.Result.Verdict
		}
		return fmt.Sprintf("# Historical application run\n\nRun: `%s`; stored state: %s; stored verdict: %s.\n\nThis application plan is unsupported or changed. The stored result is not revalidated under the current profile. Use the original run JSON and captured artifacts; no new reproduction steps are inferred.\n", r.ID, markdownText(r.State), markdownText(verdict))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Tsunami lab assessment\n\nRun: `%s`\n\nMode: %s\n\nScenario: %s\n\nState: %s\n\n", r.ID, r.Mode, r.Scenario, r.State)
	if r.Model == nil {
		b.WriteString("This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.\n\n")
	} else {
		if r.Model.Provider == "fixture" {
			b.WriteString("This uses a local Responses wire fixture, not a live model. It tests the real adapter and Mecatl loop without paid inference.\n\n")
		} else {
			b.WriteString("A live model selected tools for a reviewed synthetic workflow. This is not general autonomous vulnerability discovery or a customer assessment.\n\n")
		}
		fmt.Fprintf(&b, "Workflow: %s; provider: %s; model: %s.\n\nModel calls reserved: %d / %d; upstream attempts: %d / %d. Reported input/output tokens: %d / %d; usage uncertain: %t. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.\n\n", r.Workflow, r.Model.Provider, r.Model.Name, r.Model.CallsReserved, r.Model.MaxCalls, r.Model.WireAttempts, r.Model.MaxCalls, r.Model.InputTokens, r.Model.OutputTokens, r.Model.UsageUncertain)
		if r.Model.PricePolicy != "" {
			fmt.Fprintf(&b, "Price policy: %s (free model required; zero price ceilings; provider/model fallback disabled).\n\n", r.Model.PricePolicy)
		}
		for _, event := range r.Events {
			if event.Kind == "model_error" {
				fmt.Fprintf(&b, "Model error category: %s.\n\n", event.Detail)
			}
		}
		if len(r.Model.Calls) > 0 {
			b.WriteString("\n## Gateway call diagnostics\n\nTimes are measured at the gateway, not provider compute time or time to first token. Pending calls after restart have an unknown outcome.\n\n| Call | Outcome | Code | Phase | HTTP | Attempts | Elapsed ms | Headers ms | First adapter chunk ms | Released tools |\n|---|---|---|---|---|---|---|---|---|---|\n")
			for _, call := range r.Model.Calls {
				headers, first := "unknown", "unknown"
				if call.HeadersMS != nil {
					headers = fmt.Sprint(*call.HeadersMS)
				}
				if call.FirstChunkMS != nil {
					first = fmt.Sprint(*call.FirstChunkMS)
				}
				fmt.Fprintf(&b, "| %d | %s | %s | %s | %d | %d | %d | %s | %s | %d |\n", call.Sequence, call.Outcome, call.Code, call.Phase, call.HTTPStatus, call.WireAttempts, call.ElapsedMS, headers, first, call.ReleasedToolCalls)
			}
		}
	}
	if r.Result != nil {
		fmt.Fprintf(&b, "## Result\n\n%s: %s\n\n%s\n\n", r.Result.Verdict, markdownText(r.Result.Title), markdownText(r.Result.Reason))
		if r.Application != nil {
			fmt.Fprintf(&b, "Application: `%s`; plan digest: `%s`. This checks configured identities and one approved access rule.\n\n", r.Application.ID, r.ApplicationDigest)
			for i, read := range r.Application.Reads() {
				fmt.Fprintf(&b, "%d. Read `%s` using `%s`.\n", i+1, read.Path, read.Account)
			}
			b.WriteString("\nThe owner is checked again after the peer probe. Check application authorization on each object read. Impact/severity require review.\n\n")
		} else if r.Assessment != nil {
			fmt.Fprintf(&b, "Procedure: `%s`; digest: `%s`\n\nReview: %s\n\nApproved access rule: %s\n\nMissing-evidence feedback issued: %d / 2\n\n## Reproduction\n\n", r.Assessment.ID, r.AssessmentDigest, r.Assessment.Review, r.Assessment.ExpectedAccess, r.CompletionFeedback)
			for i, read := range r.Assessment.Reads() {
				fmt.Fprintf(&b, "%d. Read `%s` using the `%s` account alias.\n", i+1, read.Path, read.Account)
			}
			fmt.Fprintf(&b, "\nCompare the captured identity, owner, and content with the approved fixture facts. An HTTP 200 alone is insufficient.\n\n## Remediation or next action\n\n%s\n\n", r.Assessment.Remediation)
		} else if r.Result.Verdict == "confirmed" {
			b.WriteString("## Reproduction\n\n1. Read `doc-alice` using Alice's test session.\n2. Read `doc-bob` using Bob's test session.\n3. Read `doc-alice` using Bob's test session.\n4. Compare the returned ID, owner and synthetic private content with Alice's baseline. The expected result is access denied.\n\n## Remediation\n\nEnforce object ownership/authorization on the server for every document read. An authenticated session alone is not sufficient. Re-run the same known-owner checks against the fixed fixture.\n\n")
		}
	}
	fmt.Fprintf(&b, "## Evidence\n\nRequests reserved: %d / %d\n\n| Request | Account | HTTP | Blocked | Outcome |\n|---|---|---|---|---|\n", r.RequestCount, r.MaxRequests)
	for _, c := range r.Captures {
		fmt.Fprintf(&b, "| `%s` | %s | %d | %t | %s |\n", c.ID, markdownText(c.Account), c.Status, c.Blocked, markdownText(c.Error))
	}
	for _, c := range r.Captures {
		fmt.Fprintf(&b, "\n### Request %s\n\nGET %s\n\nAccount alias: %s; connected peer: %s\n\nStarted: %s; finished: %s\n\n%s", c.ID, markdownText(c.URL), markdownText(c.Account), markdownText(c.Peer), c.StartedAt.Format(time.RFC3339Nano), c.FinishedAt.Format(time.RFC3339Nano), evidenceBlock(c.Body))
	}
	if r.StopAcceptedAt != nil {
		fmt.Fprintf(&b, "\n## Stop\n\nDurable stop accepted: %s\n\n", r.StopAcceptedAt.Format(time.RFC3339Nano))
	}
	b.WriteString("\n## Limits\n\nThe validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.\n")
	return b.String()
}

func markdownText(text string) string {
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
	text = html.EscapeString(text)
	return strings.NewReplacer("\\", "\\\\", "`", "\\`", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)", "*", "\\*", "_", "\\_", "#", "\\#", "|", "\\|", "!", "\\!").Replace(text)
}

func evidenceBlock(body string) string {
	longest, current := 0, 0
	for _, r := range body {
		if r == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	length := longest + 1
	if length < 3 {
		length = 3
	}
	fence := strings.Repeat("`", length)
	return fence + "json\n" + body + "\n" + fence + "\n"
}
