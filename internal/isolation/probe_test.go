package isolation

import (
	"context"
	"net"
	"strings"
	"testing"
)

func TestProbeRejectsNonLabDestinations(t *testing.T) {
	for _, raw := range []string{"http://example.com:80", "http://8.8.8.8", "http://127.0.0.1", "http://169.254.169.254", "http://user@10.0.0.1", "http://10.0.0.1/path", "http://10.0.0.1?target=x", "http://[::ffff:127.0.0.1]", "ftp://10.0.0.1"} {
		if _, err := Address(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, raw := range []string{"http://10.0.0.1:8080", "https://172.18.0.2:6443", "http://[fd00::1]:8080"} {
		if _, err := Address(raw); err != nil {
			t.Errorf("rejected %q: %v", raw, err)
		}
	}
}

func TestRefusedConnectionIsNotAQualifiedDenial(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if got := attempt(context.Background(), address); got != "reachable" {
		t.Fatalf("positive control: %s", got)
	}
	_ = listener.Close()
	if got := attempt(context.Background(), address); !strings.HasPrefix(got, "unproven:") {
		t.Fatalf("refusal incorrectly accepted: %s", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := attempt(ctx, address); !strings.HasPrefix(got, "unproven:") {
		t.Fatalf("cancelled probe incorrectly accepted: %s", got)
	}
}
