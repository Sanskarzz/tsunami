package demo

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sanskarzz/tsunami/internal/lab"
	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/worker"
)

// Individual run responses are bounded separately from inline evidence bundles.
const maxOperatorResponseBytes = 16 << 20

var ErrOperatorResponseTooLarge = errors.New("operator response exceeds byte limit")

var operatorTransport = &http.Transport{Proxy: nil, TLSHandshakeTimeout: 5 * time.Second, IdleConnTimeout: 30 * time.Second, MaxIdleConns: 16, MaxIdleConnsPerHost: 4}

type Check struct {
	Scenario    string `json:"scenario"`
	RunID       string `json:"run_id"`
	Verdict     string `json:"verdict"`
	Requests    int    `json:"requests"`
	StopDrainMS int64  `json:"stop_drain_ms,omitempty"`
}
type Summary struct {
	Mode        string    `json:"mode"`
	CheckedAt   time.Time `json:"checked_at"`
	Checks      []Check   `json:"checks"`
	Limitations []string  `json:"limitations"`
}
type Operator struct {
	BaseURL, Token      string
	RequiredPricePolicy string
	Connectivity        string
	Client              *http.Client
	ReadyForStop        func(context.Context) error
}

// LabStats is used by the trusted test driver, never exposed as an agent tool.
func LabStats(ctx context.Context, rawURL string) (map[string]int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New("lab stats unavailable")
	}
	var stats map[string]int64
	err = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&stats)
	return stats, err
}

func (o Operator) Call(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, e := json.Marshal(in)
		if e != nil {
			return e
		}
		body = bytes.NewReader(data)
	}
	req, e := http.NewRequestWithContext(ctx, method, o.BaseURL+path, body)
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "Bearer "+o.Token)
	req.Header.Set("Content-Type", "application/json")
	limit := int64(maxOperatorResponseBytes)
	timeout := 5 * time.Second
	if strings.HasPrefix(path, "/v1/engagements/") && strings.HasSuffix(path, "/export") {
		limit = platform.MaxInlineExportBytes
		timeout = 30 * time.Second
	}
	client := http.Client{Timeout: timeout, Transport: operatorTransport}
	if o.Client != nil {
		client = *o.Client // Preserve injected transports without mutating callers.
		if client.Transport == nil {
			client.Transport = operatorTransport
		}
	}
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	resp, e := client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("operator API returned %d", resp.StatusCode)
	}
	if out != nil {
		return decodeOperatorResponse(resp.Body, out, limit)
	}
	return nil
}

func decodeOperatorResponse(body io.Reader, out any, limit int64) error {
	reader := &io.LimitedReader{R: body, N: limit + 1}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(out)
	if reader.N == 0 {
		return ErrOperatorResponseTooLarge
	}
	if err != nil {
		return err
	}
	err = decoder.Decode(&struct{}{})
	if reader.N == 0 {
		return ErrOperatorResponseTooLarge
	}
	if err != io.EOF {
		return errors.New("operator response has invalid trailing data")
	}
	return nil
}

func Wait(ctx context.Context, o Operator, id string, predicate func(platform.Run) bool) (platform.Run, error) {
	timer := time.NewTicker(25 * time.Millisecond)
	defer timer.Stop()
	for {
		var r platform.Run
		if e := o.Call(ctx, "GET", "/v1/runs/"+id, nil, &r); e != nil {
			return r, e
		}
		if predicate(r) {
			return r, nil
		}
		select {
		case <-ctx.Done():
			return r, ctx.Err()
		case <-timer.C:
		}
	}
}

