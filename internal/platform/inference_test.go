package platform

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/stacklok/mecatl/engine/session"
)

func liveRun(t *testing.T, s *Server, cfg *inference.Config) Run {
	t.Helper()
	s.Inference = cfg
	r, err := s.Store.CreateLive("a", "vulnerable", s.Targets["vulnerable"].URL, "Responses wire fixture / no live model", ModelUsage{Provider: cfg.Provider, Name: cfg.Model, ConfigID: cfg.Identity(), MaxCalls: inference.MaxCalls, MaxOutputTokens: inference.MaxOutputTokens})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Store.Update(r.ID, func(r *Run) error { r.State = Running; return nil }); err != nil {
		t.Fatal(err)
	}
	r, _ = s.Store.Get(r.ID)
	return r
}
func modelInput() inference.Request {
	return inference.Request{Messages: []session.Message{session.NewUserMessage("Assess this approved API.")}}
}

type notifiedBody struct {
	io.ReadCloser
	started chan struct{}
	once    sync.Once
}

func (b *notifiedBody) Read(p []byte) (int, error) {
	b.once.Do(func() { close(b.started) })
	return b.ReadCloser.Read(p)
}

func TestSlowModelBodiesDoNotBlockStop(t *testing.T) {
	s, _ := fixture(t)
	provider := httptest.NewServer(&inference.Fixture{})
	defer provider.Close()
	cfg, _ := inference.NewConfig("fixture", "fixture", provider.URL+"/v1/", "")
	r := liveRun(t, s, cfg)
	token, _ := s.Store.Credential(r.ID)
	done := make(chan struct{}, 2)
	for range 2 {
		reader, writer := io.Pipe()
		defer reader.Close()
		defer writer.Close()
		body := &notifiedBody{ReadCloser: reader, started: make(chan struct{})}
		req := httptest.NewRequest("POST", "/v1/runs/"+r.ID+"/model", nil)
		req.Body = body
		req.Header.Set("Authorization", "Bearer "+token)
		go func() { s.ServeHTTP(httptest.NewRecorder(), req); done <- struct{}{} }()
		select {
		case <-body.started:
		case <-time.After(time.Second):
			t.Fatal("body reader did not start")
		}
	}
	if w := request(s, "POST", "/v1/runs/"+r.ID+"/model", token, `{"messages":[]}`); w.Code != 429 {
		t.Fatal("unbounded model-body admission")
	}
	stopped := make(chan int, 1)
	go func() { stopped <- request(s, "POST", "/v1/runs/"+r.ID+"/stop", "operator-a", "").Code }()
	select {
	case code := <-stopped:
		if code != 200 {
			t.Fatalf("stop failed: %d", code)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop waited for an agent's request body")
	}
}

func TestLiveModeRequiresExplicitOperatorRequest(t *testing.T) {
	s, _ := fixture(t)
	if w := request(s, "POST", "/v1/runs", "operator-a", `{"scenario":"vulnerable","execution":"live"}`); w.Code != 409 {
		t.Fatal("live mode enabled without configuration")
	}
	f := httptest.NewServer(&inference.Fixture{})
	defer f.Close()
	s.Inference, _ = inference.NewConfig("fixture", "fixture", f.URL+"/v1/", "")
	w := request(s, "POST", "/v1/runs", "operator-a", `{"scenario":"vulnerable"}`)
	var r Run
	if err := json.Unmarshal(w.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Execution != "scripted" || r.Model != nil {
		t.Fatal("default verification could incur model calls")
	}
	token, _ := s.Store.Credential(r.ID)
	if w := request(s, "POST", "/v1/runs/"+r.ID+"/model", token, `{"messages":[{"Role":"user","Text":"hello"}]}`); w.Code == 200 {
		t.Fatal("scripted run used model gateway")
	}
	live := liveRun(t, s, s.Inference)
	other := running(t, s, "b")
	otherToken, _ := s.Store.Credential(other.ID)
	if w := request(s, "POST", "/v1/runs/"+live.ID+"/model", otherToken, `{"messages":[]}`); w.Code != 401 {
		t.Fatal("cross-run model credential accepted")
	}
	workerToken, _ := s.Store.Credential(live.ID)
	if w := request(s, "POST", "/v1/runs/"+live.ID+"/model", workerToken, `{"messages":[],"model":"expensive-model","endpoint":"http://other"}`); w.Code != 400 {
		t.Fatal("worker chose provider/model")
	}
}

func TestRequiredFreePolicyCannotCreatePaidRun(t *testing.T) {
	s, store := fixture(t)
	s.Inference, _ = inference.NewConfig("responses", "paid-model", "https://provider.invalid/v1/", "synthetic-key")
	w := request(s, "POST", "/v1/runs", "operator-a", `{"scenario":"vulnerable","execution":"live","required_price_policy":"openrouter-free/v1"}`)
	if w.Code != 409 {
		t.Fatal("required free price policy was ignored")
	}
	runs, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Fatal("paid run was created before price policy check")
	}
}

func TestModelQuotaSurvivesRestart(t *testing.T) {
	s, store := fixture(t)
	var calls atomic.Int64
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		inference.EmitToolTurn(w, "test", "finish_assessment", "{}")
	}))
	defer provider.Close()
	cfg, _ := inference.NewConfig("fixture", "fixture", provider.URL+"/v1/", "")
	r := liveRun(t, s, cfg)
	for range inference.MaxCalls {
		if _, err := s.Infer(context.Background(), r.ID, modelInput()); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, _ := store.Get(r.ID)
	if snapshot.Model.CallsReserved != 8 || snapshot.Model.WireAttempts != 8 || snapshot.Model.CompletedCalls != 8 || snapshot.Model.UsageUncertain {
		t.Fatalf("bad counters: %+v", snapshot.Model)
	}
	path := store.db.Path()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted := NewServer(reopened, s.Targets, s.AdminTokens)
	restarted.Inference = cfg
	if _, err = restarted.Infer(context.Background(), r.ID, modelInput()); err == nil {
		t.Fatal("quota reset on restart")
	}
	if calls.Load() != 8 {
		t.Fatal("unreserved upstream request")
	}
}

