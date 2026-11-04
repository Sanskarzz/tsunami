package connector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CheckTransport is a trusted operator diagnostic for the fixed lab service
// through loopback port-forward. It is not registered as an agent tool and
// does not send target requests or change the emergency latch.
func CheckTransport(ctx context.Context, dir, endpoint, emergencyKey, out string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || len(emergencyKey) < 32 {
		return errors.New("TLS diagnostic requires loopback HTTPS and an operator credential")
	}
	load := func(role string) (*tls.Config, error) {
		return LoadClientTLS(filepath.Join(dir, "ca.crt"), filepath.Join(dir, role+".crt"), filepath.Join(dir, role+".key"), LabServerName, LabTenant, LabAudience, role)
	}
	broker, err := load("broker")
	if err != nil {
		return err
	}
	operator, err := load("emergency-operator")
	if err != nil {
		return err
	}
	type check struct {
		Name        string `json:"name"`
		Passed      bool   `json:"passed"`
		HTTPStatus  int    `json:"http_status,omitempty"`
		TLSRejected bool   `json:"tls_rejected,omitempty"`
	}
	checks := []check{}
	for _, name := range []string{"broker-health", "operator-status", "broker-cannot-use-emergency-role", "operator-cannot-execute-grants", "missing-client-certificate", "untrusted-server-ca", "incorrect-server-name", "incorrect-server-audience", "obsolete-tls-version", "plaintext-refused", "broker-health-after", "operator-status-after"} {
		cfg := broker.Clone()
		method, path := "GET", "/healthz"
		bearer := ""
		body := io.Reader(nil)
		expected := 200
		rejectTLS := false
		address := endpoint
		switch name {
		case "operator-status", "operator-status-after":
			cfg = operator.Clone()
			path = "/v1/status"
			bearer = emergencyKey
		case "broker-cannot-use-emergency-role":
			path = "/v1/status"
			bearer = emergencyKey
			expected = 403
		case "operator-cannot-execute-grants":
			cfg = operator.Clone()
			method = "POST"
			path = "/v1/request"
			body = strings.NewReader("{}")
			expected = 403
		case "missing-client-certificate":
			cfg.Certificates = nil
			rejectTLS = true
		case "untrusted-server-ca":
			cfg.RootCAs = x509.NewCertPool()
			rejectTLS = true
		case "incorrect-server-name":
			cfg.ServerName = "not-the-connector.invalid"
			rejectTLS = true
		case "incorrect-server-audience":
			cfg, err = LoadClientTLS(filepath.Join(dir, "ca.crt"), filepath.Join(dir, "broker.crt"), filepath.Join(dir, "broker.key"), LabServerName, LabTenant, "customer-b", "broker")
			if err != nil {
				return err
			}
			rejectTLS = true
		case "obsolete-tls-version":
			cfg.MinVersion = tls.VersionTLS12
			cfg.MaxVersion = tls.VersionTLS12
			rejectTLS = true
		case "plaintext-refused":
			address = strings.Replace(endpoint, "https://", "http://", 1)
			expected = 400
		}
		transport := &http.Transport{Proxy: nil, TLSClientConfig: cfg, DisableKeepAlives: true, TLSHandshakeTimeout: 2 * time.Second}
		client := &http.Client{Transport: transport, Timeout: 3 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
		req, _ := http.NewRequestWithContext(ctx, method, address+path, body)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		response, callErr := client.Do(req)
		result := check{Name: name}
		if response != nil {
			result.HTTPStatus = response.StatusCode
			data, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
			_ = response.Body.Close()
			result.Passed = !rejectTLS && callErr == nil && readErr == nil && response.StatusCode == expected
			if result.Passed && (name == "broker-health" || name == "broker-health-after") {
				result.Passed = response.TLS != nil && response.TLS.Version == tls.VersionTLS13
			}
			if result.Passed && (name == "operator-status" || name == "operator-status-after") {
				var state struct {
					Cutoff *bool `json:"cutoff"`
				}
				result.Passed = json.Unmarshal(data, &state) == nil && state.Cutoff != nil && !*state.Cutoff
			}
		} else if callErr != nil && rejectTLS && authenticatedTLSRejection(callErr) {
			result.TLSRejected = true
			result.Passed = true
		}
		transport.CloseIdleConnections()
		checks = append(checks, result)
	}
	record := map[string]any{"checked_at": time.Now().UTC(), "tenant": LabTenant, "audience": LabAudience, "server_name": LabServerName, "transport": "TLS 1.3 with verified client and server URI identities", "checks": checks, "target_requests_sent": false, "cutoff_changed": false}
	data, _ := json.MarshalIndent(record, "", "  ")
	if err = os.MkdirAll(filepath.Dir(out), 0700); err != nil {
		return err
	}
	if err = os.WriteFile(out, append(data, '\n'), 0600); err != nil {
		return err
	}
	for _, c := range checks {
		if !c.Passed {
			return errors.New("connector TLS diagnostic failed; inspect sanitized record")
		}
	}
	return nil
}

func authenticatedTLSRejection(err error) bool {
	var verification *tls.CertificateVerificationError
	if errors.As(err, &verification) {
		return true
	}
	// Remote TLS alerts are wrapped by net/url with an unexported TLS alert
	// type. Match only known rejection messages; refusal/timeouts are unproven.
	for _, message := range []string{"remote error: tls: certificate required", "remote error: tls: protocol version not supported", "unexpected connector server identity"} {
		if strings.Contains(err.Error(), message) {
			return true
		}
	}
	return false
}
