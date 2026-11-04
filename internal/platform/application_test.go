package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
)

type appFlags struct {
	fixed, wrongIdentity, deleted, generic atomic.Bool
	expiredPeer                            atomic.Bool
	requests                               atomic.Int32
	block                                  atomic.Bool
	started, cancelled                     chan struct{}
}

func applicationFixture(t *testing.T) (*Server, *Store, ApplicationConfig, *appFlags) {
	t.Helper()
	s, store := fixture(t)
	flags := &appFlags{started: make(chan struct{}), cancelled: make(chan struct{})}
	ownerToken := strings.Repeat("a", 48)
	peerToken := strings.Repeat("b", 48)
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flags.requests.Add(1)
		if flags.block.Swap(false) {
			close(flags.started)
			<-r.Context().Done()
			close(flags.cancelled)
			return
		}
		if flags.expiredPeer.Load() && r.Header.Get("Authorization") == "Bearer "+peerToken {
			w.WriteHeader(401)
			return
		}
		who := ""
		switch r.Header.Get("Authorization") {
		case "Bearer " + ownerToken:
			who = "first"
		case "Bearer " + peerToken:
			who = "second"
		default:
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/me" {
			if flags.wrongIdentity.Load() {
				who = "same-user"
			}
			fmt.Fprintf(w, `{"account":{"name":%q}}`, who)
			return
		}
		if r.URL.Path == "/objects/a" {
			if flags.deleted.Load() && who == "first" {
				w.WriteHeader(404)
				return
			}
			if who == "second" && flags.fixed.Load() {
				w.WriteHeader(404)
				return
			}
			if who == "second" && flags.generic.Load() {
				fmt.Fprint(w, `{"error":"generic 200"}`)
				return
			}
			fmt.Fprint(w, `{"identifier":"a","owner":"first","private":"opaque-proof-one"}`)
			return
		}
		if r.URL.Path == "/objects/b" {
			fmt.Fprint(w, `{"identifier":"b","owner":"second","private":"opaque-proof-two"}`)
			return
		}
		w.WriteHeader(404)
	}))
	t.Cleanup(app.Close)
	hash := func(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }
	owner := application.Read{Account: "owner", Path: "/objects/a", Equals: map[string]string{"/identifier": "a", "/owner": "first"}, Hashes: map[string]string{"/private": hash("opaque-proof-one")}}
	peer := application.Read{Account: "peer", Path: "/objects/b", Equals: map[string]string{"/identifier": "b", "/owner": "second"}, Hashes: map[string]string{"/private": hash("opaque-proof-two")}}
	probe := owner
	probe.Account = "peer"
	c := ApplicationConfig{Tenant: "a", Credentials: map[string]string{"owner": ownerToken, "peer": peerToken}, Plan: application.Plan{Version: "http-object-access/v2", ID: "independent-api", Title: "Configured cross-user object read", Origin: app.URL, ApprovedAddresses: []string{"127.0.0.1"}, SyntheticDataOnly: true, OwnerPointer: "/owner", Identity: []application.Read{{Account: "owner", Path: "/me", Equals: map[string]string{"/account/name": "first"}}, {Account: "peer", Path: "/me", Equals: map[string]string{"/account/name": "second"}}}, Baselines: []application.Read{owner, peer}, Probe: probe, ExpectedAccess: "deny", DenialStatuses: []int{401, 403, 404}}}
	if err := s.RegisterApplication(c); err != nil {
		t.Fatal(err)
	}
	return s, store, c, flags
}

func applicationEngagement(t *testing.T, s *Server, c ApplicationConfig) (Engagement, Run) {
	t.Helper()
	data, _ := json.Marshal(EngagementInput{Name: "Configured owned application", AuthorizationRef: "owned-lab/application-test", Application: c.Plan.ID})
	w := request(s, "POST", "/v1/engagements", "operator-a", string(data))
	if w.Code != 201 {
		t.Fatalf("engagement: %s", w.Body)
	}
	var e Engagement
	_ = json.Unmarshal(w.Body.Bytes(), &e)
	approveEngagement(t, s, e)
	w = request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", `{"assessment":"`+e.Application.ProcedureID()+`"}`)
	if w.Code != 201 {
		t.Fatalf("run: %s", w.Body)
	}
	var r Run
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	if err := s.Store.Update(r.ID, func(r *Run) error { r.State = Running; return nil }); err != nil {
		t.Fatal(err)
	}
	return e, r
}

