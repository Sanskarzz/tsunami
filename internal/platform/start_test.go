package platform

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

func startBody(in StartInput) string { data, _ := json.Marshal(in); return string(data) }

func startCounts(t *testing.T, store *Store) [3]int {
	t.Helper()
	var counts [3]int
	if err := store.db.View(func(tx *bolt.Tx) error {
		for i, name := range []string{"runs", "credentials", "engagement_starts"} {
			counts[i] = tx.Bucket([]byte(name)).Stats().KeyN
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return counts
}

func TestStartRequiresValidActionKey(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	approveEngagement(t, s, e)
	for _, key := range []string{"", "short", strings.Repeat("A", 32), strings.Repeat("g", 32), strings.Repeat("a", 33)} {
		in := StartInput{IdempotencyKey: key, Assessment: e.Items[0].Procedure}
		if w := request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", startBody(in)); w.Code != 400 {
			t.Fatal("invalid Start key accepted")
		}
	}
	if startCounts(t, store) != [3]int{} {
		t.Fatal("invalid Start left durable work or action state")
	}
}

func TestConcurrentStartRecoveryReservesOneRun(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	approveEngagement(t, s, e)
	in := StartInput{IdempotencyKey: NewStartKey(), Assessment: e.Items[0].Procedure, Execution: "scripted"}
	type answer struct {
		run Run
		err error
	}
	answers := make(chan answer, 24)
	var wg sync.WaitGroup
	for range 24 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := store.StartEngagementRun("a", e.ID, in, nil)
			answers <- answer{r, err}
		}()
	}
	wg.Wait()
	close(answers)
	var id string
	for a := range answers {
		if a.err != nil {
			t.Fatal(a.err)
		}
		if id != "" && id != a.run.ID {
			t.Fatal("same action reserved multiple runs")
		}
		if a.run.AllowJobCreation {
			t.Fatal("Start response granted dispatch capability")
		}
		id = a.run.ID
	}
	stored, _ := store.GetEngagement("a", e.ID)
	if len(stored.RunIDs) != 1 || stored.RunIDs[0] != id || startCounts(t, store) != [3]int{1, 1, 1} {
		t.Fatal("concurrent retries consumed extra run, credential or action reservations")
	}
	// An omitted execution mode and explicit scripted mode are the same intent.
	in.Execution = ""
	if w := request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", startBody(in)); w.Code != 201 {
		t.Fatal("normalized same-action recovery failed")
	}
	for _, changed := range []StartInput{
		{IdempotencyKey: in.IdempotencyKey, Assessment: "role-boundary/v1"},
		{IdempotencyKey: in.IdempotencyKey, Assessment: in.Assessment, Execution: "live"},
		{IdempotencyKey: in.IdempotencyKey, Assessment: in.Assessment, RequiredProvider: "fixture"},
		{IdempotencyKey: in.IdempotencyKey, Assessment: in.Assessment, RequiredPricePolicy: "changed"},
	} {
		if w := request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", startBody(changed)); w.Code != 409 {
			t.Fatal("key reused for a different request")
		}
	}
	if startCounts(t, store) != [3]int{1, 1, 1} {
		t.Fatal("conflicting intent left durable work")
	}
}

