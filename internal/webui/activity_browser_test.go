package webui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/toolcontract"
	"github.com/Sanskarzz/tsunami/internal/worker"
)

// Opt-in browser integration uses the real control plane/broker/validator and
// Mecatl LocalScheduler, disposable loopback targets and a temporary database.
// It does not qualify Kubernetes, gVisor, Cilium or the frozen D2 deployment.
func TestLocalBrowserActivityAndStop(t *testing.T) {
	runLocalBrowser(t, "activity")
}

func TestLocalBrowserOnboardingAndReview(t *testing.T) {
	runLocalBrowser(t, "onboarding")
}

func TestLocalBrowserFindingsAndEvidence(t *testing.T) {
	runLocalBrowser(t, "findings")
}

func TestLocalBrowserRetest(t *testing.T) {
	runLocalBrowser(t, "retest")
}

func runLocalBrowser(t *testing.T, mode string) {
	t.Helper()
	onboarding, findings := mode == "onboarding", mode == "findings"
	retest := mode == "retest"
	enabled := os.Getenv("TSUNAMI_UI_BROWSER_TEST") == "1"
	if onboarding {
		enabled = os.Getenv("TSUNAMI_UI_ONBOARDING_TEST") == "1"
	}
	if findings {
		enabled = os.Getenv("TSUNAMI_UI_FINDINGS_TEST") == "1"
	}
	if retest {
		enabled = os.Getenv("TSUNAMI_UI_RETEST_TEST") == "1"
	}
	if !enabled {
		t.Skip("run make verify-ui-activity-browser with Node 22 and Chrome")
	}
	for _, tool := range []string{"node", "google-chrome"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatal("browser prerequisite missing: " + tool)
		}
	}
	var held, cancelled, unexpected atomic.Int32
	var requests, probes atomic.Int32
	credentials := map[string]string{"owner": strings.Repeat("a", 48), "peer": strings.Repeat("b", 48)}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		account := ""
		for alias, key := range credentials {
			if r.Header.Get("Authorization") == "Bearer "+key {
				account = alias
			}
		}
		if account == "" {
			w.WriteHeader(401)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) == 2 && parts[1] == "me" {
			fmt.Fprintf(w, `{"name":%q}`, account)
			return
		}
		if len(parts) != 3 || parts[1] != "records" {
			unexpected.Add(1)
			w.WriteHeader(404)
			return
		}
		mode, identifier := parts[0], parts[2]
		owner := strings.TrimPrefix(identifier, "doc-")
		if owner != "owner" && owner != "peer" {
			unexpected.Add(1)
			w.WriteHeader(404)
			return
		}
		if account == "peer" && owner == "owner" {
			probes.Add(1)
			if mode == "held" {
				held.Add(1)
				<-r.Context().Done()
				cancelled.Add(1)
				return
			}
			if mode == "fixed" {
				w.WriteHeader(403)
				return
			}
			if mode == "property-filtered" {
				fmt.Fprintf(w, `{"id":%q,"owner":%q}`, identifier, owner)
				return
			}
			if mode == "property-ambiguous" {
				fmt.Fprint(w, `{"error":"generic success"}`)
				return
			}
		}
		fmt.Fprintf(w, `{"id":%q,"owner":%q,"proof":%q}`, identifier, owner, "synthetic-private-proof-"+owner)
	}))
	defer target.Close()
	// The retest explicitly changes origin. Both targets are owned disposable
	// fixtures and contribute to the same independent request counters.
	var fixedTarget *httptest.Server
	if retest {
		fixedTarget = httptest.NewServer(target.Config.Handler)
		defer fixedTarget.Close()
	}
	store, err := platform.OpenStore(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	operator := strings.Repeat("c", 48)
	s := platform.NewServer(store, nil, map[string]string{operator: "owned-ui-test"})
	s.SetupKey = []byte(strings.Repeat("k", 32))
	s.EnableCurlAdapter = onboarding
	modes := []string{"vulnerable", "fixed", "held"}
	if findings || retest {
		modes = append(modes, "property", "property-filtered", "property-ambiguous")
	}
	for _, mode := range modes {
		object := func(alias string) application.Read {
			id := alias
			if onboarding {
				id = "doc-" + alias
			}
			sum := sha256.Sum256([]byte("synthetic-private-proof-" + alias))
			return application.Read{Account: alias, Path: "/" + mode + "/records/" + id, Equals: map[string]string{"/id": id, "/owner": alias}, Hashes: map[string]string{"/proof": hex.EncodeToString(sum[:])}}
		}
		owner, peer := object("owner"), object("peer")
		probe := owner
		probe.Account = "peer"
		plan := application.Plan{Version: application.ObjectAccessProfile, ID: "browser-" + mode, Title: "Owned browser " + mode + " test", Origin: target.URL, ApprovedAddresses: []string{"127.0.0.1"}, SyntheticDataOnly: true, OwnerPointer: "/owner",
			Identity: []application.Read{{Account: "owner", Path: "/" + mode + "/me", Equals: map[string]string{"/name": "owner"}}, {Account: "peer", Path: "/" + mode + "/me", Equals: map[string]string{"/name": "peer"}}}, Baselines: []application.Read{owner, peer}, Probe: probe, ExpectedAccess: "deny", DenialStatuses: []int{401, 403, 404}}
		if retest && mode == "fixed" {
			plan.Origin = fixedTarget.URL
		}
		if strings.HasPrefix(mode, "property") {
			plan.Version = application.PropertyAccessProfile
			plan.ExpectedAccess = "filtered"
			plan.DenialStatuses = nil
		}
		if err = s.RegisterApplication(platform.ApplicationConfig{Tenant: "owned-ui-test", Plan: plan, Credentials: credentials}); err != nil {
			t.Fatal(err)
		}
	}
	api := httptest.NewServer(Handler{Next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test/observations" {
			if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer "+operator {
				w.WriteHeader(401)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]int32{"requests": requests.Load(), "probes": probes.Load()})
			return
		}
		s.ServeHTTP(w, r)
	})})
	defer api.Close()
	s.Scheduler = &worker.LocalScheduler{ClientURL: api.URL, Log: io.Discard}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.Reconcile(ctx) }()
	defer func() { _, _ = s.StopGroup("owned-ui-test", false); cancel(); <-done }()
	deadline, stop := context.WithTimeout(context.Background(), 90*time.Second)
	defer stop()
	cmd := exec.CommandContext(deadline, "node", "scripts/verify-ui-activity.mjs")
	if onboarding {
		cmd.Args = append(cmd.Args, "--onboarding")
	}
	if findings {
		cmd.Args = append(cmd.Args, "--findings")
	}
	if retest {
		cmd.Args = append(cmd.Args, "--retest")
	}
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(), "TSUNAMI_TEST_ORIGIN="+api.URL, "TSUNAMI_TEST_TOKEN="+operator)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("local browser check failed: %v\n%s", err, output)
	}
	t.Log(strings.TrimSpace(string(output)))
	if onboarding {
		if requests.Load() != 11 || probes.Load() != 1 || unexpected.Load() != 0 {
			t.Fatalf("onboarding target observations: requests=%d probes=%d unexpected=%d", requests.Load(), probes.Load(), unexpected.Load())
		}
	} else if retest {
		if requests.Load() != 14 || probes.Load() != 2 || unexpected.Load() != 0 {
			t.Fatalf("retest target observations: requests=%d probes=%d unexpected=%d", requests.Load(), probes.Load(), unexpected.Load())
		}
	} else if findings {
		if requests.Load() != 35 || probes.Load() != 5 || unexpected.Load() != 0 {
			t.Fatalf("finding inspection replayed or changed target requests: requests=%d probes=%d unexpected=%d", requests.Load(), probes.Load(), unexpected.Load())
		}
	} else if held.Load() != 3 || cancelled.Load() != 3 || unexpected.Load() != 0 {
		t.Fatalf("owned target observations: held=%d cancelled=%d unexpected=%d", held.Load(), cancelled.Load(), unexpected.Load())
	}
	runs, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	wantRuns := 5
	if onboarding {
		wantRuns = 1
	}
	if retest {
		wantRuns = 2
	}
	if len(runs) != wantRuns {
		t.Fatalf("expected exactly %d explicit attempts, got %d", wantRuns, len(runs))
	}
	for _, run := range runs {
		if run.State != platform.Completed && run.State != platform.Stopped {
			t.Fatal("browser left active work")
		}
		if onboarding && (run.Executor != toolcontract.CurlHTTP || !run.ProcedureValid() || run.Result == nil || run.Result.Verdict != "confirmed") {
			t.Fatal("onboarded attempt did not use the validated plan and reviewed curl executor")
		}
	}
}
