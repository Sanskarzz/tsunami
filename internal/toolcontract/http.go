// Package toolcontract defines reviewed executor identities, not permissions.
package toolcontract

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

const CurlHTTP = "curl-http/v1"

type HTTPIntent struct {
	URL     string `json:"url"`
	Account string `json:"account"`
}

func DecodeHTTPIntent(data []byte) (HTTPIntent, error) {
	var in HTTPIntent
	bad := errors.New("invalid HTTP intent")
	if len(data) > 4096 {
		return in, bad
	}
	d := json.NewDecoder(bytes.NewReader(data))
	if token, err := d.Token(); err != nil || token != json.Delim('{') {
		return in, bad
	}
	seen := map[string]bool{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return HTTPIntent{}, bad
		}
		key, ok := token.(string)
		if !ok || seen[key] || (key != "url" && key != "account") {
			return HTTPIntent{}, bad
		}
		seen[key] = true
		var value string
		if d.Decode(&value) != nil || value == "" {
			return HTTPIntent{}, bad
		}
		if key == "url" {
			in.URL = value
		} else {
			in.Account = value
		}
	}
	if token, err := d.Token(); err != nil || token != json.Delim('}') || d.Decode(&struct{}{}) != io.EOF || len(seen) != 2 {
		return HTTPIntent{}, bad
	}
	return in, nil
}
