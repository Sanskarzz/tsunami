package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Sanskarzz/tsunami/internal/platform"
)

func TestBadOptionalConfigurationStillBootsRevocationAPI(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("startup unexpectedly contacted target") }))
	defer target.Close()
	t.Setenv("TSUNAMI_VULNERABLE_URL", target.URL)
	t.Setenv("TSUNAMI_FIXED_URL", target.URL)
	t.Setenv("TSUNAMI_PRIVATE_CONFIG_FILE", filepath.Join(t.TempDir(), "missing-private.json"))
	t.Setenv("TSUNAMI_ALLOW_PRIVATE_LAB", "true")
	t.Setenv("TSUNAMI_MODEL_PROVIDER", "openrouter-free")
	t.Setenv("TSUNAMI_ALLOW_MODEL_CALLS", "false")
	t.Setenv("TSUNAMI_APPLICATION_CONFIG_FILE", "")
	t.Setenv("TSUNAMI_ENABLE_CURL_ADAPTER", "false")
	store, err := platform.OpenStore(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	s := platform.NewServer(store, nil, map[string]string{"synthetic-operator": "a"})
	r, err := store.CreatePrivate("a", "vulnerable", "http://private.invalid", "unavailable route test")
	if err != nil {
		t.Fatal(err)
	}
	configureRoutes(context.Background(), s)
	if s.PrivateTargets != nil || s.Inference != nil {
		t.Fatal("invalid dependency retained an execution route")
	}
	for _, path := range []string{"/readyz", "/v1/status"} {
		request := httptest.NewRequest("GET", path, nil)
		request.Header.Set("Authorization", "Bearer synthetic-operator")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, request)
		if w.Code != 200 {
			t.Fatalf("administrative API did not start: %s", path)
		}
	}
	if err := s.Stop(r.ID); err != nil {
		t.Fatal("revocation failed without optional route", err)
	}
	status, _ := s.Status(r.ID)
	if status.Permission != "revoked" {
		t.Fatal("private route failure hid revocation")
	}
}
