package toolrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sanskarzz/tsunami/internal/toolcontract"
)

func testRunner(t *testing.T, forward Forward) *Runner {
	t.Helper()
	if _, err := os.Stat("/usr/bin/curl"); err != nil {
		t.Fatal("these Linux adapter tests require /usr/bin/curl")
	}
	path := filepath.Join(t.TempDir(), "http-read.sh")
	if err := os.WriteFile(path, script, 0400); err != nil {
		t.Fatal(err)
	}
	r, err := newRunner(context.Background(), forward, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Close)
	return r
}

func TestCurlIntentIsDataAndEnvironmentIsNotInherited(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "executed")
	// An ambient curl config or proxy must not change this fixed invocation.
	config := t.TempDir()
	_ = os.WriteFile(filepath.Join(config, ".curlrc"), []byte("url = file:///etc/passwd\n"), 0600)
	t.Setenv("CURL_HOME", config)
	t.Setenv("ALL_PROXY", "http://127.0.0.1:1")
	want := toolcontract.HTTPIntent{URL: "http://approved.invalid/$(touch " + marker + ");'\"\n--output /tmp/unwanted", Account: "peer"}
	var got toolcontract.HTTPIntent
	r := testRunner(t, func(_ context.Context, in toolcontract.HTTPIntent) (json.RawMessage, error) {
		got = in
		return json.RawMessage(`{"status":200}`), nil
	})
	out, err := r.Execute(context.Background(), want)
	if err != nil || string(out) != `{"status":200}` || got != want {
		t.Fatalf("intent changed or invocation failed: %s %v", out, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("input executed as shell source")
	}
	mode, _ := os.Stat(r.socket)
	if mode.Mode().Perm() != 0600 {
		t.Fatal("bridge socket permissions")
	}
}

func TestBridgeRejectsOtherCapabilitiesAndBoundsInvocations(t *testing.T) {
	var calls atomic.Int32
	r := testRunner(t, func(context.Context, toolcontract.HTTPIntent) (json.RawMessage, error) {
		calls.Add(1)
		return json.RawMessage(`{}`), nil
	})
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", r.socket)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	for _, tc := range []struct{ path, body string }{
		{"/v1/runs/other/request", `{"url":"x","account":"peer"}`},
		{"/request?run=other", `{"url":"x","account":"peer"}`},
		{"/request", `{"url":"x","account":"peer","method":"DELETE"}`},
		{"/request", `{"url":"x","account":"peer","headers":{"Authorization":"x"}}`},
		{"/request", `{"url":"x","account":"peer","body":"arbitrary"}`},
		{"/request", `{"url":"x","account":"peer"} {}`},
		{"/request", `{"url":"x","account":"owner","account":"peer"}`},
		{"/request", strings.Repeat("a", 5000)},
	} {
		resp, err := client.Post("http://localhost"+tc.path, "application/json", strings.NewReader(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != 400 {
			t.Fatalf("unsupported bridge input accepted: %s", tc.path)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("unsupported input reached broker")
	}
	for range 8 {
		if _, err := r.Execute(context.Background(), toolcontract.HTTPIntent{URL: "x", Account: "peer"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.Execute(context.Background(), toolcontract.HTTPIntent{URL: "x", Account: "peer"}); err == nil || calls.Load() != 8 {
		t.Fatal("invocation bound not enforced")
	}
}

func TestCurlCancellationClosesBridgeRequestAndDoesNotRetry(t *testing.T) {
	started, cancelled := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	r := testRunner(t, func(ctx context.Context, _ toolcontract.HTTPIntent) (json.RawMessage, error) {
		calls.Add(1)
		close(started)
		<-ctx.Done()
		close(cancelled)
		return nil, ctx.Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := r.Execute(ctx, toolcontract.HTTPIntent{URL: "http://approved.invalid", Account: "peer"})
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("curl never reached bridge")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled call succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("curl did not stop")
	}
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("upstream context did not cancel")
	}
	if calls.Load() != 1 {
		t.Fatal("request retried")
	}
}

func TestCommandLimitsOutputAndKillsOrdinaryDescendants(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		timeout      time.Duration
	}{
		{"output", "while :; do printf '0123456789012345678901234567890123456789'; done", 2 * time.Second},
		{"timeout", "sleep 30 & wait", 100 * time.Millisecond},
		{"inherited-pipe", "sleep 30 & exit 0", 2 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()
			start := time.Now()
			out, err := command(ctx, "/bin/sh", []string{"-c", tc.source}, nil)
			if err == nil || len(out) > outputLimit || time.Since(start) > 3*time.Second {
				t.Fatalf("command unbounded: %v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := command(ctx, "/bin/sh", []string{"-c", "exit 0"}, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled command started: %v", err)
	}
}

func TestChangedAdapterRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script")
	_ = os.WriteFile(path, bytes.Repeat([]byte("x"), 8), 0600)
	if _, err := newRunner(context.Background(), func(context.Context, toolcontract.HTTPIntent) (json.RawMessage, error) { return nil, nil }, path); err == nil {
		t.Fatal("changed script executed")
	}
}
