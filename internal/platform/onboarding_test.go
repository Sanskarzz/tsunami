package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/binding"
	setup "github.com/Sanskarzz/tsunami/internal/onboarding"
	bolt "go.etcd.io/bbolt"
)

type setupFixtureState struct {
	mu                 sync.Mutex
	requests           []string
	block              bool
	started, cancelled chan struct{}
}

func setupFixture(t *testing.T) (*Server, setup.Details, *setupFixtureState) {
	t.Helper()
	state := &setupFixtureState{started: make(chan struct{}), cancelled: make(chan struct{})}
	tokens := map[string]string{"owner": strings.Repeat("owner-token-", 5), "peer": strings.Repeat("peer-token-", 5)}
	server, store := fixture(t)
	server.AdminTokens = map[string]string{"admin-a": "demo-a", "admin-b": "demo-b"}
	server.SetupKey = bytes.Repeat([]byte{42}, 32)
	server.EnableCurlAdapter = true
	server.SetRouteHealth("applications", "ready", "")
	owner := []byte(`{"id":"record-a","owner":"user-a","secret":"synthetic-private-proof-value-for-owner-a"}`)
	peer := []byte(`{"id":"record-b","owner":"user-b","secret":"synthetic-private-proof-value-for-peer-b"}`)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		account := ""
		for k, v := range tokens {
			if r.Header.Get("Authorization") == "Bearer "+v {
				account = k
			}
		}
		if account == "" {
			w.WriteHeader(401)
			return
		}
		state.mu.Lock()
		state.requests = append(state.requests, account+" "+r.URL.Path)
		blocked := state.block
		state.block = false
		state.mu.Unlock()
		if blocked {
			close(state.started)
			<-r.Context().Done()
			close(state.cancelled)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/me":
			who := "user-a"
			if account == "peer" {
				who = "user-b"
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"id": who})
		case "/objects/record-a":
			_, _ = w.Write(owner)
		case "/objects/record-b":
			_, _ = w.Write(peer)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(api.Close)
	in := binding.Input{Version: "tsunami.application-binding/v1", ID: "integration", Title: "Owned integration", Profile: application.ObjectAccessProfile, Origin: api.URL, ApprovedAddresses: []string{"127.0.0.1"}, SyntheticDataOnly: true, ExpectedAccess: "deny", IdentityPath: "/me", IdentityPointer: "/id", ObjectPathTemplate: "/objects/{id}", ObjectIDPointer: "/id", OwnerPointer: "/owner", ProtectedPointers: []string{"/secret"}, Owner: binding.Account{Identity: binding.Document{Status: 200, Body: json.RawMessage(`{"id":"user-a"}`)}, Object: binding.Document{Status: 200, Body: owner}}, Peer: binding.Account{Identity: binding.Document{Status: 200, Body: json.RawMessage(`{"id":"user-b"}`)}, Object: binding.Document{Status: 200, Body: peer}}}
	raw, _ := json.Marshal(in)
	plan, e := binding.Compile(raw)
	if e != nil {
		t.Fatal(e)
	}
	if e = server.RegisterApplication(ApplicationConfig{Tenant: "demo-a", Plan: plan, Credentials: tokens}); e != nil {
		t.Fatal(e)
	}
	_ = store
	d := setup.Details{Name: "New owned API", Environment: "dev", FrontendURL: api.URL, AuthorizationRef: "owned-lab/onboarding", SourceApplication: "integration", SyntheticOnly: true}
	return server, d, state
}
func setupHTTP(t *testing.T, s *Server, method, path string, body any, status int) []byte {
	t.Helper()
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(b))
	r.Header.Set("Authorization", "Bearer admin-a")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != status {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, w.Code, status, w.Body.String())
	}
	return w.Body.Bytes()
}
func createOwnedSetup(t *testing.T, s *Server, d setup.Details) setup.Session {
	t.Helper()
	var v setup.Session
	if e := json.Unmarshal(setupHTTP(t, s, "POST", "/v1/onboarding", d, 201), &v); e != nil {
		t.Fatal(e)
	}
	return v
}
func validateOwnedSetup(t *testing.T, s *Server, v setup.Session) setup.Session {
	t.Helper()
	var out setup.Session
	raw := setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/validate", setupMutation{Revision: v.Revision, Confirm: true, Digest: v.PreflightDigest}, 200)
	if e := json.Unmarshal(raw, &out); e != nil {
		t.Fatal(e)
	}
	return out
}

