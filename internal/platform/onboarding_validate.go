package platform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/Sanskarzz/tsunami/internal/binding"
	setup "github.com/Sanskarzz/tsunami/internal/onboarding"
	bolt "go.etcd.io/bbolt"
)

func (s *Server) validateSetup(parent context.Context, tenant, id string, revision int) (setup.Session, error) {
	item, e := s.Store.GetSetup(tenant, id)
	if e != nil {
		return item, e
	}
	if !editableSetup(item, revision) || item.ValidationAttempts >= 3 {
		return item, errSetupConflict
	}
	if field, _ := item.Missing(); field != "" {
		return item, errors.New("answer missing setup prerequisites before validating")
	}
	if e = s.setupNetwork(tenant, item.Details); e != nil {
		return item, e
	}
	access, e := s.Store.getSetupAccess(tenant, id, s.SetupKey)
	if e != nil || access.Origin != item.Details.APIOrigin {
		return item, errors.New("protected access unavailable for this origin")
	}
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()
	attempt := randomID()
	s.mu.Lock()
	if s.shuttingDown {
		s.mu.Unlock()
		return item, errors.New("control plane is shutting down")
	}
	e = s.Store.db.Update(func(tx *bolt.Tx) error {
		v, err := getSetup(tx, tenant, id)
		if err != nil {
			return err
		}
		if !editableSetup(v, revision) || v.ValidationAttempts >= 3 {
			return errSetupConflict
		}
		g := tx.Bucket([]byte("gates"))
		if g.Get([]byte("global")) != nil || g.Get([]byte("tenant:"+tenant)) != nil {
			return errors.New("new work is disabled")
		}
		v.State = "validating"
		v.ValidationAttempts++
		v.ValidationRequests = 0
		v.ValidationID = attempt
		v.ValidationDeadline = &deadline
		v.ValidationProcess = s.processID
		v.Plan = nil
		v.ValidatedAt = nil
		v.ValidationCode = ""
		v.Revision++
		v.Question = nil
		item = v
		return putSetup(tx, v)
	})
	if e == nil {
		if s.setupActive == nil {
			s.setupActive = map[string]context.CancelFunc{}
		}
		s.setupActive[id] = cancel
	}
	s.mu.Unlock()
	if e != nil {
		return item, e
	}
	defer func() { s.mu.Lock(); delete(s.setupActive, id); s.mu.Unlock() }()
	addresses := []netip.Prefix{}
	for _, raw := range item.Details.ApprovedAddresses {
		ip, _ := netip.ParseAddr(raw)
		addresses = append(addresses, netip.PrefixFrom(ip, ip.BitLen()))
	}
	paths := []string{item.Details.IdentityPath, item.Details.IdentityPath, item.Details.OwnerPath, item.Details.PeerPath}
	accounts := []string{"owner", "peer", "owner", "peer"}
	policy, e := NewPolicy(item.Details.APIOrigin, addresses, nil)
	if e != nil {
		return item, e
	}
	client, closeTransport := policy.WithPaths(paths).HTTPClient()
	defer closeTransport()
	var docs [4]binding.Document
	captures := []Capture{}
	code := ""
	for i, path := range paths {
		if i > 0 {
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				code = "validation_cancelled_or_expired"
			case <-timer.C:
			}
			if code != "" {
				break
			}
		}
		e = s.Store.db.Update(func(tx *bolt.Tx) error {
			if ctx.Err() != nil {
				return errors.New("preflight context ended")
			}
			g := tx.Bucket([]byte("gates"))
			if g.Get([]byte("global")) != nil || g.Get([]byte("tenant:"+tenant)) != nil {
				return errors.New("preflight permission ended")
			}
			v, err := getSetup(tx, tenant, id)
			if err != nil {
				return err
			}
			if v.State != "validating" || v.ValidationID != attempt || v.ValidationRequests >= 4 || time.Now().After(deadline) || time.Now().After(v.ExpiresAt) {
				return errors.New("preflight permission ended")
			}
			v.ValidationRequests++
			return putSetup(tx, v)
		})
		if e != nil {
			code = "preflight_permission_ended"
			break
		}
		capture := Capture{ID: randomID(), URL: item.Details.APIOrigin + path, Account: accounts[i], StartedAt: time.Now().UTC()}
		u, err := policy.WithPaths(paths).Validate(capture.URL)
		if err != nil {
			code = "preflight_scope_refused"
			break
		}
		req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		req.Header.Set("Authorization", "Bearer "+access.Tokens[accounts[i]])
		req.Header.Set("Accept", "application/json")
		response, err := client.Do(req)
		if err != nil {
			capture.Error = "request failed or cancelled"
			capture.FinishedAt = time.Now().UTC()
			captures = append(captures, capture)
			code = "validation_connection_failed"
			break
		}
		capture.Status = response.StatusCode
		raw, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
		_ = response.Body.Close()
		capture.FinishedAt = time.Now().UTC()
		if err != nil || len(raw) > 64<<10 {
			capture.Error = "incomplete or oversized baseline"
			code = "validation_response_invalid"
		} else {
			capture.Body = string(raw)
		}
		if reflectsCredential(capture.Body, access.Tokens) {
			capture.Body = ""
			capture.Error = "response reflected an assigned credential"
			code = "baseline_reflected_credentials"
		}
		captures = append(captures, capture)
		if code != "" {
			break
		}
		if response.StatusCode != 200 {
			code = "baseline_not_successful"
			break
		} // Redirects never forward credentials.
		docs[i] = binding.Document{Status: response.StatusCode, Body: json.RawMessage(raw)}
	}
	var planRaw []byte
	if code == "" {
		p, err := setup.Infer(item, docs)
		if err != nil {
			code = "baseline_contract_ambiguous_or_unsupported"
		} else {
			p.MinIntervalMillis = 1000
			a, err := prepareApplication(ApplicationConfig{Tenant: tenant, Plan: p, Credentials: access.Tokens})
			if err != nil {
				code = "baseline_contract_invalid"
			} else {
				planRaw, _ = json.Marshal(a.Plan)
			}
		}
	}
	sealed, err := sealSetup(s.SetupKey, setupAAD(tenant, id, "evidence/"+attempt), captures)
	if err != nil {
		return item, errors.New("preflight evidence could not be protected")
	}
	e = s.Store.db.Update(func(tx *bolt.Tx) error {
		v, err := getSetup(tx, tenant, id)
		if err != nil {
			return err
		}
		if err = tx.Bucket([]byte("onboarding_evidence")).Put(setupKey(tenant, id+"/"+attempt), sealed); err != nil {
			return err
		}
		if v.State != "validating" || v.ValidationID != attempt {
			return errors.New("preflight revoked or replaced; captured responses were not promoted to readiness")
		}
		if ctx.Err() != nil || time.Now().After(deadline) || time.Now().After(v.ExpiresAt) {
			code = "validation_cancelled_or_expired"
		}
		v.State = "reviewing"
		v.Revision++
		v.ValidationCode = code
		if code == "" {
			if err = json.Unmarshal(planRaw, &v.Plan); err != nil {
				return err
			}
			now := time.Now().UTC()
			v.ValidatedAt = &now
			v.State = "ready_for_approval"
			v.Messages = append(v.Messages, setup.Message{Role: "assistant", Text: "Four legitimate identity/own-resource reads passed and an unambiguous supported plan was compiled. No cross-user probe has run. Review the exact assessment scope before approving and starting; sessions will be rechecked during assessment.", At: now})
		} else {
			v.State = "validation_failed"
			v.Messages = append(v.Messages, setup.Message{Role: "assistant", Text: "Prerequisite validation did not establish readiness (" + code + "). No security probe ran. Correct the access/contract or revise the setup, then explicitly approve another bounded validation if appropriate. Ambiguous JSON mappings require a reviewed integration; they are not guessed.", At: time.Now().UTC()})
		}
		item = v
		return putSetup(tx, v)
	})
	return item, e
}

