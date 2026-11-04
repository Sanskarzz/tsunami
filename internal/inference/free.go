package inference

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
)

const OpenRouterBaseURL = "https://openrouter.ai/api/v1/"
const FreePolicyID = "openrouter-free/v1"

var freeModelID = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+:free$`)

// enforceFree is the last check before reserving/forwarding an HTTP attempt.
// This policy is server-owned; conversation data cannot change it. Restricting
// tool types also excludes separately priced hosted capabilities/plugins.
func enforceFree(req *http.Request, model string) (*http.Request, error) {
	if !freeModelID.MatchString(model) {
		return nil, errors.New("an explicit :free model is required")
	}
	data, err := io.ReadAll(io.LimitReader(req.Body, 128<<10+1))
	_ = req.Body.Close()
	if err != nil || len(data) > 128<<10 {
		return nil, errors.New("free model request byte limit")
	}
	var body map[string]json.RawMessage
	if json.Unmarshal(data, &body) != nil {
		return nil, errors.New("invalid model payload")
	}
	var requested string
	if json.Unmarshal(body["model"], &requested) != nil || requested != model {
		return nil, errors.New("free model selection changed")
	}
	for _, field := range []string{"models", "plugins", "route", "service_tier"} {
		if value, ok := body[field]; ok && string(value) != "null" {
			return nil, errors.New("fallbacks, plugins and service tiers are disabled")
		}
	}
	var tools []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if json.Unmarshal(body["tools"], &tools) != nil || len(tools) != 2 {
		return nil, errors.New("unexpected free model tools")
	}
	seen := map[string]bool{}
	for _, tool := range tools {
		if tool.Type != "function" || (tool.Name != "http_request" && tool.Name != "finish_assessment") || seen[tool.Name] {
			return nil, errors.New("hosted or unknown model tool denied")
		}
		seen[tool.Name] = true
	}
	set := func(name string, value any) { body[name], _ = json.Marshal(value) }
	set("provider", map[string]any{"allow_fallbacks": false, "require_parameters": true, "max_price": map[string]int{"prompt": 0, "completion": 0, "request": 0}})
	set("reasoning", map[string]bool{"enabled": false})
	delete(body, "include") // Encrypted reasoning replay is unused in this profile.
	set("store", false)
	set("stream", true)
	set("max_output_tokens", MaxOutputTokens)
	data, err = json.Marshal(body)
	if err != nil {
		return nil, err
	}
	clone := req.Clone(req.Context())
	clone.Body = io.NopCloser(bytes.NewReader(data))
	clone.ContentLength = int64(len(data))
	clone.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil }
	return clone, nil
}
