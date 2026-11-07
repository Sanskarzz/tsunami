package binding_test

import (
	"encoding/json"
	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/binding"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"path/filepath"
	"strings"
	"testing"
)

type Input = binding.Input
type Document = binding.Document
type Account = binding.Account

func integrationInput(profile string) Input {
	doc := func(body string) Document { return Document{Status: 200, Body: json.RawMessage(body)} }
	access := "deny"
	if profile == application.PropertyAccessProfile {
		access = "filtered"
	}
	return Input{Version: "tsunami.application-binding/v1", ID: "prepared-api", Title: "Prepared owned test", Profile: profile, Origin: "http://owned.example.test:8080", ApprovedAddresses: []string{"192.0.2.20"}, SyntheticDataOnly: true, ExpectedAccess: access, IdentityPath: "/me", IdentityPointer: "/name", ObjectPathTemplate: "/records/{id}", ObjectIDPointer: "/id", OwnerPointer: "/owner", ProtectedPointers: []string{"/private"}, Owner: Account{Identity: doc(`{"name":"first"}`), Object: doc(`{"id":"record_first","owner":"first","private":"synthetic-owner-proof-123456789"}`)}, Peer: Account{Identity: doc(`{"name":"second"}`), Object: doc(`{"id":"record_second","owner":"second","private":"synthetic-peer-proof-123456789"}`)}}
}

func TestPreparedBindingStillRequiresServerCredentialsAndApproval(t *testing.T) {
	for _, profile := range []string{application.ObjectAccessProfile, application.PropertyAccessProfile} {
		t.Run(profile, func(t *testing.T) {
			raw, _ := json.Marshal(integrationInput(profile))
			p, err := binding.Compile(raw)
			if err != nil {
				t.Fatal(err)
			}
			store, err := platform.OpenStore(filepath.Join(t.TempDir(), "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			s := platform.NewServer(store, nil, nil)
			if err := s.RegisterApplication(platform.ApplicationConfig{Tenant: "owned", Plan: p, Credentials: map[string]string{"owner": strings.Repeat("a", 48), "peer": strings.Repeat("b", 48)}}); err != nil {
				t.Fatal(err)
			}
			registered := s.Applications["owned"][p.ID].Plan
			if registered.CredentialSetID == "" || registered.CredentialSetID == strings.Repeat("0", 64) {
				t.Fatal("server did not bind assigned credentials")
			}
			e, err := store.CreateEngagement("owned", p.Origin, platform.EngagementInput{Name: "Owned preparation test", AuthorizationRef: "owned-lab/test", Application: p.ID}, registered)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.CreateEngagementRun("owned", e.ID, registered.ProcedureID(), nil); err == nil {
				t.Fatal("local preparation bypassed scope approval")
			}
		})
	}
}
