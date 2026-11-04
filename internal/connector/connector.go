// Package connector is a lab-only private HTTP execution point. It accepts
// signed single-request grants, never arbitrary proxy traffic or shell tools.
package connector

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/platform"
	bolt "go.etcd.io/bbolt"
)

const MaxGrantLifetime = 10 * time.Second

type Grant struct {
	ID               string    `json:"id"`
	Audience         string    `json:"audience"`
	Tenant           string    `json:"tenant"`
	RunID            string    `json:"run_id"`
	URL              string    `json:"url"`
	Account          string    `json:"account"`
	IssuedAt         time.Time `json:"issued_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	Assessment       string    `json:"assessment,omitempty"`
	AssessmentDigest string    `json:"assessment_digest,omitempty"`
}

type Envelope struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

func Sign(key ed25519.PrivateKey, g Grant) (Envelope, error) {
	if len(key) != ed25519.PrivateKeySize {
		return Envelope{}, errors.New("invalid signing key")
	}
	data, err := json.Marshal(g)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{base64.RawURLEncoding.EncodeToString(data), base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, data))}, nil
}

type Reply struct {
	Status   int    `json:"status"`
	Body     string `json:"body"`
	Location string `json:"location,omitempty"`
	Peer     string `json:"peer"`
}

type Server struct {
	Tenant, Audience string
	PublicKey        ed25519.PublicKey
	EmergencyKey     string
	Policies         []*platform.Policy
	TLSNotAfter      time.Time
	db               *bolt.DB
	mu               sync.Mutex // serializes cutoff persistence with action admission
	active           map[string]context.CancelFunc
}

func New(path, tenant, audience string, key ed25519.PublicKey, emergencyKey string, policies ...*platform.Policy) (*Server, error) {
	if tenant == "" || audience == "" || len(key) != ed25519.PublicKeySize || len(emergencyKey) < 32 || len(policies) == 0 {
		return nil, errors.New("connector identity, verification key, emergency key, and policies required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range []string{"used", "state"} {
			if _, e := tx.CreateBucketIfNotExists([]byte(name)); e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Server{Tenant: tenant, Audience: audience, PublicKey: append(ed25519.PublicKey(nil), key...), EmergencyKey: emergencyKey, Policies: policies, db: db, active: map[string]context.CancelFunc{}}, nil
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, cancel := range s.active {
		cancel()
	}
	return s.db.Close()
}

func (s *Server) decode(e Envelope) (Grant, *platform.Policy, error) {
	var g Grant
	data, err := base64.RawURLEncoding.DecodeString(e.Payload)
	sig, sigErr := base64.RawURLEncoding.DecodeString(e.Signature)
	if err != nil || sigErr != nil || !ed25519.Verify(s.PublicKey, data, sig) || json.Unmarshal(data, &g) != nil {
		return g, nil, errors.New("invalid permission signature")
	}
	now := time.Now()
	if g.Tenant != s.Tenant || g.Audience != s.Audience || len(g.ID) != 32 || len(g.RunID) != 32 ||
		g.IssuedAt.After(now.Add(250*time.Millisecond)) || !now.Before(g.ExpiresAt) || !g.ExpiresAt.After(g.IssuedAt) || g.ExpiresAt.Sub(g.IssuedAt) > MaxGrantLifetime ||
		(g.Account != "alice" && g.Account != "bob" && g.Account != "anonymous") {
		return g, nil, errors.New("permission identity or deadline denied")
	}
	for _, policy := range s.Policies {
		if g.Assessment != "" {
			d, err := assessment.Get(g.Assessment)
			if err != nil || d.Digest() != g.AssessmentDigest {
				return g, nil, errors.New("unknown or changed grant procedure")
			}
			policy = policy.WithPaths(d.Paths())
		}
		if u, err := policy.Validate(g.URL); err == nil {
			if g.Assessment != "" {
				d, _ := assessment.Get(g.Assessment)
				if !d.Allows(g.Account, u.Path) {
					return g, nil, errors.New("grant account/path pair denied")
				}
			} else if g.Account == "anonymous" || g.AssessmentDigest != "" {
				return g, nil, errors.New("anonymous requires a scoped procedure")
			}
			return g, policy, nil
		}
	}
	return g, nil, errors.New("connector target denied")
}

func (s *Server) admit(parent context.Context, g Grant) (context.Context, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.active) >= 8 || !time.Now().Before(g.ExpiresAt) {
		return nil, nil, errors.New("connector busy or permission expired")
	}
	err := s.db.Update(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte("state")).Get([]byte("cutoff")) != nil {
			return errors.New("emergency cutoff active")
		}
		b := tx.Bucket([]byte("used"))
		// Expired entries can be removed: expired grants are rejected separately.
		cursor := b.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var deadline time.Time
			if deadline.UnmarshalText(v) != nil {
				return errors.New("invalid replay state")
			}
			if !time.Now().Before(deadline) {
				if e := cursor.Delete(); e != nil {
					return e
				}
			}
		}
		if b.Get([]byte(g.ID)) != nil || b.Stats().KeyN >= 1024 {
			return errors.New("permission already consumed or replay store full")
		}
		value, _ := g.ExpiresAt.MarshalText()
		return b.Put([]byte(g.ID), value)
	})
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithDeadline(parent, g.ExpiresAt)
	s.active[g.ID] = cancel
	return ctx, func() { cancel(); s.mu.Lock(); delete(s.active, g.ID); s.mu.Unlock() }, nil
}

func (s *Server) cutoff() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte("state")).Put([]byte("cutoff"), []byte("disabled")) }); err != nil {
		return err
	}
	for _, cancel := range s.active {
		cancel()
	}
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	certificateDeadline, err := s.transportPermission(r)
	if err != nil {
		http.Error(w, "authenticated connector transport/role required", 403)
		return
	}
	transportCtx, cancelTransport := context.WithDeadline(r.Context(), certificateDeadline)
	defer cancelTransport()
	r = r.WithContext(transportCtx)
	if r.Method == "GET" && r.URL.Path == "/healthz" {
		w.WriteHeader(200)
		return
	}
	if (r.URL.Path == "/v1/cutoff" && r.Method == "POST") || (r.URL.Path == "/v1/status" && r.Method == "GET") {
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(s.EmergencyKey)) != 1 {
			http.Error(w, "emergency authentication required", 401)
			return
		}
		if r.Method == "POST" {
			if err := s.cutoff(); err != nil {
				http.Error(w, "cutoff persistence failed", 503)
				return
			}
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		var stopped bool
		if err := s.db.View(func(tx *bolt.Tx) error { stopped = tx.Bucket([]byte("state")).Get([]byte("cutoff")) != nil; return nil }); err != nil {
			http.Error(w, "state unavailable", 503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"cutoff": stopped, "active": len(s.active)})
		return
	}
	if r.Method != "POST" || r.URL.Path != "/v1/request" {
		http.NotFound(w, r)
		return
	}
	var envelope Envelope
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	d.DisallowUnknownFields()
	if d.Decode(&envelope) != nil || d.Decode(&struct{}{}) != io.EOF {
		http.Error(w, "invalid request", 400)
		return
	}
	g, policy, err := s.decode(envelope)
	if err != nil {
		http.Error(w, "permission or target denied", 403)
		return
	}
	ctx, release, err := s.admit(r.Context(), g)
	if err != nil {
		http.Error(w, "permission unavailable", 403)
		return
	}
	defer release()
	req, err := http.NewRequestWithContext(ctx, "GET", g.URL, nil)
	if err != nil {
		http.Error(w, "invalid target", 400)
		return
	}
	// Synthetic account secrets only. Agents and grant callers cannot supply
	// arbitrary Authorization, Host, methods, bodies, or extra target headers.
	if g.Account != "anonymous" {
		req.Header.Set("Authorization", "Bearer lab-"+g.Account)
	}
	req.Header.Set("X-Tsunami-Request-ID", g.ID)
	var peer string
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{GotConn: func(c httptrace.GotConnInfo) { peer = c.Conn.RemoteAddr().String() }}))
	client, closeTransport := policy.HTTPClient()
	defer closeTransport()
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "target request failed or permission expired", 502)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, (64<<10)+1))
	if err != nil || len(body) > 64<<10 || ctx.Err() != nil {
		http.Error(w, "target response interrupted or too large", 502)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Reply{resp.StatusCode, string(body), resp.Header.Get("Location"), peer})
}

// Client belongs only to the broker. Grants never reach workers. A private
// connector outage returns an error; it never triggers direct target dialing.
type Client struct {
	Endpoint, Tenant, Audience string
	Key                        ed25519.PrivateKey
	Lifetime                   time.Duration
	TLS                        *tls.Config
}

func (c Client) Permission(run platform.Run, req *http.Request) (Envelope, error) {
	if run.Tenant != c.Tenant || run.Connectivity != "private" {
		return Envelope{}, errors.New("private route identity denied")
	}
	life := c.Lifetime
	if life <= 0 || life > MaxGrantLifetime {
		return Envelope{}, errors.New("invalid grant lifetime")
	}
	now := time.Now().UTC()
	expiry := now.Add(life)
	if run.Deadline.Before(expiry) {
		expiry = run.Deadline
	}
	if deadline, ok := req.Context().Deadline(); ok && deadline.Before(expiry) {
		expiry = deadline
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return Envelope{}, err
	}
	g := Grant{ID: hex.EncodeToString(id), Audience: c.Audience, Tenant: c.Tenant, RunID: run.ID, URL: req.URL.String(), Account: strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer lab-"), IssuedAt: now, ExpiresAt: expiry}
	if run.Assessment != nil {
		if !run.Assessment.Canonical() || run.Assessment.Digest() != run.AssessmentDigest {
			return Envelope{}, errors.New("procedure changed")
		}
		g.Assessment = run.Assessment.ID
		g.AssessmentDigest = run.AssessmentDigest
		if g.Account == "" {
			g.Account = "anonymous"
		}
	}
	return Sign(c.Key, g)
}

func (c Client) Forward(ctx context.Context, run platform.Run, req *http.Request) (*http.Response, string, error) {
	if !strings.HasPrefix(c.Endpoint, "https://") || c.TLS == nil || c.TLS.InsecureSkipVerify || c.TLS.RootCAs == nil || len(c.TLS.Certificates) == 0 || c.TLS.VerifyConnection == nil {
		return nil, "", errors.New("verified connector mTLS required; plaintext fallback disabled")
	}
	envelope, err := c.Permission(run, req)
	if err != nil {
		return nil, "", err
	}
	data, _ := json.Marshal(envelope)
	request, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(c.Endpoint, "/")+"/v1/request", bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext, TLSClientConfig: c.TLS.Clone(), TLSHandshakeTimeout: 3 * time.Second, DisableKeepAlives: true, ResponseHeaderTimeout: 11 * time.Second, MaxResponseHeaderBytes: 16 << 10}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 12 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(request)
	if err != nil {
		return nil, "", errors.New("private connector unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, "", errors.New("private connector denied or interrupted request")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (128<<10)+1))
	if err != nil || len(body) > 128<<10 {
		return nil, "", errors.New("invalid connector response")
	}
	var out Reply
	if json.Unmarshal(body, &out) != nil || out.Status < 100 || out.Status > 599 || len(out.Body) > 64<<10 {
		return nil, "", errors.New("invalid connector response")
	}
	header := http.Header{}
	header.Set("Location", out.Location)
	return &http.Response{StatusCode: out.Status, Header: header, Body: io.NopCloser(strings.NewReader(out.Body)), Request: req}, out.Peer, nil
}
