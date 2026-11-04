package demo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Sanskarzz/tsunami/internal/platform"
)

// StopInference waits for provider HTTP 200 headers before pressing Stop. It
// proves broker drain and durable revocation, not the vendor's compute/billing.
func StopInference(ctx context.Context, o Operator, out string, log io.Writer) error {
	var run platform.Run
	if err := o.Call(ctx, "POST", "/v1/runs", map[string]string{"scenario": "vulnerable", "execution": "live", "required_price_policy": o.RequiredPricePolicy}, &run); err != nil {
		return err
	}
	if log != nil {
		fmt.Fprintf(log, "EVAL_CREATED run=%s scenario=model-stop\n", run.ID)
	}
	waitCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	run, err := Wait(waitCtx, o, run.ID, func(r platform.Run) bool {
		opened := r.ActiveModelRequests > 0 && r.Model != nil && r.Model.StreamsOpened == r.Model.CallsReserved && r.Model.WireAttempts == r.Model.CallsReserved && r.Model.CompletedCalls+1 == r.Model.CallsReserved
		return opened || r.State == platform.Completed || r.State == platform.Failed || r.State == platform.Stopped
	})
	if err != nil {
		return err
	}
	if run.ActiveModelRequests == 0 || run.State != platform.Running {
		return errors.New("missed the live inference window; no stop claim made")
	}
	start := time.Now()
	if err = o.Call(ctx, "POST", "/v1/runs/"+run.ID+"/stop", nil, &run); err != nil {
		return err
	}
	run, err = Wait(waitCtx, o, run.ID, func(r platform.Run) bool { return r.State == platform.Stopped && r.ActiveRequests == 0 })
	if err != nil {
		return err
	}
	drain := time.Since(start).Milliseconds()
	attempts, requests := run.Model.WireAttempts, run.RequestCount
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Second):
	}
	if err = o.Call(ctx, "GET", "/v1/runs/"+run.ID, nil, &run); err != nil {
		return err
	}
	if run.State != platform.Stopped || run.ActiveRequests != 0 || run.Model.WireAttempts != attempts || run.RequestCount != requests {
		return errors.New("run resumed or counters increased after stop")
	}
	if err = os.MkdirAll(out, 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(run, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "stop.json"), data, 0600); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(out, "stop.md"), []byte(platform.Report(run)), 0600); err != nil {
		return err
	}
	data, _ = json.MarshalIndent(map[string]any{"run_id": run.ID, "provider_stream_headers_observed": true, "broker_stop_drain_ms": drain, "no_counter_growth_for_ms": 1000, "vendor_cancellation_verified": false}, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "stop-check.json"), data, 0600); err != nil {
		return err
	}
	if log != nil {
		fmt.Fprintf(log, "PASS live inference stop run=%s broker_drain_ms=%d; vendor billing/compute cancellation is unverified\n", run.ID, drain)
	}
	return nil
}
