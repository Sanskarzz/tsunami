package platform

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"time"

	setup "github.com/Sanskarzz/tsunami/internal/onboarding"
	bolt "go.etcd.io/bbolt"
)

func (s *Server) sourceSetup(tenant string, d setup.Details) (setup.Details, map[string]string, error) {
	if d.SourceApplication == "" {
		return d, nil, d.Validate()
	}
	a, ok := s.Applications[tenant][d.SourceApplication]
	if !ok {
		return d, nil, errors.New("registered integration unavailable for this tenant")
	}
	if d.APIOrigin != "" && d.APIOrigin != a.Plan.Origin {
		return d, nil, errors.New("registered integration credentials cannot be assigned to another origin")
	}
	d.APIOrigin = a.Plan.Origin
	d.ApprovedAddresses = append([]string{}, a.Plan.ApprovedAddresses...)
	d.AuthMethod = "bearer"
	d.IdentityPath = a.Plan.Identity[0].Path
	d.OwnerPath = a.Plan.Baselines[0].Path
	d.PeerPath = a.Plan.Baselines[1].Path
	d.ExpectedAccess = a.Plan.ExpectedAccess
	d.ProtectedFields = []string{}
	for p := range a.Plan.Baselines[0].Hashes {
		d.ProtectedFields = append(d.ProtectedFields, p)
	}
	sort.Strings(d.ProtectedFields)
	return d, a.credentials, d.Validate()
}

// Public setup addresses are restricted to HTTPS and public destinations.
// The only private exception is a tenant's exact existing lab integration:
// origin, destination addresses and legitimate read paths cannot be widened.
func (s *Server) setupNetwork(tenant string, d setup.Details) error {
	if !setup.Origin(d.APIOrigin) || len(d.ApprovedAddresses) == 0 {
		return errors.New("explicit API origin and reviewed destination addresses required")
	}
	allowed := map[string]bool{}
	paths := map[string]bool{}
	if d.SourceApplication != "" {
		a, ok := s.Applications[tenant][d.SourceApplication]
		if !ok || a.Plan.Origin != d.APIOrigin {
			return errors.New("registered integration origin changed")
		}
		for _, a := range a.Plan.ApprovedAddresses {
			allowed[a] = true
		}
		for _, p := range a.Plan.Paths() {
			paths[p] = true
		}
		for _, p := range []string{d.IdentityPath, d.OwnerPath, d.PeerPath} {
			if p != "" && !paths[p] {
				return errors.New("registered integration read scope cannot be widened")
			}
		}
	} else {
		u, _ := url.Parse(d.APIOrigin)
		if u.Scheme != "https" {
			return errors.New("new external API integrations require HTTPS")
		}
	}
	seen := map[string]bool{}
	for _, raw := range d.ApprovedAddresses {
		ip, e := netip.ParseAddr(raw)
		if e != nil || ip.Unmap().String() != raw || seen[raw] {
			return errors.New("invalid or duplicate destination address")
		}
		seen[raw] = true
		if len(allowed) > 0 {
			if !allowed[raw] {
				return errors.New("registered integration destination cannot be widened")
			}
			continue
		}
		for _, denied := range s.SetupDenied {
			if denied.Contains(ip.Unmap()) {
				return errors.New("destination is excluded by trusted infrastructure policy")
			}
		}
		if !publicSetupIP(ip) {
			return errors.New("destination is outside this profile's public-address policy")
		}
	}
	return nil
}
func publicSetupIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, raw := range []string{"0.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/3", "168.63.129.16/32", "::/96", "64:ff9b::/96", "100::/64", "2001:db8::/32"} {
		if netip.MustParsePrefix(raw).Contains(ip) {
			return false
		}
	}
	return true
}

