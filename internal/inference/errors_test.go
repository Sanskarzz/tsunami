package inference

import (
	"context"
	"errors"
	"testing"
)

func TestDiagnosticCodesAreClosedAndDoNotExposeMessages(t *testing.T) {
	for _, code := range []string{"upstream_http_429 " + FaultCanary, "upstream_http_0429", "upstream_http_999", FaultCanary, "model_deadline_exceeded\nsecret"} {
		if KnownErrorCode(code) {
			t.Fatal("untrusted category accepted")
		}
	}
	for _, code := range []string{"model_cancelled", "model_deadline_exceeded", "upstream_http_503", "upstream_event_http_429"} {
		if !KnownErrorCode(code) {
			t.Fatal("known category missing")
		}
	}
	if code := ErrorCode(classify(context.Background(), errors.New(FaultCanary), 200, "model_stream_failed")); code != "model_stream_failed" {
		t.Fatal("raw provider text classified as a trusted condition")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := ErrorCode(classify(ctx, errors.New(FaultCanary), 503, "model_stream_failed")); code != "model_cancelled" {
		t.Fatal("observed cancellation lost to upstream text")
	}
}
