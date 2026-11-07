package journey

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/worker"
)

type lab struct {
	server       *platform.Server
	store        *platform.Store
	api          *httptest.Server
	credential   string
	requests     atomic.Int32
	fullRunReads atomic.Int32
	startPosts   atomic.Int32
	loseStart    atomic.Bool
	held, ended  chan struct{}
}

func ownedLab(t *testing.T, mode string) *lab {
	t.Helper()
	l := &lab{held: make(chan struct{}), ended: make(chan struct{})}
	keys := map[string]string{"owner": strings.Repeat("a", 48), "peer": strings.Repeat("b", 48)}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.requests.Add(1)
		alias := ""
		for who, key := range keys {
			if r.Header.Get("Authorization") == "Bearer "+key {
				alias = who
			}
		}
		if alias == "" || (mode == "expired" && alias == "peer") {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path == "/me" {
			fmt.Fprintf(w, `{"name":%q}`, alias)
			return
		}
		owner := strings.TrimPrefix(r.URL.Path, "/records/")
		if owner != "owner" && owner != "peer" {
			w.WriteHeader(404)
			return
		}
		if owner == "owner" && alias == "peer" {
			if mode == "held" || mode == "property-held" {
				close(l.held)
				<-r.Context().Done()
				close(l.ended)
				return
			}
			if mode == "fixed" {
				w.WriteHeader(403)
				return
			}
			if mode == "property-filtered" {
				fmt.Fprintf(w, `{"id":%q,"owner":%q}`, owner, owner)
				return
			}
			if mode == "property-ambiguous" {
				fmt.Fprintf(w, `{"id":%q,"owner":%q,"proof":"[redacted]"}`, owner, owner)
				return
			}
		}
		fmt.Fprintf(w, `{"id":%q,"owner":%q,"proof":%q}`, owner, owner, "private-proof-"+owner)
	}))
	store, err := platform.OpenStore(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	l.store = store
	t.Cleanup(func() { _ = store.Close() })
	t.Cleanup(target.Close)
	op := strings.Repeat("c", 48)
	l.server = platform.NewServer(store, nil, map[string]string{op: "owned-a", strings.Repeat("d", 48): "other-tenant"})
	object := func(alias string) application.Read {
		sum := sha256.Sum256([]byte("private-proof-" + alias))
		return application.Read{Account: alias, Path: "/records/" + alias, Equals: map[string]string{"/id": alias, "/owner": alias}, Hashes: map[string]string{"/proof": hex.EncodeToString(sum[:])}}
	}
	owner, peer := object("owner"), object("peer")
	probe := owner
	probe.Account = "peer"
	p := application.Plan{Version: "http-object-access/v2", ID: "owned-api", Title: "Owned cross-user access test", Origin: target.URL, ApprovedAddresses: []string{"127.0.0.1"}, SyntheticDataOnly: true, OwnerPointer: "/owner", Identity: []application.Read{{Account: "owner", Path: "/me", Equals: map[string]string{"/name": "owner"}}, {Account: "peer", Path: "/me", Equals: map[string]string{"/name": "peer"}}}, Baselines: []application.Read{owner, peer}, Probe: probe, ExpectedAccess: "deny", DenialStatuses: []int{401, 403, 404}}
	if strings.HasPrefix(mode, "property-") {
		p.Version = application.PropertyAccessProfile
		p.ExpectedAccess = "filtered"
		p.DenialStatuses = nil
	}
	if err := l.server.RegisterApplication(platform.ApplicationConfig{Tenant: "owned-a", Plan: p, Credentials: keys}); err != nil {
		t.Fatal(err)
	}
	l.api = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/runs/") && len(strings.Split(strings.Trim(r.URL.Path, "/"), "/")) == 3 && r.Method == "GET" {
			l.fullRunReads.Add(1)
		}
		if strings.HasSuffix(r.URL.Path, "/runs") && r.Method == "POST" {
			l.startPosts.Add(1)
			if l.loseStart.Swap(false) {
				buffer := httptest.NewRecorder()
				l.server.ServeHTTP(buffer, r) // Commit, then lose its successful response.
				if buffer.Code != 201 {
					t.Error("test failed to create the uncertain run")
				}
				w.WriteHeader(502)
				return
			}
		}
		l.server.ServeHTTP(w, r)
	}))
	t.Cleanup(l.api.Close)
	l.server.Scheduler = &worker.LocalScheduler{ClientURL: l.api.URL, OnError: func(id string, _ error) { _ = store.Fail(id) }}
	ctx, cancel := context.WithCancel(context.Background())
	reconciled := make(chan struct{})
	go func() { defer close(reconciled); l.server.Reconcile(ctx) }()
	t.Cleanup(func() { cancel(); <-reconciled })
	l.credential = filepath.Join(t.TempDir(), "operator")
	if err := os.WriteFile(l.credential, []byte(op+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return l
}

func (l *lab) invoke(ctx context.Context, args ...string) (string, error) {
	var out, diagnostic bytes.Buffer
	args = append(args, "--url", l.api.URL, "--token-file", l.credential)
	err := Run(ctx, args, &out, &diagnostic)
	return out.String(), err
}

func (l *lab) call(t *testing.T, result any, args ...string) string {
	t.Helper()
	if result != nil {
		args = append(args, "--json")
	}
	out, err := l.invoke(context.Background(), args...)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		if err := json.Unmarshal([]byte(out), result); err != nil {
			t.Fatal(err)
		}
	}
	for _, forbidden := range []string{strings.Repeat("a", 48), strings.Repeat("b", 48), strings.Repeat("c", 48), "private-proof-owner", "private-proof-peer"} {
		if strings.Contains(out, forbidden) {
			t.Fatal("CLI output exposed a credential or raw object proof")
		}
	}
	return out
}

