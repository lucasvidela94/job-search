// Package store provides atomic JSON file-based persistence for jobsearch state.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Store manages file-based state.
type Store struct {
	dir string
}

// New creates a Store rooted at dir. The directory is created if missing.
func New(dir string) *Store {
	return &Store{dir: dir}
}

// SeenEntry tracks a job posting seen during scraping.
type SeenEntry struct {
	Title     string `json:"title"`
	Company   string `json:"company"`
	URL       string `json:"url"`
	FirstSeen string `json:"first_seen"`
	Fit       string `json:"fit,omitempty"`
	Status    string `json:"status"` // new, skipped, ranked, expired
	Source    string `json:"portal,omitempty"`
	RankScore int    `json:"rank_score,omitempty"`
	RankDate  string `json:"rank_date,omitempty"`
}

// TrackerEntry records a job application.
type TrackerEntry struct {
	Company string `json:"company"`
	Role    string `json:"role"`
	URL     string `json:"url,omitempty"`
	Date    string `json:"date,omitempty"`
	Status  string `json:"status,omitempty"` // applied, interview, offer, rejected
}

// LoadJSON reads and unmarshals a JSON file.
func (s *Store) loadJSON(path string, dest any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// saveJSON writes data atomically to path (write tmp → rename).
func (s *Store) saveJSON(path string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0600); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename tmp: %w", err)
	}
	return nil
}

// LoadSeenJobs reads seen_jobs.json.
func (s *Store) LoadSeenJobs() (map[string]SeenEntry, error) {
	entries := map[string]SeenEntry{}
	path := filepath.Join(s.dir, "seen_jobs.json")
	if err := s.loadJSON(path, &entries); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = map[string]SeenEntry{}
	}
	return entries, nil
}

// SaveSeenJobs writes seen_jobs.json atomically.
func (s *Store) SaveSeenJobs(entries map[string]SeenEntry) error {
	path := filepath.Join(s.dir, "seen_jobs.json")
	return s.saveJSON(path, entries)
}

// LoadTracker reads tracker.json.
func (s *Store) LoadTracker() ([]TrackerEntry, error) {
	var entries []TrackerEntry
	path := filepath.Join(s.dir, "tracker.json")
	if err := s.loadJSON(path, &entries); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []TrackerEntry{}
	}
	return entries, nil
}

// SaveTracker writes tracker.json atomically.
func (s *Store) SaveTracker(entries []TrackerEntry) error {
	path := filepath.Join(s.dir, "tracker.json")
	return s.saveJSON(path, entries)
}
