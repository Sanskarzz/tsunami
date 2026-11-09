package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/netip"
	"os"
	"strings"

	"github.com/Sanskarzz/tsunami/internal/application"
)

type ApplicationConfig struct {
	Tenant      string            `json:"tenant"`
	Plan        application.Plan  `json:"plan"`
	Credentials map[string]string `json:"credentials"`
}
type RegisteredApplication struct {
	Plan        application.Plan
	Target      Target
	credentials map[string]string
}

// RegisterApplication is trusted startup configuration, never an agent/API
// registration route. Credentials never enter the public plan or run record.
func (s *Server) RegisterApplication(c ApplicationConfig) error {
	a, err := prepareApplication(c)
	if err != nil {
		return err
	}
	if s.Applications == nil {
		s.Applications = map[string]map[string]RegisteredApplication{}
	}
	if s.Applications[c.Tenant] == nil {
		s.Applications[c.Tenant] = map[string]RegisteredApplication{}
	}
	if _, exists := s.Applications[c.Tenant][a.Plan.ID]; exists {
		return errors.New("duplicate application ID for tenant")
	}
	s.Applications[c.Tenant][a.Plan.ID] = a
	return nil
}

func prepareApplication(c ApplicationConfig) (RegisteredApplication, error) {
	if c.Tenant == "" || len(c.Credentials) != 2 {
		return RegisteredApplication{}, errors.New("two tenant-bound application credentials required")
	}
	keys := []string{}
	for _, r := range c.Plan.Identity {
		keys = append(keys, r.Account)
	}
	if len(keys) != 2 || keys[0] == keys[1] {
		return RegisteredApplication{}, errors.New("distinct identity aliases required")
	}
	secret := map[string]string{}
	for _, key := range keys {
		token := c.Credentials[key]
		if len(token) < 32 || len(token) > 4096 || strings.ContainsAny(token, "\r\n\t ") {
			return RegisteredApplication{}, errors.New("invalid application credential")
		}
		secret[key] = token
	}
	if secret[keys[0]] == secret[keys[1]] {
		return RegisteredApplication{}, errors.New("distinct test identities required")
	}
	data, _ := json.Marshal(struct {
		Tenant, ID  string
		Credentials map[string]string
	}{c.Tenant, c.Plan.ID, secret})
	sum := sha256.Sum256(data)
	c.Plan.CredentialSetID = hex.EncodeToString(sum[:])
	if err := c.Plan.Validate(); err != nil {
		return RegisteredApplication{}, err
	}
	approved := []netip.Prefix{}
	for _, raw := range c.Plan.ApprovedAddresses {
		if raw == os.Getenv("KUBERNETES_SERVICE_HOST") && raw != "" {
			return RegisteredApplication{}, errors.New("Kubernetes API is not an application target")
		}
		ip, _ := netip.ParseAddr(raw)
		approved = append(approved, netip.PrefixFrom(ip, ip.BitLen()))
	}
	policy, err := NewPolicy(c.Plan.Origin, approved, nil)
	if err != nil {
		return RegisteredApplication{}, err
	}
	// Own immutable copy of all configuration, including slices/maps.
	public, _ := json.Marshal(c.Plan)
	var plan application.Plan
	_ = json.Unmarshal(public, &plan)
	publicPlan, _ := json.Marshal(plan)
	for _, credential := range secret {
		if strings.Contains(string(publicPlan), credential) {
			return RegisteredApplication{}, errors.New("public plan cannot contain an assigned credential")
		}
	}
	return RegisteredApplication{Plan: plan, Target: Target{URL: plan.Origin, Policy: policy.WithPaths(plan.Paths())}, credentials: secret}, nil
}

func (s *Server) LoadApplications(path string) error {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("application configuration unavailable")
	}
	f, err := os.Open(path)
	if err != nil {
		return errors.New("application configuration unavailable")
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 128<<10))
	d.DisallowUnknownFields()
	var entries []ApplicationConfig
	if d.Decode(&entries) != nil || len(entries) == 0 || len(entries) > 16 {
		return errors.New("invalid bounded application configuration")
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return errors.New("extra application configuration data")
	}
	for _, entry := range entries {
		if err := s.RegisterApplication(entry); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) targetForRun(r Run) (Target, error) {
	if r.Application == nil {
		return s.target(r.Tenant, r.Scenario, r.Connectivity)
	}
	a, ok := s.application(r.Tenant, r.Application.ID)
	if !ok || r.Connectivity != "" || r.Application.Validate() != nil || a.Plan.Digest() != r.ApplicationDigest || r.Application.Digest() != r.ApplicationDigest {
		return Target{}, errors.New("application permission unavailable or configuration changed")
	}
	return a.Target, nil
}

