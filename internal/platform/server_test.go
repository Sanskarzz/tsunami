package platform

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Server, *Store) {
	t.Helper()
	store, err := OpenStore(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"doc-alice","owner":"alice","content":"synthetic"}`))
	}))
	t.Cleanup(target.Close)
	lt, err := LabTarget(context.Background(), target.URL)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, map[string]Target{"vulnerable": lt, "fixed": lt}, map[string]string{"operator-a": "a", "operator-b": "b"})
	server.GlobalToken = "global"
	return server, store
}

func running(t *testing.T, s *Server, tenant string) Run {
	t.Helper()
	r, err := s.Store.Create(tenant, "vulnerable", s.Targets["vulnerable"].URL, s.Mode)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Store.Update(r.ID, func(r *Run) error { r.State = Running; return nil }); err != nil {
		t.Fatal(err)
	}
	r, _ = s.Store.Get(r.ID)
	return r
}

func request(s *Server, method, path, token, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

func TestTenantIdentityAndRunCredentialsCannotBeSubstituted(t *testing.T) {
	s, _ := fixture(t)
	r := running(t, s, "a")
	other := running(t, s, "b")
	otherToken, _ := s.Store.Credential(other.ID)
	if w := request(s, "GET", "/v1/runs/"+r.ID, "operator-b", ""); w.Code != 404 {
		t.Fatalf("cross-tenant read: %d", w.Code)
	}
	if w := request(s, "POST", "/v1/runs/"+r.ID+"/stop", "operator-b", ""); w.Code != 404 {
		t.Fatalf("cross-tenant stop: %d", w.Code)
	}
	if w := request(s, "GET", "/v1/runs/"+r.ID+"/config", otherToken, ""); w.Code != 401 {
		t.Fatalf("other run credential accepted: %d", w.Code)
	}
	if w := request(s, "POST", "/v1/runs", "operator-a", `{"scenario":"vulnerable","tenant":"b"}`); w.Code != 400 {
		t.Fatalf("caller-selected tenant accepted: %d", w.Code)
	}
	if w := request(s, "POST", "/v1/runs", "operator-a", `{"scenario":"vulnerable","target":"http://external.test"}`); w.Code != 400 {
		t.Fatalf("caller-selected target accepted: %d", w.Code)
	}
}

func TestWorkerCannotForgeConfirmedVerdict(t *testing.T) {
	s, _ := fixture(t)
	r := running(t, s, "a")
	token, _ := s.Store.Credential(r.ID)
	if w := request(s, "POST", "/v1/runs/"+r.ID+"/finish", token, `{"verdict":"confirmed"}`); w.Code != 400 {
		t.Fatalf("forged result accepted: %d", w.Code)
	}
	result, err := s.Finish(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != "inconclusive" {
		t.Fatalf("evidenceless finding: %+v", result)
	}
}

func TestRequestBudgetAndStoppedRunFence(t *testing.T) {
	s, _ := fixture(t)
	r := running(t, s, "a")
	for i := 0; i < r.MaxRequests; i++ {
		if _, err := s.Execute(context.Background(), r.ID, Request{r.Target + "/api/documents/doc-alice", "alice"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Execute(context.Background(), r.ID, Request{r.Target + "/api/documents/doc-alice", "alice"}); err == nil {
		t.Fatal("budget bypass")
	}
	if err := s.Stop(r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Execute(context.Background(), r.ID, Request{r.Target + "/api/documents/doc-alice", "alice"}); err == nil {
		t.Fatal("stopped run executed")
	}
	token, _ := s.Store.Credential(r.ID)
	if strings.Contains(Report(r), token) {
		t.Fatal("credential in report")
	}
}

func TestStopCancelsActiveRequestWithoutScheduler(t *testing.T) {
	s, _ := fixture(t)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done(); close(cancelled) }))
	defer target.Close()
	lt, err := LabTarget(context.Background(), target.URL)
	if err != nil {
		t.Fatal(err)
	}
	s.Targets["vulnerable"] = lt
	r := running(t, s, "a")
	done := make(chan error, 1)
	go func() {
		_, err := s.Execute(context.Background(), r.ID, Request{r.Target + "/api/slow", "alice"})
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request never started")
	}
	if err := s.Stop(r.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("target connection not cancelled")
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled request succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("broker did not drain")
	}
}

func TestStopStateAndTenantGlobalGatesSurviveRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	a, err := store.Create("a", "vulnerable", "http://fixture.test", "test")
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Create("b", "fixed", "http://fixture.test", "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.StopGroup("a", false); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, _ := store.Get(a.ID)
	if got.State != Stopped {
		t.Fatal("lost durable stop")
	}
	if _, err = store.Create("a", "fixed", "http://fixture.test", "test"); err == nil {
		t.Fatal("lost tenant gate")
	}
	got, _ = store.Get(b.ID)
	if got.State != Pending {
		t.Fatal("tenant stop affected another tenant")
	}
	if _, err = store.StopGroup("", true); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Create("b", "fixed", "http://fixture.test", "test"); err == nil {
		t.Fatal("global gate bypass")
	}
}

func TestRunDeadlineCancelsWithoutController(t *testing.T) {
	s, _ := fixture(t)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	target := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done(); close(cancelled) }))
	defer target.Close()
	lt, err := LabTarget(context.Background(), target.URL)
	if err != nil {
		t.Fatal(err)
	}
	s.Targets["vulnerable"] = lt
	r := running(t, s, "a")
	defer s.Stop(r.ID)
	if err = s.Store.Update(r.ID, func(r *Run) error { r.Deadline = time.Now().Add(300 * time.Millisecond); return nil }); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, e := s.Execute(context.Background(), r.ID, Request{r.Target + "/api/slow", "alice"})
		done <- e
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request never started")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("run deadline did not cancel target connection")
	}
	if err = <-done; err == nil {
		t.Fatal("expired request succeeded")
	}
	if _, err = s.Finish(r.ID); err == nil {
		t.Fatal("expired run finalized")
	}
}

func TestMissingOrFabricatedObjectsDoNotProveAuthorization(t *testing.T) {
	r := Run{Captures: []Capture{{Account: "bob", URL: "http://fixture/api/documents/unknown", Status: 404}}}
	if result := Validate(r); result.Verdict != "inconclusive" {
		t.Fatalf("false assurance: %+v", result)
	}
}
