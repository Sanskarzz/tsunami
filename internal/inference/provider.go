package inference

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sanskarzz/tsunami/internal/assessment"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/openai/openai-go/v3/option"
	"github.com/stacklok/mecatl/engine/port"
	"github.com/stacklok/mecatl/engine/session"
	responses "github.com/stacklok/mecatl/provider/openai"
)

type Config struct {
	Provider, Model, BaseURL string
	key                      string
}

func NewConfig(provider, model, baseURL, key string) (*Config, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("invalid model endpoint")
	}
	if strings.TrimSpace(model) == "" || len(model) > 160 {
		return nil, errors.New("explicit model ID required")
	}
	if provider != "responses" && provider != "fixture" && provider != "openrouter-free" {
		return nil, errors.New("unsupported provider; use responses, openrouter-free, or fixture")
	}
	if provider != "fixture" && (u.Scheme != "https" || strings.TrimSpace(key) == "") {
		return nil, errors.New("real model mode requires HTTPS and a key file")
	}
	if provider == "openrouter-free" && (strings.TrimRight(baseURL, "/")+"/" != OpenRouterBaseURL || !freeModelID.MatchString(model)) {
		return nil, errors.New("openrouter-free requires the official endpoint and an explicit :free model")
	}
	if strings.EqualFold(u.Hostname(), "openrouter.ai") && provider != "openrouter-free" {
		return nil, errors.New("use openrouter-free for OpenRouter; generic routing is disabled")
	}
	if strings.HasPrefix(key, "sk-or-") && provider == "responses" {
		return nil, errors.New("OpenRouter credentials require the pinned free-only profile")
	}
	if provider == "fixture" {
		// This exception sends only a fixed synthetic key. It cannot downgrade a
		// real provider credential to cleartext, even if misconfigured.
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, errors.New("invalid fixture scheme")
		}
		ip, parseErr := netip.ParseAddr(u.Hostname())
		if (parseErr != nil || (!ip.IsLoopback() && !ip.IsPrivate())) && u.Hostname() != "tsunami-model-fixture" && u.Hostname() != "tsunami-model-fixture.tsunami.svc.cluster.local" {
			return nil, errors.New("fixture endpoint must be in the local lab")
		}
		key = FixtureKey
	}
	return &Config{Provider: provider, Model: model, BaseURL: strings.TrimRight(baseURL, "/") + "/", key: key}, nil
}

func (c *Config) Identity() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\n%s\n%s\n%s\n%d\n%d\n%s\n%s", c.Provider, c.Model, c.BaseURL, WorkflowID, MaxCalls, MaxOutputTokens, c.PricePolicy(), CatalogProtocol)))
	return hex.EncodeToString(sum[:])
}

func (c *Config) PricePolicy() string {
	if c.Provider == "openrouter-free" {
		return FreePolicyID
	}
	return ""
}

type guardedTransport struct {
	inner       http.RoundTripper
	endpoint    string
	reserve     func(context.Context) error
	freeModel   string
	status      *atomic.Int32
	progress    func(context.Context, int) error
	observation *Observation
}

func (t guardedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "POST" || req.URL.String() != t.endpoint {
		return nil, callError{"model_destination_denied"}
	}
	if t.freeModel != "" {
		var err error
		req, err = enforceFree(req, t.freeModel)
		if err != nil {
			return nil, callError{"model_free_policy_denied"}
		}
	}
	if err := t.reserve(req.Context()); err != nil {
		return nil, classify(req.Context(), err, 0, "model_attempt_denied")
	}
	if t.observation != nil {
		t.observation.Record.WireAttempts++
	}
	resp, err := t.inner.RoundTrip(req)
	if err == nil {
		if t.observation != nil {
			t.observation.headers(resp.StatusCode)
		}
		if t.status != nil {
			t.status.Store(int32(resp.StatusCode))
		}
		if t.progress != nil {
			if progressErr := t.progress(req.Context(), resp.StatusCode); progressErr != nil {
				_ = resp.Body.Close()
				return nil, progressErr
			}
		}
		resp.Body = boundedBody{Reader: io.LimitReader(resp.Body, 2<<20), Closer: resp.Body}
	}
	return resp, err
}

type boundedBody struct {
	io.Reader
	io.Closer
}

// Complete buffers a bounded completed turn. No partial function call is
// released to a worker if the provider stream fails or Stop cancels it.
func (c *Config) Complete(ctx context.Context, target string, req Request, reserve func(context.Context) error, progress ...func(context.Context, int) error) (Reply, *session.Usage, error) {
	return c.complete(ctx, target, req, nil, nil, reserve, progress...)
}

