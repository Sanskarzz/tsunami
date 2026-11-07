// Package webui serves the opt-in local operator interface. It supplies no
// credentials, proxy authority, target registration or new assessment API.
package webui

import (
	"embed"
	"net"
	"net/http"
	"strings"
)

//go:embed static/index.html static/app.js static/onboarding.js static/styles.css
var assets embed.FS

type Handler struct{ Next http.Handler }

// Quiesce preserves the control server's admission shutdown behind this wrapper.
func (h Handler) Quiesce() {
	if q, ok := h.Next.(interface{ Quiesce() }); ok {
		q.Quiesce()
	}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path != "/ui" && !strings.HasPrefix(r.URL.Path, "/ui/") {
		h.Next.ServeHTTP(w, r)
		return
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'; object-src 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	// A loopback Host is required for port-forward use. This is an exposure
	// guard, not authentication; the API still requires an operator token.
	host := r.Host
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		host = hostname
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		http.Error(w, "Use a loopback port-forward for the local interface.", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.URL.RawQuery != "" {
		http.Error(w, "The interface accepts no query parameters or credentials in URLs.", http.StatusBadRequest)
		return
	}
	if r.URL.Path == "/ui" {
		http.Redirect(w, r, "/ui/", http.StatusTemporaryRedirect)
		return
	}
	files := map[string]string{"/ui/": "index.html", "/ui/app.js": "app.js", "/ui/onboarding.js": "onboarding.js", "/ui/styles.css": "styles.css"}
	name, ok := files[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	data, err := assets.ReadFile("static/" + name)
	if err != nil {
		http.Error(w, "Interface unavailable.", http.StatusInternalServerError)
		return
	}
	types := map[string]string{"index.html": "text/html; charset=utf-8", "app.js": "text/javascript; charset=utf-8", "onboarding.js": "text/javascript; charset=utf-8", "styles.css": "text/css; charset=utf-8"}
	w.Header().Set("Content-Type", types[name])
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
}
