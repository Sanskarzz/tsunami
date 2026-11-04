package toolrunner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/toolcontract"
)

func TestBrokerStopCancelsTargetBehindCurlAndFencesLaterCalls(t *testing.T) {
	started, cancelled := make(chan struct{}), make(chan struct{})
	var count atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer target.Close()
	approved, err := platform.LabTarget(context.Background(), target.URL)
	if err != nil {
		t.Fatal(err)
	}
	store, err := platform.OpenStore(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	s := platform.NewServer(store, map[string]platform.Target{"vulnerable": approved}, nil)
	run, err := store.Create("owned", "vulnerable", target.URL, "adapter cancellation test")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(run.ID, func(r *platform.Run) error { r.State = platform.Running; return nil }); err != nil {
		t.Fatal(err)
	}
	r := testRunner(t, func(ctx context.Context, in toolcontract.HTTPIntent) (json.RawMessage, error) {
		captures, err := s.Execute(ctx, run.ID, platform.Request{URL: in.URL, Account: in.Account})
		out, _ := json.Marshal(captures)
		return out, err
	})
	done := make(chan error, 1)
	intent := toolcontract.HTTPIntent{URL: target.URL + "/api/slow", Account: "alice"}
	go func() { _, err := r.Execute(context.Background(), intent); done <- err }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("target never reached")
	}
	if err := s.Stop(run.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("stop did not cancel target")
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled call succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("curl did not return")
	}
	if _, err := r.Execute(context.Background(), intent); err == nil {
		t.Fatal("stopped run made another request")
	}
	if count.Load() != 1 {
		t.Fatal("extra request reached target")
	}
	stored, _ := store.Get(run.ID)
	if stored.State != platform.Stopped || stored.RequestCount != 1 {
		t.Fatal("durable stop/reservation lost")
	}
}

func TestCurlAdapterDoesNotWeakenBrokerTLSOrRedirectPolicy(t *testing.T) {
	for _, mode := range []string{"untrusted_tls", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			var reached atomic.Int32
			outside := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Add(1); w.WriteHeader(200) }))
			defer outside.Close()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, outside.URL+"/api/documents/doc-alice", 302)
			})
			var target *httptest.Server
			if mode == "untrusted_tls" {
				target = httptest.NewTLSServer(handler)
				target.Config.ErrorLog = nil
			} else {
				target = httptest.NewServer(handler)
			}
			defer target.Close()
			approved, err := platform.LabTarget(context.Background(), target.URL)
			if err != nil {
				t.Fatal(err)
			}
			store, err := platform.OpenStore(filepath.Join(t.TempDir(), "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			s := platform.NewServer(store, map[string]platform.Target{"vulnerable": approved}, nil)
			run, err := store.Create("owned", "vulnerable", target.URL, "adapter boundary test")
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Update(run.ID, func(r *platform.Run) error { r.State = platform.Running; return nil }); err != nil {
				t.Fatal(err)
			}
			r := testRunner(t, func(ctx context.Context, in toolcontract.HTTPIntent) (json.RawMessage, error) {
				captures, err := s.Execute(ctx, run.ID, platform.Request{URL: in.URL, Account: in.Account})
				out, _ := json.Marshal(captures)
				return out, err
			})
			_, _ = r.Execute(context.Background(), toolcontract.HTTPIntent{URL: target.URL + "/api/documents/doc-alice", Account: "alice"})
			stored, _ := store.Get(run.ID)
			if reached.Load() != 0 || len(stored.Captures) == 0 {
				t.Fatal("outside target reached or evidence missing")
			}
			if mode == "untrusted_tls" && stored.Captures[0].Error == "" {
				t.Fatal("invalid upstream TLS accepted")
			}
			if mode == "redirect" && stored.Captures[0].Status != 302 {
				t.Fatal("redirect response lost")
			}
		})
	}
}