// Exact assigned-token reflection is refused, including JSON unicode escaping.
// This is not general redaction of encoded/transformed or unknown secrets.
func reflectsCredential(body string, tokens map[string]string) bool {
	for _, token := range tokens {
		if token != "" && strings.Contains(body, token) {
			return true
		}
	}
	var decoded any
	if json.Unmarshal([]byte(body), &decoded) != nil {
		return false
	}
	var walk func(any) bool
	walk = func(v any) bool {
		switch x := v.(type) {
		case string:
			for _, token := range tokens {
				if token != "" && strings.Contains(x, token) {
					return true
				}
			}
		case []any:
			for _, c := range x {
				if walk(c) {
					return true
				}
			}
		case map[string]any:
			for k, c := range x {
				if walk(k) || walk(c) {
					return true
				}
			}
		}
		return false
	}
	return walk(decoded)
}

func (s *Server) stopSetup(tenant, id string) (setup.Session, error) {
	s.mu.Lock()
	var out setup.Session
	e := s.Store.db.Update(func(tx *bolt.Tx) error {
		var err error
		out, err = getSetup(tx, tenant, id)
		if err != nil {
			return err
		}
		if out.State != "stopped" {
			out.State = "stopped"
			out.Revision++
			out.Question = nil
			out.Messages = append(out.Messages, setup.Message{Role: "assistant", Text: "Setup permission was revoked. Request cancellation and any assessment workload cleanup are separate observations.", At: time.Now().UTC()})
		}
		return putSetup(tx, out)
	})
	if e == nil && s.setupActive[id] != nil {
		s.setupActive[id]()
	}
	s.mu.Unlock()
	if e == nil && out.EngagementID != "" {
		_, e = s.StopEngagement(tenant, out.EngagementID)
	}
	return out, e
}

