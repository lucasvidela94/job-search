package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	// Verify DB file was created
	dbPath := filepath.Join(dir, "state", "candidate.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("DB file not created: %v", err)
	}
}

func TestMigrateCreatesTables(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	// Verify tables exist
	tables := []string{"applications", "events"}
	for _, name := range tables {
		var exists int
		err := d.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %s: %v", name, err)
		}
		if exists != 1 {
			t.Errorf("table %s not found", name)
		}
	}
}

func TestOpenTwice(t *testing.T) {
	dir := t.TempDir()
	d1, err := Open(dir)
	if err != nil {
		t.Fatalf("Open 1: %v", err)
	}
	d1.Close()

	// Re-open should work (already migrated)
	d2, err := Open(dir)
	if err != nil {
		t.Fatalf("Open 2: %v", err)
	}
	d2.Close()
}

func TestDir(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	expected := filepath.Join(dir, "state")
	if d.Dir() != expected {
		t.Errorf("Dir() = %q; want %q", d.Dir(), expected)
	}
}
