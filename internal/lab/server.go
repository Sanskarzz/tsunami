// Package lab is an intentionally vulnerable synthetic fixture, never a scanner.
package lab

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Server struct {
	Fixed      bool
	RedirectTo string
	Requests   atomic.Int64
	Cancelled  atomic.Int64
	SlowActive atomic.Int64
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/stats" {
		_ = json.NewEncoder(w).Encode(map[string]int64{"requests": s.Requests.Load(), "cancelled": s.Cancelled.Load(), "slow_active": s.SlowActive.Load()})
		return
	}
	s.Requests.Add(1)
	if strings.HasPrefix(r.URL.Path, "/api/reviewed/") {
		s.reviewed(w, r)
		return
	}
	account := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer lab-")
	if account != "alice" && account != "bob" {
		http.Error(w, `{"error":"authentication required"}`, 401)
		return
	}
	if r.Method != "GET" {
		http.Error(w, `{"error":"read-only fixture"}`, 405)
		return
	}
	if r.URL.Path == "/api/slow" {
		s.SlowActive.Add(1)
		defer s.SlowActive.Add(-1)
		select {
		case <-r.Context().Done():
			s.Cancelled.Add(1)
			return
		case <-time.After(20 * time.Second):
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "finished"})
			return
		}
	}
	if r.URL.Path == "/api/redirect" {
		http.Redirect(w, r, s.RedirectTo, http.StatusFound)
		return
	}
	owner := ""
	switch r.URL.Path {
	case "/api/documents/doc-alice":
		owner = "alice"
	case "/api/documents/doc-bob":
		owner = "bob"
	}
	if owner == "" {
		http.Error(w, `{"error":"not found"}`, 404)
		return
	}
	if s.Fixed && account != owner {
		http.Error(w, `{"error":"forbidden"}`, 403)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"id": "doc-" + owner, "owner": owner, "content": "synthetic private invoice for " + owner})
}

func (s *Server) reviewed(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, `{"error":"read-only fixture"}`, 405)
		return
	}
	account := "anonymous"
	if token := r.Header.Get("Authorization"); token != "" {
		if token == "Bearer lab-alice" {
			account = "alice"
		} else if token == "Bearer lab-bob" {
			account = "bob"
		} else {
			http.Error(w, `{"error":"authentication required"}`, 401)
			return
		}
	}
	// Fixture data is independent of the procedure catalog. A mistaken
	// procedure expectation must fail evaluation, not change the application.
	type objectData struct {
		ID      string `json:"id"`
		Owner   string `json:"owner"`
		Content string `json:"content"`
	}
	objects := map[string]objectData{
		"/api/reviewed/objects/r-7f3c9a":     {"r-7f3c9a", "alice", "synthetic private ledger A"},
		"/api/reviewed/objects/r-b82d16":     {"r-b82d16", "bob", "synthetic private ledger B"},
		"/api/reviewed/objects/r-shared-40a": {"r-shared-40a", "alice", "synthetic team handover"},
		"/api/reviewed/admin/summary":        {"admin-summary", "administrators", "synthetic administrator audit summary"},
	}
	object, exists := objects[r.URL.Path]
	if !exists {
		http.Error(w, `{"error":"not found"}`, 404)
		return
	}
	if s.Fixed {
		if account == "anonymous" {
			http.Error(w, `{"error":"authentication required"}`, 401)
			return
		}
		if object.Owner == "administrators" && account != "alice" {
			http.Error(w, `{"error":"forbidden"}`, 403)
			return
		}
		if object.Owner != "administrators" && object.ID != "r-shared-40a" && object.Owner != account {
			http.Error(w, `{"error":"not found"}`, 404)
			return
		}
	}
	_ = json.NewEncoder(w).Encode(object)
}
