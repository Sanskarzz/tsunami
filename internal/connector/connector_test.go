package connector

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/platform"
)

const emergency = "synthetic-emergency-key-for-native-tests"

type testLab struct {
	key                      ed25519.PrivateKey
	pub                      ed25519.PublicKey
	target                   platform.Target
	server                   *Server
	http                     *httptest.Server
	path                     string
	tlsDir                   string
	ca                       *authority
	brokerHTTP, operatorHTTP *http.Client
}

func newLab(t *testing.T, handler http.Handler) *testLab {
	t.Helper()
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	target, err := platform.LabTarget(context.Background(), web.URL)
	if err != nil {
		t.Fatal(err)
	}
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	ca, err := newAuthority(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "ca.crt"), ca.certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	writeTestLeaf(t, dir, ca, "connector", ConnectorIdentity("demo-a", "customer-a"), []string{"127.0.0.1"}, x509.ExtKeyUsageServerAuth, time.Now().Add(-time.Minute), time.Now().Add(time.Hour))
	for _, role := range []string{"broker", "emergency-operator"} {
		writeTestLeaf(t, dir, ca, role, Identity("demo-a", role), nil, x509.ExtKeyUsageClientAuth, time.Now().Add(-time.Minute), time.Now().Add(time.Hour))
	}
	l := &testLab{key: key, pub: pub, target: target, path: filepath.Join(dir, "connector.db"), tlsDir: dir, ca: ca}
	l.brokerHTTP = testHTTPClient(t, dir, "broker", "127.0.0.1", "demo-a", "customer-a")
	l.operatorHTTP = testHTTPClient(t, dir, "emergency-operator", "127.0.0.1", "demo-a", "customer-a")
	l.restart(t)
	t.Cleanup(func() { l.http.Close(); _ = l.server.Close() })
	return l
}

func (l *testLab) restart(t *testing.T) {
	t.Helper()
	if l.server != nil {
		l.http.Close()
		if err := l.server.Close(); err != nil {
			t.Fatal(err)
		}
	}
	s, err := New(l.path, "demo-a", "customer-a", l.pub, emergency, l.target.Policy)
	if err != nil {
		t.Fatal(err)
	}
	l.server = s
	config, expiry, err := LoadServerTLS(filepath.Join(l.tlsDir, "ca.crt"), filepath.Join(l.tlsDir, "connector.crt"), filepath.Join(l.tlsDir, "connector.key"), "127.0.0.1", "demo-a", "customer-a")
	if err != nil {
		t.Fatal(err)
	}
	s.TLSNotAfter = expiry
	l.http = httptest.NewUnstartedServer(s)
	l.http.TLS = config
	l.http.StartTLS()
}

func (l *testLab) grant(path string, lifetime time.Duration) Grant {
	now := time.Now().UTC()
	return Grant{ID: fmt.Sprintf("%032x", now.UnixNano()), Audience: "customer-a", Tenant: "demo-a", RunID: strings.Repeat("a", 32), URL: l.target.URL + path, Account: "alice", IssuedAt: now, ExpiresAt: now.Add(lifetime)}
}

func post(t *testing.T, client *http.Client, endpoint string, envelope Envelope) int {
	t.Helper()
	data, _ := json.Marshal(envelope)
	resp, err := client.Post(endpoint+"/v1/request", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestGrantCannotChangeTenantAudienceDestinationOrLifetime(t *testing.T) {
	var hits atomic.Int64
	l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(200) }))
	for _, name := range []string{"tenant", "audience", "target", "account", "expired", "too-long", "future", "signature"} {
		t.Run(name, func(t *testing.T) {
			g := l.grant("/api/documents/doc-alice", time.Second)
			switch name {
			case "tenant":
				g.Tenant = "demo-b"
			case "audience":
				g.Audience = "customer-b"
			case "target":
				g.URL = "http://not-approved.invalid/api/documents/doc-alice"
			case "account":
				g.Account = "other-customer-secret"
			case "expired":
				g.IssuedAt = time.Now().Add(-time.Second)
				g.ExpiresAt = time.Now().Add(-time.Millisecond)
			case "too-long":
				g.ExpiresAt = g.IssuedAt.Add(11 * time.Second)
			case "future":
				g.IssuedAt = time.Now().Add(time.Hour)
				g.ExpiresAt = g.IssuedAt.Add(time.Second)
			}
			e, err := Sign(l.key, g)
			if err != nil {
				t.Fatal(err)
			}
			if name == "signature" {
				e.Payload += "A"
			}
			if code := post(t, l.brokerHTTP, l.http.URL, e); code != 403 {
				t.Fatalf("accepted %s: %d", name, code)
			}
		})
	}
	if hits.Load() != 0 {
		t.Fatal("denied grant reached target")
	}
}

