package lab

import (
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

type PropertyRecord struct {
	ID          string `json:"id"`
	PrivateNote string `json:"private_note"`
}

type PropertyFixtureConfig struct {
	Credentials map[string]string         `json:"credentials"`
	Records     map[string]PropertyRecord `json:"records"`
}

// PropertyFixture is an owned read-only lab. Both records are shared, but each
// private_note belongs only to its owner. No arbitrary URLs or commands exist.
type PropertyFixture struct {
	config PropertyFixtureConfig
	mu     sync.Mutex
	stats  map[string]*StopStats
}

func NewPropertyFixture(c PropertyFixtureConfig) (*PropertyFixture, error) {
	if len(c.Credentials) != 2 || len(c.Records) != 2 {
		return nil, errors.New("two owned property-test identities required")
	}
	owned := PropertyFixtureConfig{Credentials: map[string]string{}, Records: map[string]PropertyRecord{}}
	for _, alias := range []string{"owner", "peer"} {
		token, record := c.Credentials[alias], c.Records[alias]
		if len(token) < 32 || len(token) > 4096 || strings.ContainsAny(token, "\r\n\t ") || len(record.PrivateNote) < 24 || len(record.PrivateNote) > 256 || len(record.ID) < 16 || len(record.ID) > 64 {
			return nil, errors.New("invalid owned property fixture configuration")
		}
		if _, err := hex.DecodeString(record.ID); err != nil {
			return nil, errors.New("opaque hex record IDs required")
		}
		owned.Credentials[alias], owned.Records[alias] = token, record
	}
	if owned.Credentials["owner"] == owned.Credentials["peer"] || owned.Records["owner"].ID == owned.Records["peer"].ID || owned.Records["owner"].PrivateNote == owned.Records["peer"].PrivateNote {
		return nil, errors.New("distinct owned identities and records required")
	}
	f := &PropertyFixture{config: owned, stats: map[string]*StopStats{}}
	for _, variant := range []string{"vulnerable", "filtered", "ambiguous", "stop"} {
		f.stats[variant] = &StopStats{}
	}
	return f, nil
}

func (f *PropertyFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	f.mu.Lock()
	stats, ok := f.stats[parts[1]]
	f.mu.Unlock()
	if !ok {
		w.WriteHeader(404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if len(parts) == 3 && parts[2] == "stats" {
		f.mu.Lock()
		snapshot := *stats
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(snapshot)
		return
	}
	alias := ""
	if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		for name, key := range f.config.Credentials {
			if subtle.ConstantTimeCompare([]byte(token), []byte(key)) == 1 {
				alias = name
			}
		}
	}
	if alias == "" {
		w.WriteHeader(401)
		return
	}
	identity := len(parts) == 3 && parts[2] == "me"
	owner := ""
	if len(parts) == 4 && parts[2] == "records" {
		for name, record := range f.config.Records {
			if record.ID == parts[3] {
				owner = name
			}
		}
	}
	if !identity && owner == "" {
		w.WriteHeader(404)
		return
	}
	now := time.Now().UTC()
	f.mu.Lock()
	stats.Requests++
	stats.LastRequestAt = &now
	f.mu.Unlock()
	if identity {
		_ = json.NewEncoder(w).Encode(map[string]string{"name": alias})
		return
	}
	probe := owner == "owner" && alias == "peer"
	if probe {
		f.mu.Lock()
		stats.ProbeStarted++
		stats.Active++
		stats.ProbeStartedAt = &now
		f.mu.Unlock()
		defer func() { f.mu.Lock(); stats.Active--; f.mu.Unlock() }()
		if parts[1] == "stop" {
			timer := time.NewTimer(30 * time.Second)
			defer timer.Stop()
			select {
			case <-r.Context().Done():
				at := time.Now().UTC()
				f.mu.Lock()
				stats.ProbeCancelled++
				stats.ProbeCancelledAt = &at
				f.mu.Unlock()
				return
			case <-timer.C:
			}
		}
		f.mu.Lock()
		stats.ProbeCompleted++
		f.mu.Unlock()
	}
	record := f.config.Records[owner]
	body := map[string]string{"id": record.ID, "owner": owner, "title": "Shared lab record", "private_note": record.PrivateNote}
	if owner != alias {
		switch parts[1] {
		case "filtered":
			delete(body, "private_note")
		case "ambiguous":
			body["private_note"] = "[redacted]"
		}
	}
	_ = json.NewEncoder(w).Encode(body)
}
