package demo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/stacklok/mecatl/engine/session"
)

type FaultResult struct {
	Mode                 string               `json:"mode"`
	Expected             string               `json:"expected"`
	Actual               string               `json:"actual"`
	Passed               bool                 `json:"passed"`
	ProviderAttempts     int64                `json:"fixture_requests"`
	TargetRequests       int                  `json:"target_requests"`
	ReleasedChunks       int                  `json:"released_chunks"`
	CancellationObserved bool                 `json:"fixture_observed_cancellation"`
	UsageUncertain       bool                 `json:"usage_uncertain"`
	Call                 inference.CallRecord `json:"call"`
}
type FaultSummary struct {
	CheckedAt   time.Time     `json:"checked_at"`
	Mode        string        `json:"mode"`
	Results     []FaultResult `json:"results"`
	Limitations []string      `json:"limitations"`
}

// DiagnoseInference exercises the real adapter and gateway against loopback
// fixtures. It never loads a real key, runs an agent tool, or calls a target.
func DiagnoseInference(ctx context.Context, out string, log io.Writer) (FaultSummary, error) {
	summary := FaultSummary{CheckedAt: time.Now().UTC(), Mode: "local Responses fault fixtures; no model or target calls", Results: []FaultResult{}, Limitations: []string{"Short run deadlines are set only by this trusted local harness; deployed call limits remain unchanged.", "Timing is observed at the gateway, not provider compute time or time to first token.", "Historical model_error records cannot be assigned a more specific cause retroactively."}}
	if err := os.MkdirAll(out, 0700); err != nil {
		return summary, err
	}
	cases := []struct{ mode, expected string }{{"success", "ok"}, {"missing_usage", "ok"}, {"http_401", "upstream_http_401"}, {"http_429", "upstream_http_429"}, {"http_503", "upstream_http_503"}, {"no_headers", "model_deadline_exceeded"}, {"headers_only", "model_deadline_exceeded"}, {"stop_after_headers", "model_cancelled"}, {"truncated_tool", "model_stream_truncated"}, {"output_limit", "model_output_token_limit"}, {"unknown_incomplete", "model_response_incomplete"}, {"event_rate_limit", "upstream_event_http_429"}, {"invalid_usage", "model_invalid_usage"}, {"reply_limit", "model_reply_limit"}}
	allPassed := true
	for _, tc := range cases {
		result, err := diagnoseOne(ctx, tc.mode, tc.expected)
		if err != nil {
			return summary, err
		}
		summary.Results = append(summary.Results, result)
		allPassed = allPassed && result.Passed
		data, _ := json.MarshalIndent(summary, "", "  ")
		if err = os.WriteFile(filepath.Join(out, "diagnostics.json"), append(data, '\n'), 0600); err != nil {
			return summary, err
		}
		if log != nil {
			fmt.Fprintf(log, "FAULT mode=%s expected=%s actual=%s passed=%t wire_attempts=%d released_tools=%d\n", tc.mode, tc.expected, result.Actual, result.Passed, result.Call.WireAttempts, result.Call.ReleasedToolCalls)
		}
	}
	if !allPassed {
		return summary, errors.New("diagnostic mismatches retained; inspect diagnostics.json")
	}
	return summary, nil
}

