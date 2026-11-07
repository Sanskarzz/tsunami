package application

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestJSONProofRejectsAmbiguousAndWrongShapes(t *testing.T) {
	sum := sha256.Sum256([]byte("proof"))
	r := Read{Equals: map[string]string{"/data/name": "owner"}, Hashes: map[string]string{"/data/private": hex.EncodeToString(sum[:])}}
	if !r.Matches(`{"data":{"name":"owner","private":"proof"}}`) {
		t.Fatal("valid JSON proof rejected")
	}
	for _, body := range []string{
		`{"data":{"name":"other","name":"owner","private":"proof"}}`,
		`{"data":{"name":"owner","private":"proof"},"x":1,"x":2}`,
		`{"data":{"name":"owner","private":"proof"}} {}`,
		`{"data":[{"name":"owner","private":"proof"}]}`,
		`{"data":{"name":"owner","private":123}}`,
		`{"data":{"name":"owner","private":"wrong"}}`,
	} {
		if r.Matches(body) {
			t.Fatal("ambiguous/unproven response accepted")
		}
	}
	r.Equals = map[string]string{"not-a-pointer": "owner"}
	if r.Matches(`{"data":{"name":"owner","private":"proof"}}`) {
		t.Fatal("invalid selector accepted")
	}
}

func TestPropertyFilteringRequiresThePublicObjectAndKnownPrivateValue(t *testing.T) {
	hash := func(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }
	r := Read{Equals: map[string]string{"/id": "record-a", "/owner": "owner"}, Hashes: map[string]string{"/private": hash("known-private-value")}}
	for _, test := range []struct{ body, want string }{
		{`{"id":"record-a","owner":"owner","private":"known-private-value"}`, "leaked"},
		{`{"id":"record-a","owner":"owner","public":"hello"}`, "filtered"},
		{`{"id":"record-a","owner":"owner","private":"[redacted]"}`, "unknown"},
		{`{"id":"record-a","owner":"owner","private": null}`, "unknown"},
		{`{"id":"record-a","owner":"owner","private":123}`, "unknown"},
		{`{"id":"record-a","owner":"owner","private":{"nested":"known-private-value"}}`, "leaked"},
		{`{"id":"record-a","owner":"owner","renamed":["known-private-value"]}`, "leaked"},
		{`{"id":"other","owner":"owner"}`, "unknown"},
		{`{"error":"generic success"}`, "unknown"},
		{`[{"id":"record-a","owner":"owner"}]`, "unknown"},
		{`{"id":"record-a","owner":"owner","private":"known-private-value","private":"[redacted]"}`, "unknown"},
		{`{"id":"record-a","owner":"owner"} {}`, "unknown"},
	} {
		if got := r.PropertyResult(test.body); got != test.want {
			t.Fatalf("property verdict %s, expected %s for %s", got, test.want, test.body)
		}
	}
	r.Hashes["/second"] = hash("another-private-value")
	if r.PropertyResult(`{"id":"record-a","owner":"owner","second":"another-private-value","private":null}`) != "leaked" {
		t.Fatal("one proven private leak was hidden by another ambiguous property")
	}
	r.Hashes["/nested/private"] = hash("x")
	if r.PropertyResult(`{"id":"record-a","owner":"owner","second":"another-private-value"}`) != "unknown" {
		t.Fatal("unsupported nested policy accepted")
	}
}