func (c *Config) CompleteAssessment(ctx context.Context, target string, req Request, d *assessment.Definition, reserve func(context.Context) error, progress ...func(context.Context, int) error) (Reply, *session.Usage, error) {
	if d == nil || !d.Canonical() {
		return Reply{}, nil, errors.New("unknown or changed assessment")
	}
	return c.complete(ctx, target, req, d, nil, reserve, progress...)
}

func (c *Config) CompleteObserved(ctx context.Context, target string, req Request, d *assessment.Definition, observation *Observation, reserve func(context.Context) error, progress ...func(context.Context, int) error) (Reply, *session.Usage, error) {
	if d != nil && !d.Canonical() {
		return Reply{}, nil, callError{"model_policy_changed"}
	}
	return c.complete(ctx, target, req, d, observation, reserve, progress...)
}

func (c *Config) complete(ctx context.Context, target string, req Request, d *assessment.Definition, observation *Observation, reserve func(context.Context) error, progress ...func(context.Context, int) error) (replyOut Reply, usageOut *session.Usage, errOut error) {
	if observation == nil {
		deadline, _ := ctx.Deadline()
		observation = NewObservation(0, deadline)
	}
	defer func() {
		observation.Record.UsageAccepted = errOut == nil && usageOut != nil
		observation.Finish(errOut)
	}()
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second, MaxResponseHeaderBytes: 16 << 10, DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	freeModel := ""
	if c.Provider == "openrouter-free" {
		freeModel = c.Model
	}
	var status atomic.Int32
	failure := func(cause error, fallback string) error { return classify(ctx, cause, int(status.Load()), fallback) }
	var opened func(context.Context, int) error
	if len(progress) > 0 {
		opened = progress[0]
	}
	client := &http.Client{Transport: guardedTransport{inner: transport, endpoint: c.BaseURL + "responses", reserve: reserve, freeModel: freeModel, status: &status, progress: opened, observation: observation}, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	provider := responses.New(responses.WithAPIKey(c.key), responses.WithBaseURL(c.BaseURL), responses.WithHTTPClient(client), responses.WithMaxRetries(0), responses.WithRequestOption(option.WithJSONSet("max_output_tokens", MaxOutputTokens)), responses.WithProviderCapabilities(port.ProviderCapabilities{}))
	stream, err := provider.Stream(ctx, port.LLMRequest{Model: c.Model, System: PromptFor(target, d), Tools: ToolsFor(d), Messages: req.Messages})
	if err != nil {
		return Reply{}, nil, failure(err, "model_request_failed")
	}
	reply := Reply{Chunks: []port.Chunk{}}
	var usage *session.Usage
	bytes := 0
	done := false
	for chunk, err := range stream {
		if err != nil {
			return Reply{}, usage, failure(err, "model_stream_failed")
		}
		observation.chunk()
		if chunk.Kind == port.ChunkToolCall {
			observation.Record.BufferedToolCalls++
		}
		data, err := json.Marshal(chunk)
		if err != nil {
			return Reply{}, nil, failure(nil, "model_invalid_reply")
		}
		bytes += len(data)
		if bytes > MaxReplyBytes || len(reply.Chunks) >= 512 {
			return Reply{}, usage, failure(nil, "model_reply_limit")
		}
		if chunk.Kind == port.ChunkUsage && chunk.Usage != nil {
			observation.Record.UsageChunkObserved = true
			if chunk.Usage.InputTokens < 0 || chunk.Usage.InputTokens > 1_000_000 || chunk.Usage.OutputTokens < 0 || chunk.Usage.OutputTokens > MaxOutputTokens {
				return Reply{}, nil, failure(nil, "model_invalid_usage")
			}
			// The adapter also emits zero-valued usage when the response omitted
			// usage. Zero/zero accounting for our nonempty request stays uncertain.
			if chunk.Usage.InputTokens == 0 && chunk.Usage.OutputTokens == 0 {
				usage = nil
			} else {
				copy := *chunk.Usage
				usage = &copy
			}
		}
		if chunk.Kind == port.ChunkDone {
			done = true
			if chunk.Stop != session.StopEndTurn {
				return Reply{}, usage, failure(nil, "model_response_incomplete")
			}
		}
		reply.Chunks = append(reply.Chunks, chunk)
	}
	if !done || ctx.Err() != nil {
		return Reply{}, usage, failure(nil, "model_stream_truncated")
	}
	data, err := json.Marshal(reply)
	if err != nil || len(data) > MaxReplyBytes {
		return Reply{}, usage, failure(nil, "model_reply_limit")
	}
	return reply, usage, nil
}