func diagnoseOne(ctx context.Context, mode, expected string) (FaultResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	fixtureMode := mode
	if mode == "stop_after_headers" {
		fixtureMode = "headers_only"
	}
	f := &inference.FaultFixture{Mode: fixtureMode, Started: make(chan struct{})}
	provider := httptest.NewServer(f)
	defer func() { cancel(); provider.Close() }()
	cfg, err := inference.NewConfig("fixture", "tsunami-fault-fixture", provider.URL+"/v1/", "")
	if err != nil {
		return FaultResult{}, err
	}
	dir, err := os.MkdirTemp("", "tsunami-inference-fault-")
	if err != nil {
		return FaultResult{}, err
	}
	defer os.RemoveAll(dir)
	store, err := platform.OpenStore(filepath.Join(dir, "state.db"))
	if err != nil {
		return FaultResult{}, err
	}
	defer store.Close()
	s := platform.NewServer(store, nil, nil)
	s.Inference = cfg
	r, err := store.CreateLive("diagnostic", "vulnerable", "http://synthetic-target.invalid", "trusted local diagnostic", platform.ModelUsage{Provider: cfg.Provider, Name: cfg.Model, ConfigID: cfg.Identity(), MaxCalls: inference.MaxCalls, MaxOutputTokens: inference.MaxOutputTokens})
	if err != nil {
		return FaultResult{}, err
	}
	if err = store.Update(r.ID, func(r *platform.Run) error {
		r.State = platform.Running
		if mode == "no_headers" || mode == "headers_only" {
			r.Deadline = time.Now().Add(400 * time.Millisecond)
		}
		return nil
	}); err != nil {
		return FaultResult{}, err
	}
	input := inference.Request{Messages: []session.Message{session.NewUserMessage("Connection diagnostic only. No HTTP target requests.")}}
	var reply inference.Reply
	var callErr error
	if mode == "stop_after_headers" {
		done := make(chan struct{})
		go func() { reply, callErr = s.Infer(ctx, r.ID, input); close(done) }()
		select {
		case <-f.Started:
		case <-ctx.Done():
			return FaultResult{}, ctx.Err()
		case <-time.After(3 * time.Second):
			return FaultResult{}, errors.New("diagnostic provider did not start")
		}
		// Wait for recorded headers, independently from a provider accepting a
		// TCP request. No fixed sleep is used as evidence of an active stream.
		for {
			snap, _ := s.Snapshot(r.ID)
			if snap.Model.StreamsOpened > 0 {
				break
			}
			select {
			case <-done:
				return FaultResult{}, errors.New("diagnostic model ended before headers were recorded")
			case <-ctx.Done():
				return FaultResult{}, ctx.Err()
			case <-time.After(time.Millisecond):
			}
		}
		if err = s.Stop(r.ID); err != nil {
			return FaultResult{}, err
		}
		select {
		case <-done:
		case <-ctx.Done():
			return FaultResult{}, ctx.Err()
		case <-time.After(3 * time.Second):
			return FaultResult{}, errors.New("diagnostic Stop did not drain")
		}
	} else {
		reply, callErr = s.Infer(ctx, r.ID, input)
	}
	snapshot, err := s.Snapshot(r.ID)
	if err != nil {
		return FaultResult{}, err
	}
	if len(snapshot.Model.Calls) != 1 {
		return FaultResult{}, errors.New("diagnostic call record missing")
	}
	call := snapshot.Model.Calls[0]
	if mode == "no_headers" || mode == "headers_only" || mode == "stop_after_headers" {
		deadline := time.Now().Add(time.Second)
		for !f.Cancelled.Load() && time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return FaultResult{}, ctx.Err()
			case <-time.After(time.Millisecond):
			}
		}
	}
	passed := call.Code == expected && call.WireAttempts == 1 && snapshot.Model.WireAttempts == 1 && f.Requests.Load() == 1 && snapshot.RequestCount == 0 && call.FinishedAt != nil
	if expected == "ok" {
		passed = passed && callErr == nil && call.ReleasedToolCalls == 1 && snapshot.Model.UsageUncertain == (mode == "missing_usage")
		if mode == "missing_usage" {
			passed = passed && !call.UsageAccepted
		}
	} else {
		passed = passed && callErr != nil && len(reply.Chunks) == 0 && call.ReleasedToolCalls == 0 && snapshot.Model.UsageUncertain
	}
	if mode == "no_headers" {
		passed = passed && call.HeadersMS == nil && call.FirstChunkMS == nil
	}
	if mode == "headers_only" || mode == "stop_after_headers" {
		passed = passed && call.HeadersMS != nil && call.FirstChunkMS == nil && f.Cancelled.Load()
	}
	if mode == "truncated_tool" || mode == "output_limit" || mode == "unknown_incomplete" {
		passed = passed && call.BufferedToolCalls == 1
	}
	if err = s.Stop(r.ID); err != nil {
		return FaultResult{}, err
	} // explicit diagnostic cleanup, not a completed assessment
	return FaultResult{Mode: mode, Expected: expected, Actual: call.Code, Passed: passed, ProviderAttempts: f.Requests.Load(), TargetRequests: snapshot.RequestCount, ReleasedChunks: len(reply.Chunks), CancellationObserved: f.Cancelled.Load(), UsageUncertain: snapshot.Model.UsageUncertain, Call: call}, nil
}
