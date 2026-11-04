package connector

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/platform"
)

func TestMTLSRejectsUnknownMissingWrongTenantAndWrongServer(t *testing.T) {
	var hits atomic.Int64
	l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(200) }))
	for _, name := range []string{"no-client-cert", "other-ca-client", "wrong-tenant-client", "expired-client", "wrong-server-name", "untrusted-server-ca", "wrong-server-audience"} {
		t.Run(name, func(t *testing.T) {
			cfg := l.brokerHTTP.Transport.(*http.Transport).TLSClientConfig.Clone()
			switch name {
			case "no-client-cert":
				cfg.Certificates = nil
			case "other-ca-client", "wrong-tenant-client", "expired-client":
				ca := l.ca
				identity := Identity("demo-a", "broker")
				start, end := time.Now().Add(-time.Minute), time.Now().Add(time.Hour)
				if name == "other-ca-client" {
					var err error
					ca, err = newAuthority(time.Now())
					if err != nil {
						t.Fatal(err)
					}
				}
				if name == "wrong-tenant-client" {
					identity = Identity("demo-b", "broker")
				}
				if name == "expired-client" {
					start = time.Now().Add(-time.Hour)
					end = time.Now().Add(-time.Minute)
				}
				cert, key, err := ca.issue(identity, nil, x509.ExtKeyUsageClientAuth, start, end)
				if err != nil {
					t.Fatal(err)
				}
				pair, err := tls.X509KeyPair(cert, key)
				if err != nil {
					t.Fatal(err)
				}
				cfg.Certificates = []tls.Certificate{pair}
			case "wrong-server-name":
				cfg.ServerName = "not-this-connector.invalid"
			case "untrusted-server-ca":
				cfg.RootCAs = x509.NewCertPool()
			case "wrong-server-audience":
				var err error
				cfg, err = LoadClientTLS(filepath.Join(l.tlsDir, "ca.crt"), filepath.Join(l.tlsDir, "broker.crt"), filepath.Join(l.tlsDir, "broker.key"), "127.0.0.1", "demo-a", "customer-b", "broker")
				if err != nil {
					t.Fatal(err)
				}
			}
			client := &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: cfg, Proxy: nil, DisableKeepAlives: true}}
			e, _ := Sign(l.key, l.grant("/api/documents/doc-alice", time.Second))
			data, _ := json.Marshal(e)
			resp, err := client.Post(l.http.URL+"/v1/request", "application/json", bytes.NewReader(data))
			if resp != nil {
				defer resp.Body.Close()
			}
			if err == nil {
				t.Fatalf("untrusted transport reached HTTP handler: %d", resp.StatusCode)
			}
		})
	}
	if hits.Load() != 0 {
		t.Fatal("untrusted transport reached target")
	}
}

func TestTLSRolesAndPlaintextCannotSubstituteForAuthorization(t *testing.T) {
	var hits atomic.Int64
	l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(200) }))
	e, _ := Sign(l.key, l.grant("/api/documents/doc-alice", time.Second))
	if code := post(t, l.operatorHTTP, l.http.URL, e); code != 403 {
		t.Fatal("operator TLS role executed a broker grant")
	}
	req, _ := http.NewRequest("POST", l.http.URL+"/v1/cutoff", nil)
	req.Header.Set("Authorization", "Bearer "+emergency)
	resp, err := l.brokerHTTP.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatal("broker certificate used emergency path")
	}
	// A signed grant alone cannot bypass transport authentication.
	w := httptest.NewRecorder()
	data, _ := json.Marshal(e)
	l.server.ServeHTTP(w, httptest.NewRequest("POST", "/v1/request", bytes.NewReader(data)))
	if w.Code != 403 {
		t.Fatal("plaintext handler accepted a grant")
	}
	client := Client{Endpoint: strings.Replace(l.http.URL, "https://", "http://", 1), Tenant: "demo-a", Audience: "customer-a", Key: l.key, Lifetime: MaxGrantLifetime, TLS: l.brokerHTTP.Transport.(*http.Transport).TLSClientConfig}
	r := platform.Run{ID: strings.Repeat("a", 32), Tenant: "demo-a", Connectivity: "private", Deadline: time.Now().Add(time.Second)}
	request, _ := http.NewRequest("GET", l.target.URL+"/api/documents/doc-alice", nil)
	request.Header.Set("Authorization", "Bearer lab-alice")
	if _, _, err := client.Forward(context.Background(), r, request); err == nil {
		t.Fatal("plaintext fallback accepted")
	}
	if hits.Load() != 0 {
		t.Fatal("role denial reached target")
	}
}

