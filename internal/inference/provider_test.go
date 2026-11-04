package inference

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stacklok/mecatl/engine/session"
)

func request() Request {
	return Request{Messages: []session.Message{session.NewUserMessage("Assess only the approved synthetic API.")}}
}

func TestRealAdapterUsesFixedControls(t *testing.T) {
	f := &Fixture{}
	server := httptest.NewServer(f)
	defer server.Close()
	c, err := NewConfig("fixture", "fixture-model", server.URL+"/v1/", "")
	if err != nil {
		t.Fatal(err)
	}
	attempts := 0
	reply, usage, err := c.Complete(context.Background(), "http://lab-api:8080", request(), func(context.Context) error { attempts++; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.Chunks) == 0 || usage == nil || usage.InputTokens != 40 || usage.OutputTokens != 20 || attempts != 1 || f.Requests.Load() != 1 {
		t.Fatalf("unexpected accounting: attempts=%d usage=%+v", attempts, usage)
	}
}

func TestNoSDKRetryOrRawProviderError(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "sensitive upstream error "+FixtureKey, 500)
	}))
	defer server.Close()
	c, _ := NewConfig("fixture", "fixture", server.URL+"/v1/", "")
	_, _, err := c.Complete(context.Background(), "http://lab:8080", request(), func(context.Context) error { return nil })
	if err == nil || strings.Contains(err.Error(), FixtureKey) || calls.Load() != 1 {
		t.Fatalf("retry or error exposure: calls=%d err=%v", calls.Load(), err)
	}
}

func TestModelRedirectCannotForwardCredential(t *testing.T) {
	var trap atomic.Int64
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { trap.Add(1) }))
	defer second.Close()
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, second.URL+"/v1/responses", http.StatusTemporaryRedirect)
	}))
	defer first.Close()
	c, _ := NewConfig("fixture", "fixture", first.URL+"/v1/", "")
	if _, _, err := c.Complete(context.Background(), "http://lab:8080", request(), func(context.Context) error { return nil }); err == nil {
		t.Fatal("redirect accepted")
	}
	if trap.Load() != 0 {
		t.Fatal("redirect destination reached")
	}
}

func TestTruncatedStreamReleasesNoTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "event: response.output_item.done\ndata: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":{\"id\":\"fc_test\",\"type\":\"function_call\",\"call_id\":\"test\",\"name\":\"finish_assessment\",\"arguments\":\"{}\",\"status\":\"completed\"}}\n\n")
	}))
	defer server.Close()
	c, _ := NewConfig("fixture", "fixture", server.URL+"/v1/", "")
	reply, _, err := c.Complete(context.Background(), "http://lab:8080", request(), func(context.Context) error { return nil })
	if err == nil || len(reply.Chunks) != 0 {
		t.Fatal("partial model tool escaped failed stream")
	}
}

func TestRealProviderRequiresTLSAndVerifiesCertificate(t *testing.T) {
	if _, err := NewConfig("responses", "explicit-model", "http://example.com/v1/", "synthetic-key"); err == nil {
		t.Fatal("real key permitted over HTTP")
	}
	if _, err := NewConfig("fixture", "fixture", "http://example.com/v1/", ""); err == nil {
		t.Fatal("public fixture permitted")
	}
	var calls atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer server.Close()
	c, err := NewConfig("responses", "explicit-model", server.URL+"/v1/", "synthetic-key")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = c.Complete(context.Background(), "http://lab:8080", request(), func(context.Context) error { return nil }); err == nil {
		t.Fatal("untrusted TLS certificate accepted")
	}
	if calls.Load() != 0 {
		t.Fatal("TLS verification bypassed")
	}
}

func TestMediaReferencesAndBadToolPairingRejected(t *testing.T) {
	r := request()
	image, err := session.NewImageURLContent("image/png", "https://not-approved.invalid/private.png")
	if err != nil {
		t.Fatal(err)
	}
	r.Messages[0].Parts = []session.Content{image}
	if ValidateRequest(r) == nil {
		t.Fatal("remote media reference accepted")
	}
	r = request()
	r.Messages[0].Role = session.Role("system")
	if ValidateRequest(r) == nil {
		t.Fatal("worker system message accepted")
	}
	r = Request{Messages: []session.Message{session.NewToolMessage(session.ToolResult{CallID: "orphan", Content: "x"})}}
	if ValidateRequest(r) == nil {
		t.Fatal("orphan tool result accepted")
	}
	r = request()
	r.Messages[0].Text = strings.Repeat("x", MaxRequestBytes)
	if ValidateRequest(r) == nil {
		t.Fatal("oversized conversation accepted")
	}
}