func (r Run) ProcedureID() string {
	if r.Application != nil {
		return r.Application.ProcedureID()
	}
	if r.Assessment != nil {
		return r.Assessment.ID
	}
	return r.Workflow
}
func (r Run) ProcedureDigest() string {
	if r.Application != nil {
		return r.ApplicationDigest
	}
	return r.AssessmentDigest
}
func (r Run) ProcedureValid() bool {
	if r.Application != nil {
		return r.Assessment == nil && r.Application.Validate() == nil && r.Application.Digest() == r.ApplicationDigest
	}
	return r.Assessment != nil && r.Assessment.Canonical() && r.Assessment.Digest() == r.AssessmentDigest
}
func (r Run) ProcedureRemediation() string {
	if r.Application != nil {
		if r.Application.Version == application.PropertyAccessProfile {
			return "Authorize each response property for the requesting principal. Use an explicit public response projection and omit the protected fields from the peer's view."
		}
		return "Authorize every object read against the authenticated principal and intended sharing rules on the server."
	}
	return r.Assessment.Remediation
}

func validateApplication(r Run) Result {
	return validateApplicationEvidence(r, nil)
}

// Optional trace records the same selected captures/checks as validation. It is
// a read-only explanation, never a second path for confirming a finding.
func validateApplicationEvidence(r Run, checks *[]EvidenceCheck) Result {
	p := r.Application
	if p == nil {
		return Result{Verdict: "inconclusive", Reason: "Application plan unavailable.", EvidenceIDs: []string{}}
	}
	if !r.ProcedureValid() || (p.Version != application.PropertyAccessProfile && p.ExpectedAccess != "deny") {
		return validateApplicationAtProbe(r, -1, checks)
	}
	// A worker may repeat an allowed read within its budget. A later denial,
	// filtered reply or failure cannot erase an earlier exposure. Validate each
	// matching exposure against its own ordered prerequisites and rechecks.
	unproven := []string{}
	for i, capture := range r.Captures {
		if capture.URL != r.Target+p.Probe.Path || capture.Account != p.Probe.Account || capture.Blocked || capture.Error != "" || capture.Status != 200 {
			continue
		}
		if p.Version == application.PropertyAccessProfile {
			if p.Probe.PropertyResult(capture.Body) != "leaked" {
				continue
			}
		} else if !p.Probe.Matches(capture.Body) {
			continue
		}
		var selected []EvidenceCheck
		result := validateApplicationAtProbe(r, i, &selected)
		if result.Verdict == "confirmed" {
			if checks != nil {
				*checks = selected
			}
			return result
		}
		unproven = append(unproven, capture.ID)
	}
	result := validateApplicationAtProbe(r, -1, checks)
	if len(unproven) > 0 && result.Verdict == "no_issue_observed" {
		result.Verdict = "inconclusive"
		result.Reason = "An earlier private-value exposure lacked complete ordered prerequisites. A later filtered reply cannot erase that observation."
		if p.Version == application.ObjectAccessProfile {
			result.Reason = "An earlier protected-object exposure lacked complete ordered prerequisites. A later denied reply cannot erase that observation."
		}
	}
	// Keep the unresolved exposure inspectable even when the latest probe is
	// already inconclusive, rather than referencing only the later response.
	for _, id := range unproven {
		found := false
		for _, existing := range result.EvidenceIDs {
			if existing == id {
				found = true
			}
		}
		if !found {
			result.EvidenceIDs = append(result.EvidenceIDs, id)
		}
		if checks != nil {
			mapped := false
			for _, check := range *checks {
				if check.CaptureID == id {
					mapped = true
					break
				}
			}
			if !mapped {
				*checks = append(*checks, EvidenceCheck{Role: "earlier_unresolved_probe", Account: p.Probe.Account, Path: p.Probe.Path, CaptureID: id, Outcome: "unresolved_prior_exposure"})
			}
		}
	}
	return result
}

