package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/Sanskarzz/tsunami/internal/lab"
	bolt "go.etcd.io/bbolt"
)

func engagementFixture(t *testing.T, s *Server, ids ...string) Engagement {
	t.Helper()
	if len(ids) == 0 {
		for _, d := range assessment.List() {
			ids = append(ids, d.ID)
		}
	}
	in := EngagementInput{Name: "Synthetic engagement", AuthorizationRef: "owned-lab/operator-test", Scenario: "vulnerable", Assessments: ids}
	data, _ := json.Marshal(in)
	w := request(s, "POST", "/v1/engagements", "operator-a", string(data))
	if w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var e Engagement
	if json.Unmarshal(w.Body.Bytes(), &e) != nil {
		t.Fatal("invalid response")
	}
	return e
}

func approveEngagement(t *testing.T, s *Server, e Engagement) {
	t.Helper()
	if w := request(s, "POST", "/v1/engagements/"+e.ID+"/approve", "operator-a", `{"scope_digest":"`+e.ScopeDigest+`"}`); w.Code != 200 {
		t.Fatalf("approve: %d %s", w.Code, w.Body)
	}
}

func TestEngagementAuthorizationAndTenantBoundaries(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	if _, err := store.CreateEngagementRun("a", e.ID, "opaque-ownership/v1", nil); err == nil {
		t.Fatal("draft dispatched")
	}
	r := running(t, s, "a")
	token, _ := store.Credential(r.ID)
	for _, action := range []struct{ method, path, body string }{{"GET", "", ""}, {"GET", "/export", ""}, {"POST", "/approve", `{}`}, {"POST", "/stop", `{}`}, {"POST", "/skip", `{"procedure":"opaque-ownership/v1","reason":"skip"}`}, {"POST", "/runs", `{"assessment":"opaque-ownership/v1"}`}} {
		for _, credential := range []string{"operator-b", token} {
			w := request(s, action.method, "/v1/engagements/"+e.ID+action.path, credential, action.body)
			if w.Code != 401 && w.Code != 404 {
				t.Fatalf("unauthorized access: %s %d", action.path, w.Code)
			}
		}
	}
	for _, extra := range []string{`"target":"http://unapproved.invalid"`, `"tenant":"b"`, `"authorized":true`, `"max_runs":1000`} {
		body := `{"name":"x","authorization_ref":"lab","scenario":"vulnerable","assessments":["opaque-ownership/v1"],` + extra + `}`
		if w := request(s, "POST", "/v1/engagements", "operator-a", body); w.Code != 400 {
			t.Fatalf("authority field accepted: %s", extra)
		}
	}
	approveEngagement(t, s, e)
	if _, err := store.CreateEngagementRun("a", e.ID, "role-boundary/v1", nil); err == nil {
		t.Fatal("unapproved procedure dispatched")
	}
	if w := request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", `{"assessment":"opaque-ownership/v1","execution":"live"}`); w.Code != 409 {
		t.Fatal("unconfigured model dispatched")
	}
	child, err := store.CreateEngagementRun("a", e.ID, "opaque-ownership/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if child.Tenant != "a" || child.Target != e.Target || child.EngagementID != e.ID || child.AssessmentDigest != e.Items[0].Digest {
		t.Fatal("child scope not pinned")
	}
}

