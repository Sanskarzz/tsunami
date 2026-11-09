package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/assessment"

	bolt "go.etcd.io/bbolt"
)

// StartInput identifies one operator action. A different attempt needs a new
// key; transport recovery must reuse the original key and request fields.
type StartInput struct {
	IdempotencyKey      string `json:"idempotency_key"`
	Assessment          string `json:"assessment"`
	Execution           string `json:"execution"`
	RequiredProvider    string `json:"required_provider,omitempty"`
	RequiredPricePolicy string `json:"required_price_policy,omitempty"`
}

type startRecord struct {
	RequestDigest string `json:"request_digest"`
	RunID         string `json:"run_id"`
}

// The same action can be recovered after evidence exists. Return only launch
// metadata/public scope, never capture bodies, engine/model traces or events.
type engagementStartView struct {
	ID                string                 `json:"id"`
	EngagementID      string                 `json:"engagement_id"`
	Tenant            string                 `json:"tenant"`
	Target            string                 `json:"target"`
	State             string                 `json:"state"`
	Execution         string                 `json:"execution"`
	Executor          string                 `json:"executor,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	Deadline          time.Time              `json:"deadline"`
	Application       *application.Plan      `json:"application,omitempty"`
	ApplicationDigest string                 `json:"application_digest,omitempty"`
	Assessment        *assessment.Definition `json:"assessment,omitempty"`
	AssessmentDigest  string                 `json:"assessment_digest,omitempty"`
	RequestCount      int                    `json:"request_count"`
	MaxRequests       int                    `json:"max_requests"`
}

func startView(r Run) engagementStartView {
	return engagementStartView{ID: r.ID, EngagementID: r.EngagementID, Tenant: r.Tenant,
		Target: r.Target, State: r.State, Execution: r.Execution, Executor: r.Executor,
		CreatedAt: r.CreatedAt, Deadline: r.Deadline, Application: r.Application,
		ApplicationDigest: r.ApplicationDigest, Assessment: r.Assessment,
		AssessmentDigest: r.AssessmentDigest, RequestCount: r.RequestCount, MaxRequests: r.MaxRequests}
}

// NewStartKey names a new explicit action; retain it for transport recovery.
func NewStartKey() string { return randomID() }

func (in StartInput) normalized() (StartInput, error) {
	if len(in.IdempotencyKey) != 32 {
		return in, errors.New("idempotency_key must be 32 lowercase hexadecimal characters")
	}
	for _, c := range in.IdempotencyKey {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return in, errors.New("idempotency_key must be 32 lowercase hexadecimal characters")
		}
	}
	if in.Execution == "" {
		in.Execution = "scripted"
	}
	return in, nil
}

func startKey(tenant, id, key string) []byte {
	return []byte(tenant + "\x00" + id + "\x00" + key)
}

func startDigest(e Engagement, in StartInput) string {
	in.IdempotencyKey = "" // The key names the action; these bytes bind its intent.
	data, _ := json.Marshal(struct {
		Scope string
		Input StartInput
	}{e.ScopeDigest, in})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func lookupStart(tx *bolt.Tx, tenant, id string, in StartInput) (Run, bool, error) {
	e, err := getEngagement(tx, tenant, id)
	if err != nil {
		return Run{}, false, err
	}
	data := tx.Bucket([]byte("engagement_starts")).Get(startKey(tenant, id, in.IdempotencyKey))
	if data == nil {
		return Run{}, false, nil
	}
	var record startRecord
	if json.Unmarshal(data, &record) != nil || e.digest() != e.ScopeDigest || e.ApprovedDigest != e.ScopeDigest || !slices.Contains(e.RunIDs, record.RunID) {
		return Run{}, true, errors.New("Start recovery state unavailable")
	}
	if record.RequestDigest != startDigest(e, in) {
		return Run{}, true, errors.New("idempotency key already used for a different Start request")
	}
	run, err := getBoundRun(tx, e, record.RunID)
	return run, true, err
}

// LookupEngagementStart permits read-only recovery before checking current
// execution-route health, expiry or Stop gates. It never grants permission.
func (s *Store) LookupEngagementStart(tenant, id string, in StartInput) (Run, bool, error) {
	in, err := in.normalized()
	if err != nil {
		return Run{}, false, err
	}
	var run Run
	var found bool
	err = s.db.View(func(tx *bolt.Tx) error {
		var err error
		run, found, err = lookupStart(tx, tenant, id, in)
		return err
	})
	return run, found, err
}

// StartEngagementRun coalesces concurrent identical requests in the same
// transaction as child reservation, run state and worker credential creation.
func (s *Store) StartEngagementRun(tenant, id string, in StartInput, model *ModelUsage) (Run, error) {
	in, err := in.normalized()
	if err != nil {
		return Run{}, err
	}
	if previous, found, err := s.LookupEngagementStart(tenant, id, in); err != nil || found {
		return previous, err
	}
	if in.Execution == "scripted" {
		if model != nil || in.RequiredProvider != "" || in.RequiredPricePolicy != "" {
			return Run{}, errors.New("model policy supplied for scripted run")
		}
	} else if in.Execution != "live" || model == nil || model.Provider != in.RequiredProvider || model.PricePolicy != in.RequiredPricePolicy {
		return Run{}, errors.New("explicit matching model policy required")
	}
	return s.createEngagementRun(tenant, id, in.Assessment, model, &in)
}