func TestOnboardingFormQuestionsAreDurableScopedAndNoTraffic(t *testing.T) {
	s, d, state := setupFixture(t)
	d.SourceApplication = ""
	d.FrontendURL = "https://frontend.example.test"
	v := createOwnedSetup(t, s, d)
	if v.Question == nil || v.Question.Field != "api_origin" || v.State != "needs_input" {
		t.Fatal("missing backend not asked")
	}
	answer := setupMutation{Revision: v.Revision, QuestionID: v.Question.ID, Value: "https://api.example.test"}
	raw := setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/answers", answer, 200)
	var next setup.Session
	_ = json.Unmarshal(raw, &next)
	if next.Question.Field != "addresses" {
		t.Fatal("network permission not explicit")
	}
	setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/answers", answer, 409)
	r := httptest.NewRequest("GET", "/v1/onboarding/"+v.ID, nil)
	r.Header.Set("Authorization", "Bearer admin-b")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatal("cross-tenant setup readable")
	}
	r = httptest.NewRequest("GET", "/v1/onboarding", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("unauthenticated setup readable")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.requests) != 0 {
		t.Fatal("metadata contacted target")
	}
	stored, e := s.Store.GetSetup("demo-a", v.ID)
	if e != nil || stored.Question.ID != next.Question.ID {
		t.Fatal("question not persisted")
	}
}

func TestOnboardingValidationEncryptionAndAtomicDraftRecovery(t *testing.T) {
	s, d, state := setupFixture(t)
	v := createOwnedSetup(t, s, d)
	setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/validate", setupMutation{Revision: v.Revision, Confirm: true, Digest: strings.Repeat("0", 64)}, 409)
	state.mu.Lock()
	n := len(state.requests)
	state.mu.Unlock()
	if n != 0 {
		t.Fatal("wrong grant contacted target")
	}
	v = validateOwnedSetup(t, s, v)
	if v.State != "ready_for_approval" || v.Plan == nil || v.ValidationRequests != 4 || v.Plan.MinIntervalMillis != 1000 {
		t.Fatalf("validation failed: %+v", v)
	}
	state.mu.Lock()
	got := append([]string{}, state.requests...)
	state.mu.Unlock()
	expected := []string{"owner /me", "peer /me", "owner /objects/record-a", "peer /objects/record-b"}
	if strings.Join(got, ";") != strings.Join(expected, ";") {
		t.Fatal("preflight escaped legitimate baselines")
	}
	_ = s.Store.db.View(func(tx *bolt.Tx) error {
		for _, name := range []string{"onboarding", "onboarding_secrets", "onboarding_evidence"} {
			_ = tx.Bucket([]byte(name)).ForEach(func(_, b []byte) error {
				for _, secret := range []string{strings.Repeat("owner-token-", 5), strings.Repeat("peer-token-", 5), "synthetic-private-proof-value-for-owner-a"} {
					if bytes.Contains(b, []byte(secret)) {
						t.Error("secret/proof stored unencrypted")
					}
				}
				return nil
			})
		}
		return nil
	})
	var prepared, recovered setup.Session
	_ = json.Unmarshal(setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/prepare", setupMutation{Revision: v.Revision}, 200), &prepared)
	_ = json.Unmarshal(setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/prepare", setupMutation{Revision: v.Revision}, 200), &recovered)
	if prepared.EngagementID == "" || recovered.EngagementID != prepared.EngagementID {
		t.Fatal("lost prepare response duplicated draft")
	}
	eng, e := s.Store.GetEngagement("demo-a", prepared.EngagementID)
	if e != nil || eng.State != "draft" || len(eng.RunIDs) != 0 {
		t.Fatal("preparation started assessment")
	}
	a, ok := s.application("demo-a", v.Plan.ID)
	if !ok || a.Plan.Digest() != v.Plan.Digest() {
		t.Fatal("registered application unavailable")
	}
	if _, ok = s.application("demo-b", v.Plan.ID); ok {
		t.Fatal("cross-tenant application accessible")
	}
	badKey := bytes.Repeat([]byte{1}, 32)
	if _, e = s.Store.getSetupAccess("demo-a", v.ID, badKey); e == nil {
		t.Fatal("wrong encryption key accepted")
	}
	setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/stop", struct{}{}, 200)
	if _, ok = s.application("demo-a", v.Plan.ID); ok {
		t.Fatal("stopped application still available")
	}
	eng, _ = s.Store.GetEngagement("demo-a", prepared.EngagementID)
	if eng.State != "stopped" {
		t.Fatal("setup Stop did not close assessment")
	}
}

func TestOnboardingStopDuringPreflightAndGlobalStopGate(t *testing.T) {
	for _, global := range []bool{false, true} {
		t.Run(map[bool]string{false: "setup", true: "global"}[global], func(t *testing.T) {
			s, d, state := setupFixture(t)
			v := createOwnedSetup(t, s, d)
			state.mu.Lock()
			state.block = true
			state.mu.Unlock()
			done := make(chan error, 1)
			go func() { _, e := s.validateSetup(context.Background(), "demo-a", v.ID, v.Revision); done <- e }()
			select {
			case <-state.started:
			case <-time.After(3 * time.Second):
				t.Fatal("preflight did not start")
			}
			if global {
				if _, e := s.StopGroup("", true); e != nil {
					t.Fatal(e)
				}
			} else {
				setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/stop", struct{}{}, 200)
			}
			select {
			case <-state.cancelled:
			case <-time.After(3 * time.Second):
				t.Fatal("active request not cancelled")
			}
			select {
			case e := <-done:
				if e == nil {
					t.Fatal("late validation became ready")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("validation remained active")
			}
			stored, _ := s.Store.GetSetup("demo-a", v.ID)
			if stored.State != "stopped" || stored.Plan != nil {
				t.Fatal("revoked preflight promoted readiness")
			}
			state.mu.Lock()
			n := len(state.requests)
			state.mu.Unlock()
			if n != 1 {
				t.Fatal("preflight continued after Stop")
			}
		})
	}
}

func TestOnboardingPreflightDoesNotFollowRedirects(t *testing.T) {
	s, d, _ := setupFixture(t)
	var reached bool
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	defer other.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, other.URL, 302) }))
	defer origin.Close()
	a := s.Applications["demo-a"]["integration"]
	a.Plan.Origin = origin.URL
	s.Applications["demo-a"]["integration"] = a
	v := createOwnedSetup(t, s, d)
	v = validateOwnedSetup(t, s, v)
	if reached || v.State != "validation_failed" || v.ValidationCode != "baseline_not_successful" {
		t.Fatal("redirect was followed or called ready")
	}
}

