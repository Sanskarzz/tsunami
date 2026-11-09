package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/toolcontract"
)

func activityPage(t *testing.T, s *Server, id string, after uint64, limit int) ActivityPage {
	t.Helper()
	w := request(s, "GET", fmt.Sprintf("/v1/runs/%s/events?after=%d&limit=%d", id, after, limit), "operator-a", "")
	var page ActivityPage
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil {
		t.Fatalf("activity: %d %s", w.Code, w.Body)
	}
	return page
}

func activityKinds(page ActivityPage) []string {
	out := []string{}
	for _, e := range page.Items {
		out = append(out, e.Kind)
	}
	return out
}

func TestActivityRealApplicationEvidenceAndSafePagination(t *testing.T) {
	for _, tc := range []struct {
		name, verdict              string
		property, fixed, ambiguous bool
	}{
		{"object_leak", "confirmed", false, false, false},
		{"object_fixed", "no_issue_observed", false, true, false},
		{"property_leak", "confirmed", true, false, false},
		{"property_filtered", "no_issue_observed", true, true, false},
		{"property_ambiguous", "inconclusive", true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, store, config, flags := applicationFixture(t)
			if tc.property {
				config.Plan.ID = "property-activity-api"
				config.Plan.Version = application.PropertyAccessProfile
				config.Plan.ExpectedAccess = "filtered"
				config.Plan.DenialStatuses = nil
				if err := s.RegisterApplication(config); err != nil {
					t.Fatal(err)
				}
			}
			flags.property.Store(tc.property)
			flags.fixed.Store(tc.fixed)
			flags.ambiguousProperty.Store(tc.ambiguous)
			e, run := applicationEngagement(t, s, config)
			executeAppReads(t, s, run, run.Application.Reads())
			result, err := s.Finish(run.ID)
			if err != nil || result.Verdict != tc.verdict {
				t.Fatalf("validation: %+v %v", result, err)
			}
			before, _ := store.Get(run.ID)
			page := activityPage(t, s, run.ID, 0, MaxActivityPage)
			if page.HasMore || page.Gap || !page.TrackedFromCreation || page.NextCursor != page.LatestSequence {
				t.Fatalf("invalid page: %+v", page)
			}
			if len(page.Items) != 23 {
				t.Fatalf("expected creation + 7*(admission,send,capture) + validation, got %v", activityKinds(page))
			}
			for i, event := range page.Items {
				if event.Sequence != uint64(i+1) || event.SchemaVersion != 1 || event.RecordedAt.IsZero() || event.RunID != run.ID || event.EngagementID != e.ID || event.ProcedureDigest != run.ApplicationDigest {
					t.Fatalf("bad envelope: %+v", event)
				}
				raw, _ := json.Marshal(event)
				if len(raw) > MaxActivityEventBytes {
					t.Fatal("oversized event")
				}
			}
			first, last := page.Items[0], page.Items[len(page.Items)-1]
			if first.Kind != "run.created" || first.Data.RequestLimit != run.MaxRequests || !first.Data.Deadline.Equal(run.Deadline) || first.Data.Executor != "native" {
				t.Fatalf("creation: %+v", first)
			}
			if last.Kind != "validation.completed" || last.Source != "validator" || last.Data.Verdict != tc.verdict || last.Data.Validation.Required != result.Validation.BaselinesRequired || last.Data.Validation.Verified != result.Validation.BaselinesVerified || !reflect.DeepEqual(last.Data.EvidenceIDs, result.EvidenceIDs) {
				t.Fatalf("untrusted validation projection: %+v", last)
			}
			for i := 1; i < len(page.Items)-1; i += 3 {
				admitted, sent, captured := page.Items[i], page.Items[i+1], page.Items[i+2]
				if admitted.Kind != "request.admitted" || admitted.Data.Reservation != (i-1)/3+1 || admitted.Data.ReadIndex < 1 || admitted.Data.AccountSlot < 1 || admitted.Source != "broker" {
					t.Fatalf("admission: %+v", admitted)
				}
				if sent.Kind != "request.sent" || sent.Data.ObservedAt == nil || captured.Kind != "request.capture_persisted" || captured.Data.Outcome != "response_captured" || captured.Data.Status == 0 || captured.CaptureID != sent.CaptureID || captured.CaptureID != admitted.CaptureID || admitted.OperationID == "" || admitted.OperationID != sent.OperationID || sent.OperationID != captured.OperationID {
					t.Fatalf("bad request correlation: %+v %+v %+v", admitted, sent, captured)
				}
			}
			// Legacy text/raw evidence remain explicitly inspectable elsewhere, never in this projection.
			if err = store.Update(run.ID, func(r *Run) error {
				r.Events = append(r.Events, Event{time.Now(), "legacy", "secret-legacy-error"})
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			body := request(s, "GET", "/v1/runs/"+run.ID+"/events", "operator-a", "").Body.String()
			credential, _ := store.Credential(run.ID)
			for _, secret := range []string{config.Credentials["owner"], config.Credentials["peer"], credential, "opaque-proof-one", "opaque-proof-two", "secret-legacy-error", config.Plan.Origin, "/objects/", "Authorization", "private"} {
				if strings.Contains(body, secret) {
					t.Fatalf("activity leaked %q", secret)
				}
			}
			all := []ActivityEvent{}
			cursor := uint64(0)
			for {
				part := activityPage(t, s, run.ID, cursor, 4)
				all = append(all, part.Items...)
				cursor = part.NextCursor
				if !part.HasMore {
					break
				}
			}
			if !reflect.DeepEqual(all, page.Items) {
				t.Fatal("pagination skipped/repeated/reordered events")
			}
			empty := activityPage(t, s, run.ID, cursor, 4)
			if len(empty.Items) != 0 || empty.NextCursor != cursor || empty.HasMore {
				t.Fatal("invalid empty polling page")
			}
			after, _ := store.Get(run.ID)
			if !reflect.DeepEqual(before.Activity, after.Activity) || flags.requests.Load() != 7 {
				t.Fatal("reads mutated history or contacted target")
			}
		})
	}
}

func TestActivityDeniesEscapeBeforeDNSWithoutEchoingInput(t *testing.T) {
	s, store := fixture(t)
	resolver := &fakeResolver{ips: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}
	target := s.Targets["vulnerable"]
	target.Policy.resolver = resolver
	run := running(t, s, "a")
	for _, raw := range []string{"http://unapproved.invalid/secret-url-canary", run.Target + "/api/documents/doc-alice?secret=secret-url-canary"} {
		if _, err := s.Execute(context.Background(), run.ID, Request{raw, "secret-account-canary"}); err == nil {
			t.Fatal("scope escape accepted")
		}
	}
	page := activityPage(t, s, run.ID, 0, 50)
	if !reflect.DeepEqual(activityKinds(page), []string{"run.created", "request.rejected", "request.capture_persisted", "request.rejected", "request.capture_persisted"}) {
		t.Fatalf("denial claimed networking: %v", activityKinds(page))
	}
	for _, event := range page.Items {
		if event.Kind == "request.rejected" && (event.Source != "broker" || event.Data.Stage != "scope" || event.Data.Code != "scope_not_approved" || event.OperationID == "" || event.CaptureID == "") {
			t.Fatal("missing rejection references/code")
		}
	}
	raw, _ := json.Marshal(page)
	if resolver.calls != 0 || strings.Contains(string(raw), "canary") || strings.Contains(string(raw), "unapproved.invalid") {
		t.Fatal("denial dialed or echoed unsafe arguments")
	}
	stored, _ := store.Get(run.ID)
	if stored.RequestCount != 0 || len(stored.Captures) != 2 {
		t.Fatal("denied request consumed forwarding reservation or lost capture")
	}
}

func TestActivityRedirectEscapeAndUnsafeDNSDoNotClaimSend(t *testing.T) {
	s, store := fixture(t)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://unapproved.invalid/secret-redirect-canary", http.StatusFound)
	}))
	defer target.Close()
	approved, err := LabTarget(context.Background(), target.URL)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &fakeResolver{ips: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}
	approved.Policy.resolver = resolver
	s.Targets["vulnerable"] = approved
	run := running(t, s, "a")
	if _, err = s.Execute(context.Background(), run.ID, Request{run.Target + "/api/redirect", "alice"}); err == nil {
		t.Fatal("followed scope escape")
	}
	page := activityPage(t, s, run.ID, 0, 50)
	if !reflect.DeepEqual(activityKinds(page), []string{"run.created", "request.admitted", "request.sent", "request.capture_persisted", "request.rejected", "request.capture_persisted"}) || resolver.calls != 1 {
		t.Fatalf("redirect lost hop enforcement: %v DNS=%d", activityKinds(page), resolver.calls)
	}
	raw, _ := json.Marshal(page)
	if strings.Contains(string(raw), "canary") || strings.Contains(string(raw), "unapproved.invalid") {
		t.Fatal("echoed rejected redirect")
	}
	resolver.ips = []netip.Addr{netip.MustParseAddr("169.254.169.254")}
	blocked := running(t, s, "a")
	if _, err = s.Execute(context.Background(), blocked.ID, Request{blocked.Target + "/api/documents/doc-alice", "alice"}); err == nil {
		t.Fatal("unsafe DNS accepted")
	}
	page = activityPage(t, s, blocked.ID, 0, 50)
	if !reflect.DeepEqual(activityKinds(page), []string{"run.created", "request.admitted", "request.capture_persisted"}) || page.Items[2].Data.Outcome != "failed_or_interrupted" {
		t.Fatalf("failed dial invented send: %v", activityKinds(page))
	}
	r, _ := store.Get(blocked.ID)
	if r.RequestCount != 1 {
		t.Fatal("ambiguous/failed reservation was refunded")
	}
}

