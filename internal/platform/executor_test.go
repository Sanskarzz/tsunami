package platform

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Sanskarzz/tsunami/internal/toolcontract"
)

func TestExecutorSelectionIsApprovedAndInherited(t *testing.T) {
	s, store, c, _ := applicationFixture(t)
	in := EngagementInput{Name: "CLI adapter test", AuthorizationRef: "owned-lab/test", Application: c.Plan.ID, Executor: toolcontract.CurlHTTP}
	data, _ := json.Marshal(in)
	if w := request(s, "POST", "/v1/engagements", "operator-a", string(data)); w.Code != 409 {
		t.Fatal("disabled adapter accepted")
	}
	s.EnableCurlAdapter = true
	w := request(s, "POST", "/v1/engagements", "operator-a", string(data))
	if w.Code != 201 {
		t.Fatalf("draft: %s", w.Body)
	}
	var e Engagement
	_ = json.Unmarshal(w.Body.Bytes(), &e)
	native := e
	native.Executor = ""
	if native.digest() == e.ScopeDigest {
		t.Fatal("executor not bound to approval")
	}
	approveEngagement(t, s, e)
	path := "/v1/engagements/" + e.ID + "/runs"
	if w := request(s, "POST", path, "operator-a", `{"assessment":"`+e.Application.ProcedureID()+`","executor":"native"}`); w.Code != 400 {
		t.Fatal("child selected its executor")
	}
	w = request(s, "POST", path, "operator-a", `{"assessment":"`+e.Application.ProcedureID()+`"}`)
	if w.Code != 201 {
		t.Fatal(w.Body)
	}
	var r Run
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	if r.Executor != toolcontract.CurlHTTP {
		t.Fatal("child executor lost")
	}
	if err := store.Update(r.ID, func(r *Run) error { r.Executor = ""; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := store.EngagementSnapshot("a", e.ID); err == nil {
		t.Fatal("changed child binding accepted")
	}
	if _, err := s.StopEngagement("a", e.ID); err != nil {
		t.Fatal("stop failed for changed binding", err)
	}
	in.Application = ""
	in.Scenario = "vulnerable"
	in.Assessments = []string{"opaque-ownership/v1"}
	data, _ = json.Marshal(in)
	if w := request(s, "POST", "/v1/engagements", "operator-a", string(data)); w.Code != 400 {
		t.Fatal("unsupported built-in executor accepted")
	}
	in.Application = c.Plan.ID
	in.Scenario = ""
	in.Assessments = nil
	in.Executor = "arbitrary-shell"
	data, _ = json.Marshal(in)
	if w := request(s, "POST", "/v1/engagements", "operator-a", string(data)); w.Code != 409 || strings.Contains(w.Body.String(), "token") {
		t.Fatal("unknown executor accepted")
	}
}