type setupMutation struct {
	Revision     int      `json:"revision"`
	QuestionID   string   `json:"question_id,omitempty"`
	Value        string   `json:"value,omitempty"`
	IdentityPath string   `json:"identity_path,omitempty"`
	OwnerPath    string   `json:"owner_path,omitempty"`
	PeerPath     string   `json:"peer_path,omitempty"`
	Fields       []string `json:"fields,omitempty"`
	Confirm      bool     `json:"confirm,omitempty"`
	Digest       string   `json:"digest,omitempty"`
}

func writeSetup(w http.ResponseWriter, status int, item setup.Session) {
	item.PreflightDigest = item.Digest()
	writeJSON(w, status, item)
}

func (s *Server) serveOnboarding(w http.ResponseWriter, r *http.Request) {
	tenant, ok := s.admin(r)
	if !ok {
		problem(w, 401, errors.New("operator authentication required"))
		return
	}
	if len(s.SetupKey) != 32 && r.Method != "GET" && !strings.HasSuffix(r.URL.Path, "/stop") {
		problem(w, 503, errors.New("protected onboarding is not configured; no credentials accepted"))
		return
	}
	if r.URL.Path == "/v1/onboarding" {
		switch r.Method {
		case "GET":
			items, e := s.Store.ListSetup(tenant)
			if e != nil {
				problem(w, 503, errors.New("setup state unavailable"))
				return
			}
			// Lists project metadata only; conversations/plans are fetched by ID.
			type summary struct {
				ID        string            `json:"id"`
				Revision  int               `json:"revision"`
				Details   map[string]string `json:"details"`
				State     string            `json:"state"`
				CreatedAt time.Time         `json:"created_at"`
				ExpiresAt time.Time         `json:"expires_at"`
			}
			projected := []summary{}
			for _, item := range items {
				projected = append(projected, summary{ID: item.ID, Revision: item.Revision, Details: map[string]string{"name": item.Details.Name, "environment": item.Details.Environment}, State: item.State, CreatedAt: item.CreatedAt, ExpiresAt: item.ExpiresAt})
			}
			writeJSON(w, 200, projected)
		case "POST":
			var d setup.Details
			if !decode(w, r, &d) {
				return
			}
			d, tokens, e := s.sourceSetup(tenant, d)
			if e != nil {
				problem(w, 400, e)
				return
			}
			item, e := s.Store.CreateSetup(tenant, d)
			if e != nil {
				problem(w, 409, e)
				return
			}
			if tokens != nil {
				item, e = s.saveSetupAccess(tenant, item.ID, item.Revision, tokens)
				if e != nil {
					problem(w, 409, errors.New("setup saved but integration access not assigned; recover its saved ID"))
					return
				}
			}
			writeSetup(w, 201, item)
		default:
			w.WriteHeader(405)
		}
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || len(parts) > 4 {
		w.WriteHeader(404)
		return
	}
	id := parts[2]
	item, e := s.Store.GetSetup(tenant, id)
	if e != nil {
		problem(w, 404, ErrNotFound)
		return
	}
	if item.State == "validating" && item.ValidationProcess != s.processID {
		_ = s.Store.db.Update(func(tx *bolt.Tx) error {
			v, e := getSetup(tx, tenant, id)
			if e != nil {
				return e
			}
			if v.State == "validating" && v.ValidationProcess != s.processID {
				v.State = "interrupted"
				v.Revision++
				v.ValidationCode = "previous_validation_uncertain"
				v.Messages = append(v.Messages, setup.Message{Role: "assistant", Text: "A previous process ended during validation. Its request outcome is uncertain. Nothing was replayed; inspect the attempt and explicitly approve a fresh validation if appropriate.", At: time.Now().UTC()})
				return putSetup(tx, v)
			}
			return nil
		})
		item, _ = s.Store.GetSetup(tenant, id)
	}
	if len(parts) == 3 && r.Method == "GET" {
		if time.Now().After(item.ExpiresAt) && item.State != "stopped" {
			item.State = "expired"
			item.Question = nil
		}
		writeSetup(w, 200, item)
		return
	}
	if len(parts) != 4 || r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	action := parts[3]
	if action == "access" {
		var in struct {
			Revision int    `json:"revision"`
			Owner    string `json:"owner"`
			Peer     string `json:"peer"`
		}
		if !decodeLimit(w, r, &in, 12<<10) {
			return
		}
		item, e = s.saveSetupAccess(tenant, id, in.Revision, map[string]string{"owner": in.Owner, "peer": in.Peer})
		if e != nil {
			problem(w, 409, e)
			return
		}
		writeSetup(w, 200, item)
		return
	}
	if action == "details" {
		var in struct {
			Revision int           `json:"revision"`
			Details  setup.Details `json:"details"`
		}
		if !decode(w, r, &in) {
			return
		}
		if e = in.Details.Validate(); e != nil {
			problem(w, 400, e)
			return
		}
		item, e = s.Store.changeSetup(tenant, id, in.Revision, func(tx *bolt.Tx, v *setup.Session) error {
			if v.Details.SourceApplication != in.Details.SourceApplication {
				return errors.New("create a new setup to change a registered integration")
			}
			if in.Details.SourceApplication != "" {
				if e := s.setupNetwork(tenant, in.Details); e != nil {
					return e
				}
			}
			if v.Details.APIOrigin != in.Details.APIOrigin {
				in.Details.ApprovedAddresses = nil
				v.AccessReady = false
				v.AccessOrigin = ""
				if e := tx.Bucket([]byte("onboarding_secrets")).Delete(setupKey(tenant, id)); e != nil {
					return e
				}
			}
			v.Details = in.Details
			v.Plan = nil
			v.ValidatedAt = nil
			v.ValidationCode = ""
			v.Question = nil
			v.Messages = append(v.Messages, setup.Message{Role: "user", Text: "Application details revised. Previous readiness observations were invalidated.", At: time.Now().UTC()})
			return nil
		})
	} else if action == "stop" {
		item, e = s.stopSetup(tenant, id)
	} else {
		var in setupMutation
		if !decode(w, r, &in) {
			return
		}
		switch action {
		case "answers":
			item, e = s.Store.changeSetup(tenant, id, in.Revision, func(_ *bolt.Tx, v *setup.Session) error {
				if v.Question == nil || v.Question.ID != in.QuestionID {
					return errSetupConflict
				}
				switch v.Question.Field {
				case "api_origin":
					if !setup.Origin(in.Value) {
						return errors.New("supply an API origin without path, query or credentials")
					}
					v.Details.APIOrigin = in.Value
					v.Details.ApprovedAddresses = nil
					v.AccessReady = false
				case "auth_method", "unsupported_auth":
					v.Details.AuthMethod = in.Value
				case "routes":
					v.Details.IdentityPath = in.IdentityPath
					v.Details.OwnerPath = in.OwnerPath
					v.Details.PeerPath = in.PeerPath
				case "access_rule":
					v.Details.ExpectedAccess = in.Value
				case "protected_fields":
					v.Details.ProtectedFields = in.Fields
				case "test_data":
					v.Details.SyntheticOnly = in.Confirm
				default:
					return errors.New("use the corresponding protected/permission form for this setup step")
				}
				if e := v.Details.Validate(); e != nil {
					return e
				}
				v.Plan = nil
				v.ValidatedAt = nil
				v.ValidationCode = ""
				v.Messages = append(v.Messages, setup.Message{Role: "user", Text: "Updated the requested " + v.Question.Field + " information.", At: time.Now().UTC()})
				v.Question = nil
				return nil
			})
		case "resolve":
			if !in.Confirm || item.Revision != in.Revision || !editableSetup(item, in.Revision) || item.Details.APIOrigin == "" {
				e = errSetupConflict
				break
			}
			u, _ := url.Parse(item.Details.APIOrigin)
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", u.Hostname())
			cancel()
			if err != nil {
				e = errors.New("API DNS resolution failed; no API requests made")
				break
			}
			addresses := []string{}
			seen := map[string]bool{}
			for _, ip := range ips {
				raw := ip.Unmap().String()
				if !seen[raw] {
					seen[raw] = true
					addresses = append(addresses, raw)
				}
			}
			sort.Strings(addresses)
			d := item.Details
			d.ApprovedAddresses = addresses
			if len(addresses) > 8 {
				e = errors.New("too many DNS answers for this bounded integration")
				break
			}
			if e = s.setupNetwork(tenant, d); e != nil {
				break
			}
			item, e = s.Store.changeSetup(tenant, id, in.Revision, func(_ *bolt.Tx, v *setup.Session) error {
				v.Details.ApprovedAddresses = addresses
				v.Plan = nil
				v.ValidatedAt = nil
				v.ValidationCode = ""
				v.Messages = append(v.Messages, setup.Message{Role: "assistant", Text: "Resolved candidate destination addresses. Review them in the preflight grant; no API connection or credential forwarding occurred.", At: time.Now().UTC()})
				return nil
			})
		case "validate":
			if !in.Confirm || in.Digest != item.Digest() {
				e = errors.New("review and explicitly authorize the current preflight digest")
				break
			}
			item, e = s.validateSetup(r.Context(), tenant, id, in.Revision)
		case "prepare":
			if item.EngagementID != "" {
				writeSetup(w, 200, item)
				return
			}
			if item.Plan == nil || !s.EnableCurlAdapter {
				e = errors.New("validated plan and reviewed curl executor required")
				break
			}
			a, err := s.Store.getSetupAccess(tenant, id, s.SetupKey)
			if err != nil {
				e = errors.New("protected access unavailable")
				break
			}
			prepared, err := prepareApplication(ApplicationConfig{Tenant: tenant, Plan: *item.Plan, Credentials: a.Tokens})
			if err != nil {
				e = err
				break
			}
			item, e = s.Store.prepareSetup(tenant, id, in.Revision, prepared)
		default:
			w.WriteHeader(404)
			return
		}
	}
	if e != nil {
		problem(w, 409, e)
		return
	}
	writeSetup(w, 200, item)
}

func (s *Server) saveSetupAccess(tenant, id string, revision int, tokens map[string]string) (setup.Session, error) {
	if len(s.SetupKey) != 32 || tokens["owner"] == tokens["peer"] {
		return setup.Session{}, errors.New("two distinct protected bearer credentials required")
	}
	for _, k := range []string{"owner", "peer"} {
		v := tokens[k]
		if len(v) < 32 || len(v) > 4096 || strings.ContainsAny(v, "\r\n\t ") {
			return setup.Session{}, errors.New("bearer credentials must be 32–4096 bytes without whitespace")
		}
	}
	return s.Store.changeSetup(tenant, id, revision, func(tx *bolt.Tx, v *setup.Session) error {
		if v.Details.APIOrigin == "" || v.Details.AuthMethod != "bearer" {
			return errors.New("select an explicit API origin and supported bearer authentication first")
		}
		b, e := sealSetup(s.SetupKey, setupAAD(tenant, id, "access"), setupAccess{Origin: v.Details.APIOrigin, Tokens: tokens})
		if e != nil {
			return errors.New("protected access could not be saved")
		}
		if e = tx.Bucket([]byte("onboarding_secrets")).Put(setupKey(tenant, id), b); e != nil {
			return e
		}
		v.AccessReady = true
		v.AccessOrigin = v.Details.APIOrigin
		v.Plan = nil
		v.ValidatedAt = nil
		v.ValidationCode = ""
		v.Messages = append(v.Messages, setup.Message{Role: "assistant", Text: "Two bearer credentials were assigned to this API origin through the protected form. Sessions are not verified until you approve baseline validation.", At: time.Now().UTC()})
		return nil
	})
}