func (l *lab) approved(t *testing.T) platform.Engagement {
	t.Helper()
	var view platform.EngagementView
	l.call(t, &view, "plan", "--application", "owned-api", "--authorization-ref", "owned-lab/test")
	e := view.Engagement
	if view.Progress.Total != 1 || view.Progress.Pending != 1 {
		t.Fatal("draft preview hid its pending procedure")
	}
	if l.requests.Load() != 0 {
		t.Fatal("draft creation contacted the target")
	}
	if _, err := l.invoke(context.Background(), "start", "--engagement", e.ID); err == nil {
		t.Fatal("draft started without approval")
	}
	if _, err := l.invoke(context.Background(), "approve", "--engagement", e.ID, "--scope-digest", strings.Repeat("0", 64)); err == nil {
		t.Fatal("approval accepted an unreviewed digest")
	}
	if l.startPosts.Load() != 0 || l.requests.Load() != 0 {
		t.Fatal("a rejected launch or approval caused work")
	}
	l.call(t, &view, "approve", "--engagement", e.ID, "--scope-digest", e.ScopeDigest)
	if view.Engagement.ApprovedDigest != e.ScopeDigest {
		t.Fatal("approved a different scope")
	}
	return view.Engagement
}

func TestConfiguredJourneyThroughMecatl(t *testing.T) {
	for _, test := range []struct{ mode, want string }{{"vulnerable", "confirmed"}, {"fixed", "no_issue_observed"}, {"expired", "inconclusive"}, {"property-leaked", "confirmed"}, {"property-filtered", "no_issue_observed"}, {"property-ambiguous", "inconclusive"}} {
		t.Run(test.mode, func(t *testing.T) {
			l := ownedLab(t, test.mode)
			l.call(t, nil, "applications")
			e := l.approved(t)
			var started struct {
				RunID string `json:"run_id"`
			}
			l.call(t, &started, "start", "--engagement", e.ID)
			l.call(t, nil, "watch", "--run", started.RunID, "--timeout", "10s")
			var status platform.RunStatus
			l.call(t, &status, "status", "--run", started.RunID)
			if status.Assessment == nil || status.Assessment.Verdict != test.want || status.ExecutionMode != "scripted" {
				t.Fatalf("wrong assessment: %+v", status)
			}
			// The worker legitimately fetches its run at startup; the CLI's
			// read-only observation path itself must use only /status.
			before := l.fullRunReads.Load()
			l.call(t, nil, "status", "--run", started.RunID)
			if l.fullRunReads.Load() != before {
				t.Fatal("status downloaded captured response bodies")
			}
			out := filepath.Join(t.TempDir(), "export")
			l.call(t, nil, "export", "--engagement", e.ID, "--out", out)
			data, err := os.ReadFile(filepath.Join(out, "bundle.json"))
			if err != nil {
				t.Fatal(err)
			}
			var bundle platform.EngagementBundle
			if err := json.Unmarshal(data, &bundle); err != nil {
				t.Fatal(err)
			}
			if len(bundle.Findings) != 0 || ((len(bundle.Candidates) == 1) != (test.want == "confirmed")) {
				t.Fatal("CLI invented or hid a review decision")
			}
			info, _ := os.Stat(out)
			file, _ := os.Stat(filepath.Join(out, "bundle.json"))
			if info.Mode().Perm() != 0700 || file.Mode().Perm() != 0600 {
				t.Fatal("export permissions are not private")
			}
			if _, err := l.invoke(context.Background(), "export", "--engagement", e.ID, "--out", out); err == nil {
				t.Fatal("export overwrote an existing directory")
			}
			unchanged, _ := os.ReadFile(filepath.Join(out, "bundle.json"))
			if !bytes.Equal(data, unchanged) {
				t.Fatal("existing evidence changed")
			}
			l.call(t, nil, "stop", "--engagement", e.ID)
			if _, err := l.invoke(context.Background(), "start", "--engagement", e.ID); err == nil {
				t.Fatal("stopped engagement launched work")
			}
			if l.startPosts.Load() != 1 {
				t.Fatal("CLI retried or launched extra work")
			}
		})
	}
}

