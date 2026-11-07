package platform

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"time"

	"github.com/Sanskarzz/tsunami/internal/application"
	setup "github.com/Sanskarzz/tsunami/internal/onboarding"
	bolt "go.etcd.io/bbolt"
)

var errSetupConflict = errors.New("setup changed, expired, stopped or already has an assessment; reload its current revision")

type setupDisk struct {
	Session setup.Session `json:"session"`
	Process string        `json:"process,omitempty"`
}
type setupAccess struct {
	Origin string            `json:"origin"`
	Tokens map[string]string `json:"tokens"`
}
type dynamicApp struct {
	Plan      application.Plan `json:"plan"`
	SessionID string           `json:"session_id"`
}

func setupKey(tenant, id string) []byte { return []byte(tenant + "\x00" + id) }
func setupAAD(tenant, id, kind string) []byte {
	return []byte("tsunami/setup/v1/" + tenant + "/" + id + "/" + kind)
}
func sealSetup(key, aad []byte, value any) ([]byte, error) {
	block, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	g, e := cipher.NewGCM(block)
	if e != nil {
		return nil, e
	}
	n := make([]byte, g.NonceSize())
	if _, e = rand.Read(n); e != nil {
		return nil, e
	}
	raw, e := json.Marshal(value)
	if e != nil {
		return nil, e
	}
	return g.Seal(n, n, raw, aad), nil
}
func openSetup(key, aad, raw []byte, value any) error {
	block, e := aes.NewCipher(key)
	if e != nil {
		return e
	}
	g, e := cipher.NewGCM(block)
	if e != nil {
		return e
	}
	if len(raw) < g.NonceSize() {
		return errors.New("encrypted setup material unavailable")
	}
	b, e := g.Open(nil, raw[:g.NonceSize()], raw[g.NonceSize():], aad)
	if e != nil {
		return errors.New("encrypted setup material unavailable")
	}
	return json.Unmarshal(b, value)
}
func getSetup(tx *bolt.Tx, tenant, id string) (setup.Session, error) {
	raw := tx.Bucket([]byte("onboarding")).Get(setupKey(tenant, id))
	var disk setupDisk
	if raw == nil {
		return setup.Session{}, ErrNotFound
	}
	if json.Unmarshal(raw, &disk) != nil || disk.Session.Tenant != tenant || disk.Session.ID != id {
		return setup.Session{}, errors.New("setup state unavailable")
	}
	disk.Session.ValidationProcess = disk.Process
	return disk.Session, nil
}
func putSetup(tx *bolt.Tx, s setup.Session) error {
	b, e := json.Marshal(setupDisk{Session: s, Process: s.ValidationProcess})
	if e != nil {
		return e
	}
	return tx.Bucket([]byte("onboarding")).Put(setupKey(s.Tenant, s.ID), b)
}
func (s *Store) GetSetup(tenant, id string) (setup.Session, error) {
	var out setup.Session
	e := s.db.View(func(tx *bolt.Tx) error { var e error; out, e = getSetup(tx, tenant, id); return e })
	return out, e
}
func (s *Store) ListSetup(tenant string) ([]setup.Session, error) {
	out := []setup.Session{}
	e := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("onboarding")).ForEach(func(_, raw []byte) error {
			var d setupDisk
			if json.Unmarshal(raw, &d) != nil {
				return errors.New("setup state unavailable")
			}
			if d.Session.Tenant == tenant {
				out = append(out, d.Session)
			}
			return nil
		})
	})
	return out, e
}
func (s *Store) CreateSetup(tenant string, d setup.Details) (setup.Session, error) {
	if e := d.Validate(); e != nil {
		return setup.Session{}, e
	}
	now := time.Now().UTC()
	out := setup.Session{ID: randomID(), Tenant: tenant, Revision: 1, Details: d, State: "draft", CreatedAt: now, ExpiresAt: now.Add(time.Hour), AdvisorMode: "checked_workflow", Messages: []setup.Message{{Role: "assistant", Text: "I will review the supplied information and ask only for missing prerequisites. No target requests or security tests have started. This setup advisor uses checked rules, not a live model.", At: now}}}
	out.Advance()
	e := s.db.Update(func(tx *bolt.Tx) error {
		n := 0
		if e := tx.Bucket([]byte("onboarding")).ForEach(func(_, raw []byte) error {
			var d setupDisk
			if json.Unmarshal(raw, &d) != nil {
				return errors.New("setup state unavailable")
			}
			if d.Session.Tenant == tenant {
				n++
			}
			return nil
		}); e != nil {
			return e
		}
		if n >= 64 {
			return errors.New("lab tenant setup allowance reached; preserve history through explicit maintenance")
		}
		return putSetup(tx, out)
	})
	return out, e
}
func editableSetup(s setup.Session, revision int) bool {
	return s.Revision == revision && s.Revision < 128 && s.State != "stopped" && s.State != "validating" && s.EngagementID == "" && time.Now().Before(s.ExpiresAt)
}
func (s *Store) changeSetup(tenant, id string, revision int, fn func(*bolt.Tx, *setup.Session) error) (setup.Session, error) {
	var out setup.Session
	e := s.db.Update(func(tx *bolt.Tx) error {
		var e error
		out, e = getSetup(tx, tenant, id)
		if e != nil {
			return e
		}
		if !editableSetup(out, revision) {
			return errSetupConflict
		}
		if e = fn(tx, &out); e != nil {
			return e
		}
		out.Revision++
		out.Advance()
		return putSetup(tx, out)
	})
	return out, e
}
func (s *Store) getSetupAccess(tenant, id string, key []byte) (setupAccess, error) {
	var a setupAccess
	e := s.db.View(func(tx *bolt.Tx) error {
		return openSetup(key, setupAAD(tenant, id, "access"), tx.Bucket([]byte("onboarding_secrets")).Get(setupKey(tenant, id)), &a)
	})
	return a, e
}
func (s *Store) onboardedPlans(tenant string) ([]application.Plan, error) {
	out := []application.Plan{}
	e := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("onboarded_apps")).ForEach(func(k, b []byte) error {
			var a dynamicApp
			if json.Unmarshal(b, &a) != nil {
				return errors.New("application state unavailable")
			}
			session, e := getSetup(tx, tenant, a.SessionID)
			if e == ErrNotFound {
				return nil
			}
			if e != nil {
				return e
			}
			if string(k) != string(setupKey(tenant, a.Plan.ID)) {
				return nil
			}
			if session.State != "stopped" && time.Now().Before(session.ExpiresAt) {
				out = append(out, a.Plan)
			}
			return nil
		})
	})
	return out, e
}
func (s *Server) application(tenant, id string) (RegisteredApplication, bool) {
	if a, ok := s.Applications[tenant][id]; ok {
		return a, true
	}
	if len(s.SetupKey) != 32 {
		return RegisteredApplication{}, false
	}
	var c ApplicationConfig
	e := s.Store.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("onboarded_apps")).Get(setupKey(tenant, id))
		if b == nil {
			return ErrNotFound
		}
		var a dynamicApp
		if json.Unmarshal(b, &a) != nil {
			return ErrNotFound
		}
		session, e := getSetup(tx, tenant, a.SessionID)
		if e != nil || session.State == "stopped" || time.Now().After(session.ExpiresAt) {
			return ErrNotFound
		}
		if e = s.setupNetwork(tenant, session.Details); e != nil {
			return e
		}
		var access setupAccess
		if e = openSetup(s.SetupKey, setupAAD(tenant, a.SessionID, "access"), tx.Bucket([]byte("onboarding_secrets")).Get(setupKey(tenant, a.SessionID)), &access); e != nil || access.Origin != a.Plan.Origin {
			return errors.New("application access unavailable")
		}
		c = ApplicationConfig{Tenant: tenant, Plan: a.Plan, Credentials: access.Tokens}
		return nil
	})
	if e != nil {
		return RegisteredApplication{}, false
	}
	a, e := prepareApplication(c)
	return a, e == nil
}