func TestOnboardingUnsupportedAuthPrivateAddressesAndKeyLoss(t *testing.T) {
	s, d, _ := setupFixture(t)
	d.SourceApplication = ""
	d.APIOrigin = "https://api.example.test"
	d.ApprovedAddresses = []string{"1.1.1.1"}
	d.AuthMethod = "password"
	v := createOwnedSetup(t, s, d)
	if v.State != "unsupported" || v.Question.Field != "unsupported_auth" {
		t.Fatal("password adapter invented")
	}
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.100.100.200", "168.63.129.16", "::1", "fd00::1", "2001:db8::1"} {
		if publicSetupIP(netip.MustParseAddr(raw)) {
			t.Fatalf("unsafe public target accepted: %s", raw)
		}
	}
	if _, e := sealSetup(bytes.Repeat([]byte{2}, 32), setupAAD("a", "one", "access"), setupAccess{Tokens: map[string]string{"owner": "secret"}}); e != nil {
		t.Fatal(e)
	}
	s.SetupKey = nil
	setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/stop", struct{}{}, 200)
	setupHTTP(t, s, "POST", "/v1/onboarding", d, 503)
}

func TestOnboardingPersistsAcrossStoreReopen(t *testing.T) {
	file := filepath.Join(t.TempDir(), "onboard.db")
	store, e := OpenStore(file)
	if e != nil {
		t.Fatal(e)
	}
	d := setup.Details{Name: "Example", Environment: "dev", FrontendURL: "https://frontend.example.test", AuthorizationRef: "authorized/dev"}
	v, e := store.CreateSetup("a", d)
	if e != nil {
		t.Fatal(e)
	}
	_ = store.Close()
	store, e = OpenStore(file)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	next, e := store.GetSetup("a", v.ID)
	if e != nil || next.Question.ID != v.Question.ID || next.Revision != v.Revision {
		t.Fatal("pending question lost on restart")
	}
	_ = os.Chmod(file, 0600)
}

