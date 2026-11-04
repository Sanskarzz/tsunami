package inference

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

const FaultCanary = "synthetic-private-provider-canary"

// FaultFixture serves only fixed Responses protocol examples. It never calls
// a model or fetches a URL from the request. Used by the trusted diagnostics.
type FaultFixture struct {
	Mode      string
	Requests  atomic.Int64
	Started   chan struct{}
	Cancelled atomic.Bool
	once      sync.Once
}

func (f *FaultFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" || r.URL.Path != "/v1/responses" || r.Header.Get("Authorization") != "Bearer "+FixtureKey {
		http.Error(w, "fixture request denied", 401)
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(r.Body, 128<<10))
	f.Requests.Add(1)
	if f.Started != nil {
		f.once.Do(func() { close(f.Started) })
	}
	wait := func() { <-r.Context().Done(); f.Cancelled.Store(true) }
	switch f.Mode {
	case "http_401":
		http.Error(w, FaultCanary, 401)
		return
	case "http_429":
		w.Header().Set("Retry-After", "1")
		http.Error(w, FaultCanary, 429)
		return
	case "http_503":
		http.Error(w, FaultCanary, 503)
		return
	case "no_headers":
		wait()
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(200)
	w.(http.Flusher).Flush()
	if f.Mode == "headers_only" {
		wait()
		return
	}
	emit := func(kind string, payload map[string]any) {
		payload["type"] = kind
		data, _ := json.Marshal(payload)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, data)
	}
	partialTool := func() {
		emit("response.output_item.done", map[string]any{"output_index": 0, "item": map[string]any{"id": "fc_partial", "type": "function_call", "call_id": "partial", "name": "finish_assessment", "arguments": "{}", "status": "completed"}})
	}
	switch f.Mode {
	case "success":
		EmitToolTurn(w, "diagnostic", "finish_assessment", "{}")
	case "missing_usage":
		partialTool()
		emit("response.completed", map[string]any{"response": map[string]any{"status": "completed"}})
	case "truncated_tool":
		partialTool()
	case "output_limit", "unknown_incomplete":
		partialTool()
		reason := "max_output_tokens"
		if f.Mode == "unknown_incomplete" {
			reason = FaultCanary
		}
		emit("response.incomplete", map[string]any{"response": map[string]any{"id": "sensitive-provider-id", "status": "incomplete", "incomplete_details": map[string]string{"reason": reason}, "usage": map[string]int{"input_tokens": 40, "output_tokens": 20}}})
	case "event_rate_limit":
		emit("response.failed", map[string]any{"response": map[string]any{"id": "sensitive-provider-id", "status": "failed", "error": map[string]string{"code": "rate_limit_exceeded", "message": FaultCanary}}})
	case "invalid_usage":
		emit("response.completed", map[string]any{"response": map[string]any{"status": "completed", "usage": map[string]int{"input_tokens": -1, "output_tokens": 20}}})
	case "reply_limit":
		emit("response.output_text.delta", map[string]any{"delta": strings.Repeat("x", MaxReplyBytes+1)})
	default:
		emit("error", map[string]any{"code": "unknown_provider_code", "message": FaultCanary})
	}
}