func TestEngagementLimitsAreAtomicAndFailuresKeepReservations(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s)
	approveEngagement(t, s, e)
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for _, d := range assessment.List() {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if _, err := store.CreateEngagementRun("a", e.ID, id, nil); err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}(d.ID)
	}
	wg.Wait()
	if success != EngagementMaxConcurrent {
		t.Fatalf("concurrency admitted %d", success)
	}
	view, err := store.EngagementSnapshot("a", e.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range view.Engagement.RunIDs {
		if err := store.Fail(id); err != nil {
			t.Fatal(err)
		}
	}
	for i := success; i < EngagementMaxRuns; i++ {
		r, err := store.CreateEngagementRun("a", e.ID, "opaque-ownership/v1", nil)
		if err != nil {
			t.Fatal(err)
		}
		if err = store.Fail(r.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CreateEngagementRun("a", e.ID, "opaque-ownership/v1", nil); err == nil {
		t.Fatal("failed runs refunded run budget")
	}
	runs, _ := store.List()
	view, _ = store.EngagementSnapshot("a", e.ID)
	if len(runs) != EngagementMaxRuns || len(view.Engagement.RunIDs) != EngagementMaxRuns {
		t.Fatal("orphan child or missing reservation")
	}
}

func TestEngagementStopRacesCreationAndSurvivesRestart(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s)
	approveEngagement(t, s, e)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _ = store.CreateEngagementRun("a", e.ID, "opaque-ownership/v1", nil)
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		if _, err := s.StopEngagement("a", e.ID); err != nil {
			t.Error(err)
		}
	}()
	close(start)
	wg.Wait()
	view, err := store.EngagementSnapshot("a", e.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range view.Engagement.RunIDs {
		r, _ := store.Get(id)
		if r.State != Stopped {
			t.Fatal("child survived committed Stop")
		}
	}
	path := store.db.Path()
	_ = store.Close()
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err := reopened.CreateEngagementRun("a", e.ID, "opaque-ownership/v1", nil); err == nil {
		t.Fatal("restart reopened stopped engagement")
	}
	if err := reopened.ApproveEngagement("a", e.ID, e.ScopeDigest); err == nil {
		t.Fatal("stopped engagement reapproved")
	}
}

func TestEngagementStopStillWorksAfterProcedureOrTargetDrift(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s)
	approveEngagement(t, s, e)
	r, err := store.CreateEngagementRun("a", e.ID, "opaque-ownership/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Update(r.ID, func(r *Run) error {
		r.Assessment.Title = "no longer canonical"
		r.Target = "http://changed.invalid"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ExportEngagement("a", e.ID); err == nil {
		t.Fatal("changed binding exported")
	}
	w := request(s, "POST", "/v1/engagements/"+e.ID+"/stop", "operator-a", `{}`)
	if w.Code != 200 {
		t.Fatalf("invalid policy blocked Stop: %d %s", w.Code, w.Body)
	}
	r, _ = store.Get(r.ID)
	if r.State != Stopped {
		t.Fatal("drifted child remained permitted")
	}
}

func TestEngagementStopCancelsActiveTargetRequest(t *testing.T) {
	s, store := fixture(t)
	started, cancelled := make(chan struct{}), make(chan struct{})
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done(); close(cancelled) }))
	defer target.Close()
	lt, err := LabTarget(context.Background(), target.URL)
	if err != nil {
		t.Fatal(err)
	}
	s.Targets["vulnerable"] = lt
	e := engagementFixture(t, s, "opaque-ownership/v1")
	approveEngagement(t, s, e)
	r, err := store.CreateEngagementRun("a", e.ID, "opaque-ownership/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = store.Update(r.ID, func(r *Run) error { r.State = Running; return nil })
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = s.Execute(context.Background(), r.ID, Request{r.Target + r.Assessment.Baselines[0].Path, r.Assessment.Baselines[0].Account})
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request not started")
	}
	if _, err := s.StopEngagement("a", e.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("broker did not drain")
	}
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("target did not observe cancellation")
	}
	if _, err := s.Execute(context.Background(), r.ID, Request{r.Target + r.Assessment.Probe.Path, "bob"}); err == nil {
		t.Fatal("request accepted after Stop")
	}
}