func TestActivityAuthorizationAndStrictCursorBounds(t *testing.T) {
	s, store := fixture(t)
	run := running(t, s, "a")
	path := "/v1/runs/" + run.ID + "/events"
	token, _ := store.Credential(run.ID)
	for _, tc := range []struct {
		token string
		code  int
	}{{"", 401}, {token, 401}, {"operator-b", 404}} {
		if w := request(s, "GET", path, tc.token, ""); w.Code != tc.code {
			t.Fatalf("unauthorized events: %d", w.Code)
		}
	}
	if _, err := store.ReadActivity("b", run.ID, 0, 50); !errors.Is(err, ErrNotFound) {
		t.Fatal("store bypassed tenant check")
	}
	for _, query := range []string{"after=-1", "after=+1", "after=1.5", "after=2", "after=9007199254740992", "after=18446744073709551616", "after=", "limit=0", "limit=101", "limit=", "limit=1&limit=2", "after=0&after=0", "cursor=0", "after=%zz", "after=0;limit=1"} {
		if w := request(s, "GET", path+"?"+query, "operator-a", ""); w.Code != 400 {
			t.Fatalf("accepted %q: %d %s", query, w.Code, w.Body)
		}
	}
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		if w := request(s, method, path, "operator-a", `{"kind":"validation.completed","verdict":"confirmed"}`); w.Code != 405 {
			t.Fatalf("event mutation route: %d", w.Code)
		}
	}
}

