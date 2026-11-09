package platform

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/Sanskarzz/tsunami/internal/toolcontract"
	bolt "go.etcd.io/bbolt"
)

const (
	MaxActivityEvents     = 256
	MaxActivityEventBytes = 2048
	MaxActivityPage       = 100
)

// Activity is a bounded metadata journal, separate from legacy free-text Events.
// It shares the run's transaction; no consumer or external telemetry service is
// involved in admission, validation, or revocation.
type Activity struct {
	TrackedFromCreation bool            `json:"tracked_from_creation"`
	LastSequence        uint64          `json:"last_sequence"`
	Items               []ActivityEvent `json:"items"`
}

type ActivityEvent struct {
	SchemaVersion   int          `json:"schema_version"`
	Sequence        uint64       `json:"sequence"`
	RecordedAt      time.Time    `json:"recorded_at"`
	Kind            string       `json:"kind"`
	Source          string       `json:"source"`
	EngagementID    string       `json:"engagement_id,omitempty"`
	RunID           string       `json:"run_id"`
	ProcedureDigest string       `json:"procedure_digest,omitempty"`
	OperationID     string       `json:"operation_id,omitempty"`
	CaptureID       string       `json:"capture_id,omitempty"`
	Data            ActivityData `json:"data"`
}

// Only trusted call sites construct these fields. No URL, account alias, error
// text, target body, prompt, header, or model/worker-supplied message belongs here.
type ActivityData struct {
	ExecutionMode  string            `json:"execution_mode,omitempty"`
	Executor       string            `json:"executor,omitempty"`
	RequestLimit   int               `json:"request_limit,omitempty"`
	Deadline       *time.Time        `json:"deadline,omitempty"`
	ReadIndex      int               `json:"read_index,omitempty"`
	AccountSlot    int               `json:"account_slot,omitempty"`
	Reservation    int               `json:"reservation,omitempty"`
	Stage          string            `json:"stage,omitempty"`
	Code           string            `json:"code,omitempty"`
	ObservedAt     *time.Time        `json:"observed_at,omitempty"`
	Status         int               `json:"status,omitempty"`
	DurationMillis int64             `json:"duration_ms,omitempty"`
	Outcome        string            `json:"outcome,omitempty"`
	Verdict        string            `json:"verdict,omitempty"`
	Validation     *ValidationCounts `json:"validation,omitempty"`
	EvidenceIDs    []string          `json:"evidence_ids,omitempty"`
}

type ValidationCounts struct {
	Required      int  `json:"required"`
	Verified      int  `json:"verified"`
	ProbeCaptured bool `json:"probe_captured"`
}

var activityDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)

func activityRef(id string) string {
	if runCursor.MatchString(id) {
		return id
	}
	return ""
}

// appendActivity never rejects a security transition because retention is full.
// Oldest records are evicted; readers receive explicit gap/retention metadata.
func appendActivity(r *Run, kind, op, capture string, data ActivityData) {
	source := ""
	switch kind {
	case "run.created", "permission.revoked", "activity.drained_locally":
		source = "control_plane"
	case "execution.dispatch_requested", "execution.worker_observed", "execution.exit_observed", "execution.failed", "cleanup.delete_accepted", "cleanup.resources_absent":
		source = "controller"
	case "request.admitted", "request.sent", "request.rejected", "request.capture_persisted":
		source = "broker"
	case "validation.completed":
		source = "validator"
	default:
		return
	}
	if r.Activity == nil {
		r.Activity = &Activity{Items: []ActivityEvent{}}
	}
	r.Activity.LastSequence++
	e := ActivityEvent{SchemaVersion: 1, Sequence: r.Activity.LastSequence, RecordedAt: time.Now().UTC(), Kind: kind, Source: source,
		RunID: activityRef(r.ID), EngagementID: activityRef(r.EngagementID), OperationID: activityRef(op), CaptureID: activityRef(capture), Data: data}
	if digest := r.ProcedureDigest(); activityDigest.MatchString(digest) {
		e.ProcedureDigest = digest
	}
	// Fields above are closed codes, numbers, timestamps and bounded references.
	// Defensively preserve the transition even if a future producer violates the
	// byte contract. Its details then remain explicitly unavailable.
	if raw, _ := json.Marshal(e); len(raw) > MaxActivityEventBytes {
		e.Data = ActivityData{Code: "details_unavailable"}
	}
	r.Activity.Items = append(r.Activity.Items, e)
	if len(r.Activity.Items) > MaxActivityEvents {
		r.Activity.Items = append([]ActivityEvent(nil), r.Activity.Items[len(r.Activity.Items)-MaxActivityEvents:]...)
	}
}

func activityCreated(r *Run) {
	mode := "unknown"
	if r.Execution == "scripted" || r.Execution == "live" {
		mode = r.Execution
	}
	executor := "unknown"
	switch r.Executor {
	case "", "native":
		executor = "native"
	case toolcontract.CurlHTTP:
		executor = toolcontract.CurlHTTP
	}
	r.Activity = &Activity{TrackedFromCreation: true, Items: []ActivityEvent{}}
	appendActivity(r, "run.created", "", "", ActivityData{ExecutionMode: mode, Executor: executor, RequestLimit: r.MaxRequests, Deadline: &r.Deadline})
}

