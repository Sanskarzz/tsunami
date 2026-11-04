package kube

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/toolcontract"
)

func TestMissingDispatchedJobIsNotRecreated(t *testing.T) {
	var posts atomic.Int64
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			posts.Add(1)
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"reason":"NotFound"}`))
	}))
	defer api.Close()
	tokenPath := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenPath, []byte("synthetic-kube-token"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Scheduler{Client: api.Client(), Host: api.URL, Namespace: "tsunami", tokenPath: tokenPath}
	r := platform.Run{ID: "fixture", State: platform.Running}
	if err := s.Ensure(context.Background(), r, "run-token"); !errors.Is(err, platform.ErrWorkerFailed) {
		t.Fatalf("missing Job not fenced: %v", err)
	}
	if posts.Load() != 0 {
		t.Fatal("recreated a dispatched Job")
	}
	r.AllowJobCreation = true
	if err := s.Ensure(context.Background(), r, "run-token"); err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 2 {
		t.Fatalf("expected initial Secret + Job creation, got %d", posts.Load())
	}
}

func TestRejectedCleanupStaysPendingAndIsRetried(t *testing.T) {
	var allow atomic.Bool
	var deletes atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Error("unexpected dispatch after Stop")
		}
		deletes.Add(1)
		if !allow.Load() {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer api.Close()
	path := filepath.Join(t.TempDir(), "token")
	_ = os.WriteFile(path, []byte("synthetic-kube-identity"), 0600)
	scheduler := &Scheduler{Client: api.Client(), Host: api.URL, Namespace: "tsunami", tokenPath: path}
	store, err := platform.OpenStore(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := platform.NewServer(store, nil, nil)
	server.Scheduler = scheduler
	run, err := store.Create("a", "vulnerable", "http://owned.invalid", "cleanup test")
	if err != nil {
		t.Fatal(err)
	}
	if err = server.Stop(run.ID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); server.Reconcile(ctx) }()
	defer func() { cancel(); <-done }()
	wait := func(predicate func() bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if predicate() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("cleanup reconciliation did not progress")
	}
	wait(func() bool { return deletes.Load() >= 2 })
	stored, _ := store.Get(run.ID)
	if stored.DeletionRequestedAt != nil || stored.State != platform.Stopped {
		t.Fatal("rejected deletion recorded success or reopened permission")
	}
	allow.Store(true)
	wait(func() bool { r, _ := store.Get(run.ID); return r.DeletionRequestedAt != nil })
	if deletes.Load() < 4 {
		t.Fatal("rejected cleanup not retried")
	}
}

func TestToolJobRequiresSandboxAndKeepsNativeImageSeparate(t *testing.T) {
	var posts atomic.Int64
	var job map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		if r.URL.Path == "/apis/batch/v1/namespaces/tsunami/jobs" {
			_ = json.NewDecoder(r.Body).Decode(&job)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer api.Close()
	path := filepath.Join(t.TempDir(), "token")
	_ = os.WriteFile(path, []byte("synthetic-token"), 0600)
	s := &Scheduler{Client: api.Client(), Host: api.URL, Namespace: "tsunami", Image: "native-image", ToolImage: "reviewed-image", tokenPath: path}
	r := platform.Run{ID: "adapter", AllowJobCreation: true, Executor: toolcontract.CurlHTTP, Application: &application.Plan{}, Execution: "scripted"}
	if err := s.Ensure(context.Background(), r, "own-run"); err == nil || posts.Load() != 0 {
		t.Fatal("unsandboxed tool dispatch attempted")
	}
	s.RuntimeClass = "gvisor"
	if err := s.Ensure(context.Background(), r, "own-run"); err != nil {
		t.Fatal(err)
	}
	pod := job["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)
	c := pod["containers"].([]any)[0].(map[string]any)
	if c["image"] != "reviewed-image" || pod["runtimeClassName"] != "gvisor" || pod["automountServiceAccountToken"] != false || len(pod["volumes"].([]any)) != 2 {
		t.Fatalf("invalid tool boundary: %+v", pod)
	}
	r.Executor = ""
	r.Application = nil
	if err := s.Ensure(context.Background(), r, "own-run"); err != nil {
		t.Fatal(err)
	}
	pod = job["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)
	c = pod["containers"].([]any)[0].(map[string]any)
	if c["image"] != "native-image" || len(pod["volumes"].([]any)) != 1 {
		t.Fatal("native image or mounts changed")
	}
}
