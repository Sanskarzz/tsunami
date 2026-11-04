package platform

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
)

func reviewedCandidate(t *testing.T) (*Server, *Store, Engagement, FindingCandidate) {
	t.Helper()
	s, store, c, _ := applicationFixture(t)
	e, r := applicationEngagement(t, s, c)
	executeAppReads(t, s, r, r.Application.Reads())
	if _, err := s.Finish(r.ID); err != nil {
		t.Fatal(err)
	}
	b, err := store.ExportEngagement("a", e.ID)
	if err != nil || len(b.Candidates) != 1 {
		t.Fatal("missing evidence candidate", err)
	}
	if len(b.Findings) != 0 {
		t.Fatal("unreviewed finding exported")
	}
	return s, store, e, b.Candidates[0]
}

func testReview(c FindingCandidate) ReviewInput {
	return ReviewInput{CandidateID: c.ID, CandidateDigest: c.Digest, Decision: "confirm", Purpose: "automated_lab_test", Rationale: "Automated synthetic review exercises the contract; not human signoff.", Impact: "A synthetic authenticated peer read another synthetic account's seeded proof data.", CVSSVector: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N"}
}

func TestFindingReviewPinsEvidenceAndSurvivesRestart(t *testing.T) {
	s, store, e, c := reviewedCandidate(t)
	in := testReview(c)
	if _, err := store.ReviewFinding("a", e.ID, in); err == nil {
		t.Fatal("active engagement reviewed")
	}
	if _, err := s.StopEngagement("a", e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReviewFinding("b", e.ID, in); err == nil {
		t.Fatal("cross-tenant review accepted")
	}
	bad := in
	bad.CandidateDigest = strings.Repeat("0", 64)
	if _, err := store.ReviewFinding("a", e.ID, bad); err == nil {
		t.Fatal("stale evidence approved")
	}
	review, err := store.ReviewFinding("a", e.ID, in)
	if err != nil || review.CVSSScore == nil || *review.CVSSScore != 6.5 || review.Severity != "Medium" {
		t.Fatal("incorrect review", err)
	}
	if _, err := store.ReviewFinding("a", e.ID, in); err == nil {
		t.Fatal("concurrent review overwrite")
	}
	path := store.db.Path()
	_ = store.Close()
	reopened, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	bundle, err := reopened.ExportEngagement("a", e.ID)
	if err != nil || len(bundle.Findings) != 1 || len(bundle.Evidence) != 7 {
		t.Fatal("review/evidence did not survive restart", err)
	}
	if bundle.Candidates[0].Review.Actor != "automated_lab_verifier" {
		t.Fatal("test review masquerades as human")
	}
	if bundle.Findings[0].CVSSScore != 6.5 || bundle.Findings[0].Evidence[0].FindingID != c.ID {
		t.Fatal("broken finding export")
	}
}

func TestWorkerCannotReviewAndFalsePositiveDoesNotReceiveMadeUpScore(t *testing.T) {
	s, store, e, c := reviewedCandidate(t)
	_, _ = s.StopEngagement("a", e.ID)
	runID, _ := store.GetEngagement("a", e.ID)
	token, _ := store.Credential(runID.RunIDs[0])
	in := testReview(c)
	data, _ := json.Marshal(in)
	if w := request(s, "POST", "/v1/engagements/"+e.ID+"/reviews", token, string(data)); w.Code != 401 {
		t.Fatal("worker reviewed its finding")
	}
	in.Decision = "false_positive"
	in.CVSSVector = ""
	in.Impact = ""
	if _, err := store.ReviewFinding("a", e.ID, in); err != nil {
		t.Fatal(err)
	}
	b, err := store.ExportEngagement("a", e.ID)
	if err != nil || len(b.Findings) != 0 || b.Candidates[0].Status != "reviewed_false_positive" {
		t.Fatal("false positive included in findings")
	}
	if b.Candidates[0].Review.CVSSScore != nil {
		t.Fatal("false positive received a fabricated zero score")
	}
}

func TestCSVDoesNotPromoteFormulaTextToSpreadsheetExpression(t *testing.T) {
	var out bytes.Buffer
	if err := FindingsCSV(&out, []ToolkitFinding{{ID: "F-1", Title: "=1+1", Impact: "  @SUM(1,1)"}}); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&out).ReadAll()
	if err != nil || rows[1][1] != "'=1+1" || rows[1][7] != "'  @SUM(1,1)" {
		t.Fatal("unsafe CSV cells")
	}
}
