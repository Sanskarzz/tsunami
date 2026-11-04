package connector

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/netip"
	"net/url"
	"os"
	"strings"

	"github.com/Sanskarzz/tsunami/internal/platform"
)

// LabConfig is installed by the operator. Address approvals are explicit exact
// IPs, not authorization inferred from whichever DNS answer happens to appear.
type LabConfig struct {
	Tenant       string `json:"tenant"`
	Audience     string `json:"audience"`
	Endpoint     string `json:"endpoint"`
	PublicKey    string `json:"public_key"`
	TLSDirectory string `json:"tls_directory"`
	Targets      map[string]struct {
		URL       string   `json:"url"`
		Addresses []string `json:"addresses"`
	} `json:"targets"`
}

func LoadConfig(path string) (LabConfig, map[string]platform.Target, ed25519.PublicKey, error) {
	var cfg LabConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil, nil, err
	}
	if json.Unmarshal(data, &cfg) != nil || len(cfg.Targets) != 2 || cfg.Tenant == "" || cfg.Audience == "" {
		return cfg, nil, nil, errors.New("invalid private lab configuration")
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Hostname() == "" || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || cfg.TLSDirectory == "" {
		return cfg, nil, nil, errors.New("connector endpoint requires an explicit HTTPS origin and TLS directory")
	}
	key, err := base64.StdEncoding.DecodeString(cfg.PublicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return cfg, nil, nil, errors.New("invalid verification key")
	}
	targets := map[string]platform.Target{}
	for _, name := range []string{"vulnerable", "fixed"} {
		t := cfg.Targets[name]
		prefixes := []netip.Prefix{}
		for _, address := range t.Addresses {
			ip, err := netip.ParseAddr(address)
			if err != nil {
				return cfg, nil, nil, errors.New("private lab approvals require exact IPs")
			}
			ip = ip.Unmap()
			prefixes = append(prefixes, netip.PrefixFrom(ip, ip.BitLen()))
		}
		policy, err := platform.NewPolicy(t.URL, prefixes, nil)
		if err != nil {
			return cfg, nil, nil, err
		}
		targets[name] = platform.Target{URL: strings.TrimRight(t.URL, "/"), Policy: policy}
	}
	return cfg, targets, ed25519.PublicKey(key), nil
}

func ConfigureBroker(s *platform.Server, cfg LabConfig, targets map[string]platform.Target, keyFile string) error {
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return err
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	pub, pubErr := base64.StdEncoding.DecodeString(cfg.PublicKey)
	if err != nil || pubErr != nil || len(key) != ed25519.PrivateKeySize || !ed25519.PublicKey(pub).Equal(ed25519.PrivateKey(key).Public()) {
		return errors.New("signing key does not match connector configuration")
	}
	client := Client{Endpoint: cfg.Endpoint, Tenant: cfg.Tenant, Audience: cfg.Audience, Key: ed25519.PrivateKey(key), Lifetime: MaxGrantLifetime}
	u, _ := url.Parse(cfg.Endpoint)
	client.TLS, err = LoadClientTLS(cfg.TLSDirectory+"/ca.crt", cfg.TLSDirectory+"/tls.crt", cfg.TLSDirectory+"/tls.key", u.Hostname(), cfg.Tenant, cfg.Audience, "broker")
	if err != nil {
		return err
	}
	for name, target := range targets {
		target.Forward = client.Forward
		targets[name] = target
	}
	s.PrivateTargets = map[string]map[string]platform.Target{cfg.Tenant: targets}
	return nil
}