func TestStopCancelsInferenceWithoutKubernetes(t *testing.T) {
	for _, stopKind := range []string{"run", "tenant", "global", "deadline"} {
		t.Run(stopKind, func(t *testing.T) {
			s, store := fixture(t)
			started := make(chan struct{})
			cancelled := make(chan struct{})
			abort := make(chan struct{})
			var calls atomic.Int64
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				calls.Add(1)
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(200)
				w.(http.Flusher).Flush()
				close(started)
				select {
				case <-r.Context().Done():
					close(cancelled)
				case <-abort:
				}
			}))
			defer provider.Close()
			defer close(abort)
			cfg, _ := inference.NewConfig("fixture", "fixture", provider.URL+"/v1/", "")
			r := liveRun(t, s, cfg)
			if stopKind == "deadline" {
				if err := store.Update(r.ID, func(r *Run) error { r.Deadline = time.Now().Add(500 * time.Millisecond); return nil }); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan error, 1)
			go func() { _, err := s.Infer(context.Background(), r.ID, modelInput()); done <- err }()
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("model did not start")
			}
			if _, err := s.Infer(context.Background(), r.ID, modelInput()); err == nil {
				t.Fatal("overlapping inference allowed for one run")
			}
			if stopKind == "run" {
				if err := s.Stop(r.ID); err != nil {
					t.Fatal(err)
				}
			} else if stopKind != "deadline" {
				if _, err := s.StopGroup("a", stopKind == "global"); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("cancelled inference succeeded")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("inference did not drain")
			}
			select {
			case <-cancelled:
			case <-time.After(2 * time.Second):
				t.Fatal("provider did not observe cancellation")
			}
			if _, err := s.Infer(context.Background(), r.ID, modelInput()); err == nil {
				t.Fatal("stopped run used model")
			}
			snapshot, _ := store.Get(r.ID)
			if (stopKind != "deadline" && snapshot.State != Stopped) || snapshot.Model.CallsReserved != 1 || snapshot.Model.WireAttempts != 1 || !snapshot.Model.UsageUncertain || calls.Load() != 1 {
				t.Fatalf("incorrect stop/accounting: %+v", snapshot.Model)
			}
		})
	}
}

func TestFailedCallReservationStaysUncertainAfterSuccess(t *testing.T) {
	s, store := fixture(t)
	var count atomic.Int64
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if count.Add(1) == 1 {
			http.Error(w, "temporary provider error", 500)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		inference.EmitToolTurn(w, "ok", "finish_assessment", "{}")
	}))
	defer provider.Close()
	cfg, _ := inference.NewConfig("fixture", "fixture", provider.URL+"/v1/", "")
	r := liveRun(t, s, cfg)
	if _, err := s.Infer(context.Background(), r.ID, modelInput()); err == nil {
		t.Fatal("failure accepted")
	}
	if _, err := s.Infer(context.Background(), r.ID, modelInput()); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := store.Get(r.ID)
	if snapshot.Model.CallsReserved != 2 || snapshot.Model.WireAttempts != 2 || snapshot.Model.CompletedCalls != 1 || !snapshot.Model.UsageUncertain {
		t.Fatalf("failed reservation refunded: %+v", snapshot.Model)
	}
}

func TestModelConfigChangeAndDeadlineFailClosed(t *testing.T) {
	s, store := fixture(t)
	provider := httptest.NewServer(&inference.Fixture{})
	defer provider.Close()
	cfg, _ := inference.NewConfig("fixture", "model-a", provider.URL+"/v1/", "")
	r := liveRun(t, s, cfg)
	s.Inference, _ = inference.NewConfig("fixture", "model-b", provider.URL+"/v1/", "")
	if _, err := s.Infer(context.Background(), r.ID, modelInput()); err == nil {
		t.Fatal("changed model accepted")
	}
	s.Inference = cfg
	if err := store.Update(r.ID, func(r *Run) error { r.Deadline = time.Now().Add(-time.Second); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Infer(context.Background(), r.ID, modelInput()); err == nil {
		t.Fatal("expired model permission accepted")
	}
}
