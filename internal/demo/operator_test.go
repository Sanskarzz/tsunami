package demo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Sanskarzz/tsunami/internal/platform"
)

func TestOperatorNeverForwardsCredentialsOnRedirect(t *testing.T) {
	var reached atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached.Add(1)
		t.Error("redirect receiver was contacted")
	}))
	defer other.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-operator" {
			t.Error("missing initial identity")
		}
		http.Redirect(w, r, other.URL+"/receive", http.StatusFound)
	}))
	defer source.Close()
	for _, client := range []*http.Client{nil, {CheckRedirect: func(*http.Request, []*http.Request) error { return nil }}} {
		if err := (Operator{BaseURL: source.URL, Token: "synthetic-operator", Client: client}).Call(context.Background(), "GET", "/v1/runs", nil, nil); err == nil {
			t.Fatal("redirect accepted")
		}
	}
	if reached.Load() != 0 {
		t.Fatal("redirect leaked operator authority")
	}
}

func TestOperatorBoundsResponsesAndRejectsTrailingData(t *testing.T) {
	for _, body := range []string{`{"value":"` + strings.Repeat("x", 100) + `"}`, `{}` + strings.Repeat(" ", 100)} {
		var out any
		if err := decodeOperatorResponse(strings.NewReader(body), &out, 64); !errors.Is(err, ErrOperatorResponseTooLarge) {
			t.Fatalf("oversize not identified: %v", err)
		}
	}
	var out any
	if err := decodeOperatorResponse(strings.NewReader(`{} {}`), &out, 64); err == nil {
		t.Fatal("extra response document accepted")
	}
	if err := decodeOperatorResponse(strings.NewReader(`{} `), &out, 64); err != nil {
		t.Fatal(err)
	}
}

func TestOperatorCollectsLegalEngagementAboveFourMiB(t *testing.T) {
	store, err := platform.OpenStore(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	e, err := store.CreateEngagement("a", "http://owned.invalid", platform.EngagementInput{Name: "large export", AuthorizationRef: "owned-lab/test", Scenario: "vulnerable", Assessments: []string{"opaque-ownership/v1"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.ApproveEngagement("a", e.ID, e.ScopeDigest); err != nil {
		t.Fatal(err)
	}
	for range 12 {
		r, err := store.CreateEngagementRun("a", e.ID, "opaque-ownership/v1", nil)
		if err != nil {
			t.Fatal(err)
		}
		if err = store.Update(r.ID, func(r *platform.Run) error {
			r.State = platform.Failed
			r.RequestCount = 6
			for n := range 6 {
				r.Captures = append(r.Captures, platform.Capture{ID: fmt.Sprintf("%032x", n), URL: r.Target + "/api/reviewed/objects/r-7f3c9a", Account: "alice", Status: 200, Body: strings.Repeat("a", 48<<10)})
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	want, err := store.ExportEngagement("a", e.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) <= 4<<20 {
		t.Fatal("large engagement control is too small")
	}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(data) }))
	defer api.Close()
	var got platform.EngagementBundle
	if err := (Operator{BaseURL: api.URL, Token: "synthetic-operator"}).Call(context.Background(), "GET", "/v1/engagements/"+e.ID+"/export", nil, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Artifacts) != 72 || string(got.Artifacts[71].Data) != string(want.Artifacts[71].Data) {
		t.Fatal("large engagement evidence was truncated")
	}
	t.Logf("collected legal %d-byte engagement with %d captures", len(data), len(got.Artifacts))
}
