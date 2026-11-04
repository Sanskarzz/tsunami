package platform

import (
	"testing"
)

func TestPrivateTargetRequiresTenantBindingAndNoNetworkFallback(t *testing.T) {
	s, _ := fixture(t)
	for _, body := range []string{`{"scenario":"vulnerable","connectivity":"private"}`, `{"scenario":"vulnerable","connectivity":"private","execution":"live"}`} {
		if w := request(s, "POST", "/v1/runs", "operator-a", body); w.Code < 400 {
			t.Fatal("unconfigured private run accepted")
		}
	}
	// Even a configured private origin is insufficient without its connector.
	s.PrivateTargets = map[string]map[string]Target{"a": {"vulnerable": s.Targets["vulnerable"]}}
	if _, err := s.target("a", "vulnerable", "private"); err == nil {
		t.Fatal("private route without connector accepted")
	}
	if _, err := s.target("b", "vulnerable", "private"); err == nil {
		t.Fatal("another tenant inherited private route")
	}
}
