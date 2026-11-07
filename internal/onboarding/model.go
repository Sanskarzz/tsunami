// Package onboarding defines the typed setup workflow. It grants no network
// permission and accepts no password/token in conversation answers.
package onboarding

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Sanskarzz/tsunami/internal/application"
	"github.com/Sanskarzz/tsunami/internal/binding"
)

type Details struct {
	Name              string   `json:"name"`
	Environment       string   `json:"environment"`
	FrontendURL       string   `json:"frontend_url"`
	AuthorizationRef  string   `json:"authorization_ref"`
	APIOrigin         string   `json:"api_origin"`
	ApprovedAddresses []string `json:"approved_addresses,omitempty"`
	AuthMethod        string   `json:"auth_method"`
	IdentityPath      string   `json:"identity_path"`
	OwnerPath         string   `json:"owner_path"`
	PeerPath          string   `json:"peer_path"`
	ExpectedAccess    string   `json:"expected_access"`
	ProtectedFields   []string `json:"protected_fields,omitempty"`
	SyntheticOnly     bool     `json:"synthetic_only"`
	SourceApplication string   `json:"source_application,omitempty"`
}

type Message struct {
	Role string    `json:"role"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}
type Question struct {
	ID    string `json:"id"`
	Field string `json:"field"`
	Text  string `json:"text"`
}
type Session struct {
	ID                 string            `json:"id"`
	Tenant             string            `json:"tenant"`
	Revision           int               `json:"revision"`
	Details            Details           `json:"details"`
	State              string            `json:"state"`
	CreatedAt          time.Time         `json:"created_at"`
	ExpiresAt          time.Time         `json:"expires_at"`
	Messages           []Message         `json:"messages"`
	Question           *Question         `json:"question,omitempty"`
	AccessReady        bool              `json:"access_ready"`
	AccessOrigin       string            `json:"access_origin,omitempty"`
	ValidationAttempts int               `json:"validation_attempts"`
	ValidationRequests int               `json:"validation_requests"`
	ValidationProcess  string            `json:"-"` // persisted separately by the trusted store
	ValidationID       string            `json:"validation_id,omitempty"`
	ValidationDeadline *time.Time        `json:"validation_deadline,omitempty"`
	ValidatedAt        *time.Time        `json:"validated_at,omitempty"`
	ValidationCode     string            `json:"validation_code,omitempty"`
	Plan               *application.Plan `json:"plan,omitempty"`
	EngagementID       string            `json:"engagement_id,omitempty"`
	AdvisorMode        string            `json:"advisor_mode"`
	PreflightDigest    string            `json:"preflight_digest,omitempty"`
}

func Text(s string, max int) bool {
	if strings.TrimSpace(s) == "" || len(s) > max {
		return false
	}
	for _, c := range s {
		if unicode.IsControl(c) {
			return false
		}
	}
	return true
}
func Origin(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && len(raw) <= 256 && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.Opaque == "" && u.Path == "" && (u.Scheme == "https" || u.Scheme == "http")
}
func Path(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && len(raw) <= 256 && strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") && !strings.Contains(raw, "..") && u.Host == "" && u.RawQuery == "" && u.Fragment == "" && u.RawPath == "" && u.Path == raw
}
func (d Details) Validate() error {
	u, e := url.Parse(d.FrontendURL)
	if !Text(d.Name, 120) || !Text(d.Environment, 40) || !Text(d.AuthorizationRef, 256) || e != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") || len(d.FrontendURL) > 256 {
		return errors.New("application name, environment, permission reference and a credential-free HTTP(S) frontend URL are required")
	}
	if d.APIOrigin != "" && !Origin(d.APIOrigin) {
		return errors.New("API must be an explicit HTTP(S) origin without path, query or credentials")
	}
	if d.AuthMethod != "" && d.AuthMethod != "bearer" && d.AuthMethod != "password" && d.AuthMethod != "sso" {
		return errors.New("unsupported authentication selection")
	}
	for _, p := range []string{d.IdentityPath, d.OwnerPath, d.PeerPath} {
		if p != "" && !Path(p) {
			return errors.New("read paths must be exact paths without query, escaping or traversal")
		}
	}
	if d.ExpectedAccess != "" && d.ExpectedAccess != "deny" && d.ExpectedAccess != "allow" && d.ExpectedAccess != "filtered" {
		return errors.New("choose the intended access rule")
	}
	if len(d.ProtectedFields) > 6 || len(d.ApprovedAddresses) > 8 {
		return errors.New("setup field/address allowance exceeded")
	}
	for _, field := range d.ProtectedFields {
		if len(field) > 120 || !strings.HasPrefix(field, "/") {
			return errors.New("protected fields use JSON pointer paths, for example /secret")
		}
	}
	if d.SourceApplication != "" && !Text(d.SourceApplication, 64) {
		return errors.New("invalid registered integration")
	}
	return nil
}
func (s Session) Digest() string {
	b, _ := json.Marshal(struct {
		ID           string
		Revision     int
		Details      Details
		AccessOrigin string
		AccessReady  bool
	}{s.ID, s.Revision, s.Details, s.AccessOrigin, s.AccessReady})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (s Session) Missing() (string, string) {
	d := s.Details
	if d.APIOrigin == "" {
		return "api_origin", "I have the frontend URL. Which backend API does this environment use? Enter its origin; this answer does not approve a connection."
	}
	if d.AuthMethod != "" && d.AuthMethod != "bearer" {
		return "unsupported_auth", "This login method is not supported yet. Supply the sanitized API/authentication contract for an adapter, or use dedicated short-lived bearer test tokens. No login attempt will be guessed."
	}
	if len(d.ApprovedAddresses) == 0 {
		return "addresses", "Resolve the proposed API hostname to preview destination addresses. This permits a bounded DNS lookup only; it does not send credentials or contact API routes."
	}
	if d.AuthMethod == "" {
		return "auth_method", "How do the two test identities access the API? Bearer tokens are supported. Password/SSO login needs a reviewed adapter; do not paste credentials into this conversation."
	}
	if d.AuthMethod != "bearer" {
		return "unsupported_auth", "This login method is not supported yet. Supply the sanitized API/authentication contract for an adapter, or use dedicated short-lived bearer test tokens. No login attempt will be guessed."
	}
	if d.IdentityPath == "" || d.OwnerPath == "" || d.PeerPath == "" {
		return "routes", "Which GET path identifies the current account, and which legitimate resource path belongs to each test account? Supply only disposable records; the cross-user test is built in."
	}
	if d.ExpectedAccess == "" {
		return "access_rule", "What is the intended rule for account B reading account A's record: deny, intentional sharing, or sharing with private fields omitted? Different owners alone do not prove forbidden access."
	}
	if len(d.ProtectedFields) == 0 {
		return "protected_fields", "Which synthetic string field is private, for example /secret or /private_note? Supply the field name, not its value. The bounded validator needs a distinct 24–256 byte synthetic proof in each account's own record."
	}
	if !d.SyntheticOnly {
		return "test_data", "Confirm these are disposable test identities/records containing synthetic data. Real customer-data handling is not qualified by this release."
	}
	if !s.AccessReady || s.AccessOrigin != d.APIOrigin {
		return "access", "Provide the two bearer test tokens through the protected access form. They will not be added to the conversation, worker configuration or model context."
	}
	return "", ""
}
func (s *Session) Advance() {
	if s.State == "stopped" || s.EngagementID != "" || s.State == "validating" {
		return
	}
	if s.ValidationCode != "" && s.Plan == nil {
		s.State = "validation_failed"
		s.Question = nil
		return
	}
	field, text := s.Missing()
	if field == "" {
		if s.Plan != nil && s.ValidatedAt != nil {
			s.State = "ready_for_approval"
		} else {
			s.State = "ready_for_validation"
		}
		s.Question = nil
		return
	}
	s.State = "needs_input"
	if field == "unsupported_auth" {
		s.State = "unsupported"
	}
	if s.Question != nil && s.Question.Field == field {
		return
	}
	s.Question = &Question{ID: s.ID + "-" + time.Now().UTC().Format("150405.000000000"), Field: field, Text: text}
	s.Messages = append(s.Messages, Message{Role: "assistant", Text: text, At: time.Now().UTC()})
	if len(s.Messages) > 48 {
		s.Messages = s.Messages[len(s.Messages)-48:]
	}
}

// Infer compiles public assertions and proof hashes from four observed own-
// account responses. Ambiguous identity/owner relationships are refused.
func Infer(s Session, docs [4]binding.Document) (application.Plan, error) {
	parsed := make([]any, 4)
	for i, d := range docs {
		var e error
		parsed[i], e = application.DecodeDocument(string(d.Body))
		if e != nil || d.Status != 200 {
			return application.Plan{}, errors.New("four successful strict JSON baselines are required")
		}
	}
	leaves := func(raw any) map[string]string {
		out := map[string]string{}
		var walk func(any, string)
		walk = func(v any, p string) {
			switch x := v.(type) {
			case map[string]any:
				for k, c := range x {
					walk(c, p+"/"+strings.ReplaceAll(strings.ReplaceAll(k, "~", "~0"), "/", "~1"))
				}
			case string:
				out[p] = x
			}
		}
		walk(raw, "")
		return out
	}
	i0, i1, o0, o1 := leaves(parsed[0]), leaves(parsed[1]), leaves(parsed[2]), leaves(parsed[3])
	type pair struct{ identity, owner string }
	candidates := []pair{}
	for ip, a := range i0 {
		b, ok := i1[ip]
		if !ok || a == b || a == "" {
			continue
		}
		for op, v := range o0 {
			if v == a && o1[op] == b {
				candidates = append(candidates, pair{ip, op})
			}
		}
	}
	if len(candidates) != 1 {
		return application.Plan{}, errors.New("identity/owner mapping is ambiguous or unsupported; a reviewed application adapter is needed")
	}
	u0, _ := url.Parse(s.Details.OwnerPath)
	u1, _ := url.Parse(s.Details.PeerPath)
	a0, a1 := strings.Split(u0.Path, "/"), strings.Split(u1.Path, "/")
	if len(a0) != len(a1) || len(a0) < 2 || strings.Join(a0[:len(a0)-1], "/") != strings.Join(a1[:len(a1)-1], "/") {
		return application.Plan{}, errors.New("resource IDs must be the last segment of the same reviewed path template")
	}
	ids := []string{}
	for p, a := range o0 {
		if a == a0[len(a0)-1] && o1[p] == a1[len(a1)-1] {
			ids = append(ids, p)
		}
	}
	sort.Strings(ids)
	if len(ids) != 1 {
		return application.Plan{}, errors.New("resource ID mapping is ambiguous or unsupported")
	}
	profile := application.ObjectAccessProfile
	if s.Details.ExpectedAccess == "filtered" {
		profile = application.PropertyAccessProfile
	}
	in := binding.Input{Version: "tsunami.application-binding/v1", ID: "onboard-" + s.ID, Title: s.Details.Name, Profile: profile, Origin: s.Details.APIOrigin, ApprovedAddresses: s.Details.ApprovedAddresses, SyntheticDataOnly: true, ExpectedAccess: s.Details.ExpectedAccess, IdentityPath: s.Details.IdentityPath, IdentityPointer: candidates[0].identity, ObjectPathTemplate: strings.Join(a0[:len(a0)-1], "/") + "/{id}", ObjectIDPointer: ids[0], OwnerPointer: candidates[0].owner, ProtectedPointers: s.Details.ProtectedFields, Owner: binding.Account{Identity: docs[0], Object: docs[2]}, Peer: binding.Account{Identity: docs[1], Object: docs[3]}}
	raw, _ := json.Marshal(in)
	return binding.Compile(raw)
}
