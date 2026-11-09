package platform

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/Sanskarzz/tsunami/internal/toolcontract"
)

func (s *Server) StopEngagement(tenant, id string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids, err := s.Store.StopEngagement(tenant, id)
	if err != nil {
		return nil, err
	}
	for _, runID := range ids {
		for _, cancel := range s.active[runID] {
			cancel()
		}
	}
	return ids, nil
}

func (s *Server) serveEngagements(w http.ResponseWriter, r *http.Request) {
	tenant, ok := s.admin(r)
	if !ok {
		problem(w, 401, errors.New("operator authentication required"))
		return
	}
	if r.URL.Path == "/v1/engagements" {
		switch r.Method {
		case "GET":
			items, err := s.Store.ListEngagementSummaries(tenant)
			if err != nil {
				problem(w, 500, errors.New("engagement state unavailable"))
				return
			}
			writeJSON(w, 200, items)
		case "POST":
			var in EngagementInput
			if !decode(w, r, &in) {
				return
			}
			if in.Executor != "" && (in.Executor != toolcontract.CurlHTTP || !s.EnableCurlAdapter) {
				problem(w, 409, errors.New("reviewed executor unavailable"))
				return
			}
			if in.Application != "" {
				configured, ok := s.application(tenant, in.Application)
				if !ok {
					problem(w, 400, errors.New("configured application unavailable"))
					return
				}
				e, err := s.Store.CreateEngagement(tenant, configured.Plan.Origin, in, configured.Plan)
				if err != nil {
					problem(w, 400, err)
					return
				}
				writeJSON(w, 201, e)
				return
			}
			target, err := s.target(tenant, in.Scenario, in.Connectivity)
			if err != nil {
				problem(w, 400, errors.New("configured lab target required"))
				return
			}
			e, err := s.Store.CreateEngagement(tenant, target.URL, in)
			if err != nil {
				problem(w, 400, err)
				return
			}
			writeJSON(w, 201, e)
		default:
			w.WriteHeader(405)
		}
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || len(parts) > 5 {
		w.WriteHeader(404)
		return
	}
	id := parts[2]
	engagement, err := s.Store.GetEngagement(tenant, id)
	if err != nil {
		problem(w, 404, ErrNotFound)
		return
	}
	if len(parts) == 3 && r.Method == "GET" {
		view, err := s.engagementMetadataView(tenant, id)
		if err != nil {
			problem(w, 500, errors.New("engagement state unavailable"))
			return
		}
		writeJSON(w, 200, view)
		return
	}
	if len(parts) == 4 && parts[3] == "findings" {
		s.serveFindings(w, r, tenant, id, "")
		return
	}
	if len(parts) == 4 && parts[3] == "retest" {
		s.serveRetest(w, r, tenant, id)
		return
	}
	if len(parts) == 5 && parts[3] == "evidence" {
		s.serveFindings(w, r, tenant, id, parts[4])
		return
	}
	if len(parts) == 4 && parts[3] == "export" && r.Method == "GET" {
		bundle, err := s.Store.ExportEngagement(tenant, id)
		if err != nil {
			problem(w, 500, errors.New("engagement export unavailable"))
			return
		}
		writeJSON(w, 200, bundle)
		return
	}
	if len(parts) == 4 && parts[3] == "findings.csv" && r.Method == "GET" {
		bundle, err := s.Store.ExportEngagement(tenant, id)
		if err != nil {
			problem(w, 500, errors.New("finding export unavailable"))
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="findings.csv"`)
		_ = FindingsCSV(w, bundle.Findings)
		return
	}
	if len(parts) == 4 && r.Method == "POST" {
		switch parts[3] {
		case "reviews":
			var in ReviewInput
			if !decode(w, r, &in) {
				return
			}
			review, err := s.Store.ReviewFinding(tenant, id, in)
			if err != nil {
				problem(w, 409, err)
				return
			}
			writeJSON(w, 200, review)
			return
		case "approve":
			var in ApprovalInput
			if !decode(w, r, &in) {
				return
			}
			if err := s.Store.ApproveEngagement(tenant, id, in.ScopeDigest); err != nil {
				problem(w, 409, err)
				return
			}
			snapshot, err := s.engagementMetadataView(tenant, id)
			if err != nil {
				problem(w, 500, errors.New("engagement state unavailable"))
				return
			}
			writeJSON(w, 200, snapshot)
			return
		case "stop":
			var in struct{}
			if !decode(w, r, &in) {
				return
			}
			ids, err := s.StopEngagement(tenant, id)
			if err != nil {
				problem(w, 500, errors.New("engagement stop was not persisted"))
				return
			}
			writeJSON(w, 200, map[string]any{"stopped": ids})
			return
		case "skip":
			var in struct {
				Procedure string `json:"procedure"`
				Reason    string `json:"reason"`
			}
			if !decode(w, r, &in) {
				return
			}
			if err := s.Store.SkipEngagementItem(tenant, id, in.Procedure, in.Reason); err != nil {
				problem(w, 409, err)
				return
			}
			writeJSON(w, 200, map[string]bool{"recorded": true})
			return
		case "runs":
			if err := s.CheckAdmission(routeFor(Run{Application: engagement.Application, Connectivity: engagement.Connectivity}), false, engagement.Executor != ""); err != nil {
				problem(w, 503, err)
				return
			}
			var in struct {
				Assessment          string `json:"assessment"`
				Execution           string `json:"execution"`
				RequiredProvider    string `json:"required_provider"`
				RequiredPricePolicy string `json:"required_price_policy"`
			}
			if !decode(w, r, &in) {
				return
			}
			if in.Execution != "" && in.Execution != "scripted" && in.Execution != "live" {
				problem(w, 400, errors.New("unknown execution mode"))
				return
			}
			if engagement.Executor != "" && (!s.EnableCurlAdapter || engagement.Executor != toolcontract.CurlHTTP) {
				problem(w, 409, errors.New("reviewed executor unavailable"))
				return
			}
			var current Target
			var err error
			if engagement.Application != nil {
				configured, ok := s.application(tenant, engagement.Application.ID)
				if !ok || configured.Plan.Digest() != engagement.Application.Digest() {
					problem(w, 409, errors.New("application configuration changed"))
					return
				}
				if in.Execution == "live" {
					problem(w, 409, errors.New("application model execution awaits data-handling qualification"))
					return
				}
				current = configured.Target
			} else {
				current, err = s.target(tenant, engagement.Scenario, engagement.Connectivity)
			}
			if err != nil || current.URL != engagement.Target {
				problem(w, 409, errors.New("approved target configuration changed"))
				return
			}
			var model *ModelUsage
			if in.Execution == "live" {
				if engagement.Connectivity == "private" || s.Inference == nil || (s.Inference.Provider != "fixture" && s.Inference.Provider != "openrouter-free") || in.RequiredProvider != s.Inference.Provider {
					problem(w, 409, errors.New("explicit supported model policy required; private model execution unavailable"))
					return
				}
				if (s.Inference.Provider == "openrouter-free" && in.RequiredPricePolicy != inference.FreePolicyID) || (s.Inference.Provider == "fixture" && in.RequiredPricePolicy != "") {
					problem(w, 409, errors.New("required price policy differs"))
					return
				}
				model = &ModelUsage{Provider: s.Inference.Provider, Name: s.Inference.Model, ConfigID: s.Inference.Identity(), MaxCalls: inference.MaxCalls, MaxOutputTokens: inference.MaxOutputTokens, PricePolicy: s.Inference.PricePolicy()}
			} else if in.RequiredProvider != "" || in.RequiredPricePolicy != "" {
				problem(w, 400, errors.New("model policy supplied for scripted run"))
				return
			}
			run, err := s.Store.CreateEngagementRun(tenant, id, in.Assessment, model)
			if err != nil {
				problem(w, 409, err)
				return
			}
			writeJSON(w, 201, run)
			return
		}
	}
	w.WriteHeader(404)
}