func TestReplayRejectedAcrossConnectorRestart(t *testing.T) {
	var hits atomic.Int64
	l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(200) }))
	e, err := Sign(l.key, l.grant("/api/documents/doc-alice", MaxGrantLifetime))
	if err != nil {
		t.Fatal(err)
	}
	if code := post(t, l.brokerHTTP, l.http.URL, e); code != 200 {
		t.Fatal(code)
	}
	if code := post(t, l.brokerHTTP, l.http.URL, e); code != 403 {
		t.Fatal("replay accepted")
	}
	l.restart(t)
	if code := post(t, l.brokerHTTP, l.http.URL, e); code != 403 {
		t.Fatal("replay accepted after restart")
	}
	if hits.Load() != 1 {
		t.Fatal("replayed target request")
	}
}

func TestLeaseExpiryCancelsTargetWithoutControlPlane(t *testing.T) {
	started, cancelled := make(chan struct{}), make(chan struct{})
	l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Expire an already-open response body, not only a connection waiting
		// for headers. No partial response may be released as usable evidence.
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"partial":`))
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	g := l.grant("/api/slow", 250*time.Millisecond)
	e, _ := Sign(l.key, g)
	done := make(chan int, 1)
	go func() { done <- post(t, l.brokerHTTP, l.http.URL, e) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("target not started")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("lease expiry did not cancel target")
	}
	if code := <-done; code != 502 {
		t.Fatalf("expired response accepted: %d", code)
	}
	if code := post(t, l.brokerHTTP, l.http.URL, e); code != 403 {
		t.Fatal("expired grant reused")
	}
}

func TestEmergencyCutoffCancelsActiveWorkAndSurvivesRestart(t *testing.T) {
	started, cancelled := make(chan struct{}), make(chan struct{})
	l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done(); close(cancelled) }))
	e, _ := Sign(l.key, l.grant("/api/slow", 3*time.Second))
	done := make(chan int, 1)
	go func() { done <- post(t, l.brokerHTTP, l.http.URL, e) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("target not started")
	}
	for _, token := range []string{"wrong-key", emergency} {
		req, _ := http.NewRequest("POST", l.http.URL+"/v1/cutoff", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := l.operatorHTTP.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		want := 200
		if token == "wrong-key" {
			want = 401
		}
		if resp.StatusCode != want {
			t.Fatal(resp.StatusCode)
		}
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("emergency cutoff did not cancel target")
	}
	if code := <-done; code != 502 {
		t.Fatal(code)
	}
	l.restart(t)
	fresh, _ := Sign(l.key, l.grant("/api/documents/doc-alice", time.Second))
	if code := post(t, l.brokerHTTP, l.http.URL, fresh); code != 403 {
		t.Fatal("restart cleared emergency cutoff")
	}
}

func privateBroker(t *testing.T, l *testLab) (*platform.Server, platform.Run) {
	t.Helper()
	store, err := platform.OpenStore(filepath.Join(t.TempDir(), "broker.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	c := Client{Endpoint: l.http.URL, Tenant: "demo-a", Audience: "customer-a", Key: l.key, Lifetime: MaxGrantLifetime, TLS: l.brokerHTTP.Transport.(*http.Transport).TLSClientConfig.Clone()}
	target := l.target
	target.Forward = c.Forward
	s := platform.NewServer(store, map[string]platform.Target{"vulnerable": l.target, "fixed": l.target}, nil)
	s.PrivateTargets = map[string]map[string]platform.Target{"demo-a": {"vulnerable": target, "fixed": target}}
	r, err := store.CreatePrivate("demo-a", "vulnerable", target.URL, "private native test")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Update(r.ID, func(r *platform.Run) error { r.State = platform.Running; return nil }); err != nil {
		t.Fatal(err)
	}
	return s, r
}

func TestBrokerStopCancelsPrivateRequest(t *testing.T) {
	for _, mode := range []string{"run", "tenant", "global"} {
		t.Run(mode, func(t *testing.T) {
			started, cancelled := make(chan struct{}), make(chan struct{})
			l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done(); close(cancelled) }))
			s, r := privateBroker(t, l)
			done := make(chan error, 1)
			go func() {
				_, err := s.Execute(context.Background(), r.ID, platform.Request{URL: r.Target + "/api/slow", Account: "alice"})
				done <- err
			}()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("target not started")
			}
			var err error
			switch mode {
			case "run":
				err = s.Stop(r.ID)
			case "tenant":
				_, err = s.StopGroup("demo-a", false)
			case "global":
				_, err = s.StopGroup("", true)
			}
			if err != nil {
				t.Fatal(err)
			}
			select {
			case <-cancelled:
			case <-time.After(time.Second):
				t.Fatal("stop did not reach target")
			}
			if err := <-done; err == nil {
				t.Fatal("cancelled request succeeded")
			}
			if _, err := s.Execute(context.Background(), r.ID, platform.Request{URL: r.Target + "/api/slow", Account: "alice"}); err == nil {
				t.Fatal("stopped run resumed")
			}
		})
	}
}

func TestPrivateConnectorOutageHasNoDirectFallback(t *testing.T) {
	var hits atomic.Int64
	l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(200) }))
	s, r := privateBroker(t, l)
	l.http.Close()
	if _, err := s.Execute(context.Background(), r.ID, platform.Request{URL: r.Target + "/api/documents/doc-alice", Account: "alice"}); err == nil {
		t.Fatal("unavailable connector succeeded")
	}
	if hits.Load() != 0 {
		t.Fatal("broker fell back to direct private access")
	}
}

func TestPrivateRedirectCannotRelayToAnotherOrigin(t *testing.T) {
	var hits atomic.Int64
	trap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer trap.Close()
	l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, trap.URL+"/api/documents/doc-alice", 302)
	}))
	s, r := privateBroker(t, l)
	captures, err := s.Execute(context.Background(), r.ID, platform.Request{URL: r.Target + "/api/redirect", Account: "alice"})
	if err == nil || len(captures) != 2 || !captures[1].Blocked || hits.Load() != 0 {
		t.Fatal("private redirect boundary failed")
	}
}

func TestConnectorRechecksPinnedProcedureAndAnonymousCapability(t *testing.T) {
	var hits atomic.Int64
	auth := make(chan string, 1)
	l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		auth <- r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	d, err := assessment.Get("anonymous-access/v1")
	if err != nil {
		t.Fatal(err)
	}
	g := l.grant(d.Probe.Path, time.Second)
	g.Account = "anonymous"
	g.Assessment = d.ID
	g.AssessmentDigest = d.Digest()
	e, _ := Sign(l.key, g)
	if code := post(t, l.brokerHTTP, l.http.URL, e); code != 200 {
		t.Fatal(code)
	}
	if <-auth != "" {
		t.Fatal("private anonymous request leaked Authorization")
	}
	for _, variant := range []string{"missing-procedure", "wrong-digest", "different-path", "different-account"} {
		bad := l.grant(d.Probe.Path, time.Second)
		bad.Account = "anonymous"
		bad.Assessment = d.ID
		bad.AssessmentDigest = d.Digest()
		switch variant {
		case "missing-procedure":
			bad.Assessment = ""
			bad.AssessmentDigest = ""
		case "wrong-digest":
			bad.AssessmentDigest = "modified"
		case "different-path":
			bad.URL = l.target.URL + "/api/reviewed/admin/summary"
		case "different-account":
			bad.Account = "bob"
		}
		e, _ := Sign(l.key, bad)
		if code := post(t, l.brokerHTTP, l.http.URL, e); code != 403 {
			t.Fatalf("procedure bypass %s: %d", variant, code)
		}
	}
	if hits.Load() != 1 {
		t.Fatal("denied procedure reached target")
	}
}

func writeTestLeaf(t *testing.T, dir string, ca *authority, name, identity string, names []string, usage x509.ExtKeyUsage, start, end time.Time) {
	t.Helper()
	cert, key, err := ca.issue(identity, names, usage, start, end)
	if err != nil {
		t.Fatal(err)
	}
	for filename, data := range map[string][]byte{name + ".crt": cert, name + ".key": key} {
		if err := os.WriteFile(filepath.Join(dir, filename), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func testHTTPClient(t *testing.T, dir, role, serverName, tenant, audience string) *http.Client {
	t.Helper()
	cfg, err := LoadClientTLS(filepath.Join(dir, "ca.crt"), filepath.Join(dir, role+".crt"), filepath.Join(dir, role+".key"), serverName, tenant, audience, role)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil, TLSClientConfig: cfg, DisableKeepAlives: true}, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
}
