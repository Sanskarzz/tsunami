package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/connector"
	"github.com/Sanskarzz/tsunami/internal/demo"
	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/Sanskarzz/tsunami/internal/isolation"
	"github.com/Sanskarzz/tsunami/internal/kube"
	"github.com/Sanskarzz/tsunami/internal/lab"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/worker"
	"github.com/stacklok/mecatl/engine/port"
	"github.com/stacklok/mecatl/engine/session"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "tsunami:", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func secretFile(path string) (string, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return "", e
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", errors.New("empty credential file")
	}
	return s, nil
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: tsunami local-demo | serve | worker | lab | verify")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	switch os.Args[1] {
	case "export-bundle":
		fs := flag.NewFlagSet("export-bundle", flag.ContinueOnError)
		input := fs.String("input", "", "operator-collected bundle JSON")
		out := fs.String("out", "", "new output directory")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		if *input == "" || *out == "" {
			return errors.New("input and new output directory required")
		}
		f, err := os.Open(*input)
		if err != nil {
			return err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.Size() > platform.MaxInlineExportBytes {
			return errors.New("bundle exceeds inline export byte limit")
		}
		var b platform.EngagementBundle
		decoder := json.NewDecoder(io.LimitReader(f, platform.MaxInlineExportBytes+1))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&b); err != nil {
			return errors.New("invalid bounded bundle")
		}
		if decoder.Decode(&struct{}{}) != io.EOF {
			return errors.New("extra bundle data")
		}
		return demo.WriteEngagementBundle(*out, b)
	case "application-local", "application-verify":
		fs := flag.NewFlagSet(os.Args[1], flag.ContinueOnError)
		config := fs.String("config", "", "trusted application config file; local only")
		out := fs.String("out", "artifacts/application-adapters", "new campaign parent")
		executor := fs.String("executor", "", "approved reviewed executor; Kubernetes only")
		caseFlag := fs.String("cases", "", "comma-separated application=expected pairs")
		review := fs.Bool("exercise-review", false, "explicit automated lab review, not human signoff")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		cases := map[string]string{}
		for _, entry := range strings.Split(*caseFlag, ",") {
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) != 2 || cases[parts[0]] != "" {
				return errors.New("explicit distinct evaluation cases required")
			}
			cases[parts[0]] = parts[1]
		}
		var summary demo.ApplicationSummary
		var err error
		if os.Args[1] == "application-local" {
			if *executor != "" {
				return errors.New("reviewed tool execution requires the qualified Kubernetes sandbox")
			}
			if *config == "" {
				return errors.New("trusted config file required")
			}
			summary, err = demo.ApplicationLocal(ctx, *config, *out, cases, *review, os.Stdout)
		} else {
			var token string
			token, err = secretFile(env("TSUNAMI_ADMIN_TOKEN_FILE", "/run/operator/admin-a"))
			if err == nil {
				summary, err = demo.ApplicationVerify(ctx, demo.Operator{BaseURL: env("TSUNAMI_BROKER_URL", "http://tsunami-control:8080"), Token: token}, *out, cases, *review, os.Stdout, *executor)
			}
		}
		data, _ := json.Marshal(summary)
		fmt.Printf("APPLICATION_REPORT %s\n", data)
		return err
	case "engagement-local":
		fs := flag.NewFlagSet("engagement-local", flag.ContinueOnError)
		out := fs.String("out", "artifacts/engagements", "new campaign output directory")
		fixture := fs.Bool("fixture", false, "offline Responses fixture, never an external model")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		_, err := demo.EngagementLocal(ctx, *out, *fixture, os.Stdout)
		return err
	case "engagement-verify":
		fs := flag.NewFlagSet("engagement-verify", flag.ContinueOnError)
		out := fs.String("out", "/tmp/engagements", "campaign output directory")
		connectivity := fs.String("connectivity", "", "private or empty for public lab")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		if *connectivity != "" && *connectivity != "private" {
			return errors.New("unsupported connectivity")
		}
		token, err := secretFile(env("TSUNAMI_ADMIN_TOKEN_FILE", "/run/operator/admin-a"))
		if err != nil {
			return err
		}
		summary, err := demo.EngagementVerify(ctx, demo.Operator{BaseURL: env("TSUNAMI_BROKER_URL", "http://tsunami-control:8080"), Token: token, Connectivity: *connectivity}, *out, false, os.Stdout)
		data, _ := json.Marshal(summary)
		fmt.Printf("ENGAGEMENT_REPORT %s\n", data)
		return err
	case "catalog-analyze":
		fs := flag.NewFlagSet("catalog-analyze", flag.ContinueOnError)
		source := fs.String("source", "", "directory of archived run JSON files")
		out := fs.String("out", "artifacts/catalog-analysis", "new analysis directory, outside source")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		if *source == "" {
			return errors.New("catalog-analyze requires --source")
		}
		report, err := demo.AnalyzeCatalogDirectory(*source, *out)
		if err != nil {
			return err
		}
		fmt.Printf("Analyzed %d stored runs: matched=%d operational_failures=%d assessment_misses=%d. Original records and verdicts unchanged.\n", len(report.Items), report.Matched, report.OperationalFailures, report.AssessmentMisses)
		return nil
	case "diagnose-inference":
		fs := flag.NewFlagSet("diagnose-inference", flag.ContinueOnError)
		out := fs.String("out", "artifacts/inference-diagnostics", "sanitized fault campaign output")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		_, err := demo.DiagnoseInference(ctx, *out, os.Stdout)
		return err
	case "catalog-list":
		return json.NewEncoder(os.Stdout).Encode(assessment.List())
	case "catalog-local":
		fs := flag.NewFlagSet("catalog-local", flag.ContinueOnError)
		fixture := fs.Bool("fixture", false, "use the offline Responses fixture; never an external provider")
		repeats := fs.Int("repeat", 2, "repeats of the five-procedure pair suite (1-3)")
		out := fs.String("out", "artifacts/assessment-local", "artifact directory")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		_, err := demo.CatalogLocal(ctx, *out, *fixture, *repeats, os.Stdout)
		return err
	case "catalog-eval":
		fs := flag.NewFlagSet("catalog-eval", flag.ContinueOnError)
		mode := fs.String("mode", "scripted", "scripted, fixture, or free")
		repeats := fs.Int("repeat", 2, "repetitions (1-3); free mode permits one")
		ids := fs.String("assessments", "", "comma-separated known IDs; free mode requires at most two")
		connectivity := fs.String("connectivity", "", "private or empty for the public lab")
		allow := fs.Bool("allow-model-calls", false, "explicitly permit real free inference")
		out := fs.String("out", "/tmp/catalog", "artifact directory")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		if *mode == "free" && !*allow {
			return errors.New("free catalog inference requires explicit --allow-model-calls")
		}
		if *connectivity != "" && *connectivity != "private" {
			return errors.New("invalid connectivity")
		}
		token, err := secretFile(env("TSUNAMI_ADMIN_TOKEN_FILE", "/run/operator/admin-a"))
		if err != nil {
			return err
		}
		selected := []string{}
		if *ids != "" {
			selected = strings.Split(*ids, ",")
		}
		report, err := demo.Catalog(ctx, demo.Operator{BaseURL: env("TSUNAMI_BROKER_URL", "http://tsunami-control:8080"), Token: token, Connectivity: *connectivity}, *out, demo.CatalogOptions{Mode: *mode, Repeats: *repeats, IDs: selected}, os.Stdout)
		data, _ := json.Marshal(report)
		fmt.Printf("CATALOG_REPORT %s\n", data)
		return err
	case "connector-transport-check":
		fs := flag.NewFlagSet("connector-transport-check", flag.ContinueOnError)
		dir := fs.String("pki-dir", ".local/private-pki", "local operator TLS files")
		endpoint := fs.String("endpoint", "", "loopback HTTPS port-forward")
		out := fs.String("out", "artifacts/connector-tls/transport.json", "sanitized diagnostic record")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		key, err := io.ReadAll(io.LimitReader(os.Stdin, 512))
		if err != nil || len(key) >= 512 {
			return errors.New("invalid emergency credential input")
		}
		if err := connector.CheckTransport(ctx, *dir, *endpoint, strings.TrimSpace(string(key)), *out); err != nil {
			return err
		}
		fmt.Println("PASS connector mTLS, role separation, identity/CA/version rejection, and plaintext refusal; no target requests or model calls.")
		return nil
	case "bootstrap-connector-tls":
		fs := flag.NewFlagSet("bootstrap-connector-tls", flag.ContinueOnError)
		dir := fs.String("dir", ".local/private-pki", "ignored offline lab PKI directory")
		renew := fs.Bool("renew", false, "explicitly renew leaf certificates under the same CA")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		record, err := connector.ProvisionLabPKI(*dir, *renew)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(record)
	case "connector-cutoff":
		fs := flag.NewFlagSet("connector-cutoff", flag.ContinueOnError)
		dir := fs.String("pki-dir", ".local/private-pki", "local emergency client TLS files")
		endpoint := fs.String("endpoint", "", "loopback-only operator port-forward HTTPS endpoint")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		u, err := url.Parse(*endpoint)
		if err != nil || u.Scheme != "https" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
			return errors.New("emergency helper requires a loopback HTTPS port-forward")
		}
		tlsConfig, err := connector.LoadClientTLS(filepath.Join(*dir, "ca.crt"), filepath.Join(*dir, "emergency-operator.crt"), filepath.Join(*dir, "emergency-operator.key"), connector.LabServerName, connector.LabTenant, connector.LabAudience, "emergency-operator")
		if err != nil {
			return err
		}
		key, err := io.ReadAll(io.LimitReader(os.Stdin, 512))
		if err != nil || len(strings.TrimSpace(string(key))) < 32 || len(key) >= 512 {
			return errors.New("emergency credential required on stdin")
		}
		req, _ := http.NewRequestWithContext(ctx, "POST", *endpoint+"/v1/cutoff", nil)
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(key)))
		client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil, TLSClientConfig: tlsConfig, DisableKeepAlives: true}, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
		resp, err := client.Do(req)
		if err != nil {
			return errors.New("authenticated emergency transport failed; cutoff unconfirmed")
		}
		defer resp.Body.Close()
		var state struct {
			Cutoff bool `json:"cutoff"`
		}
		if resp.StatusCode != 200 || json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&state) != nil || !state.Cutoff {
			return errors.New("emergency cutoff unconfirmed")
		}
		fmt.Println("Connector emergency cutoff persisted over mTLS; private execution remains disabled across restarts.")
		return nil
	case "bootstrap-connector-keys":
		// Only for piping into the trusted setup harness; never log this output.
		pub, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return err
		}
		emergency := make([]byte, 32)
		if _, err := rand.Read(emergency); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"signing-key": base64.StdEncoding.EncodeToString(key), "public-key": base64.StdEncoding.EncodeToString(pub), "emergency-key": hex.EncodeToString(emergency)})
	case "connector":
		if os.Getenv("TSUNAMI_ALLOW_PRIVATE_LAB") != "true" {
			return errors.New("connector still uses synthetic targets; explicit lab acknowledgement required")
		}
		cfg, targets, publicKey, err := connector.LoadConfig(env("TSUNAMI_PRIVATE_CONFIG_FILE", "/run/private/config.json"))
		if err != nil {
			return err
		}
		key, err := secretFile(env("TSUNAMI_EMERGENCY_KEY_FILE", "/run/connector/emergency-key"))
		if err != nil {
			return err
		}
		s, err := connector.New(env("TSUNAMI_CONNECTOR_STATE_PATH", "/data/connector.db"), cfg.Tenant, cfg.Audience, publicKey, key, targets["vulnerable"].Policy, targets["fixed"].Policy)
		if err != nil {
			return err
		}
		defer s.Close()
		u, _ := url.Parse(cfg.Endpoint)
		tlsConfig, notAfter, err := connector.LoadServerTLS(filepath.Join(cfg.TLSDirectory, "ca.crt"), filepath.Join(cfg.TLSDirectory, "tls.crt"), filepath.Join(cfg.TLSDirectory, "tls.key"), u.Hostname(), cfg.Tenant, cfg.Audience)
		if err != nil {
			return err
		}
		s.TLSNotAfter = notAfter
		return serve(ctx, env("LISTEN_ADDR", ":8080"), s, tlsConfig)
	case "model-check":
		fs := flag.NewFlagSet("model-check", flag.ContinueOnError)
		model := fs.String("model", "", "explicit OpenRouter :free model")
		keyPath := fs.String("key-file", "", "local key file")
		out := fs.String("out", "artifacts/openrouter/model-check.json", "sanitized diagnostic output")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		key, err := secretFile(*keyPath)
		if err != nil {
			return errors.New("model key file unavailable")
		}
		cfg, err := inference.NewConfig("openrouter-free", *model, inference.OpenRouterBaseURL, key)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		attempts := 0
		start := time.Now()
		reply, usage, err := cfg.Complete(ctx, "http://lab.invalid:8080", inference.Request{Messages: []session.Message{session.NewUserMessage("This is a connection check only. Call finish_assessment with empty arguments; do not request HTTP traffic.")}}, func(context.Context) error {
			attempts++
			if attempts > 1 {
				return errors.New("diagnostic attempt limit")
			}
			return nil
		})
		names := []string{}
		for _, chunk := range reply.Chunks {
			if chunk.Kind == port.ChunkToolCall && chunk.ToolCall != nil {
				for _, spec := range inference.Tools() {
					if chunk.ToolCall.Name == spec.Name {
						names = append(names, spec.Name)
					}
				}
			}
		}
		code := ""
		if err != nil {
			code = inference.ErrorCode(err)
		}
		record := map[string]any{"checked_at": time.Now().UTC(), "model": *model, "price_policy": inference.FreePolicyID, "attempts": attempts, "completed": err == nil, "error_code": code, "usage": usage, "tools": names, "duration_ms": time.Since(start).Milliseconds()}
		data, _ := json.MarshalIndent(record, "", "  ")
		if writeErr := os.MkdirAll(filepath.Dir(*out), 0700); writeErr != nil {
			return writeErr
		}
		if writeErr := os.WriteFile(*out, data, 0600); writeErr != nil {
			return writeErr
		}
		fmt.Println(string(data))
		if err != nil {
			return errors.New("free model check failed; inspect sanitized diagnostic")
		}
		return nil
	case "model-fixture":
		return serve(ctx, env("LISTEN_ADDR", ":8080"), &inference.Fixture{})
	case "model-fixture-demo":
		fs := flag.NewFlagSet("model-fixture-demo", flag.ContinueOnError)
		out := fs.String("out", "artifacts/model-fixture", "artifact directory")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		return demo.ModelFixture(ctx, *out, os.Stdout)
	case "live-eval":
		fs := flag.NewFlagSet("live-eval", flag.ContinueOnError)
		allow := fs.Bool("allow-model-calls", false, "explicitly allow configured model calls; external providers can incur costs")
		requireFree := fs.Bool("require-free", false, "require the server's free-only price policy")
		out := fs.String("out", "/tmp/tsunami-live", "artifact directory")
		repeats := fs.Int("repeat", 1, "number of vulnerable/fixed pairs (1-3)")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		if !*allow {
			return errors.New("live-eval requires explicit --allow-model-calls; default verification never calls a live provider")
		}
		token, err := secretFile(env("TSUNAMI_ADMIN_TOKEN_FILE", "/run/operator/admin-a"))
		if err != nil {
			return err
		}
		policy := ""
		if *requireFree {
			policy = inference.FreePolicyID
		}
		_, err = demo.Evaluate(ctx, demo.Operator{BaseURL: env("TSUNAMI_BROKER_URL", "http://tsunami-control:8080"), Token: token, RequiredPricePolicy: policy}, *out, *repeats, os.Stdout)
		return err
	case "live-stop-check":
		fs := flag.NewFlagSet("live-stop-check", flag.ContinueOnError)
		allow := fs.Bool("allow-model-calls", false, "explicitly allow one run using the configured model")
		requireFree := fs.Bool("require-free", false, "require the server's free-only price policy")
		out := fs.String("out", "/tmp/tsunami-live-stop", "artifact directory")
		if err := fs.Parse(os.Args[2:]); err != nil {
			return err
		}
		if !*allow {
			return errors.New("live-stop-check requires explicit --allow-model-calls")
		}
		token, err := secretFile(env("TSUNAMI_ADMIN_TOKEN_FILE", "/run/operator/admin-a"))
		if err != nil {
			return err
		}
		policy := ""
		if *requireFree {
			policy = inference.FreePolicyID
		}
		return demo.StopInference(ctx, demo.Operator{BaseURL: env("TSUNAMI_BROKER_URL", "http://tsunami-control:8080"), Token: token, RequiredPricePolicy: policy}, *out, os.Stdout)
	case "runtime-probe":
		return isolation.Runtime(ctx, os.Stdout)
	case "network-probe", "curl-network-probe":
		var destinations map[string]string
		if err := json.Unmarshal([]byte(os.Getenv("TSUNAMI_PROBE_DESTINATIONS")), &destinations); err != nil {
			return err
		}
		if os.Args[1] == "curl-network-probe" {
			if err := isolation.Curl(ctx, isolation.Config{Broker: os.Getenv("TSUNAMI_PROBE_BROKER"), Destinations: destinations, PositiveControl: os.Getenv("TSUNAMI_PROBE_POSITIVE") == "true"}, os.Stdout); err != nil {
				return err
			}
			return isolation.RuntimeSmoke(os.Stdout)
		}
		return isolation.Run(ctx, isolation.Config{Broker: os.Getenv("TSUNAMI_PROBE_BROKER"), DNS: os.Getenv("TSUNAMI_PROBE_DNS"), Destinations: destinations, PositiveControl: os.Getenv("TSUNAMI_PROBE_POSITIVE") == "true", OpenRouterIPv4: os.Getenv("TSUNAMI_PROBE_OPENROUTER_IPV4")}, os.Stdout)
	case "collect-k8s":
		fs := flag.NewFlagSet("collect-k8s", flag.ContinueOnError)
		kubeContext := fs.String("context", "kind-kind", "explicit kubectl context")
		out := fs.String("out", "artifacts/kubernetes", "artifact directory")
		execution := fs.String("execution", "scripted", "collect scripted, live, or all runs")
		runs := fs.String("runs", "", "comma-separated exact run IDs instead of latest per scenario")
		if e := fs.Parse(os.Args[2:]); e != nil {
			return e
		}
		var ids []string
		if *runs != "" {
			ids = strings.Split(*runs, ",")
		}
		return demo.CollectKubernetes(ctx, *kubeContext, *out, os.Stdout, demo.CollectOptions{Execution: *execution, IDs: ids})
	case "bootstrap-secret":
		data := map[string]string{}
		for _, name := range []string{"admin-a", "admin-b", "global"} {
			b := make([]byte, 32)
			if _, err := rand.Read(b); err != nil {
				return err
			}
			data[name] = hex.EncodeToString(b)
		}
		// Intended to be piped straight to kubectl; never written into Git.
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"apiVersion": "v1", "kind": "Secret", "metadata": map[string]string{"name": "tsunami-operator", "namespace": "tsunami"}, "type": "Opaque", "stringData": data})
	case "local-demo":
		fs := flag.NewFlagSet("local-demo", flag.ContinueOnError)
		out := fs.String("out", "artifacts/local", "artifact directory")
		if e := fs.Parse(os.Args[2:]); e != nil {
			return e
		}
		return demo.Local(ctx, *out, os.Stdout)
	case "lab":
		fs := flag.NewFlagSet("lab", flag.ContinueOnError)
		fixed := fs.Bool("fixed", false, "enforce object ownership")
		if e := fs.Parse(os.Args[2:]); e != nil {
			return e
		}
		return serve(ctx, env("LISTEN_ADDR", ":8080"), &lab.Server{Fixed: *fixed, RedirectTo: env("LAB_REDIRECT_TO", "http://not-approved.invalid/api/documents/doc-alice")})
	case "worker":
		token, e := secretFile(env("TSUNAMI_TOKEN_FILE", "/run/tsunami/token"))
		if e != nil {
			return e
		}
		return worker.Run(ctx, worker.Client{BaseURL: os.Getenv("TSUNAMI_BROKER_URL"), RunID: os.Getenv("TSUNAMI_RUN_ID"), Token: token}, os.Stdout)
	case "verify":
		fs := flag.NewFlagSet("verify", flag.ContinueOnError)
		out := fs.String("out", "/tmp/tsunami-artifacts", "artifact directory")
		private := fs.Bool("private", false, "verify the configured private fixture through its connector")
		if e := fs.Parse(os.Args[2:]); e != nil {
			return e
		}
		token, e := secretFile(env("TSUNAMI_ADMIN_TOKEN_FILE", "/run/operator/admin-a"))
		if e != nil {
			return e
		}
		ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
		defer cancel()
		vulnerableURL := "http://tsunami-vulnerable:8080"
		tripwireURL := "http://tsunami-tripwire:8080"
		connectivity := ""
		if *private {
			connectivity = "private"
			vulnerableURL = env("TSUNAMI_PRIVATE_VULNERABLE_URL", "http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080")
			tripwireURL = "http://tsunami-tripwire.tsunami-customer-a.svc.cluster.local:8080"
		}
		ready := func(ctx context.Context) error {
			for {
				stats, err := demo.LabStats(ctx, vulnerableURL+"/stats")
				if err != nil {
					return err
				}
				if stats["slow_active"] > 0 {
					return nil
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(20 * time.Millisecond):
				}
			}
		}
		summary, e := demo.Verify(ctx, demo.Operator{BaseURL: env("TSUNAMI_BROKER_URL", "http://tsunami-control:8080"), Token: token, ReadyForStop: ready, Connectivity: connectivity}, *out, os.Stdout)
		if e == nil {
			tripwire, err := demo.LabStats(ctx, tripwireURL+"/stats")
			if err != nil {
				return err
			}
			if tripwire["requests"] != 0 {
				return errors.New("unapproved lab service received traffic")
			}
			stats, err := demo.LabStats(ctx, vulnerableURL+"/stats")
			if err != nil {
				return err
			}
			if stats["cancelled"] == 0 {
				return errors.New("target did not observe cancellation")
			}
			fmt.Println("PASS tripwire: zero requests; lab target observed request cancellation")
			fmt.Printf("Completed %d scenario checks; reports in %s\n", len(summary.Checks), *out)
		}
		return e
	case "serve":
		if os.Getenv("TSUNAMI_ALLOW_UNSANDBOXED_LAB") != "true" {
			return errors.New("this initial fixture-only runner requires explicit TSUNAMI_ALLOW_UNSANDBOXED_LAB=true; it is not an untrusted-shell sandbox")
		}
		v, e := platform.LabTarget(ctx, env("TSUNAMI_VULNERABLE_URL", "http://tsunami-vulnerable:8080"))
		if e != nil {
			return e
		}
		f, e := platform.LabTarget(ctx, env("TSUNAMI_FIXED_URL", "http://tsunami-fixed:8080"))
		if e != nil {
			return e
		}
		admins := map[string]string{}
		for _, name := range []string{"admin-a", "admin-b"} {
			token, e := secretFile(filepath.Join(env("TSUNAMI_OPERATOR_DIR", "/run/operator"), name))
			if e != nil {
				return e
			}
			admins[token] = "demo-" + strings.TrimPrefix(name, "admin-")
		}
		global, e := secretFile(filepath.Join(env("TSUNAMI_OPERATOR_DIR", "/run/operator"), "global"))
		if e != nil {
			return e
		}
		store, e := platform.OpenStore(env("TSUNAMI_STATE_PATH", "/data/state.db"))
		if e != nil {
			return e
		}
		defer store.Close()
		server := platform.NewServer(store, map[string]platform.Target{"vulnerable": v, "fixed": f}, admins)
		server.GlobalToken = global
		server.EnableCurlAdapter = os.Getenv("TSUNAMI_ENABLE_CURL_ADAPTER") == "true"
		if server.EnableCurlAdapter && (os.Getenv("TSUNAMI_RUNTIME_CLASS") != "gvisor" || os.Getenv("TSUNAMI_TOOL_IMAGE") == "") {
			return errors.New("curl adapter requires gvisor and a trusted tool image")
		}
		if configFile := os.Getenv("TSUNAMI_APPLICATION_CONFIG_FILE"); configFile != "" {
			if err := server.LoadApplications(configFile); err != nil {
				return err
			}
		}
		if configFile := os.Getenv("TSUNAMI_PRIVATE_CONFIG_FILE"); configFile != "" {
			if os.Getenv("TSUNAMI_ALLOW_PRIVATE_LAB") != "true" {
				return errors.New("private connectivity currently requires the synthetic lab acknowledgement")
			}
			cfg, targets, _, err := connector.LoadConfig(configFile)
			if err != nil {
				return err
			}
			if err := connector.ConfigureBroker(server, cfg, targets, env("TSUNAMI_SIGNING_KEY_FILE", "/run/connector/signing-key")); err != nil {
				return err
			}
		}
		provider := os.Getenv("TSUNAMI_MODEL_PROVIDER")
		if provider != "" {
			if os.Getenv("TSUNAMI_ALLOW_MODEL_CALLS") != "true" {
				return errors.New("model configuration requires explicit TSUNAMI_ALLOW_MODEL_CALLS=true")
			}
			key := ""
			if provider == "responses" || provider == "openrouter-free" {
				key, e = secretFile(env("TSUNAMI_MODEL_KEY_FILE", "/run/model/api-key"))
				if e != nil {
					return errors.New("model key file unavailable")
				}
			}
			baseURL := env("TSUNAMI_MODEL_BASE_URL", "https://api.openai.com/v1/")
			server.Inference, e = inference.NewConfig(provider, os.Getenv("TSUNAMI_MODEL_ID"), baseURL, key)
			if e != nil {
				return e
			}
		}
		broker, err := url.Parse(env("TSUNAMI_BROKER_URL", "http://tsunami-control:8080"))
		if err != nil {
			return err
		}
		ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", broker.Hostname())
		if err != nil || len(ips) == 0 {
			return errors.New("cannot resolve broker service")
		}
		prt := broker.Port()
		if prt == "" {
			prt = "8080"
		}
		broker.Host = net.JoinHostPort(ips[0].String(), prt)
		// Workers receive the literal service IP and need no DNS egress.
		server.Scheduler, e = kube.InCluster(env("TSUNAMI_NAMESPACE", "tsunami"), env("TSUNAMI_IMAGE", "tsunami:dev"), broker.String(), os.Getenv("TSUNAMI_RUNTIME_CLASS"))
		if e != nil {
			return e
		}
		go server.Reconcile(ctx)
		fmt.Printf("LAB ONLY: scripted mode default; model path enabled=%t; runtimeClass=%q; no arbitrary shell\n", server.Inference != nil, os.Getenv("TSUNAMI_RUNTIME_CLASS"))
		return serve(ctx, env("LISTEN_ADDR", ":8080"), server)
	default:
		return errors.New("unknown subcommand")
	}
}

func serve(ctx context.Context, address string, h http.Handler, tlsConfig ...*tls.Config) error {
	s := &http.Server{Addr: address, Handler: h, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 15 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(shutdown)
	}()
	var err error
	if len(tlsConfig) > 0 {
		s.TLSConfig = tlsConfig[0]
		err = s.ListenAndServeTLS("", "")
	} else {
		err = s.ListenAndServe()
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
