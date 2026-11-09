package platform

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/assessment"
	setup "github.com/Sanskarzz/tsunami/internal/onboarding"
	bolt "go.etcd.io/bbolt"
)

func metadataView(t *testing.T, s *Server, tenantToken, id string) EngagementView {
	t.Helper()
	w := request(s, "GET", "/v1/engagements/"+id, tenantToken, "")
	var view EngagementView
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &view) != nil || view.Metadata == nil {
		t.Fatalf("metadata response unavailable: %d %s", w.Code, w.Body)
	}
	return view
}

func TestMetadataUsesStoredPolicyWithoutTrafficOrScopeChanges(t *testing.T) {
	s, store, c, flags := applicationFixture(t)
	e, run := applicationEngagement(t, s, c)
	// Model an existing attempt with a stricter stored allowance/deadline.
	// Status must project that record, not today's prospective profile defaults.
	if err := store.Update(run.ID, func(r *Run) error {
		r.MaxRequests = 5
		r.Deadline = r.CreatedAt.Add(17 * time.Second)
		r.Captures = []Capture{{ID: "capture", Body: "raw-response-body-canary"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	beforeRun, _ := store.Get(run.ID)
	beforeEngagement, _ := store.GetEngagement("a", e.ID)
	w := request(s, "GET", "/v1/applications", "operator-a", "")
	var apps []ApplicationSummary
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &apps) != nil || len(apps) != 1 {
		t.Fatalf("application metadata: %d %s", w.Code, w.Body)
	}
	app := apps[0]
	if app.Metadata.Registration != "startup" || app.Metadata.Access.Availability != "configured" || app.Metadata.Access.CredentialHealth != "unknown" || app.Metadata.Access.SetupPermissionExpiresAt != nil {
		t.Fatalf("invented access health/expiry: %+v", app.Metadata)
	}
	view := metadataView(t, s, "operator-a", e.ID)
	m := view.Metadata
	if m.Permission != "active" || m.ExecutionMode != "scripted" || m.Executor != "native" || m.MaxRunDurationSeconds != int(MaxRunDuration/time.Second) || len(m.Procedures) != 1 {
		t.Fatalf("scope policy differs: %+v", m)
	}
	p := m.Procedures[0]
	if p != app.Metadata.Procedure || p.Digest != view.Engagement.Application.Digest() || p.Version != c.Plan.Version || p.MaxRequests != view.Engagement.Application.MaxRequests() || p.PlannedReads != len(view.Engagement.Application.Reads()) || !p.Supported {
		t.Fatalf("procedure metadata differs: %+v", p)
	}
	w = request(s, "GET", "/v1/runs/"+run.ID+"/status", "operator-a", "")
	var status RunStatus
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &status) != nil {
		t.Fatalf("status: %d %s", w.Code, w.Body)
	}
	if status.TargetRequestLimit != beforeRun.MaxRequests || !status.ExpiresAt.Equal(beforeRun.Deadline) || !status.CreatedAt.Equal(beforeRun.CreatedAt) || status.State != beforeRun.State || status.ProcedureID != beforeRun.ProcedureID() || status.ProcedureVersion != c.Plan.Version || status.ProcedureDigest != beforeRun.ApplicationDigest {
		t.Fatalf("status substituted policy defaults: %+v", status)
	}
	if status.ObservedAt.IsZero() || status.ExecutorObservationAttemptedAt != nil || m.ObservedAt.IsZero() || app.Metadata.ObservedAt.IsZero() {
		t.Fatal("missing snapshot time or invented executor observation")
	}
	for _, path := range []string{"/v1/applications", "/v1/engagements/" + e.ID, "/v1/runs/" + run.ID + "/status"} {
		body := request(s, "GET", path, "operator-a", "").Body.String()
		for _, forbidden := range []string{c.Credentials["owner"], c.Credentials["peer"], "raw-response-body-canary", "opaque-proof-one", "opaque-proof-two"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("sensitive value in metadata from %s", path)
			}
		}
	}
	afterRun, _ := store.Get(run.ID)
	afterEngagement, _ := store.GetEngagement("a", e.ID)
	before, _ := json.Marshal([]any{beforeRun, beforeEngagement})
	after, _ := json.Marshal([]any{afterRun, afterEngagement})
	if string(before) != string(after) || flags.requests.Load() != 0 {
		t.Fatal("metadata read changed stored scope/state or contacted the target")
	}
	bundle, err := store.ExportEngagement("a", e.ID)
	if err != nil || bundle.View.Metadata != nil {
		t.Fatal("ephemeral metadata was added to stored export contract", err)
	}
	failedAt := time.Now().UTC()
	if err := store.Update(run.ID, func(r *Run) error {
		tracked(r).ObservedAt = &failedAt
		r.Lifecycle.ObservationCode = "executor_observation_unavailable"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	status, err = s.Status(run.ID)
	if err != nil || status.ExecutorObservationAttemptedAt == nil || !status.ExecutorObservationAttemptedAt.Equal(failedAt) || status.ObservationCode != "executor_observation_unavailable" || status.WorkerObservedAt != nil || status.Execution != "unknown" {
		t.Fatal("failed executor check was promoted to a worker observation", err)
	}
}

func TestMetadataSeparatesApprovalAttemptsAndDurableGates(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	view := metadataView(t, s, "operator-a", e.ID)
	d, _ := assessment.Get("opaque-ownership/v1")
	p := view.Metadata.Procedures[0]
	if view.Metadata.Permission != "unapproved" || view.Metadata.LatestAttempt != nil || p.MaxRequests != d.MaxRequests || p.PlannedReads != len(d.Reads()) || p.Version != "v1" || p.Digest != d.Digest() {
		t.Fatalf("draft/catalog metadata differs: %+v", view.Metadata)
	}
	w := request(s, "POST", "/v1/engagements/"+e.ID+"/approve", "operator-a", `{}`)
	var approved EngagementView
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &approved) != nil || approved.Metadata.Permission != "active" {
		t.Fatal("approval response lacks current metadata")
	}
	run, err := store.CreateEngagementRun("a", e.ID, d.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Fail(run.ID); err != nil {
		t.Fatal(err)
	}
	view = metadataView(t, s, "operator-a", e.ID)
	if view.Engagement.State != EngagementActive || view.Metadata.Permission != "active" || view.Metadata.LatestAttempt.State != Failed || view.Metadata.LatestAttempt.Verdict != "inconclusive" {
		t.Fatalf("failed attempt was confused with approval: %+v", view.Metadata)
	}
	if err = store.Update(run.ID, func(r *Run) error { r.State = Completed; return nil }); err != nil {
		t.Fatal(err)
	}
	view = metadataView(t, s, "operator-a", e.ID)
	if view.Engagement.State != EngagementActive || view.Metadata.Permission != "active" || view.Metadata.LatestAttempt.State != Completed {
		t.Fatal("completed attempt was confused with approval state")
	}
	if _, err = s.StopGroup("a", false); err != nil {
		t.Fatal(err)
	}
	view = metadataView(t, s, "operator-a", e.ID)
	if view.Engagement.State != EngagementActive || view.Metadata.Permission != "disabled" {
		t.Fatal("tenant gate hidden behind stored active state")
	}
	if _, err = s.StopEngagement("a", e.ID); err != nil {
		t.Fatal(err)
	}
	if view = metadataView(t, s, "operator-a", e.ID); view.Metadata.Permission != "revoked" {
		t.Fatal("engagement revocation not reflected")
	}
}

func TestMetadataNeverTreatsLocalAccessAsVerifiedSession(t *testing.T) {
	s, store, c, flags := applicationFixture(t)
	e, run := applicationEngagement(t, s, c)
	flags.expiredPeer.Store(true)
	view := metadataView(t, s, "operator-a", e.ID)
	if view.Metadata.Access.Availability != "configured" || view.Metadata.Access.CredentialHealth != "unknown" || view.Metadata.Access.BaselinesValidatedAt != nil {
		t.Fatal("metadata inferred live identity health")
	}
	changed := NewServer(store, s.Targets, s.AdminTokens)
	c.Credentials["peer"] = strings.Repeat("changed-peer-", 5)
	if err := changed.RegisterApplication(c); err != nil {
		t.Fatal(err)
	}
	view = metadataView(t, changed, "operator-a", e.ID)
	if view.Metadata.Access.Availability != "configuration_changed" || view.Metadata.Access.CredentialHealth != "unknown" {
		t.Fatal("changed credential binding masqueraded as available approved access")
	}
	token, _ := store.Credential(run.ID)
	for _, path := range []string{"/v1/applications", "/v1/engagements/" + e.ID, "/v1/runs/" + run.ID + "/status"} {
		for _, unauthorized := range []string{"", token, "operator-b"} {
			w := request(s, "GET", path, unauthorized, "")
			if path == "/v1/applications" && unauthorized == "operator-b" {
				if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
					t.Fatal("other tenant's registry metadata leaked")
				}
			} else if !slices.Contains([]int{401, 404}, w.Code) {
				t.Fatalf("unauthorized metadata access: %s %d", path, w.Code)
			}
		}
	}
	if flags.requests.Load() != 0 {
		t.Fatal("availability check performed target validation")
	}
}

func TestMetadataKeepsUnsupportedStoredProcedureUnknown(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	if err := store.db.Update(func(tx *bolt.Tx) error {
		current, err := getEngagement(tx, "a", e.ID)
		if err != nil {
			return err
		}
		current.Items[0].Digest = strings.Repeat("0", 64)
		return putEngagement(tx, current)
	}); err != nil {
		t.Fatal(err)
	}
	m := metadataView(t, s, "operator-a", e.ID).Metadata
	if m.Permission != "unavailable" || m.Procedures[0].Supported || m.Procedures[0].MaxRequests != 0 || m.Procedures[0].Digest != strings.Repeat("0", 64) {
		t.Fatal("stored definition was replaced by current policy")
	}
}

func TestOnboardedMetadataReportsPermissionExpiryAndHistoricalValidation(t *testing.T) {
	s, details, state := setupFixture(t)
	item := validateOwnedSetup(t, s, createOwnedSetup(t, s, details))
	var prepared setup.Session
	raw := setupHTTP(t, s, "POST", "/v1/onboarding/"+item.ID+"/prepare", setupMutation{Revision: item.Revision}, 200)
	if json.Unmarshal(raw, &prepared) != nil {
		t.Fatal("invalid prepared setup")
	}
	view := metadataView(t, s, "admin-a", prepared.EngagementID)
	access := view.Metadata.Access
	if access.Availability != "configured" || access.CredentialHealth != "unknown" || access.SetupPermissionExpiresAt == nil || !access.SetupPermissionExpiresAt.Equal(item.ExpiresAt) || access.BaselinesValidatedAt == nil || !access.BaselinesValidatedAt.Equal(*item.ValidatedAt) {
		t.Fatalf("setup observation missing or promoted to credential health: %+v", access)
	}
	key := slices.Clone(s.SetupKey)
	s.SetupKey = nil
	if metadataView(t, s, "admin-a", prepared.EngagementID).Metadata.Access.Availability != "unavailable" {
		t.Fatal("missing protected access key was hidden")
	}
	s.SetupKey = key
	if err := s.Store.db.Update(func(tx *bolt.Tx) error {
		v, err := getSetup(tx, "demo-a", item.ID)
		if err != nil {
			return err
		}
		v.ExpiresAt = time.Now().Add(-time.Second)
		return putSetup(tx, v)
	}); err != nil {
		t.Fatal(err)
	}
	access = metadataView(t, s, "admin-a", prepared.EngagementID).Metadata.Access
	if access.Availability != "unavailable" || access.SetupPermissionExpiresAt.After(time.Now()) || access.CredentialHealth != "unknown" {
		t.Fatal("expired setup was presented as usable access")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.requests) != 4 {
		t.Fatal("metadata reads sent additional baseline/probe requests")
	}
}
