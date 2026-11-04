// Package application defines bounded, operator-configured HTTP assessments.
// It contains no credential storage, network client, or executable expressions.
package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/netip"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

type Read struct {
	Account string            `json:"account"`
	Path    string            `json:"path"`
	Equals  map[string]string `json:"equals,omitempty"`
	Hashes  map[string]string `json:"sha256,omitempty"`
}

type Plan struct {
	Version           string   `json:"version"`
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	Origin            string   `json:"origin"`
	ApprovedAddresses []string `json:"approved_addresses"`
	CredentialSetID   string   `json:"credential_set_id"`
	SyntheticDataOnly bool     `json:"synthetic_data_only"`
	OwnerPointer      string   `json:"owner_pointer"`
	Identity          []Read   `json:"identity"`
	Baselines         []Read   `json:"baselines"`
	Probe             Read     `json:"probe"`
	ExpectedAccess    string   `json:"expected_access"`
	DenialStatuses    []int    `json:"denial_statuses"`
}

var slug = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
var digest = regexp.MustCompile(`^[a-f0-9]{64}$`)

func (p Plan) ProcedureID() string { return p.ID + "/object-access/v2" }
func (p Plan) Digest() string {
	data, _ := json.Marshal(p)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func (p Plan) MaxRequests() int { return 8 }
func (p Plan) Reads() []Read {
	out := append(append([]Read{}, p.Identity...), p.Baselines...)
	out = append(out, p.Probe)
	if len(p.Baselines) > 0 {
		out = append(out, p.Baselines[0], p.Identity[1])
	} // Recheck owner after probe.
	return out
}
func (p Plan) Paths() []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range p.Reads() {
		if !seen[r.Path] {
			seen[r.Path] = true
			out = append(out, r.Path)
		}
	}
	sort.Strings(out)
	return out
}
func (p Plan) Allows(alias, path string) bool {
	for _, r := range p.Reads() {
		if r.Account == alias && r.Path == path {
			return true
		}
	}
	return false
}

func (p Plan) Validate() error {
	bad := errors.New("invalid bounded application plan")
	if p.Version != "http-object-access/v2" || !slug.MatchString(p.ID) || strings.TrimSpace(p.Title) == "" || len(p.Title) > 120 || strings.ContainsAny(p.Title, "\r\n\t") || !p.SyntheticDataOnly || !digest.MatchString(p.CredentialSetID) {
		return bad
	}
	u, err := url.Parse(p.Origin)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || u.Opaque != "" || u.Path != "" || (u.Scheme != "http" && u.Scheme != "https") || len(p.Origin) > 256 {
		return bad
	}
	if len(p.ApprovedAddresses) == 0 || len(p.ApprovedAddresses) > 8 {
		return bad
	}
	seen := map[string]bool{}
	for _, raw := range p.ApprovedAddresses {
		ip, err := netip.ParseAddr(raw)
		if err != nil || ip.Zone() != "" || ip.Unmap().String() != raw || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || seen[raw] {
			return bad
		}
		seen[raw] = true
	}
	if len(p.Identity) != 2 || len(p.Baselines) != 2 || (p.ExpectedAccess != "allow" && p.ExpectedAccess != "deny") || len(p.DenialStatuses) > 3 {
		return bad
	}
	for _, status := range p.DenialStatuses {
		if status != 401 && status != 403 && status != 404 {
			return bad
		}
	}
	a, b := p.Identity[0].Account, p.Identity[1].Account
	if !slug.MatchString(a) || !slug.MatchString(b) || a == b || p.Baselines[0].Account != a || p.Baselines[1].Account != b || p.Probe.Account != b || p.Probe.Path != p.Baselines[0].Path || p.Baselines[0].Path == p.Baselines[1].Path {
		return bad
	}
	if !validPointer(p.OwnerPointer) || len(p.Identity[0].Equals) != 1 || len(p.Identity[1].Equals) != 1 || len(p.Identity[0].Hashes) != 0 || len(p.Identity[1].Hashes) != 0 {
		return bad
	}
	for key, owner := range p.Identity[0].Equals {
		peer, ok := p.Identity[1].Equals[key]
		if !ok || peer == owner || p.Baselines[0].Equals[p.OwnerPointer] != owner || p.Baselines[1].Equals[p.OwnerPointer] != peer {
			return bad
		}
	}
	for _, r := range p.Reads() {
		parsed, err := url.Parse(r.Path)
		if err != nil || !strings.HasPrefix(r.Path, "/") || strings.HasPrefix(r.Path, "//") || parsed.Path != r.Path || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.Fragment != "" || strings.Contains(r.Path, "%") || len(r.Path) > 256 || len(r.Equals) == 0 || len(r.Equals)+len(r.Hashes) > 8 {
			return bad
		}
		for _, part := range strings.Split(r.Path, "/") {
			if part == "." || part == ".." || strings.Contains(part, "\\") {
				return bad
			}
		}
		for pointer, value := range r.Equals {
			if !validPointer(pointer) || value == "" || len(value) > 256 {
				return bad
			}
		}
		for pointer, value := range r.Hashes {
			if !validPointer(pointer) || !digest.MatchString(value) {
				return bad
			}
			if _, ok := r.Equals[pointer]; ok {
				return bad
			}
		}
	}
	// Probe proof must be exactly the owner's baseline proof. Expectations
	// cannot be relaxed just for the attempted cross-user request.
	x, _ := json.Marshal(p.Baselines[0].Equals)
	y, _ := json.Marshal(p.Probe.Equals)
	u1, _ := json.Marshal(p.Baselines[0].Hashes)
	u2, _ := json.Marshal(p.Probe.Hashes)
	if !bytes.Equal(x, y) || !bytes.Equal(u1, u2) || len(p.Baselines[0].Hashes) == 0 || len(p.Baselines[1].Hashes) == 0 {
		return bad
	}
	return nil
}