// prepareSetup atomically registers a validated immutable application and its
// unapproved engagement. Retrying a lost response recovers the same ID.
func (s *Store) prepareSetup(tenant, id string, revision int, prepared RegisteredApplication) (setup.Session, error) {
	var out setup.Session
	e := s.db.Update(func(tx *bolt.Tx) error {
		var e error
		out, e = getSetup(tx, tenant, id)
		if e != nil {
			return e
		}
		if out.EngagementID != "" {
			return nil
		}
		if !editableSetup(out, revision) || out.State != "ready_for_approval" || out.Plan == nil || out.Plan.Digest() != prepared.Plan.Digest() {
			return errSetupConflict
		}
		eng, e := buildEngagement(tenant, prepared.Plan.Origin, EngagementInput{Application: prepared.Plan.ID, Name: out.Details.Name, AuthorizationRef: out.Details.AuthorizationRef, Executor: "curl-http/v1"}, prepared.Plan)
		if e != nil {
			return e
		}
		if e = putEngagementLimited(tx, eng); e != nil {
			return e
		}
		b, _ := json.Marshal(dynamicApp{Plan: prepared.Plan, SessionID: id})
		if e = tx.Bucket([]byte("onboarded_apps")).Put(setupKey(tenant, prepared.Plan.ID), b); e != nil {
			return e
		}
		out.EngagementID = eng.ID
		out.State = "assessment_created"
		out.Revision++
		out.Messages = append(out.Messages, setup.Message{Role: "assistant", Text: "The validated application and an unapproved assessment draft are saved. Review its exact scope, approve it, then explicitly Start. No security probe has run.", At: time.Now().UTC()})
		return putSetup(tx, out)
	})
	return out, e
}
