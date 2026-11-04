package inference

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/stacklok/mecatl/engine/port"
)

type callError struct{ code string }

func (e callError) Error() string { return e.code }

// ErrorCode only exposes our own closed labels, never vendor response text.
func ErrorCode(err error) string {
	var e callError
	if errors.As(err, &e) {
		return e.code
	}
	return "model_error"
}

func ContextFailure(ctx context.Context) error {
	if ctx.Err() == nil {
		return nil
	}
	return classify(ctx, ctx.Err(), 0, "model_error")
}

func KnownErrorCode(code string) bool {
	switch code {
	case "ok", "model_error", "model_deadline_exceeded", "model_cancelled", "upstream_redirect_denied", "model_destination_denied", "model_free_policy_denied", "model_attempt_denied", "model_policy_changed", "model_context_limit", "model_content_filter", "model_tls_verification_failed", "model_stream_truncated", "model_transport_timeout", "model_output_token_limit", "model_response_incomplete", "model_request_failed", "model_stream_failed", "model_invalid_reply", "model_reply_limit", "model_invalid_usage", "model_cancelled_or_deadline":
		return true
	}
	for _, prefix := range []string{"upstream_http_", "upstream_event_http_"} {
		if strings.HasPrefix(code, prefix) {
			n, err := strconv.Atoi(strings.TrimPrefix(code, prefix))
			return err == nil && n >= 400 && n <= 599 && strconv.Itoa(n) == strings.TrimPrefix(code, prefix)
		}
	}
	return false
}

// classify projects observed conditions into our vocabulary. It never returns
// vendor messages, correlation IDs, request data, or free-form stop details.
func classify(ctx context.Context, err error, status int, fallback string) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return callError{"model_deadline_exceeded"}
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return callError{"model_cancelled"}
	}
	if status >= 300 && status < 400 {
		return callError{"upstream_redirect_denied"}
	}
	if status >= 400 && status <= 599 {
		return callError{fmt.Sprintf("upstream_http_%d", status)}
	}
	var own callError
	if errors.As(err, &own) {
		return own
	}
	var metadata port.ProviderErrorMetadataError
	if errors.As(err, &metadata) {
		switch metadata.ProviderErrorCode() {
		case "context_length_exceeded":
			return callError{"model_context_limit"}
		case "content_filter", "content_policy_violation":
			return callError{"model_content_filter"}
		}
		if code := metadata.ProviderInBandStatus(); code >= 400 && code <= 599 {
			return callError{fmt.Sprintf("upstream_event_http_%d", code)}
		}
	}
	var verification *tls.CertificateVerificationError
	if errors.As(err, &verification) {
		return callError{"model_tls_verification_failed"}
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return callError{"model_stream_truncated"}
	}
	var network net.Error
	if errors.As(err, &network) && network.Timeout() {
		return callError{"model_transport_timeout"}
	}
	if err != nil {
		// The pinned adapter exposes incomplete reasons as these fixed display
		// projections rather than typed metadata. Compare only known projections;
		// never persist the provider-controlled input that produced them.
		switch err.Error() {
		case "the response was cut off at the provider's max output token limit (reason: max_output_tokens).":
			return callError{"model_output_token_limit"}
		case "response incomplete: unknown reason":
			return callError{"model_response_incomplete"}
		}
		if strings.HasPrefix(err.Error(), "the provider's content filter blocked this response (reason: content_filter). ") {
			return callError{"model_content_filter"}
		}
	}
	return callError{fallback}
}