func (s *Store) stopSetups(tenant string, global bool) ([]string, error) {
	ids := []string{}
	e := s.db.Update(func(tx *bolt.Tx) error {
		items := []setup.Session{}
		err := tx.Bucket([]byte("onboarding")).ForEach(func(_, b []byte) error {
			var d setupDisk
			if json.Unmarshal(b, &d) != nil {
				return errors.New("setup state unavailable")
			}
			v := d.Session
			v.ValidationProcess = d.Process
			if !global && v.Tenant != tenant {
				return nil
			}
			if v.State != "stopped" {
				v.State = "stopped"
				v.Question = nil
				v.Revision++
				ids = append(ids, v.ID)
				items = append(items, v)
			}
			return nil
		})
		if err != nil {
			return err
		}
		for _, v := range items {
			if e := putSetup(tx, v); e != nil {
				return e
			}
		}
		return nil
	})
	return ids, e
}

// Only onboarded plans opt into this persisted per-run interval. Reservations
// are serialized with Stop; waiting never retries an HTTP operation.
func (s *Server) reserveTarget(ctx context.Context, id string) error {
	for {
		var delay time.Duration
		s.mu.Lock()
		e := s.Store.Update(id, func(r *Run) error {
			if r.State != Running || time.Now().After(r.Deadline) || ctx.Err() != nil {
				return errors.New("run permission ended")
			}
			if r.RequestCount >= r.MaxRequests {
				return errors.New("request budget exceeded")
			}
			if r.Application != nil && r.Application.MinIntervalMillis > 0 && r.Lifecycle != nil && r.Lifecycle.LastAdmittedAt != nil {
				delay = time.Until(r.Lifecycle.LastAdmittedAt.Add(time.Duration(r.Application.MinIntervalMillis) * time.Millisecond))
				if delay > 0 {
					return nil
				}
			}
			r.RequestCount++
			now := time.Now().UTC()
			tracked(r).LastAdmittedAt = &now
			return nil
		})
		s.mu.Unlock()
		if e != nil || delay <= 0 {
			return e
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.New("run permission ended while rate limited")
		case <-timer.C:
		}
	}
}