func TestCertificateExpiryCancelsAnAlreadyOpenResponse(t *testing.T) {
	for _, role := range []string{"broker", "connector"} {
		t.Run(role, func(t *testing.T) {
			started, cancelled := make(chan struct{}), make(chan struct{})
			l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				_, _ = w.Write([]byte("partial"))
				w.(http.Flusher).Flush()
				close(started)
				<-r.Context().Done()
				close(cancelled)
			}))
			end := time.Now().Truncate(time.Second).Add(2 * time.Second)
			identity := Identity("demo-a", "broker")
			names := []string(nil)
			usage := x509.ExtKeyUsageClientAuth
			if role == "connector" {
				identity = ConnectorIdentity("demo-a", "customer-a")
				names = []string{"127.0.0.1"}
				usage = x509.ExtKeyUsageServerAuth
			}
			writeTestLeaf(t, l.tlsDir, l.ca, role, identity, names, usage, time.Now().Add(-time.Minute), end)
			if role == "connector" {
				l.restart(t)
			} else {
				l.brokerHTTP = testHTTPClient(t, l.tlsDir, "broker", "127.0.0.1", "demo-a", "customer-a")
			}
			e, _ := Sign(l.key, l.grant("/api/slow", MaxGrantLifetime))
			data, _ := json.Marshal(e)
			done := make(chan int, 1)
			go func() {
				resp, err := l.brokerHTTP.Post(l.http.URL+"/v1/request", "application/json", bytes.NewReader(data))
				if err != nil {
					done <- 0
					return
				}
				defer resp.Body.Close()
				_, _ = io.ReadAll(resp.Body)
				done <- resp.StatusCode
			}()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("TLS request not started")
			}
			select {
			case <-cancelled:
			case <-time.After(3 * time.Second):
				t.Fatal("certificate expiry did not cancel open target response")
			}
			if code := <-done; code != 502 {
				t.Fatalf("partial body accepted after cert expiry: %d", code)
			}
			// Reusing an established connection cannot re-authorize expired peers.
			if !time.Now().Before(end) {
				if _, err := LoadClientTLS(filepath.Join(l.tlsDir, "ca.crt"), filepath.Join(l.tlsDir, "broker.crt"), filepath.Join(l.tlsDir, "broker.key"), "127.0.0.1", "demo-a", "customer-a", "broker"); role == "broker" && err == nil {
					t.Fatal("expired client loaded")
				}
			}
		})
	}
}

func TestLeafRenewalPreservesAuthorityAndRequiresExplicitAction(t *testing.T) {
	dir := t.TempDir()
	first, err := ProvisionLabPKI(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	caKey, err := os.ReadFile(filepath.Join(dir, "ca.key"))
	if err != nil {
		t.Fatal(err)
	}
	oldLeaf, _ := os.ReadFile(filepath.Join(dir, "broker.crt"))
	second, err := ProvisionLabPKI(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	same, _ := os.ReadFile(filepath.Join(dir, "broker.crt"))
	if !bytes.Equal(oldLeaf, same) || first["ca_sha256"] != second["ca_sha256"] {
		t.Fatal("setup silently rotated credentials")
	}
	third, err := ProvisionLabPKI(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	newLeaf, _ := os.ReadFile(filepath.Join(dir, "broker.crt"))
	sameCA, _ := os.ReadFile(filepath.Join(dir, "ca.key"))
	if bytes.Equal(oldLeaf, newLeaf) || !bytes.Equal(caKey, sameCA) || first["ca_sha256"] != third["ca_sha256"] {
		t.Fatal("explicit leaf renewal corrupted authority")
	}
	for _, name := range []string{"ca.key", "broker.key", "connector.key", "emergency-operator.key"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("private key permissions")
		}
	}
}

func TestCertificateExpiryRecheckedOnAnExistingTLSConnection(t *testing.T) {
	for _, role := range []string{"broker", "connector"} {
		t.Run(role, func(t *testing.T) {
			l := newLab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
			end := time.Now().Truncate(time.Second).Add(2 * time.Second)
			identity, names, usage := Identity("demo-a", "broker"), []string(nil), x509.ExtKeyUsageClientAuth
			if role == "connector" {
				identity, names, usage = ConnectorIdentity("demo-a", "customer-a"), []string{"127.0.0.1"}, x509.ExtKeyUsageServerAuth
			}
			writeTestLeaf(t, l.tlsDir, l.ca, role, identity, names, usage, time.Now().Add(-time.Minute), end)
			if role == "connector" {
				l.restart(t)
			} else {
				l.brokerHTTP = testHTTPClient(t, l.tlsDir, "broker", "127.0.0.1", "demo-a", "customer-a")
			}
			u, _ := url.Parse(l.http.URL)
			conn, err := tls.Dial("tcp", u.Host, l.brokerHTTP.Transport.(*http.Transport).TLSClientConfig.Clone())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if err = conn.SetDeadline(time.Now().Add(4 * time.Second)); err != nil {
				t.Fatal(err)
			}
			reader := bufio.NewReader(conn)
			get := func() int {
				request, _ := http.NewRequest("GET", l.http.URL+"/healthz", nil)
				if err := request.Write(conn); err != nil {
					t.Fatal(err)
				}
				response, err := http.ReadResponse(reader, request)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = io.ReadAll(response.Body)
				_ = response.Body.Close()
				return response.StatusCode
			}
			if code := get(); code != 200 {
				t.Fatal(code)
			}
			time.Sleep(time.Until(end) + 20*time.Millisecond)
			if code := get(); code != 403 {
				t.Fatal("expired identity reused the same TLS connection")
			}
		})
	}
}

func TestTLSProbeDoesNotCountDeadEndpointsAsRejection(t *testing.T) {
	for _, err := range []error{context.DeadlineExceeded, io.EOF, errors.New("dial tcp: connection refused"), errors.New("connection reset by peer")} {
		if authenticatedTLSRejection(err) {
			t.Fatal("unproven network failure counted as TLS rejection")
		}
	}
	if !authenticatedTLSRejection(errors.New("remote error: tls: certificate required")) {
		t.Fatal("known client certificate alert not recognized")
	}
}
