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
