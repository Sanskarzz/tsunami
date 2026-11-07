package platform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLifecycleSeparatesRevocationLocalDrainAndExecution(t *testing.T) {
	s, store := fixture(t)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done(); close(cancelled) }))
	defer target.Close()
	approved, err := LabTarget(context.Background(), target.URL)
	if err != nil {
		t.Fatal(err)
	}
	s.Targets["vulnerable"] = approved
	r := running(t, s, "a")
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = s.Execute(context.Background(), r.ID, Request{URL: target.URL + "/api/slow", Account: "alice"})
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("target did not start")
	}
	status, err := s.Status(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Permission != "active" || status.Activity != "active_locally" || status.LastAdmittedAt == nil || status.Execution != "unknown" {
		t.Fatalf("invalid pre-stop status: %+v", status)
	}
	if err = s.Stop(r.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("local operation did not end")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("target did not observe cancellation")
	}
	status, err = s.Status(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Permission != "revoked" || status.Activity != "drained_locally" || status.Cleanup != "pending" || status.Execution != "unknown" || status.ExitObservedAt != nil {
		t.Fatalf("stop overstated termination: %+v", status)
	}
	if status.LastAdmittedAt.After(*status.RevokedAt) {
		t.Fatal("operation admitted after revocation")
	}
	token, _ := store.Credential(r.ID)
	for _, credential := range []string{token, "operator-b"} {
		w := request(s, "GET", "/v1/runs/"+r.ID+"/status", credential, "")
		if w.Code != 401 && w.Code != 404 {
			t.Fatal("run status leaked across principals")
		}
	}
}

func TestRestartKeepsUnfinishedOperationsUncertain(t *testing.T) {
	s, store := fixture(t)
	r := running(t, s, "a")
	if err := store.Update(r.ID, func(r *Run) error {
		tracked(r).Operations["old-op"] = OperationRecord{Process: "previous-process", Kind: "target", StartedAt: time.Now()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	path := store.db.Path()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	restarted := NewServer(reopened, s.Targets, s.AdminTokens)
	if err := restarted.Stop(r.ID); err != nil {
		t.Fatal(err)
	}
	status, err := restarted.Status(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.LocalOperations != 0 || status.UncertainOperations != 1 || status.Activity != "uncertain_prior_or_unpersisted_completion" || status.LocalDrainObservedAt != nil {
		t.Fatalf("restart invented a drain: %+v", status)
	}
}

func TestLegacyRunDoesNotInventLifecycleObservations(t *testing.T) {
	s, store := fixture(t)
	r := running(t, s, "a")
	if err := store.Update(r.ID, func(r *Run) error { r.Lifecycle = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := s.Stop(r.ID); err != nil {
		t.Fatal(err)
	}
	status, err := s.Status(r.ID)
	if err != nil || status.Permission != "revoked" || status.Activity != "unknown" || status.LocalDrainObservedAt != nil || status.Execution != "unknown" {
		t.Fatalf("legacy history overstated: %+v, %v", status, err)
	}
}

func TestOptionalRouteFailurePreservesStatusAndStop(t *testing.T) {
	s, _ := fixture(t)
	r := running(t, s, "a")
	s.SetRouteHealth("private", "unavailable", "private_configuration_unavailable")
	s.SetRouteHealth("model", "unavailable", "model_configuration_unavailable")
	if w := request(s, "GET", "/readyz", "", ""); w.Code != 200 {
		t.Fatal("optional failure made core unready")
	}
	w := request(s, "GET", "/v1/status", "operator-a", "")
	if w.Code != 200 {
		t.Fatal("status unavailable")
	}
	var status map[string]json.RawMessage
	if json.Unmarshal(w.Body.Bytes(), &status) != nil {
		t.Fatal("invalid status")
	}
	if w := request(s, "POST", "/v1/runs", "operator-a", `{"scenario":"vulnerable","connectivity":"private"}`); w.Code != 503 {
		t.Fatal("failed private route admitted work")
	}
	if w := request(s, "POST", "/v1/runs", "operator-a", `{"scenario":"vulnerable"}`); w.Code != 201 {
		t.Fatal("public work disabled by unrelated failure")
	}
	if w := request(s, "POST", "/v1/runs/"+r.ID+"/stop", "operator-a", `{}`); w.Code != 200 {
		t.Fatal("revocation unavailable")
	}
	if w := request(s, "GET", "/v1/status", "", ""); w.Code != 401 {
		t.Fatal("unauthenticated configuration visibility")
	}
	s.Quiesce()
	if w := request(s, "POST", "/v1/runs", "operator-a", `{"scenario":"vulnerable"}`); w.Code != 503 {
		t.Fatal("shutdown admitted new work")
	}
}
