package lab

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

// StopFixture holds a synthetic cross-user read so an operator can observe
// request cancellation independently of the worker/broker's own reports.
type StopFixture struct {
	credentials map[string]string
	mu          sync.Mutex
	cases       map[string]*StopStats
}
type StopStats struct {
	Requests         int        `json:"requests"`
	Active           int        `json:"active"`
	ProbeStarted     int        `json:"probe_started"`
	ProbeCancelled   int        `json:"probe_cancelled"`
	ProbeCompleted   int        `json:"probe_completed"`
	LastRequestAt    *time.Time `json:"last_request_at,omitempty"`
	ProbeStartedAt   *time.Time `json:"probe_started_at,omitempty"`
	ProbeCancelledAt *time.Time `json:"probe_cancelled_at,omitempty"`
}

func NewStopFixture(credentials map[string]string) (*StopFixture, error) {
	if len(credentials) != 2 || len(credentials["owner"]) < 32 || len(credentials["peer"]) < 32 || credentials["owner"] == credentials["peer"] {
		return nil, errors.New("invalid synthetic stop fixture credentials")
	}
	copy := map[string]string{"owner": credentials["owner"], "peer": credentials["peer"]}
	s := &StopFixture{credentials: copy, cases: map[string]*StopStats{}}
	for _, id := range []string{"run", "engagement", "api-outage", "restart"} {
		s.cases[id] = &StopStats{}
	}
	return s, nil
}

func (s *StopFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" || r.URL.RawQuery != "" {
		w.WriteHeader(400)
		return
	}
	if r.URL.Path == "/healthz" {
		w.WriteHeader(200)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "cases" {
		w.WriteHeader(404)
		return
	}
	s.mu.Lock()
	stats, ok := s.cases[parts[1]]
	s.mu.Unlock()
	if !ok {
		w.WriteHeader(404)
		return
	}
	if len(parts) == 3 && parts[2] == "stats" {
		s.mu.Lock()
		copy := *stats
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(copy)
		return
	}
	alias := ""
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	for name, key := range s.credentials {
		if subtle.ConstantTimeCompare([]byte(token), []byte(key)) == 1 {
			alias = name
		}
	}
	if alias == "" {
		w.WriteHeader(401)
		return
	}
	identity := len(parts) == 3 && parts[2] == "me"
	object := len(parts) == 4 && parts[2] == "objects" && (parts[3] == "owner" || parts[3] == "peer")
	if !identity && !object {
		w.WriteHeader(404)
		return
	}
	now := time.Now().UTC()
	s.mu.Lock()
	stats.Requests++
	stats.LastRequestAt = &now
	s.mu.Unlock()
	if object && parts[3] == "owner" && alias == "peer" {
		s.mu.Lock()
		stats.ProbeStarted++
		stats.Active++
		stats.ProbeStartedAt = &now
		s.mu.Unlock()
		defer func() { s.mu.Lock(); stats.Active--; s.mu.Unlock() }()
		select {
		case <-r.Context().Done():
			s.mu.Lock()
			stats.ProbeCancelled++
			at := time.Now().UTC()
			stats.ProbeCancelledAt = &at
			s.mu.Unlock()
			return
		case <-time.After(30 * time.Second):
			s.mu.Lock()
			stats.ProbeCompleted++
			s.mu.Unlock()
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if identity {
		_ = json.NewEncoder(w).Encode(map[string]string{"name": alias})
		return
	}
	owner := parts[3]
	_ = json.NewEncoder(w).Encode(map[string]string{"id": owner, "owner": owner, "proof": "synthetic-stop-proof-" + owner})
}
