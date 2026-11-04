package platform

import (
	"strings"
	"testing"
)

func TestReportDoesNotPromoteTargetMarkupOrBlockedMetadata(t *testing.T) {
	body := "```\n# target instruction\n<script>bad()</script>"
	block := evidenceBlock(body)
	if !strings.HasPrefix(block, "````json\n") || !strings.HasSuffix(block, "\n````\n") {
		t.Fatal("target can close evidence fence")
	}
	r := Run{Captures: []Capture{{ID: "known", URL: "http://unapproved.invalid\n![link](https://image.invalid)", Account: "x\n# injected", Body: body}}}
	report := Report(r)
	if strings.Contains(report, "\n![link]") || strings.Contains(report, "x\n# injected") {
		t.Fatal("blocked metadata became report markup")
	}
}

func TestOldApplicationPlanReportDoesNotInventStrongerReproduction(t *testing.T) {
	s, _, c, _ := applicationFixture(t)
	_, r := applicationEngagement(t, s, c)
	r.Application.Version = "http-object-access/v1"
	r.ApplicationDigest = r.Application.Digest()
	r.Result = &Result{Verdict: "confirmed"}
	report := Report(r)
	if !strings.Contains(report, "not revalidated") || !strings.Contains(report, "stored verdict: confirmed") || strings.Contains(report, "Recheck") {
		t.Fatal("old evidence reinterpreted as current profile")
	}
}