// Verify exercises real HTTP traffic and the trusted evidence validator. No
// external target or model provider is needed. The operator API supplies only
// fixed, server-configured synthetic scenarios.
func Verify(ctx context.Context, o Operator, outDir string, log io.Writer) (Summary, error) {
	if e := os.MkdirAll(outDir, 0700); e != nil {
		return Summary{}, e
	}
	summary := Summary{Mode: "real Mecatl engine; scripted model; synthetic targets", CheckedAt: time.Now().UTC(), Checks: []Check{}, Limitations: []string{
		"No live LLM: this is not autonomous vulnerability discovery.",
		"No arbitrary shell or external scanner execution.",
		"The workflow alone does not qualify hostile-process containment; network/runtime checks are separate.",
		"No gVisor RuntimeClass, private customer connector, or distributed permission leases have been qualified by this demo.",
	}}
	for _, scenario := range []string{"vulnerable", "fixed", "inconclusive", "redirect", "stop"} {
		var run platform.Run
		if e := o.Call(ctx, "POST", "/v1/runs", map[string]string{"scenario": scenario, "connectivity": o.Connectivity}, &run); e != nil {
			return summary, e
		}
		if log != nil {
			fmt.Fprintf(log, "VERIFY_CREATED run=%s scenario=%s connectivity=%s\n", run.ID, scenario, o.Connectivity)
		}
		var elapsed int64
		if scenario == "stop" {
			if _, e := Wait(ctx, o, run.ID, func(r platform.Run) bool {
				return r.State == platform.Running && r.ActiveRequests > 0 && r.RequestCount > 0
			}); e != nil {
				return summary, e
			}
			if o.ReadyForStop != nil {
				if e := o.ReadyForStop(ctx); e != nil {
					return summary, e
				}
			}
			start := time.Now()
			if e := o.Call(ctx, "POST", "/v1/runs/"+run.ID+"/stop", nil, &run); e != nil {
				return summary, e
			}
			var e error
			run, e = Wait(ctx, o, run.ID, func(r platform.Run) bool { return r.State == platform.Stopped && r.ActiveRequests == 0 })
			if e != nil {
				return summary, e
			}
			elapsed = time.Since(start).Milliseconds()
			count := run.RequestCount
			select {
			case <-ctx.Done():
				return summary, ctx.Err()
			case <-time.After(600 * time.Millisecond):
			}
			if e := o.Call(ctx, "GET", "/v1/runs/"+run.ID, nil, &run); e != nil {
				return summary, e
			}
			if run.RequestCount != count {
				return summary, errors.New("request count grew after stop")
			}
		} else {
			var e error
			run, e = Wait(ctx, o, run.ID, func(r platform.Run) bool {
				return r.State == platform.Completed || r.State == platform.Failed || r.State == platform.Stopped
			})
			if e != nil {
				return summary, e
			}
			want := map[string]string{"vulnerable": "confirmed", "fixed": "no_issue_observed", "inconclusive": "inconclusive", "redirect": "inconclusive"}[scenario]
			if run.State != platform.Completed || run.Result == nil || run.Result.Verdict != want {
				return summary, fmt.Errorf("%s: unexpected validation result", scenario)
			}
		}
		blocked := 0
		for _, capture := range run.Captures {
			if capture.Blocked {
				blocked++
			}
		}
		if blocked < 1 {
			return summary, errors.New("scope rejection was not recorded")
		}
		if scenario == "redirect" && blocked < 2 {
			return summary, errors.New("out-of-scope redirect was not blocked")
		}
		verdict := "stopped"
		if run.Result != nil {
			verdict = run.Result.Verdict
		}
		check := Check{scenario, run.ID, verdict, run.RequestCount, elapsed}
		summary.Checks = append(summary.Checks, check)
		data, e := json.MarshalIndent(run, "", "  ")
		if e != nil {
			return summary, e
		}
		if e = os.WriteFile(filepath.Join(outDir, scenario+".json"), data, 0600); e != nil {
			return summary, e
		}
		if e = os.WriteFile(filepath.Join(outDir, scenario+".md"), []byte(platform.Report(run)), 0600); e != nil {
			return summary, e
		}
		if log != nil {
			fmt.Fprintf(log, "PASS %-13s verdict=%-18s requests=%d scope_denials=%d stop_drain_ms=%d\n", scenario, verdict, run.RequestCount, blocked, elapsed)
		}
	}
	data, e := json.MarshalIndent(summary, "", "  ")
	if e != nil {
		return summary, e
	}
	e = os.WriteFile(filepath.Join(outDir, "summary.json"), data, 0600)
	return summary, e
}

func Local(ctx context.Context, outDir string, log io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	tripwire := &lab.Server{}
	trap := httptest.NewServer(tripwire)
	defer trap.Close()
	vulnerable := &lab.Server{RedirectTo: trap.URL + "/api/documents/doc-alice"}
	v := httptest.NewServer(vulnerable)
	defer v.Close()
	f := httptest.NewServer(&lab.Server{Fixed: true})
	defer f.Close()
	vt, e := platform.LabTarget(ctx, v.URL)
	if e != nil {
		return e
	}
	ft, e := platform.LabTarget(ctx, f.URL)
	if e != nil {
		return e
	}
	dir, e := os.MkdirTemp("", "tsunami-lab-")
	if e != nil {
		return e
	}
	// The temporary database intentionally remains available if the demo fails.
	store, e := platform.OpenStore(filepath.Join(dir, "state.db"))
	if e != nil {
		return e
	}
	defer store.Close()
	key := make([]byte, 32)
	if _, e = rand.Read(key); e != nil {
		return e
	}
	token := hex.EncodeToString(key)
	server := platform.NewServer(store, map[string]platform.Target{"vulnerable": vt, "fixed": ft}, map[string]string{token: "demo-a"})
	api := httptest.NewServer(server)
	defer api.Close()
	server.Scheduler = &worker.LocalScheduler{ClientURL: api.URL, Log: log, OnError: func(id string, _ error) { _ = store.Fail(id) }}
	go server.Reconcile(ctx)
	ready := func(ctx context.Context) error {
		for vulnerable.SlowActive.Load() == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Millisecond):
			}
		}
		return nil
	}
	_, e = Verify(ctx, Operator{BaseURL: api.URL, Token: token, ReadyForStop: ready}, outDir, log)
	cancel()
	if e != nil {
		return e
	}
	if tripwire.Requests.Load() != 0 {
		return errors.New("out-of-scope tripwire received a request")
	}
	if vulnerable.Cancelled.Load() == 0 {
		return errors.New("target did not observe cancellation of the slow request")
	}
	if log != nil {
		fmt.Fprintln(log, "PASS tripwire: zero requests; lab target observed request cancellation")
	}
	return nil
}