// References resolve to the first matching reviewed read (1-based), not a claim
// that the prescribed workflow order or exactly-once execution was enforced.
// Account slots are ordered by first appearance; aliases never leave the broker.
func activityRead(r Run, account, path string) (index, slot int) {
	type pair struct{ account, path string }
	reads := []pair{}
	if r.Application != nil {
		for _, read := range r.Application.Reads() {
			reads = append(reads, pair{read.Account, read.Path})
		}
	} else if r.Assessment != nil {
		for _, read := range r.Assessment.Reads() {
			reads = append(reads, pair{read.Account, read.Path})
		}
	} else {
		for _, alias := range []string{"alice", "bob"} {
			for _, p := range []string{"/api/documents/doc-alice", "/api/documents/doc-bob", "/api/documents/unknown", "/api/slow", "/api/redirect"} {
				reads = append(reads, pair{alias, p})
			}
		}
	}
	slots := map[string]int{}
	for i, read := range reads {
		if slots[read.account] == 0 {
			slots[read.account] = len(slots) + 1
		}
		if read.account == account && read.path == path {
			return i + 1, slots[account]
		}
	}
	return 0, 0
}

type ActivityPage struct {
	Items               []ActivityEvent `json:"items"`
	NextCursor          uint64          `json:"next_cursor"`
	OldestSequence      uint64          `json:"oldest_sequence"`
	LatestSequence      uint64          `json:"latest_sequence"`
	HasMore             bool            `json:"has_more"`
	HistoryTruncated    bool            `json:"history_truncated"`
	Gap                 bool            `json:"gap"`
	TrackedFromCreation bool            `json:"tracked_from_creation"`
}

var errActivityCursor = errors.New("invalid activity pagination")

func (s *Store) ReadActivity(tenant, id string, after uint64, limit int) (ActivityPage, error) {
	page := ActivityPage{Items: []ActivityEvent{}, NextCursor: after}
	if tenant == "" || limit < 1 || limit > MaxActivityPage {
		return page, errActivityCursor
	}
	err := s.db.View(func(tx *bolt.Tx) error {
		// Project only identity and the typed journal. Never decode/return legacy
		// event text or capture bodies on this route.
		var stored struct {
			Tenant   string
			Activity *Activity
		}
		raw := tx.Bucket([]byte("runs")).Get([]byte(id))
		if raw == nil {
			return ErrNotFound
		}
		if err := json.Unmarshal(raw, &stored); err != nil {
			return err
		}
		if stored.Tenant != tenant {
			return ErrNotFound
		}
		if stored.Activity != nil {
			a := stored.Activity
			page.LatestSequence = a.LastSequence
			page.TrackedFromCreation = a.TrackedFromCreation
			if len(a.Items) > 0 {
				page.OldestSequence = a.Items[0].Sequence
				page.HistoryTruncated = page.OldestSequence > 1
				page.Gap = after < page.OldestSequence-1
			}
			for _, item := range a.Items {
				if item.Sequence <= after {
					continue
				}
				if len(page.Items) == limit {
					page.HasMore = true
					break
				}
				page.Items = append(page.Items, item)
				page.NextCursor = item.Sequence
			}
		}
		if after > page.LatestSequence {
			return errActivityCursor
		}
		return nil
	})
	return page, err
}

func (s *Server) serveActivity(w http.ResponseWriter, r *http.Request, tenant, id string) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	q, err := urlActivityQuery(r)
	if err != nil {
		problem(w, 400, errActivityCursor)
		return
	}
	page, err := s.Store.ReadActivity(tenant, id, q.after, q.limit)
	if errors.Is(err, errActivityCursor) {
		problem(w, 400, err)
		return
	}
	if errors.Is(err, ErrNotFound) {
		problem(w, 404, ErrNotFound)
		return
	}
	if err != nil {
		problem(w, 500, errors.New("activity unavailable"))
		return
	}
	writeJSON(w, 200, page)
}

type activityQuery struct {
	after uint64
	limit int
}

func urlActivityQuery(r *http.Request) (activityQuery, error) {
	// ParseQuery errors (including malformed escapes) must not be discarded.
	query, err := url.ParseQuery(r.URL.RawQuery)
	q := activityQuery{limit: 50}
	if err != nil {
		return q, errActivityCursor
	}
	for key, values := range query {
		if len(values) != 1 || (key != "after" && key != "limit") {
			return q, errActivityCursor
		}
		raw := values[0]
		if raw == "" || len(raw) > 16 {
			return q, errActivityCursor
		}
		for _, c := range raw {
			if c < '0' || c > '9' {
				return q, errActivityCursor
			}
		}
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || n > 9007199254740991 {
			return q, errActivityCursor
		}
		if key == "after" {
			q.after = n
		} else {
			if n < 1 || n > MaxActivityPage {
				return q, errActivityCursor
			}
			q.limit = int(n)
		}
	}
	return q, nil
}
