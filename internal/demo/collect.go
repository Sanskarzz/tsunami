package demo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Sanskarzz/tsunami/internal/platform"
)

// CollectKubernetes is a local operator command, not an agent capability.
// kubectl credentials and the operator token never enter model context/output.
type CollectOptions struct {
	Execution string
	IDs       []string
}

func CollectKubernetes(ctx context.Context, kubeContext, outDir string, log io.Writer, options ...CollectOptions) error {
	opt := CollectOptions{Execution: "scripted"}
	if len(options) > 0 {
		opt = options[0]
	}
	if opt.Execution != "scripted" && opt.Execution != "live" && opt.Execution != "all" {
		return errors.New("unknown collection execution filter")
	}
	for _, id := range opt.IDs {
		if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(id) {
			return errors.New("invalid requested run ID")
		}
	}
	args := []string{"--context", kubeContext, "-n", "tsunami"}
	secretArgs := append(append([]string{}, args...), "get", "secret", "tsunami-operator", "-o", "jsonpath={.data.admin-a}")
	encoded, err := exec.CommandContext(ctx, "kubectl", secretArgs...).Output()
	if err != nil {
		return errors.New("cannot read the lab operator credential with kubectl")
	}
	token, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	address := listener.Addr().String()
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	forwardCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	forwardArgs := append(append([]string{}, args...), "port-forward", "--address=127.0.0.1", "service/tsunami-control", fmt.Sprintf("%d:8080", port))
	cmd := exec.CommandContext(forwardCtx, "kubectl", forwardArgs...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	defer func() { cancel(); <-done }()
	o := Operator{BaseURL: "http://" + address, Token: string(token)}
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	// Readiness must not download the complete run history, including for exact IDs.
	for {
		if err = o.Call(ctx, "GET", "/healthz", nil, nil); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("lab API port-forward did not become ready")
		case <-time.After(50 * time.Millisecond):
		}
	}
	return collectReports(ctx, o, outDir, log, opt)
}

// Default collection selects only the original five scenarios. Exact IDs also
// support configured applications and bypass listing entirely.
func collectReports(ctx context.Context, o Operator, outDir string, log io.Writer, opt CollectOptions) error {
	selected := map[string]platform.RunSummary{}
	if len(opt.IDs) > 0 {
		for _, id := range opt.IDs {
			selected[id] = platform.RunSummary{ID: id}
		}
	} else {
		cursor := ""
		for {
			var page platform.RunPage
			if err := o.Call(ctx, "GET", "/v1/runs?limit=100&cursor="+url.QueryEscape(cursor), nil, &page); err != nil {
				return err
			}
			for _, item := range page.Items {
				if !legacyScenario(item.Scenario) || !matchesExecution(item.Execution, opt.Execution) {
					continue
				}
				old, ok := selected[item.Scenario]
				if !ok || item.CreatedAt.After(old.CreatedAt) {
					selected[item.Scenario] = item
				}
			}
			if page.NextCursor == "" {
				break
			}
			if page.NextCursor <= cursor || !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(page.NextCursor) {
				return errors.New("invalid run-list cursor")
			}
			cursor = page.NextCursor
		}
	}
	// Fetch and validate everything before writing any report, for deterministic
	// failure rather than map-order-dependent partially written collections.
	reports := map[string]platform.Run{}
	for stem, item := range selected {
		var run platform.Run
		if err := o.Call(ctx, "GET", "/v1/runs/"+item.ID, nil, &run); err != nil {
			return err
		}
		if run.ID != item.ID || !matchesExecution(run.Execution, opt.Execution) {
			return errors.New("requested run identity or execution filter differs")
		}
		if !legacyScenario(run.Scenario) && !(len(opt.IDs) > 0 && run.Scenario == "application" && run.Application != nil) {
			return errors.New("unsupported stored scenario")
		}
		reports[stem] = run
	}
	if len(reports) == 0 {
		return errors.New("no lab runs to collect")
	}
	if err := os.MkdirAll(outDir, 0700); err != nil {
		return err
	}
	stems := make([]string, 0, len(reports))
	for stem := range reports {
		stems = append(stems, stem)
	}
	sort.Strings(stems)
	for _, stem := range stems {
		r := reports[stem]
		data, e := json.MarshalIndent(r, "", "  ")
		if e != nil {
			return e
		}
		if e = os.WriteFile(filepath.Join(outDir, stem+".json"), data, 0600); e != nil {
			return e
		}
		if e = os.WriteFile(filepath.Join(outDir, stem+".md"), []byte(platform.Report(r)), 0600); e != nil {
			return e
		}
	}
	if log != nil {
		fmt.Fprintf(log, "Collected %d synthetic run reports into %s (no credentials exported)\n", len(reports), outDir)
	}
	return nil
}

func legacyScenario(s string) bool {
	return s == "vulnerable" || s == "fixed" || s == "inconclusive" || s == "redirect" || s == "stop"
}
func matchesExecution(stored, filter string) bool {
	if stored == "" {
		stored = "scripted"
	}
	return filter == "all" || filter == stored
}
