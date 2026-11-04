package platform

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type Policy struct {
	origin   *url.URL
	allowed  []netip.Prefix
	resolver Resolver
	paths    map[string]bool
}

// WithPaths installs a closed procedure's exact path catalog.
// It does not change the approved origin or destination addresses.
func (p *Policy) WithPaths(paths []string) *Policy {
	copy := *p
	copy.paths = map[string]bool{}
	for _, path := range paths {
		copy.paths[path] = true
	}
	return &copy
}

// HTTPClient pins the validated dial address and never follows redirects. It
// is shared by the broker and private connector; each operation gets a fresh
// transport, so no tenant shares cookies or upstream connections.
func (p *Policy) HTTPClient() (*http.Client, func()) {
	t := &http.Transport{Proxy: nil, DialContext: p.dial, DisableKeepAlives: true, MaxResponseHeaderBytes: 16 << 10}
	return &http.Client{Transport: t, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, t.CloseIdleConnections
}

func NewPolicy(origin string, allowed []netip.Prefix, resolver Resolver) (*Policy, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("policy requires an explicit HTTP(S) origin")
	}
	if len(allowed) == 0 {
		return nil, errors.New("policy requires approved destination addresses")
	}
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return &Policy{origin: u, allowed: allowed, resolver: resolver}, nil
}

// LabTarget pins the initial DNS answers of an operator-configured fixture.
// This bootstrap convenience is LAB ONLY, not customer authorization of arbitrary DNS.
func LabTarget(ctx context.Context, origin string) (Target, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return Target{}, err
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", u.Hostname())
	if err != nil {
		return Target{}, err
	}
	var prefixes []netip.Prefix
	for _, ip := range ips {
		ip = ip.Unmap()
		prefixes = append(prefixes, netip.PrefixFrom(ip, ip.BitLen()))
	}
	p, err := NewPolicy(origin, prefixes, nil)
	return Target{URL: strings.TrimSuffix(origin, "/"), Policy: p}, err
}

func port(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	if u.Scheme == "https" {
		return "443"
	}
	return "80"
}

func (p *Policy) Validate(raw string) (*url.URL, error) {
	if len(raw) > 2048 {
		return nil, errors.New("URL too long")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.New("invalid URL")
	}
	if u.User != nil || u.Fragment != "" || u.Opaque != "" || u.Hostname() == "" ||
		u.Scheme != p.origin.Scheme || !strings.EqualFold(u.Hostname(), p.origin.Hostname()) || port(u) != port(p.origin) {
		return nil, errors.New("destination is outside the approved origin")
	}
	// The first catalog has a deliberately narrow read-only fixture surface.
	allowedPath := u.Path == "/api/slow" || u.Path == "/api/redirect" || u.Path == "/api/documents/doc-alice" || u.Path == "/api/documents/doc-bob" || u.Path == "/api/documents/unknown"
	if p.paths != nil {
		allowedPath = p.paths[u.Path]
	}
	if u.RawQuery != "" || u.RawPath != "" || !allowedPath {
		return nil, errors.New("path is outside the approved lab test catalog")
	}
	return u, nil
}

func (p *Policy) allowedIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range p.allowed {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func (p *Policy) dial(ctx context.Context, network, address string) (net.Conn, error) {
	host, prt, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(host, p.origin.Hostname()) || prt != port(p.origin) {
		return nil, errors.New("unexpected dial destination")
	}
	ips, err := p.resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, errors.New("DNS returned no addresses")
	}
	for _, ip := range ips {
		if !p.allowedIP(ip) {
			return nil, errors.New("DNS selected an unapproved address")
		}
	}
	// Dial the validated numeric address; do not resolve the hostname again.
	d := net.Dialer{Timeout: 2 * time.Second}
	var last error
	for _, ip := range ips {
		conn, e := d.DialContext(ctx, network, net.JoinHostPort(ip.Unmap().String(), prt))
		if e == nil {
			return conn, nil
		}
		last = e
	}
	return nil, fmt.Errorf("approved destination unavailable: %w", last)
}
