package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir) // so ConfigDir resolves inside temp

	p := &Profile{
		Name:     "Test User",
		Title:    "Senior Engineer",
		Skills:   []string{"Go", "Kubernetes"},
		Remote:   "remote",
	}

	if err := Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Name != "Test User" {
		t.Errorf("Name = %q; want Test User", loaded.Name)
	}
	if len(loaded.Skills) != 2 {
		t.Errorf("len(Skills) = %d; want 2", len(loaded.Skills))
	}
}

func TestSetSkill(t *testing.T) {
	p := &Profile{Skills: []string{"Go"}}
	p.SetSkill("Kubernetes")
	if len(p.Skills) != 2 {
		t.Errorf("len = %d; want 2", len(p.Skills))
	}

	// Dedup
	p.SetSkill("go")
	if len(p.Skills) != 2 {
		t.Errorf("len after dup = %d; want 2", len(p.Skills))
	}
}

func TestSetField(t *testing.T) {
	p := &Profile{}

	if err := p.SetField("name", "Alice"); err != nil {
		t.Fatal(err)
	}
	if p.Name != "Alice" {
		t.Errorf("Name = %q; want Alice", p.Name)
	}

	if err := p.SetField("remote", "hybrid"); err != nil {
		t.Fatal(err)
	}
	if p.Remote != "hybrid" {
		t.Errorf("Remote = %q; want hybrid", p.Remote)
	}

	if err := p.SetField("min_salary", "120000"); err != nil {
		t.Fatal(err)
	}
	if p.MinSalary != 120000 {
		t.Errorf("MinSalary = %d; want 120000", p.MinSalary)
	}

	if err := p.SetField("unknown", "x"); err == nil {
		t.Error("expected error for unknown field")
	}
}

func TestSaveCreatesDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", "")

	p := &Profile{Name: "Test"}
	if err := Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path := filepath.Join(dir, ".config", "jobsearch", "profile.json")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("profile.json not created: %v", err)
	}
}
