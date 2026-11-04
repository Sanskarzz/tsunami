package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// Curl runs trusted bypass diagnostics with actual curl. Never registered as an
// agent tool. Inputs must be literal private lab addresses, without credentials.
func Curl(ctx context.Context, cfg Config, output io.Writer) error {
	if _, err := Address(cfg.Broker); err != nil {
		return err
	}
	if len(cfg.Destinations) == 0 || len(cfg.Destinations) > 16 {
		return errors.New("bounded diagnostic destinations required")
	}
	names := []string{}
	for name, raw := range cfg.Destinations {
		if _, err := Address(raw); err != nil {
			return err
		}
		names = append(names, name)
	}
	sort.Strings(names)
	probe := func(raw, mode string) (string, bool, error) {
		address, err := Address(raw)
		if err != nil {
			return "", false, err
		}
		u, _ := url.Parse(raw)
		host, port, _ := net.SplitHostPort(address)
		args := []string{"--disable", "--silent", "--proxy", "", "--noproxy", "*", "--connect-timeout", "0.5", "--max-time", "0.8", "--output", "/dev/null", "--write-out", "%{remote_ip}", "--retry", "0"}
		// Insecure TLS is used ONLY for credential-free TCP diagnostics to the
		// lab Kubernetes API. The supported adapter and broker never use it.
		args = append(args, "--insecure")
		switch mode {
		case "resolve":
			args = append(args, "--resolve", "scope-probe.invalid:"+port+":"+host)
			raw = u.Scheme + "://scope-probe.invalid:" + port + "/"
		case "connect-to":
			args = append(args, "--connect-to", "scope-probe.invalid:"+port+":"+address)
			raw = u.Scheme + "://scope-probe.invalid:" + port + "/"
		case "literal":
			raw = u.Scheme + "://" + address + "/"
		default:
			return "", false, errors.New("unknown curl diagnostic")
		}
		args = append(args, raw)
		opctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(opctx, "/usr/bin/curl", args...)
		cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
		cmd.Stderr = io.Discard
		data, err := cmd.Output()
		peer := strings.TrimSpace(string(data))
		if peer != "" {
			if peer != host {
				return "", false, errors.New("unexpected diagnostic peer")
			}
			return "reachable", true, nil
		}
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 28 && opctx.Err() == nil {
			return "timeout", false, nil
		}
		return "unproven", false, nil
	}
	passed := true
	for _, phase := range []string{"startup", "settled", "repeat"} {
		if phase != "startup" {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}
		outcome, reachable, err := probe(cfg.Broker, "literal")
		if err != nil {
			return err
		}
		_ = json.NewEncoder(output).Encode(map[string]any{"phase": phase, "destination": "broker", "mode": "literal", "outcome": outcome})
		if !reachable {
			passed = false
		}
		for _, name := range names {
			for _, mode := range []string{"literal", "resolve", "connect-to"} {
				outcome, reachable, err := probe(cfg.Destinations[name], mode)
				if err != nil {
					return err
				}
				ok := (cfg.PositiveControl && reachable) || (!cfg.PositiveControl && outcome == "timeout")
				_ = json.NewEncoder(output).Encode(map[string]any{"phase": phase, "destination": name, "mode": mode, "outcome": outcome, "passed": ok})
				if !ok {
					passed = false
				}
			}
		}
	}
	if !passed {
		return fmt.Errorf("curl boundary unqualified; inspect positive controls and startup results")
	}
	return nil
}
