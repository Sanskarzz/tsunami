package platform

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

type fakeResolver struct {
	ips   []netip.Addr
	calls int
}

func (f *fakeResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	f.calls++
	return f.ips, nil
}

func TestScopeRejectsBypasses(t *testing.T) {
	p, err := NewPolicy("https://app.example.test", []netip.Prefix{netip.MustParsePrefix("192.0.2.1/32")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		"https://evil-app.example.test/api/documents/doc-alice",
		"https://app.example.test.evil.test/api/documents/doc-alice",
		"https://app.example.test@evil.test/api/documents/doc-alice",
		"https://user@app.example.test/api/documents/doc-alice",
		"http://app.example.test/api/documents/doc-alice",
		"https://app.example.test:444/api/documents/doc-alice",
		"https://app.example.test/api/documents/doc-alice#fragment",
		"https://app.example.test/api/documents/doc-alice?next=http://evil.test",
		"https://app.example.test/api/documents/%64oc-alice",
		"https://app.example.test/api/../admin",
		"https://169.254.169.254/api/documents/doc-alice",
	} {
		if _, err := p.Validate(raw); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if _, err := p.Validate("https://APP.EXAMPLE.TEST:443/api/documents/doc-alice"); err != nil {
		t.Fatal(err)
	}
}

func TestDialPinsValidatedAddressAndRejectsRebinding(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "fixture") }))
	defer target.Close()
	u, _ := url.Parse(target.URL)
	_, port, _ := net.SplitHostPort(u.Host)
	r := &fakeResolver{ips: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}
	p, err := NewPolicy("http://fixture.test:"+port, []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}, r)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := p.dial(context.Background(), "tcp", "fixture.test:"+port)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	if r.calls != 1 {
		t.Fatalf("resolved %d times", r.calls)
	}
	r.ips = []netip.Addr{netip.MustParseAddr("169.254.169.254")}
	if _, err = p.dial(context.Background(), "tcp", "fixture.test:"+port); err == nil || !strings.Contains(err.Error(), "unapproved") {
		t.Fatalf("rebind accepted: %v", err)
	}
	r.ips = []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("::ffff:169.254.169.254")}
	if _, err = p.dial(context.Background(), "tcp", "fixture.test:"+port); err == nil {
		t.Fatal("mixed unsafe DNS answer accepted")
	}
}

func TestTLSVerificationIsNotDisabled(t *testing.T) {
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer target.Close()
	lab, err := LabTarget(context.Background(), target.URL)
	if err != nil {
		t.Fatal(err)
	}
	transport := &http.Transport{DialContext: lab.Policy.dial}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	if resp, err := client.Get(target.URL + "/api/documents/doc-alice"); err == nil {
		resp.Body.Close()
		t.Fatal("untrusted TLS certificate accepted")
	}
}
