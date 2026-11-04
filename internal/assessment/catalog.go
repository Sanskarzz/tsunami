// Package assessment contains original, curated synthetic authorization
// procedures informed by the VAPT toolkit. It never executes toolkit scripts.
package assessment

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

//go:embed procedures/*.json
var catalog embed.FS

type Object struct {
	ID      string `json:"id"`
	Owner   string `json:"owner"`
	Content string `json:"content"`
}

type Read struct {
	Account string  `json:"account"`
	Path    string  `json:"path"`
	Object  *Object `json:"object,omitempty"`
}

// ExpectedAccess is the approved access rule, not whether the fixture has a
// bug. Actual vulnerable/fixed expectations live only in the evaluator.
type Definition struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Purpose        string `json:"purpose"`
	Risk           string `json:"risk"`
	Review         string `json:"review"`
	Method         string `json:"method"`
	MaxRequests    int    `json:"max_requests"`
	Baselines      []Read `json:"baselines"`
	Probe          Read   `json:"probe"`
	ExpectedAccess string `json:"expected_access"`
	Remediation    string `json:"remediation"`
}

func Get(id string) (Definition, error) {
	// Read from the embedded catalog each time; callers cannot mutate shared
	// slices and change another run's policy or this process's catalog.
	files, err := catalog.ReadDir("procedures")
	if err != nil {
		return Definition{}, err
	}
	for _, file := range files {
		data, err := catalog.ReadFile("procedures/" + file.Name())
		if err != nil {
			return Definition{}, err
		}
		var d Definition
		if json.Unmarshal(data, &d) != nil {
			return Definition{}, errors.New("invalid embedded assessment")
		}
		if d.ID == id {
			return d, nil
		}
	}
	return Definition{}, errors.New("unknown assessment procedure")
}

func List() []Definition {
	files, _ := catalog.ReadDir("procedures")
	out := []Definition{}
	for _, file := range files {
		data, _ := catalog.ReadFile("procedures/" + file.Name())
		var d Definition
		if json.Unmarshal(data, &d) == nil {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (d Definition) Digest() string {
	data, _ := json.Marshal(d)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func (d Definition) Canonical() bool {
	current, err := Get(d.ID)
	return err == nil && current.Digest() == d.Digest()
}
func (d Definition) Reads() []Read { return append(append([]Read{}, d.Baselines...), d.Probe) }
func (d Definition) Allows(account, path string) bool {
	for _, read := range d.Reads() {
		if read.Account == account && read.Path == path {
			return true
		}
	}
	return false
}
func (d Definition) Paths() []string {
	paths := []string{}
	seen := map[string]bool{}
	for _, read := range d.Reads() {
		if !seen[read.Path] {
			paths = append(paths, read.Path)
			seen[read.Path] = true
		}
	}
	return paths
}