func validPointer(s string) bool {
	if !strings.HasPrefix(s, "/") || len(s) > 128 || strings.Count(s, "/") > 6 {
		return false
	}
	for _, part := range strings.Split(s[1:], "/") {
		if part == "" {
			return false
		}
		for i := 0; i < len(part); i++ {
			if part[i] == '~' {
				if i+1 >= len(part) || (part[i+1] != '0' && part[i+1] != '1') {
					return false
				}
				i++
			}
		}
	}
	return true
}

// Strict JSON parsing rejects duplicate keys and trailing documents. A target
// cannot make different clients observe different identities from one object.
func strictValue(d *json.Decoder, depth int) (any, error) {
	if depth > 8 {
		return nil, errors.New("JSON nesting limit")
	}
	token, err := d.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}
	switch delim {
	case '{':
		obj := map[string]any{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return nil, err
			}
			s, ok := key.(string)
			if !ok {
				return nil, errors.New("invalid key")
			}
			if _, ok := obj[s]; ok {
				return nil, errors.New("duplicate key")
			}
			v, err := strictValue(d, depth+1)
			if err != nil {
				return nil, err
			}
			obj[s] = v
		}
		_, err = d.Token()
		return obj, err
	case '[':
		items := []any{}
		for d.More() {
			v, err := strictValue(d, depth+1)
			if err != nil {
				return nil, err
			}
			items = append(items, v)
		}
		_, err = d.Token()
		return items, err
	}
	return nil, errors.New("invalid JSON token")
}

func (r Read) Matches(body string) bool {
	if len(body) > 64<<10 {
		return false
	}
	d := json.NewDecoder(strings.NewReader(body))
	d.UseNumber()
	value, err := strictValue(d, 0)
	if err != nil {
		return false
	}
	if _, err = d.Token(); err != io.EOF {
		return false
	}
	lookup := func(pointer string) (string, bool) {
		if !validPointer(pointer) {
			return "", false
		}
		current := value
		for _, part := range strings.Split(pointer[1:], "/") {
			obj, ok := current.(map[string]any)
			if !ok {
				return "", false
			}
			part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
			current, ok = obj[part]
			if !ok {
				return "", false
			}
		}
		text, ok := current.(string)
		return text, ok
	}
	for key, want := range r.Equals {
		got, ok := lookup(key)
		if !ok || got != want {
			return false
		}
	}
	for key, want := range r.Hashes {
		got, ok := lookup(key)
		if !ok {
			return false
		}
		sum := sha256.Sum256([]byte(got))
		if hex.EncodeToString(sum[:]) != want {
			return false
		}
	}
	return true
}
