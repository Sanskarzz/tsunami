package platform

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"
)

// The inline lab contract allows 20 children, at most 12 forwarded 64 KiB
// responses per child, plus bounded denied captures and metadata. JSON escaping
// and artifact base64 can expand bytes; 192 MiB leaves room under those bounds.
// This is not a scalable artifact-delivery design; separate storage is planned.
const MaxInlineExportBytes int64 = 192 << 20

type RunSummary struct {
	ID           string    `json:"id"`
	Tenant       string    `json:"tenant"`
	Scenario     string    `json:"scenario"`
	State        string    `json:"state"`
	Execution    string    `json:"execution,omitempty"`
	EngagementID string    `json:"engagement_id,omitempty"`
	Executor     string    `json:"executor,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type RunPage struct {
	Items      []RunSummary `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

var runCursor = regexp.MustCompile(`^[a-f0-9]{32}$`)

func (s *Store) ListRunSummaries(tenant, after string, limit int) (RunPage, error) {
	page := RunPage{Items: []RunSummary{}}
	if tenant == "" || limit < 1 || limit > 100 || (after != "" && !runCursor.MatchString(after)) {
		return page, errors.New("invalid run pagination")
	}
	err := s.db.View(func(tx *bolt.Tx) error {
		cursor := tx.Bucket([]byte("runs")).Cursor()
		key, value := cursor.First()
		if after != "" {
			key, value = cursor.Seek([]byte(after))
			if bytes.Equal(key, []byte(after)) {
				key, value = cursor.Next()
			}
		}
		for ; key != nil; key, value = cursor.Next() {
			// Decode only projected metadata, never retain capture bodies in a list.
			var item RunSummary
			if json.Unmarshal(value, &item) != nil {
				return errors.New("invalid stored run metadata")
			}
			if item.ID != string(key) || !runCursor.MatchString(item.ID) {
				return errors.New("stored run identity differs from key")
			}
			if item.Tenant != tenant {
				continue
			}
			if len(page.Items) == limit {
				page.NextCursor = page.Items[len(page.Items)-1].ID
				break
			}
			page.Items = append(page.Items, item)
		}
		return nil
	})
	return page, err
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request, tenant string) {
	query := r.URL.Query()
	for key, values := range query {
		if (key != "limit" && key != "cursor") || len(values) != 1 {
			problem(w, 400, errors.New("invalid run pagination"))
			return
		}
	}
	limit := 50
	if raw := query.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			problem(w, 400, errors.New("invalid run limit"))
			return
		}
		limit = n
	}
	if limit < 1 || limit > 100 || (query.Get("cursor") != "" && !runCursor.MatchString(query.Get("cursor"))) {
		problem(w, 400, errors.New("invalid run pagination"))
		return
	}
	page, err := s.Store.ListRunSummaries(tenant, query.Get("cursor"), limit)
	if err != nil {
		problem(w, 500, errors.New("run metadata unavailable"))
		return
	}
	writeJSON(w, 200, page)
}