func TestActivityRetentionRestartAndAtomicTransitions(t *testing.T) {
	s, store := fixture(t)
	run := running(t, s, "a")
	if err := store.Update(run.ID, func(r *Run) error {
		for i := 0; i < MaxActivityEvents+10; i++ {
			appendActivity(r, "request.rejected", "", "", ActivityData{Stage: "scope", Code: "scope_not_approved"})
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before, _ := store.Get(run.ID)
	rollback := errors.New("rollback")
	if err := store.Update(run.ID, func(r *Run) error { revoke(r, "run_stop", time.Now()); r.State = Stopped; return rollback }); !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	after, _ := store.Get(run.ID)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("event/state escaped aborted transaction")
	}
	page := activityPage(t, s, run.ID, 0, 100)
	if !page.Gap || !page.HistoryTruncated || !page.HasMore || page.OldestSequence != 12 || len(page.Items) != 100 {
		t.Fatalf("missing retention notice: %+v", page)
	}
	page = activityPage(t, s, run.ID, page.NextCursor, 100)
	if page.Gap || !page.HistoryTruncated {
		t.Fatal("caught-up consumer falsely reported cursor gap")
	}
	if err := s.Stop(run.ID); err != nil {
		t.Fatal(err)
	}
	stopped, _ := store.Get(run.ID)
	if len(stopped.Activity.Items) != MaxActivityEvents || !reflect.DeepEqual(activityKinds(ActivityPage{Items: stopped.Activity.Items[len(stopped.Activity.Items)-2:]}), []string{"permission.revoked", "activity.drained_locally"}) {
		t.Fatal("full journal blocked/lost Stop events")
	}
	if err := s.Stop(run.ID); err != nil {
		t.Fatal(err)
	}
	again, _ := store.Get(run.ID)
	if again.Activity.LastSequence != stopped.Activity.LastSequence {
		t.Fatal("repeated Stop fabricated another revocation")
	}
	path := store.db.Path()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	restarted := NewServer(reopened, s.Targets, s.AdminTokens)
	persisted := activityPage(t, restarted, run.ID, stopped.Activity.LastSequence-2, 50)
	if len(persisted.Items) != 2 || !reflect.DeepEqual(persisted.Items, stopped.Activity.Items[len(stopped.Activity.Items)-2:]) {
		t.Fatal("restart lost/changed durable Stop history")
	}
	status, err := restarted.Status(run.ID)
	if err != nil || status.Permission != "revoked" {
		t.Fatal("journal restart weakened Stop", err)
	}
}

func TestActivityLegacyAndInterruptedOperationsStayUnknown(t *testing.T) {
	s, store := fixture(t)
	legacy := running(t, s, "a")
	if err := store.Update(legacy.ID, func(r *Run) error { r.Activity = nil; r.Lifecycle = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	page := activityPage(t, s, legacy.ID, 0, 50)
	if len(page.Items) != 0 || page.TrackedFromCreation || page.LatestSequence != 0 {
		t.Fatal("invented historical events")
	}
	if err := s.Stop(legacy.ID); err != nil {
		t.Fatal(err)
	}
	page = activityPage(t, s, legacy.ID, 0, 50)
	if !reflect.DeepEqual(activityKinds(page), []string{"permission.revoked"}) || page.TrackedFromCreation {
		t.Fatal("backfilled legacy activity")
	}
	run := running(t, s, "a")
	ctx, cancel, op, err := s.begin(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	if err = s.reserveTarget(ctx, run.ID, op, randomID(), "alice", "/api/slow"); err != nil {
		t.Fatal(err)
	}
	// Simulate a process crash after a durable reservation, before observing a send/completion.
	path := store.db.Path()
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	restarted := NewServer(reopened, s.Targets, s.AdminTokens)
	if err = restarted.Stop(run.ID); err != nil {
		t.Fatal(err)
	}
	page = activityPage(t, restarted, run.ID, 0, 50)
	if !reflect.DeepEqual(activityKinds(page), []string{"run.created", "request.admitted", "permission.revoked"}) {
		t.Fatalf("fabricated observations after crash: %v", activityKinds(page))
	}
	status, err := restarted.Status(run.ID)
	if err != nil || status.UncertainOperations != 1 || status.LocalDrainObservedAt != nil || status.Activity != "uncertain_prior_or_unpersisted_completion" {
		t.Fatalf("lost crash uncertainty: %+v %v", status, err)
	}
}

func TestActivityStopDuringHeldRequestAllRevocationPaths(t *testing.T) {
	for _, kind := range []string{"run_stop", "engagement_stop", "tenant_stop", "global_stop"} {
		t.Run(kind, func(t *testing.T) {
			s, store, config, flags := applicationFixture(t)
			e, run := applicationEngagement(t, s, config)
			flags.block.Store(true)
			done := make(chan error, 1)
			go func() {
				_, err := s.Execute(context.Background(), run.ID, Request{run.Target + run.Application.Identity[0].Path, run.Application.Identity[0].Account})
				done <- err
			}()
			select {
			case <-flags.started:
			case <-time.After(2 * time.Second):
				t.Fatal("target did not receive request")
			}
			page := activityPage(t, s, run.ID, 0, 50)
			until := time.Now().Add(time.Second)
			for !slices.Contains(activityKinds(page), "request.sent") && time.Now().Before(until) {
				time.Sleep(time.Millisecond)
				page = activityPage(t, s, run.ID, 0, 50)
			}
			if !slices.Contains(activityKinds(page), "request.sent") || slices.Contains(activityKinds(page), "request.capture_persisted") {
				t.Fatal("held request completion invented or send missing")
			}
			var err error
			switch kind {
			case "run_stop":
				err = s.Stop(run.ID)
			case "engagement_stop":
				_, err = s.StopEngagement("a", e.ID)
			case "tenant_stop":
				_, err = s.StopGroup("a", false)
			case "global_stop":
				_, err = s.StopGroup("", true)
			}
			if err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("interrupted request succeeded")
				}
			case <-time.After(time.Second):
				t.Fatal("Stop did not drain")
			}
			select {
			case <-flags.cancelled:
			case <-time.After(time.Second):
				t.Fatal("target did not observe cancellation")
			}
			page = activityPage(t, s, run.ID, 0, 50)
			var revoked uint64
			for _, event := range page.Items {
				if event.Kind == "permission.revoked" {
					revoked = event.Sequence
					if event.Data.Code != kind {
						t.Fatal("wrong revocation reason")
					}
				}
				if event.Kind == "request.admitted" && revoked > 0 {
					t.Fatal("admitted after durable revocation")
				}
				if event.Kind == "request.capture_persisted" && event.Data.Outcome != "failed_or_interrupted" {
					t.Fatal("interrupted capture mislabeled")
				}
				if slices.Contains([]string{"execution.exit_observed", "cleanup.resources_absent", "validation.completed"}, event.Kind) {
					t.Fatal("Stop invented execution/validation proof")
				}
			}
			if revoked == 0 || page.Items[len(page.Items)-1].Kind != "activity.drained_locally" {
				t.Fatalf("missing durable revocation/drain: %v", activityKinds(page))
			}
			stored, _ := store.Get(run.ID)
			if stored.State != Stopped || stored.StopAcceptedAt == nil {
				t.Fatal("event not atomic with Stop")
			}
			before := page.LatestSequence
			if _, err := s.Execute(context.Background(), run.ID, Request{run.Target + run.Application.Identity[0].Path, run.Application.Identity[0].Account}); err == nil {
				t.Fatal("revoked run executed")
			}
			if activityPage(t, s, run.ID, before, 50).LatestSequence != before {
				t.Fatal("refused post-Stop execution fabricated activity")
			}
		})
	}
}

// Delayed viewers hold no Store transaction and cannot delay durable Stop.
type heldActivityWriter struct {
	*httptest.ResponseRecorder
	entered, release chan struct{}
}

func (w *heldActivityWriter) Write(data []byte) (int, error) {
	close(w.entered)
	<-w.release
	return w.ResponseRecorder.Write(data)
}

func TestActivitySlowReaderCannotBlockStop(t *testing.T) {
	s, _ := fixture(t)
	run := running(t, s, "a")
	w := &heldActivityWriter{httptest.NewRecorder(), make(chan struct{}), make(chan struct{})}
	done := make(chan struct{})
	go func() {
		defer close(done)
		r := httptest.NewRequest("GET", "/v1/runs/"+run.ID+"/events", nil)
		r.Header.Set("Authorization", "Bearer operator-a")
		s.ServeHTTP(w, r)
	}()
	defer func() { close(w.release); <-done }()
	select {
	case <-w.entered:
	case <-time.After(time.Second):
		t.Fatal("read did not reach writer")
	}
	stopped := make(chan error, 1)
	go func() { stopped <- s.Stop(run.ID) }()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("viewer blocked Stop")
	}
}

func TestActivityCreationUsesBoundPolicyAndBoundsEventSize(t *testing.T) {
	r := Run{ID: randomID(), Executor: toolcontract.CurlHTTP, Execution: "scripted", MaxRequests: 3, Deadline: time.Now().Add(time.Second)}
	activityCreated(&r)
	if r.Activity.Items[0].Data.Executor != toolcontract.CurlHTTP || r.Activity.Items[0].Data.RequestLimit != 3 {
		t.Fatal("creation substituted policy")
	}
	appendActivity(&r, "permission.revoked", "unsafe-ref", "unsafe-ref", ActivityData{Code: strings.Repeat("x", MaxActivityEventBytes)})
	e := r.Activity.Items[1]
	raw, _ := json.Marshal(e)
	if len(raw) > MaxActivityEventBytes || e.Data.Code != "details_unavailable" || e.OperationID != "" || e.CaptureID != "" {
		t.Fatal("event byte/reference bound missing")
	}
}

type activityScheduler struct {
	ensured     chan Run
	observation ExecutionObservation
	err         error
}

func (s *activityScheduler) Ensure(_ context.Context, r Run, _ string) error {
	if s.ensured != nil {
		select {
		case s.ensured <- r:
		default:
		}
	}
	return nil
}
func (s *activityScheduler) Stop(context.Context, string) error { return nil }
func (s *activityScheduler) Observe(context.Context, Run) (ExecutionObservation, error) {
	return s.observation, s.err
}

func TestActivityControllerObservationsRemainSeparate(t *testing.T) {
	s, store := fixture(t)
	run, err := store.Create("a", "vulnerable", s.Targets["vulnerable"].URL, s.Mode)
	if err != nil {
		t.Fatal(err)
	}
	scheduler := &activityScheduler{ensured: make(chan Run, 1)}
	s.Scheduler = scheduler
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.Reconcile(ctx) }()
	defer func() { cancel(); <-done }()
	select {
	case dispatched := <-scheduler.ensured:
		if !dispatched.AllowJobCreation {
			t.Fatal("missing dispatch capability")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no dispatch")
	}
	page := activityPage(t, s, run.ID, 0, 50)
	if !reflect.DeepEqual(activityKinds(page), []string{"run.created", "execution.dispatch_requested"}) {
		t.Fatalf("dispatch invented worker proof: %v", activityKinds(page))
	}
	if err = s.Stop(run.ID); err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(2 * time.Second)
	for !slices.Contains(activityKinds(page), "cleanup.delete_accepted") && time.Now().Before(until) {
		time.Sleep(5 * time.Millisecond)
		page = activityPage(t, s, run.ID, 0, 50)
	}
	if !slices.Contains(activityKinds(page), "cleanup.delete_accepted") || slices.Contains(activityKinds(page), "execution.exit_observed") || slices.Contains(activityKinds(page), "cleanup.resources_absent") {
		t.Fatal("cleanup acceptance overstated termination")
	}
}

func TestActivityExecutorFailuresAbsenceAndExitAreNotInterchangeable(t *testing.T) {
	s, store := fixture(t)
	run := running(t, s, "a")
	scheduler := &activityScheduler{err: errors.New("secret-executor-error-canary")}
	s.Scheduler = scheduler
	observe := func() {
		t.Helper()
		if err := store.Update(run.ID, func(r *Run) error { tracked(r).ObservedAt = nil; return nil }); err != nil {
			t.Fatal(err)
		}
		latest, _ := store.Get(run.ID)
		s.observeExecution(context.Background(), latest)
	}
	observe()
	if len(activityPage(t, s, run.ID, 0, 50).Items) != 1 {
		t.Fatal("failed observation invented event")
	}
	scheduler.err = nil
	scheduler.observation = ExecutionObservation{Running: true}
	observe()
	observe()
	if !reflect.DeepEqual(activityKinds(activityPage(t, s, run.ID, 0, 50)), []string{"run.created", "execution.worker_observed"}) {
		t.Fatal("worker observation missing or repeated")
	}
	scheduler.observation = ExecutionObservation{ResourcesAbsent: true}
	observe()
	if len(activityPage(t, s, run.ID, 0, 50).Items) != 2 {
		t.Fatal("absence before deletion was promoted to cleanup")
	}
	if err := store.Update(run.ID, func(r *Run) error { now := time.Now().UTC(); r.DeletionRequestedAt = &now; return nil }); err != nil {
		t.Fatal(err)
	}
	observe()
	page := activityPage(t, s, run.ID, 0, 50)
	if !reflect.DeepEqual(activityKinds(page), []string{"run.created", "execution.worker_observed", "cleanup.resources_absent"}) {
		t.Fatal("object absence fabricated exit")
	}
	// An actual observed exit is a separate fact, on a run still observable.
	other := running(t, s, "a")
	scheduler.observation = ExecutionObservation{Exited: true}
	s.observeExecution(context.Background(), other)
	if !reflect.DeepEqual(activityKinds(activityPage(t, s, other.ID, 0, 50)), []string{"run.created", "execution.exit_observed"}) {
		t.Fatal("actual exit not recorded")
	}
	raw, _ := json.Marshal(page)
	if strings.Contains(string(raw), "canary") {
		t.Fatal("observation error text leaked")
	}
}
