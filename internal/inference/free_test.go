package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestFreeProfileRejectsPaidModelsAndOtherDestinations(t *testing.T) {
	for _, tc := range []struct{ provider, model, url string }{
		{"openrouter-free", "vendor/model", OpenRouterBaseURL},
		{"openrouter-free", "openrouter/auto", OpenRouterBaseURL},
		{"openrouter-free", "vendor/model:free:online", OpenRouterBaseURL},
		{"openrouter-free", "vendor/model:free", "https://other.invalid/v1/"},
		{"openrouter-free", "vendor/model:free", "https://openrouter.ai.evil.invalid/api/v1/"},
		{"responses", "vendor/model", OpenRouterBaseURL},
	} {
		if _, err := NewConfig(tc.provider, tc.model, tc.url, "synthetic-key"); err == nil {
			t.Fatalf("accepted unsafe profile: %+v", tc)
		}
	}
	if _, err := NewConfig("openrouter-free", "vendor/model:free", OpenRouterBaseURL, "synthetic-key"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewConfig("responses", "paid-model", "https://other.invalid/v1/", "sk-or-synthetic"); err == nil {
		t.Fatal("OpenRouter credential could be sent to another provider")
	}
}

func freeBody() map[string]any {
	return map[string]any{"model": "vendor/model:free", "tools": []any{map[string]string{"type": "function", "name": "http_request"}, map[string]string{"type": "function", "name": "finish_assessment"}}, "provider": map[string]any{"allow_fallbacks": true, "max_price": map[string]int{"prompt": 99}}, "max_output_tokens": 999999, "store": true}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestFreeTransportForcesZeroPricesBeforeForwarding(t *testing.T) {
	reserved, forwarded := 0, 0
	transport := guardedTransport{endpoint: OpenRouterBaseURL + "responses", freeModel: "vendor/model:free", reserve: func(context.Context) error { reserved++; return nil }, inner: transportFunc(func(req *http.Request) (*http.Response, error) {
		forwarded++
		var body struct {
			Model    string `json:"model"`
			Provider struct {
				Allow   bool           `json:"allow_fallbacks"`
				Require bool           `json:"require_parameters"`
				Max     map[string]int `json:"max_price"`
			} `json:"provider"`
			Store  bool `json:"store"`
			Stream bool `json:"stream"`
			Max    int  `json:"max_output_tokens"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "vendor/model:free" || body.Provider.Allow || !body.Provider.Require || len(body.Provider.Max) != 3 || body.Provider.Max["prompt"] != 0 || body.Provider.Max["completion"] != 0 || body.Provider.Max["request"] != 0 || body.Store || !body.Stream || body.Max != MaxOutputTokens {
			t.Fatalf("free policy missing: %+v", body)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
	})}
	data, _ := json.Marshal(freeBody())
	req, _ := http.NewRequest("POST", OpenRouterBaseURL+"responses", bytes.NewReader(data))
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if reserved != 1 || forwarded != 1 {
		t.Fatal("unexpected dispatch accounting")
	}
}

func TestFreeTransportRejectsFallbacksAndHostedToolsBeforeSpending(t *testing.T) {
	for _, field := range []string{"model", "models", "plugins", "tools", "route", "service_tier"} {
		t.Run(field, func(t *testing.T) {
			body := freeBody()
			switch field {
			case "model":
				body[field] = "vendor/paid-model"
			case "tools":
				body[field] = []any{map[string]string{"type": "web_search"}}
			default:
				body[field] = []string{"paid-feature"}
			}
			called := false
			transport := guardedTransport{endpoint: OpenRouterBaseURL + "responses", freeModel: "vendor/model:free", reserve: func(context.Context) error { called = true; return nil }, inner: transportFunc(func(*http.Request) (*http.Response, error) { called = true; return nil, nil })}
			data, _ := json.Marshal(body)
			req, _ := http.NewRequest("POST", OpenRouterBaseURL+"responses", bytes.NewReader(data))
			if _, err := transport.RoundTrip(req); err == nil || called {
				t.Fatal("unsafe payload forwarded or reserved")
			}
		})
	}
}