func executeAppReads(t *testing.T, s *Server, r Run, reads []application.Read) {
	t.Helper()
	for _, read := range reads {
		if _, err := s.Execute(context.Background(), r.ID, Request{r.Target + read.Path, read.Account}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConfiguredApplicationProofAndNegativeCases(t *testing.T) {
	for _, tc := range []struct{ name, want string }{{"vulnerable", "confirmed"}, {"fixed", "no_issue_observed"}, {"wrong_identity", "inconclusive"}, {"deleted_owner", "inconclusive"}, {"generic_body", "inconclusive"}, {"missing_identity", "inconclusive"}, {"expired_peer", "inconclusive"}} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, c, flags := applicationFixture(t)
			_, r := applicationEngagement(t, s, c)
			flags.fixed.Store(tc.name == "fixed" || tc.name == "deleted_owner" || tc.name == "expired_peer")
			flags.wrongIdentity.Store(tc.name == "wrong_identity")
			flags.generic.Store(tc.name == "generic_body")
			reads := r.Application.Reads()
			if tc.name == "missing_identity" {
				reads = reads[2:]
			}
			for i, read := range reads {
				if tc.name == "deleted_owner" && i == 5 {
					flags.deleted.Store(true)
				}
				if tc.name == "expired_peer" && i == 4 {
					flags.expiredPeer.Store(true)
				}
				executeAppReads(t, s, r, []application.Read{read})
			}
			result, err := s.Finish(r.ID)
			if err != nil || result.Verdict != tc.want {
				t.Fatalf("%s: %+v %v", tc.name, result, err)
			}
		})
	}
}

func TestApplicationScopeCredentialsAndConfigurationDrift(t *testing.T) {
	s, store, c, flags := applicationFixture(t)
	e, r := applicationEngagement(t, s, c)
	for _, in := range []Request{{r.Target + "/debug", "owner"}, {r.Target + "/objects/b", "owner"}, {r.Target + "/objects/a", "anonymous"}, {"http://not-approved.invalid/objects/a", "peer"}} {
		if _, err := s.Execute(context.Background(), r.ID, in); err == nil {
			t.Fatal("unapproved request accepted")
		}
	}
	if flags.requests.Load() != 0 {
		t.Fatal("denied request reached application")
	}
	data, _ := json.Marshal(EngagementInput{Name: "other tenant", AuthorizationRef: "lab", Application: c.Plan.ID})
	if w := request(s, "POST", "/v1/engagements", "operator-b", string(data)); w.Code != 400 {
		t.Fatal("other tenant acquired application")
	}
	token, _ := store.Credential(r.ID)
	if w := request(s, "GET", "/v1/applications", token, ""); w.Code != 401 {
		t.Fatal("worker read registry")
	}
	encoded, _ := json.Marshal(r)
	for _, secret := range c.Credentials {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("credential in worker policy")
		}
	}
	if w := request(s, "POST", "/v1/engagements/"+e.ID+"/runs", "operator-a", `{"assessment":"`+r.Application.ProcedureID()+`","execution":"live"}`); w.Code != 409 {
		t.Fatal("unqualified model path enabled")
	}
	// Model a process restart with a changed credential. The old durable run
	// must not silently use it, even under the same public application ID.
	newServer := NewServer(store, s.Targets, s.AdminTokens)
	c.Credentials["peer"] = strings.Repeat("c", 48)
	if err := newServer.RegisterApplication(c); err != nil {
		t.Fatal(err)
	}
	if _, err := newServer.Execute(context.Background(), r.ID, Request{r.Target + "/me", "owner"}); err == nil {
		t.Fatal("changed config reused old run")
	}
	if flags.requests.Load() != 0 {
		t.Fatal("configuration drift sent traffic")
	}
}

func TestApplicationLatestFailedProbeDoesNotReuseOldSuccess(t *testing.T) {
	s, _, c, flags := applicationFixture(t)
	_, r := applicationEngagement(t, s, c)
	executeAppReads(t, s, r, r.Application.Reads()[:5])
	flags.generic.Store(true)
	executeAppReads(t, s, r, []application.Read{r.Application.Probe, r.Application.Baselines[0], r.Application.Identity[1]})
	result, err := s.Finish(r.ID)
	if err != nil || result.Verdict != "inconclusive" {
		t.Fatal("stale successful probe reused")
	}
}

func TestApplicationStopCancelsBrokerWithoutKubernetes(t *testing.T) {
	s, _, c, flags := applicationFixture(t)
	e, r := applicationEngagement(t, s, c)
	flags.block.Store(true)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = s.Execute(context.Background(), r.ID, Request{r.Target + r.Application.Identity[0].Path, "owner"})
	}()
	select {
	case <-flags.started:
	case <-time.After(2 * time.Second):
		t.Fatal("application request did not start")
	}
	if _, err := s.StopEngagement("a", e.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-flags.cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("target did not observe cancellation")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("broker did not drain")
	}
	if _, err := s.Execute(context.Background(), r.ID, Request{r.Target + r.Application.Probe.Path, "peer"}); err == nil {
		t.Fatal("application request accepted after Stop")
	}
}
