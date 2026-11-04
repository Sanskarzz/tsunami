// Package isolation contains trusted lab diagnostics, not agent tools.
package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

type Config struct {
	Broker          string
	DNS             string
	Destinations    map[string]string
	PositiveControl bool
	Delay           time.Duration
	// Supplied only by the trusted operator harness after resolving the fixed
	// OpenRouter hostname. It adds one TCP/443 dependency check, not scanning.
	OpenRouterIPv4 string
}

type Observation struct {
	Phase       string `json:"phase"`
	Destination string `json:"destination"`
	Address     string `json:"address"`
	Outcome     string `json:"outcome"`
}

// Address accepts only operator-supplied literal private lab addresses. The
// probe does not resolve names, follow redirects, or consult proxy variables.
func Address(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("probe requires a plain HTTP(S) lab origin")
	}
	ip, err := netip.ParseAddr(u.Hostname())
	if err != nil || !ip.Unmap().IsPrivate() || ip.Zone() != "" {
		return "", errors.New("probe requires a literal private lab IP")
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return net.JoinHostPort(ip.String(), port), nil
}

func attempt(ctx context.Context, address string) string {
	conn, err := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "tcp", address)
	if err == nil {
		_ = conn.Close()
		return "reachable"
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() && ctx.Err() == nil {
		return "timeout"
	}
	// A refused connection is not proof of policy: the destination may be dead.
	return "unproven: " + err.Error()
}

func attemptDNS(parent context.Context, address string) string {
	ctx, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()
	resolver := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		if network != "udp" {
			return nil, errors.New("probe permits only UDP DNS")
		}
		return (&net.Dialer{}).DialContext(ctx, "udp", address)
	}}
	ips, err := resolver.LookupNetIP(ctx, "ip4", "kubernetes.default.svc.cluster.local.")
	if err == nil && len(ips) > 0 {
		return "reachable"
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) && parent.Err() == nil {
		return "timeout"
	}
	return fmt.Sprintf("unproven: DNS lookup: %v", err)
}

// Run tries fresh TCP connections at startup and after policy convergence.
// Every phase must deny the runner. Sleeping before starting an agent would
// hide a startup gap rather than fix it.
func Run(ctx context.Context, cfg Config, output io.Writer) error {
	if _, err := Address(cfg.Broker); err != nil {
		return err
	}
	if len(cfg.Destinations) == 0 {
		return errors.New("no probe destinations")
	}
	addresses := make(map[string]string, len(cfg.Destinations))
	names := make([]string, 0, len(cfg.Destinations))
	for name, raw := range cfg.Destinations {
		address, err := Address(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		addresses[name] = address
		names = append(names, name)
	}
	if cfg.DNS != "" {
		address, err := Address(cfg.DNS)
		if err != nil {
			return err
		}
		addresses["dns-udp"] = address
		names = append(names, "dns-udp")
	}
	if cfg.OpenRouterIPv4 != "" {
		ip, err := netip.ParseAddr(cfg.OpenRouterIPv4)
		if err != nil || !ip.Is4() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return errors.New("operator model probe requires a public IPv4 dependency address")
		}
		addresses["openrouter-public"] = net.JoinHostPort(ip.String(), "443")
		names = append(names, "openrouter-public")
	}
	sort.Strings(names)
	if cfg.Delay <= 0 {
		cfg.Delay = 3 * time.Second
	}
	if version, err := os.ReadFile("/proc/version"); err == nil {
		fmt.Fprintf(output, "Kernel: %s\n", strings.TrimSpace(string(version)))
	}
	passed := true
	encoder := json.NewEncoder(output)
	for _, phase := range []string{"startup", "settled", "repeat"} {
		if phase != "startup" {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(cfg.Delay):
			}
		}
		observations := make(chan Observation, len(names))
		for _, name := range names {
			go func(name string) {
				address := addresses[name]
				outcome := ""
				if name == "dns-udp" {
					outcome = attemptDNS(ctx, address)
				} else {
					outcome = attempt(ctx, address)
				}
				observations <- Observation{phase, name, address, outcome}
			}(name)
		}
		for range names {
			observation := <-observations
			if err := encoder.Encode(observation); err != nil {
				return err
			}
			want := "timeout"
			if cfg.PositiveControl {
				want = "reachable"
			}
			if observation.Outcome != want {
				passed = false
			}
		}
	}
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(cfg.Broker, "/")+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("broker positive check unavailable: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("broker positive check failed")
	}
	fmt.Fprintln(output, "PASS broker HTTP reachable")
	if !passed {
		return errors.New("network qualification failed; inspect all phases, including startup")
	}
	if cfg.PositiveControl {
		fmt.Fprintln(output, "PASS positive control: all direct lab paths reachable")
	} else {
		fmt.Fprintln(output, "PASS tested paths denied at startup and after settling; this is not full sandbox qualification")
	}
	return nil
}
