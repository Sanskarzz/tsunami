package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

// A failure in auxiliary onboarding persistence must not leave already-revoked
// assessment traffic running. The API must still report the partial failure.
func TestGroupStopCancelsRequestsDespiteSetupFailure(t *testing.T) {
	for _, global := range []bool{false, true} {
		name := "tenant"
		if global {
			name = "global"
		}
		t.Run(name, func(t *testing.T) {
			s, store := fixture(t)
			started, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			target := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				close(started)
				select {
				case <-r.Context().Done():
					close(cancelled)
				case <-release:
				}
			}))
			defer target.Close()
			defer close(release)
			approved, err := LabTarget(context.Background(), target.URL)
			if err != nil {
				t.Fatal(err)
			}
			s.Targets["vulnerable"] = approved
			run := running(t, s, "a")
			other := running(t, s, "b")
			otherCtx, cancel, op, err := s.begin(other.ID)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { cancel(); s.finishOperation(other.ID, op) }()
			done := make(chan error, 1)
			go func() {
				_, err := s.Execute(context.Background(), run.ID, Request{URL: target.URL + "/api/slow", Account: "alice"})
				done <- err
			}()
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("target did not receive request")
			}
			// Another tenant's corrupt setup currently fails the entire setup scan.
			if err := store.db.Update(func(tx *bolt.Tx) error {
				return tx.Bucket([]byte("onboarding")).Put(setupKey("b", "0123456789abcdef0123456789abcdef"), []byte("invalid-json"))
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.StopGroup("a", global); err == nil {
				t.Fatal("onboarding persistence failure was hidden")
			}
			stored, err := store.Get(run.ID)
			if err != nil || stored.State != Stopped {
				t.Fatal("assessment revocation was not persisted")
			}
			if _, err := store.Create("a", "vulnerable", target.URL, "test"); err == nil {
				t.Fatal("durable gate admitted another run")
			}
			select {
			case <-cancelled:
			case <-time.After(time.Second):
				t.Fatal("durably revoked request remained active after setup failure")
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("interrupted request succeeded")
				}
			case <-time.After(time.Second):
				t.Fatal("broker did not drain")
			}
			if _, err := s.Execute(context.Background(), run.ID, Request{URL: target.URL + "/api/slow", Account: "alice"}); err == nil {
				t.Fatal("revoked run admitted another request")
			}
			if global != (otherCtx.Err() != nil) {
				t.Fatal("cancellation did not respect tenant/global boundary")
			}
		})
	}
}