func TestOnboardingChangedScopeInvalidatesAccessAndStalePreflight(t *testing.T) {
	s, d, _ := setupFixture(t)
	v := createOwnedSetup(t, s, d)
	old := v.Revision
	changed := v.Details
	changed.OwnerPath = "/createdb"
	setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/details", map[string]any{"revision": old, "details": changed}, 409)
	setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/answers", setupMutation{Revision: old, QuestionID: "invented", Value: "https://other.example.test"}, 409)
	changed = v.Details
	changed.Name = "Revised owned application"
	var next setup.Session
	_ = json.Unmarshal(setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/details", map[string]any{"revision": old, "details": changed}, 200), &next)
	setupHTTP(t, s, "POST", "/v1/onboarding/"+v.ID+"/validate", setupMutation{Revision: old, Confirm: true, Digest: v.PreflightDigest}, 409)
	if next.Revision <= old || next.Plan != nil || next.ValidatedAt != nil {
		t.Fatal("scope edit retained readiness")
	}
}

func TestOnboardingRestartMarksUncertainValidationWithoutReplay(t *testing.T) {
	s, d, state := setupFixture(t)
	v := createOwnedSetup(t, s, d)
	_ = s.Store.db.Update(func(tx *bolt.Tx) error {
		v.State = "validating"
		v.ValidationProcess = "old-process"
		v.ValidationAttempts = 1
		v.ValidationRequests = 1
		return putSetup(tx, v)
	})
	var current setup.Session
	_ = json.Unmarshal(setupHTTP(t, s, "GET", "/v1/onboarding/"+v.ID, nil, 200), &current)
	if current.State != "interrupted" || current.ValidationCode != "previous_validation_uncertain" || current.ValidationAttempts != 1 {
		t.Fatal("uncertain work lost or called ready")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.requests) != 0 {
		t.Fatal("restart replayed a request")
	}
}

func TestCredentialReflectionAndAuthenticatedEncryption(t *testing.T) {
	tokens := map[string]string{"owner": "test-secret-credential"}
	for _, body := range []string{`{"value":"test-secret-credential"}`, `{"value":"test-secret-credenti\u0061l"}`, `{"nested":["prefix test-secret-credential suffix"]}`} {
		if !reflectsCredential(body, tokens) {
			t.Fatal("assigned credential reflection not detected")
		}
	}
	key := bytes.Repeat([]byte{4}, 32)
	b, e := sealSetup(key, setupAAD("tenant-a", "one", "access"), tokens)
	if e != nil {
		t.Fatal(e)
	}
	var out map[string]string
	if e = openSetup(key, setupAAD("tenant-b", "one", "access"), b, &out); e == nil {
		t.Fatal("cross-tenant ciphertext transplant accepted")
	}
	b[len(b)-1] ^= 1
	if e = openSetup(key, setupAAD("tenant-a", "one", "access"), b, &out); e == nil {
		t.Fatal("tampered ciphertext accepted")
	}
}

func TestOnboardedRateWaitStopsWithoutAnotherTargetRequest(t *testing.T) {
	s, store, c, flags := applicationFixture(t)
	c.Plan.ID = "rate-api"
	c.Plan.MinIntervalMillis = 5000
	if e := s.RegisterApplication(c); e != nil {
		t.Fatal(e)
	}
	_, r := applicationEngagement(t, s, c)
	if _, e := s.Execute(context.Background(), r.ID, Request{URL: r.Target + "/me", Account: "owner"}); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() {
		_, e := s.Execute(context.Background(), r.ID, Request{URL: r.Target + "/me", Account: "peer"})
		done <- e
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		view, _ := s.Snapshot(r.ID)
		if view.ActiveRequests == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("rate wait did not enter tracked operation")
		}
		time.Sleep(time.Millisecond)
	}
	if e := s.Stop(r.ID); e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("rate-limited request survived Stop")
		}
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel rate wait")
	}
	current, _ := store.Get(r.ID)
	if current.RequestCount != 1 || flags.requests.Load() != 1 {
		t.Fatal("cancelled wait reserved or sent another request")
	}
}
