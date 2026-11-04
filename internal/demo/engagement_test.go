package demo

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/platform"
)

func TestEngagementWorkflowThroughMecatlAndResponses(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	summary, err := EngagementLocal(ctx, t.TempDir(), true, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.DraftDispatchDenied || !summary.StoppedDispatchDenied || len(summary.Checks) != 2 {
		t.Fatal("engagement gates not demonstrated")
	}
	for _, check := range summary.Checks {
		if check.Completed != 5 || (check.Scenario == "vulnerable" && check.Candidates != 3) || (check.Scenario == "fixed" && check.Candidates != 0) {
			t.Fatal("incorrect application outcomes")
		}
		data, err := os.ReadFile(filepath.Join(summary.Directory, check.Scenario, "report-draft.md"))
		if err != nil || !strings.Contains(string(data), "human review") {
			t.Fatal("missing review boundary")
		}
	}
}

func TestEngagementArtifactWriterRejectsEscapeAndDigestMismatch(t *testing.T) {
	for _, path := range []string{"../outside.json", "evidence/not-a-digest.json"} {
		dir := filepath.Join(t.TempDir(), "new")
		err := saveEngagementBundle(dir, platform.EngagementBundle{Artifacts: []platform.EvidenceArtifact{{Path: path, SHA256: strings.Repeat("a", 64), Data: []byte("{}")}}})
		if err == nil {
			t.Fatal("untrusted artifact written")
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "outside.json")); !os.IsNotExist(err) {
			t.Fatal("artifact escaped campaign directory")
		}
	}
}
