package store

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketPortals   = []byte("portals")
	bucketTenders   = []byte("tenders")
	bucketSnapshots = []byte("snapshots")
)

// Tender is the canonical data structure for a single tender
type Tender struct {
	ReferenceNumber string    `json:"reference_number"`
	PortalID        string    `json:"portal_id"`
	Title           string    `json:"title"`
	IssuingEntity   string    `json:"issuing_entity"`
	Deadline        time.Time `json:"deadline"`
	EstimatedValue  string    `json:"estimated_value"`
	SourceURL       string    `json:"source_url"`
	ContentHash     string    `json:"content_hash"`
	Version         int       `json:"version"`
	LastUpdated     time.Time `json:"last_updated"`
	Status          string    `json:"status"` // "new" | "updated" | "closed"
}

// TenderEvent is what the store emits after a diff
type TenderEvent struct {
	Type   string // "new" | "updated"
	Tender Tender
}

// Store wraps BBolt
type Store struct {
	db *bolt.DB
}

// New opens the BBolt database and guarantees all buckets exist
func New(path string) (*Store, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 2 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range [][]byte{
			bucketPortals,
			bucketTenders,
			bucketSnapshots,
		} {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("init buckets: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database cleanly
func (s *Store) Close() error {
	return s.db.Close()
}

// SavePortal persists a portal definition
func (s *Store) SavePortal(p []byte, id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketPortals).Put([]byte(id), p)
	})
}

// GetAllPortals returns all portal definitions as raw JSON blobs
func (s *Store) GetAllPortals() ([][]byte, error) {
	var portals [][]byte
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketPortals).ForEach(func(_, v []byte) error {
			cp := make([]byte, len(v))
			copy(cp, v)
			portals = append(portals, cp)
			return nil
		})
	})
	return portals, err
}

// DeletePortal removes a portal by ID
func (s *Store) DeletePortal(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketPortals).Delete([]byte(id))
	})
}

// UpsertTenders diffs incoming tenders against what is stored.
// It returns a slice of TenderEvents for new or updated tenders.
// Primary key: portalID + ":" + referenceNumber
// Version key: content hash
func (s *Store) UpsertTenders(incoming []Tender) ([]TenderEvent, error) {
	var events []TenderEvent

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTenders)

		for _, t := range incoming {
			key := []byte(t.PortalID + ":" + t.ReferenceNumber)
			t.ContentHash = hash(t)

			existing := b.Get(key)

			if existing == nil {
				// brand new tender
				t.Version = 1
				t.Status = "new"
				t.LastUpdated = time.Now()

				encoded, err := json.Marshal(t)
				if err != nil {
					return fmt.Errorf("marshal tender: %w", err)
				}
				if err := b.Put(key, encoded); err != nil {
					return fmt.Errorf("put tender: %w", err)
				}
				events = append(events, TenderEvent{Type: "new", Tender: t})
				continue
			}

			// tender exists — check if content changed
			var stored Tender
			if err := json.Unmarshal(existing, &stored); err != nil {
				return fmt.Errorf("unmarshal stored tender: %w", err)
			}

			if stored.ContentHash == t.ContentHash {
				// no change, skip silently
				continue
			}

			// content changed — increment version
			t.Version = stored.Version + 1
			t.Status = "updated"
			t.LastUpdated = time.Now()

			encoded, err := json.Marshal(t)
			if err != nil {
				return fmt.Errorf("marshal updated tender: %w", err)
			}
			if err := b.Put(key, encoded); err != nil {
				return fmt.Errorf("put updated tender: %w", err)
			}
			events = append(events, TenderEvent{Type: "updated", Tender: t})
		}
		return nil
	})

	return events, err
}

// GetTendersByPortal returns all tenders for a given portal
func (s *Store) GetTendersByPortal(portalID string) ([]Tender, error) {
	var tenders []Tender
	prefix := []byte(portalID + ":")

	err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketTenders).Cursor()
		for k, v := c.Seek(prefix); k != nil && len(k) >= len(prefix) &&
			string(k[:len(prefix)]) == string(prefix); k, v = c.Next() {
			var t Tender
			if err := json.Unmarshal(v, &t); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			tenders = append(tenders, t)
		}
		return nil
	})

	return tenders, err
}

// GetAllTenders returns every tender across all portals
func (s *Store) GetAllTenders() ([]Tender, error) {
	var tenders []Tender
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketTenders).ForEach(func(_, v []byte) error {
			var t Tender
			if err := json.Unmarshal(v, &t); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			tenders = append(tenders, t)
			return nil
		})
	})
	return tenders, err
}

// hash produces a deterministic SHA256 fingerprint of a tender's content.
// It intentionally excludes Version, Status, LastUpdated, and ContentHash
// so the hash reflects only the actual tender data.
func hash(t Tender) string {
	h := sha256.New()
	h.Write([]byte(t.ReferenceNumber))
	h.Write([]byte(t.Title))
	h.Write([]byte(t.IssuingEntity))
	h.Write([]byte(t.Deadline.String()))
	h.Write([]byte(t.EstimatedValue))
	return fmt.Sprintf("%x", h.Sum(nil))
}
