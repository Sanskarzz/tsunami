package inference

import "time"

// CallRecord contains only gateway-owned metadata. No prompt, tool arguments,
// response text, headers, provider IDs, or raw error messages belong here.
type CallRecord struct {
	Sequence           int        `json:"sequence"`
	StartedAt          time.Time  `json:"started_at"`
	Deadline           time.Time  `json:"deadline"`
	DeadlineSource     string     `json:"deadline_source,omitempty"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	ElapsedMS          int64      `json:"elapsed_ms"`
	HTTPStatus         int        `json:"upstream_http_status,omitempty"`
	HeadersMS          *int64     `json:"headers_ms,omitempty"`
	FirstChunkMS       *int64     `json:"first_chunk_ms,omitempty"`
	Chunks             int        `json:"chunks"`
	BufferedToolCalls  int        `json:"buffered_tool_calls"`
	ReleasedToolCalls  int        `json:"released_tool_calls"`
	WireAttempts       int        `json:"wire_attempts"`
	UsageChunkObserved bool       `json:"usage_chunk_observed"`
	UsageAccepted      bool       `json:"usage_accepted"`
	Outcome            string     `json:"outcome"`
	Code               string     `json:"code"`
	Phase              string     `json:"phase"`
}

// Observation is local to one synchronous adapter call. The store gets a copy
// after the call; concurrent workers never share this object.
type Observation struct {
	Record CallRecord
	start  time.Time
}

func NewObservation(sequence int, deadline time.Time) *Observation {
	now := time.Now()
	return &Observation{Record: CallRecord{Sequence: sequence, StartedAt: now.UTC(), Deadline: deadline.UTC(), Outcome: "pending", Code: "unknown", Phase: "before_headers"}, start: now}
}

func (o *Observation) headers(status int) {
	ms := time.Since(o.start).Milliseconds()
	o.Record.HTTPStatus = status
	o.Record.HeadersMS = &ms
	o.Record.Phase = "awaiting_chunk"
}

func (o *Observation) chunk() {
	if o.Record.FirstChunkMS == nil {
		ms := time.Since(o.start).Milliseconds()
		o.Record.FirstChunkMS = &ms
	}
	o.Record.Chunks++
	o.Record.Phase = "reading_chunks"
}

func (o *Observation) Finish(err error) {
	now := time.Now().UTC()
	o.Record.FinishedAt = &now
	o.Record.ElapsedMS = time.Since(o.start).Milliseconds()
	if err == nil {
		o.Record.Outcome = "succeeded"
		o.Record.Code = "ok"
		o.Record.Phase = "complete"
		o.Record.ReleasedToolCalls = o.Record.BufferedToolCalls
	} else {
		o.Record.Outcome = "failed"
		o.Record.Code = ErrorCode(err)
		o.Record.ReleasedToolCalls = 0
		o.Record.UsageAccepted = false
	}
}
