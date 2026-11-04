package platform

import (
	"context"
	"errors"
	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/assessment"
	"github.com/Sanskarzz/tsunami/internal/inference"
	"net/http"
	"time"
)

var ErrWorkerFailed = errors.New("worker Job failed")

const (
	Pending   = "pending"
	Running   = "running"
	Completed = "completed"
	Stopped   = "stopped"
	Failed    = "failed"
)

// Runs carry server-owned policy. Agents cannot supply targets, tenant IDs,
// ownership expectations, budgets, or a confirmed verdict.
type Run struct {
	Executor          string            `json:"executor,omitempty"`
	EngagementID      string            `json:"engagement_id,omitempty"`
	Application       *application.Plan `json:"application,omitempty"`
	ApplicationDigest string            `json:"application_digest,omitempty"`
	// Ephemeral dispatch capability, granted only by the pending->running CAS.
	// Existing/reread runs can adopt a Job but cannot create a replacement.
	AllowJobCreation    bool                   `json:"-"`
	ID                  string                 `json:"id"`
	Tenant              string                 `json:"tenant"`
	Scenario            string                 `json:"scenario"`
	State               string                 `json:"state"`
	Target              string                 `json:"target"`
	Mode                string                 `json:"mode"`
	Execution           string                 `json:"execution,omitempty"`
	Connectivity        string                 `json:"connectivity,omitempty"`
	Workflow            string                 `json:"workflow,omitempty"`
	Assessment          *assessment.Definition `json:"assessment,omitempty"`
	AssessmentDigest    string                 `json:"assessment_digest,omitempty"`
	CompletionFeedback  int                    `json:"completion_feedback_count,omitempty"`
	Model               *ModelUsage            `json:"model,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	Deadline            time.Time              `json:"deadline"`
	StopAcceptedAt      *time.Time             `json:"stop_accepted_at,omitempty"`
	FinishedAt          *time.Time             `json:"finished_at,omitempty"`
	DeletionRequestedAt *time.Time             `json:"deletion_requested_at,omitempty"`
	RequestCount        int                    `json:"request_count"`
	MaxRequests         int                    `json:"max_requests"`
	ActiveRequests      int                    `json:"active_requests"`
	ActiveModelRequests int                    `json:"active_model_requests,omitempty"`
	Captures            []Capture              `json:"captures"`
	Events              []Event                `json:"events"`
	Result              *Result                `json:"result,omitempty"`
}

// Reservations are durable and are never refunded after an ambiguous network
// failure. Token usage is provider-reported, not a monetary billing guarantee.
type ModelUsage struct {
	PricePolicy     string                 `json:"price_policy,omitempty"`
	Provider        string                 `json:"provider"`
	Name            string                 `json:"name"`
	ConfigID        string                 `json:"config_id"`
	MaxCalls        int                    `json:"max_calls"`
	MaxOutputTokens int                    `json:"max_output_tokens"`
	CallsReserved   int                    `json:"calls_reserved"`
	WireAttempts    int                    `json:"wire_attempts"`
	StreamsOpened   int                    `json:"streams_opened"`
	CompletedCalls  int                    `json:"completed_calls"`
	InputTokens     int                    `json:"reported_input_tokens"`
	OutputTokens    int                    `json:"reported_output_tokens"`
	UsageUncertain  bool                   `json:"usage_uncertain"`
	Calls           []inference.CallRecord `json:"calls,omitempty"`
}

type Event struct {
	At     time.Time `json:"at"`
	Kind   string    `json:"kind"`
	Detail string    `json:"detail"`
}

type Capture struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	Account    string    `json:"account"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Status     int       `json:"status,omitempty"`
	Peer       string    `json:"peer,omitempty"`
	Body       string    `json:"body,omitempty"`
	Blocked    bool      `json:"blocked,omitempty"`
	Error      string    `json:"error,omitempty"`
}

type Result struct {
	Verdict     string             `json:"verdict"`
	Title       string             `json:"title"`
	Reason      string             `json:"reason"`
	EvidenceIDs []string           `json:"evidence_ids"`
	Validation  *ValidationDetails `json:"validation,omitempty"`
}

type Request struct {
	URL     string `json:"url"`
	Account string `json:"account"`
}

type Document struct {
	ID      string `json:"id"`
	Owner   string `json:"owner"`
	Content string `json:"content"`
}

type Target struct {
	URL    string
	Policy *Policy
	// A private target has no direct-network fallback. The trusted connector
	// callback opens each connection and returns the actual target peer.
	Forward func(context.Context, Run, *http.Request) (*http.Response, string, error)
}

func terminal(s string) bool { return s == Stopped || s == Completed || s == Failed }