func TestInterruptedWatchIsReadOnlyAndStopIgnoresFailedExecutionRoute(t *testing.T) {
	l := ownedLab(t, "held")
	e := l.approved(t)
	var started struct {
		RunID string `json:"run_id"`
	}
	l.call(t, &started, "start", "--engagement", e.ID)
	select {
	case <-l.held:
	case <-time.After(5 * time.Second):
		t.Fatal("target request never started")
	}
	if _, err := l.invoke(context.Background(), "watch", "--run", started.RunID, "--timeout", "1s"); err == nil || !strings.Contains(err.Error(), "work may still be active") {
		t.Fatal("watch timeout implied completion")
	}
	r, _ := l.store.Get(started.RunID)
	if r.State != platform.Running {
		t.Fatal("watch mutated run permission")
	}
	l.server.SetRouteHealth("applications", "unavailable", "application_configuration_unavailable")
	l.call(t, nil, "stop", "--engagement", e.ID)
	select {
	case <-l.ended:
	case <-time.After(2 * time.Second):
		t.Fatal("target did not observe cancellation")
	}
	var status platform.RunStatus
	l.call(t, &status, "status", "--run", started.RunID)
	if status.Permission != "revoked" {
		t.Fatal("Stop depended on failed execution admission")
	}
}

func TestUncertainStartIsNotRetriedAndChildCanBeRecovered(t *testing.T) {
	l := ownedLab(t, "held")
	e := l.approved(t)
	l.loseStart.Store(true)
	_, err := l.invoke(context.Background(), "start", "--engagement", e.ID)
	if err == nil || !strings.Contains(err.Error(), "no retry made") || !strings.Contains(err.Error(), e.ID) {
		t.Fatal("missing uncertain-start recovery instructions")
	}
	var view platform.EngagementView
	l.call(t, &view, "show", "--engagement", e.ID)
	if len(view.Engagement.RunIDs) != 1 || l.startPosts.Load() != 1 {
		t.Fatal("lost response created duplicate work or hid the saved child")
	}
	l.call(t, nil, "engagements")
	l.call(t, nil, "stop", "--engagement", e.ID)
}

func TestCredentialTransportAndInputsFailBeforeNetwork(t *testing.T) {
	for _, origin := range []string{"http://remote.example", "http://169.254.169.254", "https://user:password@example.test", "https://example.test/path", "http://127.0.0.1?token=x", "ftp://127.0.0.1"} {
		if validateOrigin(origin) == nil {
			t.Fatalf("unsafe operator URL accepted: %q", origin)
		}
	}
	for _, origin := range []string{"http://127.0.0.1:8080", "http://[::1]:8080", "http://localhost:8080", "https://control.example.test"} {
		if err := validateOrigin(origin); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte(strings.Repeat("e", 48)), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := credential(file); err == nil {
		t.Fatal("world-readable operator credential accepted")
	}
	for _, args := range [][]string{{"stop"}, {"stop", "--run", "../other"}, {"approve", "--engagement", strings.Repeat("a", 32)}, {"plan", "--execution", "live"}} {
		if Run(context.Background(), args, io.Discard, io.Discard) == nil {
			t.Fatal("invalid or unsupported CLI input accepted")
		}
	}
}

func TestOtherTenantCannotInspectOrLaunchTheSavedPlan(t *testing.T) {
	l := ownedLab(t, "fixed")
	e := l.approved(t)
	other := filepath.Join(t.TempDir(), "other-operator")
	if err := os.WriteFile(other, []byte(strings.Repeat("d", 48)), 0600); err != nil {
		t.Fatal(err)
	}
	l.credential = other
	var apps []any
	l.call(t, &apps, "applications")
	if len(apps) != 0 {
		t.Fatal("other tenant received application metadata")
	}
	for _, command := range []string{"show", "start", "stop"} {
		if _, err := l.invoke(context.Background(), command, "--engagement", e.ID); err == nil {
			t.Fatal("other tenant accessed the engagement", command)
		}
	}
	if l.startPosts.Load() != 0 || l.requests.Load() != 0 {
		t.Fatal("cross-tenant request caused execution")
	}
}

func TestPlanRefusesAnArbitraryTargetWithoutCreatingWork(t *testing.T) {
	l := ownedLab(t, "fixed")
	_, err := l.invoke(context.Background(), "plan", "--application", "https://target.example.test", "--authorization-ref", "owned-lab/test")
	if err == nil || !strings.Contains(err.Error(), "a target URL is not accepted") {
		t.Fatal("URL-only input was not clearly refused")
	}
	items, err := l.store.ListEngagements("owned-a")
	if err != nil || len(items) != 0 || l.requests.Load() != 0 {
		t.Fatal("unsupported target selection created work")
	}
}
