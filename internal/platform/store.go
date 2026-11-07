package platform

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/assessment"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

var ErrNotFound = errors.New("run not found")

type Store struct{ db *bolt.DB }

func OpenStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range []string{"runs", "credentials", "gates", "engagements", "finding_reviews", "review_history", "onboarding", "onboarding_secrets", "onboarding_evidence", "onboarded_apps"} {
			if _, e := tx.CreateBucketIfNotExists([]byte(name)); e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping() error { return s.db.View(func(*bolt.Tx) error { return nil }) }

func (s *Store) Fail(id string) error {
	return s.Update(id, func(r *Run) error {
		if terminal(r.State) {
			return nil
		}
		now := time.Now().UTC()
		r.State = Failed
		r.FinishedAt = &now
		r.Result = &Result{Verdict: "inconclusive", Title: "Assessment incomplete", Reason: "Worker ended before trusted validation; no automatic replay.", EvidenceIDs: []string{}}
		r.Events = append(r.Events, Event{now, "worker_failed", "no automatic retry of uncertain requests"})
		return nil
	})
}

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (s *Store) Create(tenant, scenario, target, mode string) (Run, error) {
	return s.create(tenant, scenario, target, mode, nil, "", createOptions{})
}

func (s *Store) CreatePrivate(tenant, scenario, target, mode string) (Run, error) {
	return s.create(tenant, scenario, target, mode, nil, "private", createOptions{})
}

func (s *Store) CreateLive(tenant, scenario, target, mode string, model ModelUsage) (Run, error) {
	return s.create(tenant, scenario, target, mode, &model, "", createOptions{})
}

func (s *Store) CreateAssessment(tenant, scenario, target, mode, connectivity, id string, model *ModelUsage) (Run, error) {
	d, err := assessment.Get(id)
	if err != nil {
		return Run{}, err
	}
	return s.create(tenant, scenario, target, mode, model, connectivity, createOptions{Definition: &d})
}

type createOptions struct {
	Definition   *assessment.Definition
	EngagementID string
	Application  *application.Plan
}

func (s *Store) create(tenant, scenario, target, mode string, model *ModelUsage, connectivity string, options createOptions) (Run, error) {
	now := time.Now().UTC()
	r := Run{ID: randomID(), Tenant: tenant, Scenario: scenario, State: Pending,
		Lifecycle: &Lifecycle{Version: 1, FromCreation: true},
		Target:    target, Mode: mode, CreatedAt: now, Deadline: now.Add(90 * time.Second),
		Execution: "scripted", Connectivity: connectivity, Workflow: "object-ownership/v1", Model: model,
		MaxRequests: 12, Captures: []Capture{}, Events: []Event{{now, "created", "server-owned lab policy; no arbitrary target selection"}}}
	if model != nil {
		r.Execution = "live"
	}
	if options.Definition != nil {
		d := *options.Definition
		if !d.Canonical() {
			return Run{}, errors.New("assessment definition changed")
		}
		r.Assessment = &d
		r.AssessmentDigest = d.Digest()
		r.Workflow = d.ID
		r.MaxRequests = d.MaxRequests
	}
	if options.Application != nil {
		if model != nil || options.Definition != nil || options.Application.Validate() != nil {
			return Run{}, errors.New("unsupported application execution")
		}
		p := *options.Application
		r.Application = &p
		r.ApplicationDigest = p.Digest()
		r.Workflow = p.ProcedureID()
		r.MaxRequests = p.MaxRequests()
	}
	err := s.db.Update(func(tx *bolt.Tx) error {
		g := tx.Bucket([]byte("gates"))
		if g.Get([]byte("global")) != nil || g.Get([]byte("tenant:"+tenant)) != nil {
			return errors.New("new runs are disabled")
		}
		if options.EngagementID != "" {
			if err := bindEngagementRun(tx, &r, options.EngagementID); err != nil {
				return err
			}
		}
		data, e := json.Marshal(r)
		if e != nil {
			return e
		}
		if e = tx.Bucket([]byte("runs")).Put([]byte(r.ID), data); e != nil {
			return e
		}
		return tx.Bucket([]byte("credentials")).Put([]byte(r.ID), []byte(randomID()+randomID()))
	})
	return r, err
}

// StopGroup updates the gate and every affected run in one durable transaction.
func (s *Store) StopGroup(tenant string, global bool) ([]string, error) {
	var ids []string
	err := s.db.Update(func(tx *bolt.Tx) error {
		key := "tenant:" + tenant
		if global {
			key = "global"
		}
		if err := tx.Bucket([]byte("gates")).Put([]byte(key), []byte("disabled")); err != nil {
			return err
		}
		b := tx.Bucket([]byte("runs"))
		updates := map[string][]byte{}
		if err := b.ForEach(func(k, v []byte) error {
			var r Run
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			if !terminal(r.State) && (global || r.Tenant == tenant) {
				now := time.Now().UTC()
				r.State = Stopped
				r.StopAcceptedAt = &now
				reason := "tenant_stop"
				if global {
					reason = "global_stop"
				}
				revoke(&r, reason, now)
				r.Events = append(r.Events, Event{now, "stopped", "durable permission revoked"})
				data, err := json.Marshal(r)
				if err != nil {
					return err
				}
				updates[string(k)] = data
				ids = append(ids, r.ID)
			}
			return nil
		}); err != nil {
			return err
		}
		for k, v := range updates {
			if err := b.Put([]byte(k), v); err != nil {
				return err
			}
		}
		return nil
	})
	return ids, err
}

func (s *Store) Get(id string) (Run, error) {
	var r Run
	err := s.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte("runs")).Get([]byte(id))
		if data == nil {
			return ErrNotFound
		}
		return json.Unmarshal(data, &r)
	})
	return r, err
}

func (s *Store) Update(id string, fn func(*Run) error) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("runs"))
		data := b.Get([]byte(id))
		if data == nil {
			return ErrNotFound
		}
		var r Run
		if err := json.Unmarshal(data, &r); err != nil {
			return err
		}
		if err := fn(&r); err != nil {
			return err
		}
		data, err := json.Marshal(r)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), data)
	})
}

func (s *Store) List() ([]Run, error) {
	runs := []Run{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("runs")).ForEach(func(_, v []byte) error {
			var r Run
			if err := json.Unmarshal(v, &r); err != nil {
				return fmt.Errorf("invalid stored run: %w", err)
			}
			runs = append(runs, r)
			return nil
		})
	})
	return runs, err
}

// Credentials are never included in the public Run representation or evidence.
// This local prototype stores them in its protected database, not a secret manager.
func (s *Store) Credential(id string) (string, error) {
	var token string
	err := s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket([]byte("credentials")).Get([]byte(id))
		if v == nil {
			return ErrNotFound
		}
		token = string(v)
		return nil
	})
	return token, err
}