func validateApplicationAtProbe(r Run, selectedProbe int, checks *[]EvidenceCheck) Result {
	p := r.Application
	if checks != nil {
		*checks = applicationChecks(r)
	}
	out := Result{Verdict: "inconclusive", Title: p.Title, Reason: "Verified identities, legitimate reads and an owner recheck are required.", EvidenceIDs: []string{}, Validation: &ValidationDetails{BaselinesRequired: 6, ExpectedAccess: p.ExpectedAccess}}
	if !r.ProcedureValid() {
		return out
	}
	probeIndex := selectedProbe
	if selectedProbe < 0 {
		for i, c := range r.Captures {
			if c.URL == r.Target+p.Probe.Path && c.Account == p.Probe.Account {
				probeIndex = i
			}
		}
	}
	if probeIndex < 0 {
		return out
	}
	probe := r.Captures[probeIndex]
	valid := func(c Capture) bool { return !c.Blocked && c.Error == "" && c.Status > 0 }
	find := func(read application.Read, start, end, checkIndex int) bool {
		var last *Capture
		for i := start; i < end; i++ {
			c := &r.Captures[i]
			if c.URL == r.Target+read.Path && c.Account == read.Account {
				last = c
			}
		}
		if last != nil {
			out.EvidenceIDs = append(out.EvidenceIDs, last.ID)
		}
		verified := last != nil && valid(*last) && last.Status == 200 && read.Matches(last.Body)
		if checks != nil {
			check := &(*checks)[checkIndex]
			check.Outcome = "missing"
			if last != nil {
				check.CaptureID = last.ID
				check.Outcome = "not_verified"
			}
			if verified {
				check.Outcome = "verified"
			}
		}
		return verified
	}
	for i, read := range append(append([]application.Read{}, p.Identity...), p.Baselines...) {
		if find(read, 0, probeIndex, i) {
			out.Validation.BaselinesVerified++
		}
	}
	if find(p.Baselines[0], probeIndex+1, len(r.Captures), 5) {
		out.Validation.BaselinesVerified++
	}
	if find(p.Identity[1], probeIndex+1, len(r.Captures), 6) {
		out.Validation.BaselinesVerified++
	}
	out.EvidenceIDs = append(out.EvidenceIDs, probe.ID)
	selected := map[string]bool{}
	for _, id := range out.EvidenceIDs {
		selected[id] = true
	}
	out.EvidenceIDs = []string{}
	for _, capture := range r.Captures {
		if selected[capture.ID] {
			out.EvidenceIDs = append(out.EvidenceIDs, capture.ID)
			delete(selected, capture.ID)
		}
	}
	out.Validation.ProbeCaptured = valid(probe)
	if checks != nil {
		check := &(*checks)[4]
		check.CaptureID, check.Outcome = probe.ID, applicationProbeOutcome(r, probe)
	}
	if out.Validation.BaselinesVerified != 6 || !out.Validation.ProbeCaptured {
		return out
	}
	if p.Version == application.PropertyAccessProfile {
		out.Reason = "The expected public object and omitted protected fields were not established. Denial, changed/redacted values or malformed content are inconclusive for this filtering rule."
		if probe.Status != 200 {
			return out
		}
		switch p.Probe.PropertyResult(probe.Body) {
		case "leaked":
			out.Verdict = "confirmed"
			out.Reason = "The peer's permitted object read included a known owner-only property value; identities, legitimate private views and follow-up access were verified."
		case "filtered":
			out.Verdict = "no_issue_observed"
			out.Reason = "The peer received the expected public object with all configured owner-only properties omitted. This covers only these fields, path and identities."
		}
		return out
	}
	if probe.Status == 200 && p.Probe.Matches(probe.Body) {
		if p.ExpectedAccess == "deny" {
			out.Verdict = "confirmed"
			out.Reason = "The authenticated peer received the owner's approved proof data; account identities and owner access were independently checked."
		} else {
			out.Verdict = "no_issue_observed"
			out.Reason = "The captured cross-user read matches explicitly approved sharing."
		}
		return out
	}
	if p.ExpectedAccess == "deny" {
		for _, status := range p.DenialStatuses {
			if probe.Status == status {
				out.Verdict = "no_issue_observed"
				out.Reason = "The prohibited read was denied and the owner could still read the same proof object afterward. This is one tested rule, not application-wide assurance."
				return out
			}
		}
	}
	out.Reason = "The probe does not prove approved access or protected-data exposure. A generic status/body is insufficient."
	return out
}