func TestStartRecoveryAfterStopRestartAndRouteFailure(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	approveEngagement(t, s, e)
	in := StartInput{IdempotencyKey: NewStartKey(), Assessment: e.Items[0].Procedure}
	r, err := store.StartEngagementRun("a", e.ID, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	credential, _ := store.Credential(r.ID)
	if err := store.Update(r.ID, func(r *Run) error { r.RequestCount = 2; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StopEngagement("a", e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StopGroup("a", false); err != nil {
		t.Fatal(err)
	}
	path := store.db.Path()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted := NewServer(reopened, s.Targets, s.AdminTokens)
	restarted.Quiesce() // New execution is unavailable, but recovery is read-only.
	w := request(restarted, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", startBody(in))
	var recovered Run
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &recovered) != nil {
		t.Fatalf("recovery: %d", w.Code)
	}
	if recovered.ID != r.ID || recovered.State != Stopped || recovered.RequestCount != 2 || !recovered.Deadline.Equal(r.Deadline) || recovered.AllowJobCreation {
		t.Fatal("recovery replaced/reactivated execution or refunded its limits")
	}
	after, _ := reopened.Credential(r.ID)
	if after != credential || startCounts(t, reopened) != [3]int{1, 1, 1} {
		t.Fatal("recovery minted work or a credential")
	}
	in.IdempotencyKey = NewStartKey()
	if _, err := reopened.StartEngagementRun("a", e.ID, in, nil); err == nil {
		t.Fatal("new action bypassed durable revocation")
	}
}

func TestStartKeysAreTenantAndEngagementBound(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	approveEngagement(t, s, e)
	in := StartInput{IdempotencyKey: NewStartKey(), Assessment: e.Items[0].Procedure}
	r, err := store.StartEngagementRun("a", e.ID, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	token, _ := store.Credential(r.ID)
	for _, credential := range []string{"operator-b", token} {
		w := request(s, "POST", "/v1/engagements/"+e.ID+"/runs", credential, startBody(in))
		if w.Code != 401 && w.Code != 404 {
			t.Fatal("action key bypassed tenant/operator authorization")
		}
	}
	other := engagementFixture(t, s, in.Assessment)
	approveEngagement(t, s, other)
	r2, err := store.StartEngagementRun("a", other.ID, in, nil)
	if err != nil || r2.ID == r.ID || r2.EngagementID != other.ID {
		t.Fatal("action key recovered another engagement's work")
	}
}

func TestFailedStartDoesNotConsumeActionKey(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	in := StartInput{IdempotencyKey: NewStartKey(), Assessment: e.Items[0].Procedure}
	if _, err := store.StartEngagementRun("a", e.ID, in, nil); err == nil {
		t.Fatal("draft started")
	}
	if startCounts(t, store) != [3]int{} {
		t.Fatal("failed admission partially persisted a reservation")
	}
	approveEngagement(t, s, e)
	if _, err := store.StartEngagementRun("a", e.ID, in, nil); err != nil {
		t.Fatal(err)
	}
	if startCounts(t, store) != [3]int{1, 1, 1} {
		t.Fatal("failed admission consumed the action identity")
	}
}

func TestStartRecoveryAfterScopeExpiry(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	e.ExpiresAt = time.Now().UTC().Add(time.Second)
	e.ScopeDigest = e.digest()
	if err := store.db.Update(func(tx *bolt.Tx) error { return putEngagement(tx, e) }); err != nil {
		t.Fatal(err)
	}
	approveEngagement(t, s, e)
	in := StartInput{IdempotencyKey: NewStartKey(), Assessment: e.Items[0].Procedure}
	r, err := store.StartEngagementRun("a", e.ID, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Until(e.ExpiresAt) + time.Millisecond)
	if recovered, err := store.StartEngagementRun("a", e.ID, in, nil); err != nil || recovered.ID != r.ID || !recovered.Deadline.Equal(r.Deadline) {
		t.Fatal("expired permission prevented read-only recovery or changed deadline")
	}
	in.IdempotencyKey = NewStartKey()
	if _, err := store.StartEngagementRun("a", e.ID, in, nil); err == nil {
		t.Fatal("new action bypassed expiry")
	}
}

func TestStartRecoveryResponseDoesNotLoadEvidence(t *testing.T) {
	s, store := fixture(t)
	e := engagementFixture(t, s, "opaque-ownership/v1")
	approveEngagement(t, s, e)
	in := StartInput{IdempotencyKey: NewStartKey(), Assessment: e.Items[0].Procedure}
	r, err := store.StartEngagementRun("a", e.ID, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	const private = "synthetic-private-capture-canary"
	if err := store.Update(r.ID, func(r *Run) error {
		r.Captures = append(r.Captures, Capture{ID: "capture", Body: private})
		r.Events = append(r.Events, Event{Detail: private})
		r.Result = &Result{Reason: private}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	stored, _ := store.Get(r.ID)
	before, _ := json.Marshal(stored)
	w := request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", startBody(in))
	if w.Code != 201 || strings.Contains(w.Body.String(), private) {
		t.Fatal("ordinary recovery loaded private evidence")
	}
	var payload map[string]json.RawMessage
	if json.Unmarshal(w.Body.Bytes(), &payload) != nil {
		t.Fatal("invalid recovery response")
	}
	for _, key := range []string{"captures", "events", "result", "activity", "model", "lifecycle"} {
		if _, ok := payload[key]; ok {
			t.Fatal("recovery returned non-launch data: " + key)
		}
	}
	stored, _ = store.Get(r.ID)
	after, _ := json.Marshal(stored)
	if !bytes.Equal(before, after) {
		t.Fatal("recovery changed stored evidence or lifecycle state")
	}
}

func TestCorruptStartRecoveryFailsClosed(t *testing.T) {
	for _, kind := range []string{"invalid-json", "foreign-child", "changed-intent"} {
		t.Run(kind, func(t *testing.T) {
			s, store := fixture(t)
			e := engagementFixture(t, s, "opaque-ownership/v1")
			approveEngagement(t, s, e)
			in := StartInput{IdempotencyKey: NewStartKey(), Assessment: e.Items[0].Procedure}
			r, err := store.StartEngagementRun("a", e.ID, in, nil)
			if err != nil {
				t.Fatal(err)
			}
			record := startRecord{RequestDigest: startDigest(e, StartInput{Assessment: in.Assessment, Execution: "scripted"}), RunID: r.ID}
			if kind == "foreign-child" {
				record.RunID = running(t, s, "b").ID
			}
			if kind == "changed-intent" {
				record.RequestDigest = strings.Repeat("0", 64)
			}
			data, _ := json.Marshal(record)
			if kind == "invalid-json" {
				data = []byte("invalid-json")
			}
			if err := store.db.Update(func(tx *bolt.Tx) error {
				return tx.Bucket([]byte("engagement_starts")).Put(startKey("a", e.ID, in.IdempotencyKey), data)
			}); err != nil {
				t.Fatal(err)
			}
			before := startCounts(t, store)
			if w := request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", startBody(in)); w.Code != 409 {
				t.Fatal("corrupt action state accepted or replaced")
			}
			if startCounts(t, store) != before {
				t.Fatal("corrupt recovery allocated replacement work")
			}
		})
	}
}

func TestStartStopRaceNeverRestoresPermission(t *testing.T) {
	for _, kind := range []string{"engagement", "tenant", "global"} {
		t.Run(kind, func(t *testing.T) {
			for range 8 {
				s, store := fixture(t)
				e := engagementFixture(t, s, "opaque-ownership/v1")
				approveEngagement(t, s, e)
				in := StartInput{IdempotencyKey: NewStartKey(), Assessment: e.Items[0].Procedure}
				ready, done := make(chan struct{}), make(chan struct{})
				go func() { defer close(done); <-ready; _, _ = store.StartEngagementRun("a", e.ID, in, nil) }()
				close(ready)
				var err error
				if kind == "engagement" {
					_, err = s.StopEngagement("a", e.ID)
				} else {
					_, err = s.StopGroup("a", kind == "global")
				}
				if err != nil {
					t.Fatal(err)
				}
				<-done
				r, err := store.StartEngagementRun("a", e.ID, in, nil)
				if err == nil && (r.State != Stopped || r.AllowJobCreation) {
					t.Fatal("same-key recovery reopened stopped work")
				}
				counts := startCounts(t, store)
				if counts != [3]int{} && counts != [3]int{1, 1, 1} {
					t.Fatal("Stop race left a partial or duplicate reservation")
				}
			}
		})
	}
}
