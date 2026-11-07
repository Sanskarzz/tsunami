package binding

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/Sanskarzz/tsunami/internal/application"
)

func example(profile string) Input {
	doc := func(body string) Document { return Document{Status: 200, Body: json.RawMessage(body)} }
	access := "deny"
	if profile == application.PropertyAccessProfile {
		access = "filtered"
	}
	return Input{Version: "tsunami.application-binding/v1", ID: "prepared-api", Title: "Prepared owned test", Profile: profile, Origin: "http://owned.example.test:8080", ApprovedAddresses: []string{"192.0.2.20"}, SyntheticDataOnly: true, ExpectedAccess: access, IdentityPath: "/me", IdentityPointer: "/name", ObjectPathTemplate: "/records/{id}", ObjectIDPointer: "/id", OwnerPointer: "/owner", ProtectedPointers: []string{"/private"}, Owner: Account{Identity: doc(`{"name":"first"}`), Object: doc(`{"id":"record_first","owner":"first","private":"synthetic-owner-proof-123456789"}`)}, Peer: Account{Identity: doc(`{"name":"second"}`), Object: doc(`{"id":"record_second","owner":"second","private":"synthetic-peer-proof-123456789"}`)}}
}

func encode(t *testing.T, input Input) []byte {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestPreparedPlansNeedRealCredentialBindingAndApproval(t *testing.T) {
	for _, profile := range []string{application.ObjectAccessProfile, application.PropertyAccessProfile} {
		t.Run(profile, func(t *testing.T) {
			input := example(profile)
			raw := encode(t, input)
			planBytes, manifestBytes, p, err := bytesFor(raw)
			if err != nil {
				t.Fatal(err)
			}
			if p.CredentialSetID != "" || p.Validate() == nil {
				t.Fatal("prepared plan masquerades as a credential-bound run plan")
			}
			for _, secret := range []string{"synthetic-owner-proof-123456789", "synthetic-peer-proof-123456789"} {
				if bytes.Contains(planBytes, []byte(secret)) || bytes.Contains(manifestBytes, []byte(secret)) {
					t.Fatal("raw private proof copied to prepared output")
				}
			}
			if p.Baselines[0].Path != "/records/record_first" || p.Probe.Account != "peer" || p.Probe.Path != p.Baselines[0].Path {
				t.Fatal("incorrect literal account/path binding")
			}
			if !p.Baselines[0].Matches(string(input.Owner.Object.Body)) || !p.Identity[1].Matches(string(input.Peer.Identity.Body)) {
				t.Fatal("compiled assertions do not match supplied baselines")
			}
			var manifest Manifest
			if json.Unmarshal(manifestBytes, &manifest) != nil || manifest.TargetRequests != 0 || manifest.PlanSHA256 != hash(planBytes) || manifest.InputSHA256 != hash(raw) {
				t.Fatal("incorrect draft provenance")
			}

		})
	}
}

func TestBadBindingsFailWithoutIncludingSourceValuesInErrors(t *testing.T) {
	for _, variant := range []string{"same_user", "wrong_owner", "same_object", "expired", "missing", "numeric_id", "path_injection", "same_proof", "weak_proof", "duplicate_pointer", "wrong_rule", "unsupported_profile", "not_synthetic", "bad_template", "nested_property"} {
		t.Run(variant, func(t *testing.T) {
			i := example(application.PropertyAccessProfile)
			switch variant {
			case "same_user":
				i.Peer = i.Owner
			case "wrong_owner":
				i.Owner.Object.Body = json.RawMessage(`{"id":"record_first","owner":"someone-else","private":"synthetic-owner-proof-123456789"}`)
			case "same_object":
				i.Peer.Object.Body = json.RawMessage(`{"id":"record_first","owner":"second","private":"synthetic-peer-proof-123456789"}`)
			case "expired":
				i.Peer.Identity.Status = 401
			case "missing":
				i.Owner.Identity.Body = json.RawMessage(`{}`)
			case "numeric_id":
				i.Owner.Object.Body = json.RawMessage(`{"id":123,"owner":"first","private":"synthetic-owner-proof-123456789"}`)
			case "path_injection":
				i.Owner.Object.Body = json.RawMessage(`{"id":"../metadata","owner":"first","private":"synthetic-owner-proof-123456789"}`)
			case "same_proof":
				i.Peer.Object.Body = json.RawMessage(`{"id":"record_second","owner":"second","private":"synthetic-owner-proof-123456789"}`)
			case "weak_proof":
				i.Owner.Object.Body = json.RawMessage(`{"id":"record_first","owner":"first","private":"short"}`)
			case "duplicate_pointer":
				i.ProtectedPointers = []string{"/private", "/private"}
			case "wrong_rule":
				i.ExpectedAccess = "allow"
			case "unsupported_profile":
				i.Profile = "raw-tcp"
			case "not_synthetic":
				i.SyntheticDataOnly = false
			case "bad_template":
				i.ObjectPathTemplate = "/records/{id}/{other}"
			case "nested_property":
				i.ProtectedPointers = []string{"/nested/private"}
			}
			_, err := Compile(encode(t, i))
			if err == nil {
				t.Fatal("unsupported prerequisites accepted")
			}
			if strings.Contains(err.Error(), "synthetic-owner-proof") || strings.Contains(err.Error(), "someone-else") {
				t.Fatal("source data echoed in an error")
			}
		})
	}
	valid := encode(t, example(application.ObjectAccessProfile))
	for _, bad := range [][]byte{append(append([]byte{}, valid...), []byte(` {}`)...), bytes.Replace(valid, []byte(`"version":`), []byte(`"version":"ignored","version":`), 1), bytes.Replace(valid, []byte(`"name":"first"`), []byte(`"name":"fake","name":"first"`), 1), bytes.Repeat([]byte("x"), (64<<10)+1)} {
		if _, err := Compile(bad); err == nil {
			t.Fatal("ambiguous, extra or oversized JSON accepted")
		}
	}
}

func TestCommandWritesPrivatePackagesWithoutOverwritingOrFollowingLinks(t *testing.T) {
	parent := t.TempDir()
	input := filepath.Join(parent, "binding.json")
	out := filepath.Join(parent, "prepared")
	if err := os.WriteFile(input, encode(t, example(application.PropertyAccessProfile)), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := Command(context.Background(), []string{"--input", input, "--out", out}, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{out, filepath.Join(out, "plan.json"), filepath.Join(out, "manifest.json")} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		want := os.FileMode(0600)
		if info.IsDir() {
			want = 0700
		}
		if info.Mode().Perm() != want {
			t.Fatal("package permissions not private")
		}
	}
	if err := Command(context.Background(), []string{"--input", input, "--out", out}, io.Discard, io.Discard); err == nil {
		t.Fatal("existing package overwritten")
	}
	link := filepath.Join(parent, "link.json")
	if err := os.Symlink(input, link); err != nil {
		t.Fatal(err)
	}
	if err := Command(context.Background(), []string{"--input", link, "--out", out + "-link"}, io.Discard, io.Discard); err == nil {
		t.Fatal("symlink input accepted")
	}
	fifo := filepath.Join(parent, "pipe")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Command(context.Background(), []string{"--input", fifo, "--out", out + "-pipe"}, io.Discard, io.Discard); err == nil {
		t.Fatal("nonregular input accepted")
	}
	if strings.Contains(stdout.String(), "synthetic-owner-proof") || !strings.Contains(stdout.String(), "No targets contacted") {
		t.Fatal("output leaked proof or claimed live verification")
	}
}
