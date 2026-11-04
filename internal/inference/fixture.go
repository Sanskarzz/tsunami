package inference

import (
	"encoding/json"
	"fmt"
	"github.com/Sanskarzz/tsunami/internal/assessment"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
)

// Fixture is a deterministic Responses wire server. It is not an LLM. No
// endpoint received from a worker is fetched here; it only emits function calls.
type Fixture struct {
	Requests        atomic.Int64
	PrematureFinish bool
}

func (f *Fixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		w.WriteHeader(200)
		return
	}
	if r.Method != "POST" || r.URL.Path != "/v1/responses" {
		w.WriteHeader(404)
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+FixtureKey {
		w.WriteHeader(401)
		return
	}
	var in struct {
		Instructions string `json:"instructions"`
		Input        []struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"input"`
		Tools []struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"tools"`
		MaxOutput int   `json:"max_output_tokens"`
		Store     *bool `json:"store"`
		Stream    bool  `json:"stream"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&in); err != nil {
		w.WriteHeader(400)
		return
	}
	if in.Store == nil || *in.Store || !in.Stream || in.MaxOutput != MaxOutputTokens || len(in.Tools) != 2 {
		http.Error(w, "unexpected model controls", 400)
		return
	}
	for _, tool := range in.Tools {
		if tool.Type != "function" || (tool.Name != "http_request" && tool.Name != "finish_assessment") {
			w.WriteHeader(400)
			return
		}
	}
	const prefix = "The only approved target origin is "
	_, suffix, ok := strings.Cut(in.Instructions, prefix)
	if !ok {
		w.WriteHeader(400)
		return
	}
	target, _, ok := strings.Cut(suffix, ". Use the approved paths")
	u, err := url.Parse(target)
	if !ok || err != nil || u.Host == "" {
		w.WriteHeader(400)
		return
	}
	step := 0
	for _, item := range in.Input {
		if item.Type == "function_call_output" {
			step++
		}
	}
	f.Requests.Add(1)
	name := "http_request"
	account := "alice"
	path := "/api/documents/doc-alice"
	switch step {
	case 0:
		target = "http://not-approved.invalid"
	case 1:
	case 2:
		account = "bob"
		path = "/api/documents/doc-bob"
	case 3:
		account = "bob"
	default:
		name = "finish_assessment"
	}
	if _, suffix, found := strings.Cut(in.Instructions, "Reviewed assessment JSON:\n"); found {
		data, _, ok := strings.Cut(suffix, "\nEnd reviewed assessment.")
		var d assessment.Definition
		if !ok || json.Unmarshal([]byte(data), &d) != nil || !d.Canonical() {
			http.Error(w, "unexpected procedure", 400)
			return
		}
		reads := d.Reads()
		if f.PrematureFinish {
			target = u.String()
			switch step {
			case 0:
				name = "http_request"
				account = d.Probe.Account
				path = d.Probe.Path
			case 1:
				name = "finish_assessment"
			case 2, 3:
				name = "http_request"
				account = d.Baselines[step-2].Account
				path = d.Baselines[step-2].Path
			default:
				name = "finish_assessment"
			}
		} else if step == 0 {
			target = "http://not-approved.invalid"
			path = reads[0].Path
			account = reads[0].Account
			name = "http_request"
		} else if step <= len(reads) {
			name = "http_request"
			account = reads[step-1].Account
			path = reads[step-1].Path
		} else {
			name = "finish_assessment"
		}
	}
	args := json.RawMessage(`{}`)
	if name == "http_request" {
		args, _ = json.Marshal(map[string]string{"url": target + path, "account": account})
	}
	w.Header().Set("Content-Type", "text/event-stream")
	EmitToolTurn(w, fmt.Sprintf("fixture-%d", step), name, string(args))
}

// EmitToolTurn is shared with offline failure tests; the shape is the documented
// Responses function-call event protocol, with synthetic usage numbers.
func EmitToolTurn(w http.ResponseWriter, id, name, args string) {
	emit := func(kind string, payload map[string]any) {
		payload["type"] = kind
		data, _ := json.Marshal(payload)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, data)
	}
	item := map[string]any{"id": "fc_" + id, "type": "function_call", "call_id": "call_" + id, "name": name, "arguments": args, "status": "completed"}
	emit("response.output_item.added", map[string]any{"output_index": 0, "item": item})
	emit("response.function_call_arguments.done", map[string]any{"output_index": 0, "item_id": "fc_" + id, "name": name, "arguments": args})
	emit("response.output_item.done", map[string]any{"output_index": 0, "item": item})
	emit("response.completed", map[string]any{"response": map[string]any{"id": "resp_" + id, "status": "completed", "usage": map[string]int{"input_tokens": 40, "output_tokens": 20, "total_tokens": 60}}})
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
