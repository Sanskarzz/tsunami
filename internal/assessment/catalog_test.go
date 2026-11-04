package assessment

import (
	"regexp"
	"testing"
)

func TestCatalogHasClosedReadOnlyProceduresAndStableCopies(t *testing.T) {
	defs := List()
	if len(defs) != 5 {
		t.Fatal("unexpected catalog size")
	}
	ids := map[string]bool{}
	path := regexp.MustCompile(`^/api/reviewed/[a-zA-Z0-9/_-]+$`)
	for _, d := range defs {
		if ids[d.ID] || !d.Canonical() || d.Method != "GET" || len(d.Baselines) != 2 || d.MaxRequests < 3 || d.MaxRequests > 12 {
			t.Fatalf("invalid procedure: %s", d.ID)
		}
		ids[d.ID] = true
		if d.ExpectedAccess != "allow" && d.ExpectedAccess != "deny" && d.ExpectedAccess != "unknown" {
			t.Fatal("invalid access rule")
		}
		for _, read := range d.Reads() {
			if !path.MatchString(read.Path) || !d.Allows(read.Account, read.Path) || (read.Account != "alice" && read.Account != "bob" && read.Account != "anonymous") {
				t.Fatal("procedure expands supported capabilities")
			}
		}
		if d.Allows("root", d.Probe.Path) || d.Allows(d.Probe.Account, "/stats") {
			t.Fatal("unexpected capability")
		}
		copy, err := Get(d.ID)
		if err != nil {
			t.Fatal(err)
		}
		copy.Baselines[0].Object.Content = "tampered"
		fresh, _ := Get(d.ID)
		if copy.Canonical() || fresh.Digest() != d.Digest() {
			t.Fatal("caller mutated shared catalog state")
		}
	}
	if _, err := Get("../../untrusted.json"); err == nil {
		t.Fatal("unknown catalog input accepted")
	}
}
