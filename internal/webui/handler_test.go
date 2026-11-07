package webui

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sanskarzz/tsunami/internal/platform"
)

func TestLocalUIAddsNoOperatorAuthority(t *testing.T) {
	store, err := platform.OpenStore(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := platform.NewServer(store, nil, map[string]string{"private-operator-test-token": "demo-a"})
	server.SetRouteHealth("applications", "ready", "")
	if err := server.CheckAdmission("applications", false, false); err != nil {
		t.Fatal(err)
	}
	handler := Handler{Next: server}
	for _, path := range []string{"/ui/", "/ui/app.js", "/ui/onboarding.js", "/ui/styles.css"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:8080"+path, nil))
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Header().Get("Content-Security-Policy"), "connect-src 'self'") || strings.Contains(w.Body.String(), "private-operator-test-token") {
			t.Fatalf("unsafe or unavailable asset %s: status %d", path, w.Code)
		}
	}
	for _, path := range []string{"/v1/applications", "/v1/engagements"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:8080"+path, nil))
		if w.Code != 401 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("UI bypassed API authentication: %s status %d", path, w.Code)
		}
		request := httptest.NewRequest("GET", "http://127.0.0.1:8080"+path, nil)
		request.Header.Set("Authorization", "Bearer private-operator-test-token")
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, request)
		if w.Code != 200 {
			t.Fatalf("UI altered operator API: %s status %d", path, w.Code)
		}
	}
	// Wrapping must preserve the control API's shutdown admission gate.
	handler.Quiesce()
	if err := server.CheckAdmission("applications", false, false); err == nil {
		t.Fatal("wrapper lost shutdown admission gate")
	}
}

func TestUIRefusesRemoteHostsMutationsQueriesAndUnknownFiles(t *testing.T) {
	handler := Handler{Next: http.NotFoundHandler()}
	for _, test := range []struct {
		method, address string
		status          int
	}{
		{"GET", "http://public.example.test/ui/", 403},
		{"POST", "http://localhost:8080/ui/", 405},
		{"GET", "http://localhost:8080/ui/?token=never-accept-this", 400},
		{"GET", "http://localhost:8080/ui/state.db", 404},
		{"GET", "http://localhost:8080/ui/../../go.mod", 404},
		{"HEAD", "http://[::1]:8080/ui/", 200},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(test.method, test.address, nil))
		if w.Code != test.status || strings.Contains(w.Body.String(), "never-accept-this") || (test.method == "HEAD" && w.Body.Len() != 0) {
			t.Fatalf("%s %s: got %d", test.method, test.address, w.Code)
		}
	}
}
