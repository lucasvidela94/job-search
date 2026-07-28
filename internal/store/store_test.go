package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeenJobsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	entries := map[string]SeenEntry{
		"job1": {
			Title:     "Engineer",
			Company:   "Acme",
			URL:       "https://example.com/job1",
			FirstSeen: "2026-07-28",
			Status:    "new",
			Source:    "linkedin",
		},
	}

	if err := s.SaveSeenJobs(entries); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.LoadSeenJobs()
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded))
	}
	entry := loaded["job1"]
	if entry.Title != "Engineer" || entry.Company != "Acme" {
		t.Errorf("got %+v, expected Engineer at Acme", entry)
	}
}

func TestSeenJobsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	entries, err := s.LoadSeenJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty map, got %d entries", len(entries))
	}
}

func TestTrackerRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	entries := []TrackerEntry{
		{Company: "Acme", Role: "Engineer", URL: "https://example.com", Date: "2026-07-28", Status: "applied"},
	}

	if err := s.SaveTracker(entries); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.LoadTracker()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded))
	}
	if loaded[0].Company != "Acme" || loaded[0].Role != "Engineer" {
		t.Errorf("got %+v, expected Acme/Engineer", loaded[0])
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	// Write initial data
	if err := s.SaveSeenJobs(map[string]SeenEntry{
		"k1": {Title: "original"},
	}); err != nil {
		t.Fatal(err)
	}

	// Verify no .tmp files remain
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".tmp" {
			t.Errorf("temporary file left behind: %s", f.Name())
		}
	}

	// Load and verify
	entries, err := s.LoadSeenJobs()
	if err != nil {
		t.Fatal(err)
	}
	if entries["k1"].Title != "original" {
		t.Errorf("expected original, got %+v", entries["k1"])
	}
}
