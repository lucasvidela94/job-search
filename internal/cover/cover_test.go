package cover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucasvidela94/jobsearch/internal/config"
	"github.com/lucasvidela94/jobsearch/internal/portal"
	"github.com/lucasvidela94/jobsearch/internal/profile"
)

func TestRender(t *testing.T) {
	p := &profile.Profile{
		Name:     "Lucas Videla",
		Title:    "Senior Go Developer",
		Skills:   []string{"Go", "Kubernetes", "PostgreSQL"},
	}
	job := portal.JobPosting{
		ID:      "123",
		Title:   "Go Backend Engineer",
		Company: "Acme Corp",
	}

	g, err := New(p, job)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	content, err := g.Render()
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if content == "" {
		t.Fatal("expected non-empty content")
	}
	if !contains(content, "Go Backend Engineer") {
		t.Error("expected job title in content")
	}
	if !contains(content, "Acme Corp") {
		t.Error("expected company in content")
	}
	if !contains(content, "Lucas Videla") {
		t.Error("expected name in content")
	}
	if !contains(content, "Go, Kubernetes, PostgreSQL") {
		t.Error("expected skills in content")
	}
}

func TestRenderToFile(t *testing.T) {
	dir := t.TempDir()
	p := &profile.Profile{
		Name:   "Test User",
		Title:  "Engineer",
		Skills: []string{"Go"},
	}
	job := portal.JobPosting{
		ID:      "456",
		Title:   "Backend Engineer",
		Company: "Test Corp",
	}

	g, err := New(p, job)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	path, err := g.RenderToFile(dir)
	if err != nil {
		t.Fatalf("RenderToFile: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestResolveTemplateOverride(t *testing.T) {
	// Create a user template override
	overrideDir := filepath.Join(config.ConfigDir(), "templates")
	os.MkdirAll(overrideDir, 0755)
	overridePath := filepath.Join(overrideDir, "cover.txt")
	customContent := "CUSTOM TEMPLATE: {{.JobTitle}}"
	os.WriteFile(overridePath, []byte(customContent), 0644)
	defer os.Remove(overridePath)

	p := &profile.Profile{Name: "Tester", Title: "Dev", Skills: []string{"Go"}}
	job := portal.JobPosting{ID: "1", Title: "Test Role", Company: "Test"}

	g, err := New(p, job)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	content, err := g.Render()
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !contains(content, "CUSTOM TEMPLATE") {
		t.Error("expected custom template content, got:", content)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