func TestEngagementProgressAndEvidenceComeFromTrustedCaptures(t *testing.T) {
	s, store := fixture(t)
	app := httptest.NewServer(&lab.Server{})
	defer app.Close()
	lt, err := LabTarget(context.Background(), app.URL)
	if err != nil {
		t.Fatal(err)
	}
	s.Targets["vulnerable"] = lt
	e := engagementFixture(t, s, "opaque-ownership/v1", "shared-access/v1", "role-boundary/v1")
	approveEngagement(t, s, e)
	if w := request(s, "POST", "/v1/engagements/"+e.ID+"/skip", "operator-a", `{"procedure":"role-boundary/v1","reason":"not selected for this pass"}`); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if _, err := store.CreateEngagementRun("a", e.ID, "role-boundary/v1", nil); err == nil {
		t.Fatal("skipped procedure dispatched")
	}
	for _, id := range []string{"opaque-ownership/v1", "shared-access/v1"} {
		r, err := store.CreateEngagementRun("a", e.ID, id, nil)
		if err != nil {
			t.Fatal(err)
		}
		_ = store.Update(r.ID, func(r *Run) error { r.State = Running; return nil })
		for _, read := range r.Assessment.Reads() {
			if _, err := s.Execute(context.Background(), r.ID, Request{r.Target + read.Path, read.Account}); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := s.Finish(r.ID); err != nil {
			t.Fatal(err)
		}
	}
	bundle, err := store.ExportEngagement("a", e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.View.Progress.Completed != 2 || bundle.View.Progress.Skipped != 1 || len(bundle.Candidates) != 1 {
		t.Fatalf("incorrect trusted progress: %+v", bundle.View.Progress)
	}
	if err := store.SkipEngagementItem("a", e.ID, "opaque-ownership/v1", "hide attempted case"); err == nil {
		t.Fatal("attempt relabeled skipped")
	}
	data, _ := json.MarshalIndent(bundle, "", "  ")
	var roundtrip EngagementBundle
	if json.Unmarshal(data, &roundtrip) != nil {
		t.Fatal("invalid export")
	}
	for _, artifact := range roundtrip.Artifacts {
		sum := sha256.Sum256(artifact.Data)
		if hex.EncodeToString(sum[:]) != artifact.SHA256 || !regexp.MustCompile(`^evidence/[0-9a-f]{64}\.json$`).MatchString(artifact.Path) {
			t.Fatal("artifact bytes/path changed on export")
		}
	}
	for _, runID := range bundle.View.Engagement.RunIDs {
		token, _ := store.Credential(runID)
		if strings.Contains(string(data), token) {
			t.Fatal("run credential exported")
		}
	}
	if strings.Contains(string(data), `"cvss_score"`) {
		t.Fatal("severity invented for unrated candidate")
	}
	if _, err := store.ExportEngagement("b", e.ID); err == nil {
		t.Fatal("cross-tenant evidence exported")
	}
	if w := request(s, "POST", "/v1/engagements/"+e.ID+"/progress", "operator-a", `{"completed":3}`); w.Code != 404 {
		t.Fatal("caller wrote progress")
	}
}

func TestEngagementRunDeadlineAndTenantGate(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	expiry := time.Now().UTC().Add(30 * time.Second)
	// Trusted test setup models a shorter approved window; the public API
	// cannot rewrite expiry, limits or the approved digest.
	if err := store.db.Update(func(tx *bolt.Tx) error {
		stored, err := getEngagement(tx, "a", e.ID)
		if err != nil {
			return err
		}
		stored.ExpiresAt = expiry
		stored.ScopeDigest = stored.digest()
		return putEngagement(tx, stored)
	}); err != nil {
		t.Fatal(err)
	}
	// Review the shortened window's new digest before approving it.
	reviewed, err := store.GetEngagement("a", e.ID)
	if err != nil {
		t.Fatal(err)
	}
	approveEngagement(t, s, reviewed)
	r, err := store.CreateEngagementRun("a", e.ID, "opaque-ownership/v1", nil)
	if err != nil || !r.Deadline.Equal(expiry) {
		t.Fatal("child outlived approved window")
	}
	if _, err := s.StopGroup("a", false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateEngagementRun("a", e.ID, "opaque-ownership/v1", nil); err == nil {
		t.Fatal("tenant gate bypassed")
	}
	bundle, err := store.ExportEngagement("a", e.ID)
	if err != nil || bundle.ToolkitEngagement["authorized"] != false {
		t.Fatal("revoked authority exported as permitted")
	}
}

func TestEngagementModelPolicyAndPrivateRouteCannotBeBypassed(t *testing.T) {
	s, store := fixture(t)
	var err error
	s.Inference, err = inference.NewConfig("openrouter-free", "nvidia/nemotron-3.5-lightning:free", inference.OpenRouterBaseURL, "synthetic-unused-key")
	if err != nil {
		t.Fatal(err)
	}
	e := engagementFixture(t, s, "opaque-ownership/v1")
	approveEngagement(t, s, e)
	base := "/v1/engagements/" + e.ID + "/runs"
	for _, body := range []string{
		`{"assessment":"opaque-ownership/v1","execution":"live"}`,
		`{"assessment":"opaque-ownership/v1","execution":"live","required_provider":"fixture"}`,
		`{"assessment":"opaque-ownership/v1","execution":"live","required_provider":"openrouter-free"}`,
	} {
		if w := request(s, "POST", base, "operator-a", body); w.Code != 409 {
			t.Fatal("free-only policy bypassed")
		}
	}
	runs, _ := store.List()
	if len(runs) != 0 {
		t.Fatal("denied model attempt created an orphan")
	}
	body, _ := json.Marshal(map[string]string{"assessment": "opaque-ownership/v1", "execution": "live", "required_provider": "openrouter-free", "required_price_policy": inference.FreePolicyID})
	if w := request(s, "POST", base, "operator-a", string(body)); w.Code != 201 {
		t.Fatalf("explicit free policy refused: %s", w.Body)
	}
	// No reconciler or worker is started: this tests admission, never inference.
	private := s.Targets["vulnerable"]
	private.Forward = func(context.Context, Run, *http.Request) (*http.Response, string, error) {
		panic("admission must not open a private connection")
	}
	s.PrivateTargets = map[string]map[string]Target{"a": {"vulnerable": private}}
	in := EngagementInput{Name: "Private lab", AuthorizationRef: "owned-lab/private", Scenario: "vulnerable", Connectivity: "private", Assessments: []string{"opaque-ownership/v1"}}
	data, _ := json.Marshal(in)
	if w := request(s, "POST", "/v1/engagements", "operator-b", string(data)); w.Code != 400 {
		t.Fatal("other customer acquired a private route")
	}
	w := request(s, "POST", "/v1/engagements", "operator-a", string(data))
	if w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	var pe Engagement
	_ = json.Unmarshal(w.Body.Bytes(), &pe)
	approveEngagement(t, s, pe)
	if w := request(s, "POST", "/v1/engagements/"+pe.ID+"/runs", "operator-a", string(body)); w.Code != 409 {
		t.Fatal("private model execution silently enabled")
	}
}
