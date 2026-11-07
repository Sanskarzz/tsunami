// Package binding prepares unapproved registration plans from operator-supplied
// synthetic baseline documents. It has no HTTP client or credential reader.
package binding

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"

	"github.com/Sanskarzz/tsunami/internal/application"
)

type Document struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}
type Account struct {
	Identity Document `json:"identity"`
	Object   Document `json:"object"`
}
type Input struct {
	Version            string   `json:"version"`
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Profile            string   `json:"profile"`
	Origin             string   `json:"origin"`
	ApprovedAddresses  []string `json:"approved_addresses"`
	SyntheticDataOnly  bool     `json:"synthetic_data_only"`
	ExpectedAccess     string   `json:"expected_access"`
	IdentityPath       string   `json:"identity_path"`
	IdentityPointer    string   `json:"identity_pointer"`
	ObjectPathTemplate string   `json:"object_path_template"`
	ObjectIDPointer    string   `json:"object_id_pointer"`
	OwnerPointer       string   `json:"owner_pointer"`
	ProtectedPointers  []string `json:"protected_pointers"`
	Owner              Account  `json:"owner"`
	Peer               Account  `json:"peer"`
}

var segment = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9_.-]{0,119}$`)

func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// Compile validates supplied facts, not the target's current behavior, the
// origin of these documents, or legal permission. Registration and the normal
// scope approval/execution remain separate, required steps.
func Compile(raw []byte) (application.Plan, error) {
	bad := func(message string) (application.Plan, error) { return application.Plan{}, errors.New(message) }
	if _, err := application.DecodeDocument(string(raw)); err != nil {
		return bad("binding must be one strict JSON document, at most 64 KiB with depth at most eight")
	}
	var input Input
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&input) != nil || d.Decode(&struct{}{}) != io.EOF {
		return bad("binding schema invalid; check field names and types")
	}
	if input.Version != "tsunami.application-binding/v1" || !input.SyntheticDataOnly {
		return bad("only explicit synthetic application-binding/v1 inputs are supported")
	}
	if input.Profile != application.ObjectAccessProfile && input.Profile != application.PropertyAccessProfile {
		return bad("choose a supported object-access or property-access profile")
	}
	if input.OwnerPointer == input.ObjectIDPointer || len(input.ProtectedPointers) < 1 || len(input.ProtectedPointers) > 6 {
		return bad("separate identity/ownership selectors and one to six protected pointers are required")
	}
	if strings.Count(input.ObjectPathTemplate, "{id}") != 1 || strings.Count(input.ObjectPathTemplate, "{") != 1 || strings.Count(input.ObjectPathTemplate, "}") != 1 {
		return bad("object_path_template must contain exactly one {id} placeholder")
	}
	seen := map[string]bool{}
	for _, pointer := range input.ProtectedPointers {
		if seen[pointer] || pointer == input.OwnerPointer || pointer == input.ObjectIDPointer {
			return bad("protected pointers must be distinct from each other and the public identity fields")
		}
		seen[pointer] = true
	}
	p := application.Plan{Version: input.Profile, ID: input.ID, Title: input.Title, Origin: input.Origin, ApprovedAddresses: input.ApprovedAddresses, SyntheticDataOnly: true, OwnerPointer: input.OwnerPointer, ExpectedAccess: input.ExpectedAccess, Identity: []application.Read{}, Baselines: []application.Read{}, DenialStatuses: []int{}}
	if input.Profile == application.ObjectAccessProfile && input.ExpectedAccess == "deny" {
		p.DenialStatuses = []int{401, 403, 404}
	}
	identities, objects := []string{}, []string{}
	privateValues := map[string]string{}
	for i, account := range []Account{input.Owner, input.Peer} {
		alias := []string{"owner", "peer"}[i]
		if account.Identity.Status != 200 || account.Object.Status != 200 {
			return bad("both supplied identity and own-object baselines must have HTTP 200 status")
		}
		identity, err := application.DecodeDocument(string(account.Identity.Body))
		if err != nil {
			return bad("identity baseline is not strict bounded JSON")
		}
		object, err := application.DecodeDocument(string(account.Object.Body))
		if err != nil {
			return bad("object baseline is not strict bounded JSON")
		}
		identityID, ok := application.StringAt(identity, input.IdentityPointer)
		if !ok || identityID == "" || len(identityID) > 256 {
			return bad("identity_pointer must select a nonempty public string identifier in both identities")
		}
		ownerID, ownerOK := application.StringAt(object, input.OwnerPointer)
		objectID, objectOK := application.StringAt(object, input.ObjectIDPointer)
		if !ownerOK || ownerID != identityID {
			return bad("an own-object owner does not match its supplied identity")
		}
		if !objectOK || !segment.MatchString(objectID) || objectID == "." || objectID == ".." {
			return bad("object IDs must be distinct, safe string path segments (no escaping or implicit conversion)")
		}
		identities, objects = append(identities, identityID), append(objects, objectID)
		proofs := map[string]string{}
		for _, pointer := range input.ProtectedPointers {
			value, ok := application.StringAt(object, pointer)
			if !ok || len(value) < 24 || len(value) > 256 || value == identityID || value == objectID {
				return bad("protected fields require distinct synthetic string proof values, 24–256 bytes long")
			}
			if prior, ok := privateValues[pointer]; ok && prior == value {
				return bad("owner and peer need distinct synthetic proof values for each protected field")
			}
			privateValues[pointer] = value
			proofs[pointer] = hash([]byte(value))
		}
		path := strings.Replace(input.ObjectPathTemplate, "{id}", objectID, 1)
		p.Identity = append(p.Identity, application.Read{Account: alias, Path: input.IdentityPath, Equals: map[string]string{input.IdentityPointer: identityID}})
		p.Baselines = append(p.Baselines, application.Read{Account: alias, Path: path, Equals: map[string]string{input.ObjectIDPointer: objectID, input.OwnerPointer: ownerID}, Hashes: proofs})
	}
	if identities[0] == identities[1] || objects[0] == objects[1] {
		return bad("distinct identities and legitimate objects are required")
	}
	p.Probe = p.Baselines[0]
	p.Probe.Account = "peer"
	// This validates the registration shape only. The server must replace this
	// temporary fingerprint with one derived from its real assigned credentials.
	p.CredentialSetID = strings.Repeat("0", 64)
	if err := p.Validate(); err != nil {
		return bad("compiled plan violates the selected profile's path, address, selector or access-rule contract")
	}
	p.CredentialSetID = ""
	return p, nil
}

type Manifest struct {
	Version           string   `json:"version"`
	InputSHA256       string   `json:"input_sha256"`
	PlanSHA256        string   `json:"plan_sha256"`
	Status            string   `json:"status"`
	TargetRequests    int      `json:"target_requests"`
	CredentialAliases []string `json:"credential_aliases"`
}

func bytesFor(raw []byte) ([]byte, []byte, application.Plan, error) {
	p, err := Compile(raw)
	if err != nil {
		return nil, nil, p, err
	}
	plan, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, nil, p, errors.New("cannot encode prepared plan")
	}
	plan = append(plan, '\n')
	m := Manifest{Version: "tsunami.prepared-application/v1", InputSHA256: hash(raw), PlanSHA256: hash(plan), Status: "pending_operator_registration_and_scope_approval", TargetRequests: 0, CredentialAliases: []string{"owner", "peer"}}
	manifest, err := json.MarshalIndent(m, "", "  ")
	return plan, append(manifest, '\n'), p, err
}
