package applications

import (
	"context"
	"testing"

	"github.com/lucasvidela94/jobsearch/internal/db"
	"github.com/lucasvidela94/jobsearch/internal/portal"
)

func setupTest(t *testing.T) (*Repository, context.Context) {
	t.Helper()
	dir := t.TempDir()
	d, err := db.Open(dir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return NewRepository(d), context.Background()
}

func makeJob(id string) portal.JobPosting {
	return portal.JobPosting{
		ID:       id,
		Title:    "Go Developer",
		Company:  "Acme Corp",
		URL:      "https://example.com/jobs/" + id,
		Source:   "linkedin",
		ApplyURL: "https://example.com/apply/" + id,
	}
}

func TestCreate(t *testing.T) {
	r, ctx := setupTest(t)
	app, err := r.Create(ctx, makeJob("123"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if app.ID == "" {
		t.Error("app.ID is empty")
	}
	if app.Title != "Go Developer" {
		t.Errorf("app.Title = %q; want Go Developer", app.Title)
	}
	if app.Status != "applied" {
		t.Errorf("app.Status = %q; want applied", app.Status)
	}
}

func TestGetByID(t *testing.T) {
	r, ctx := setupTest(t)
	created, err := r.Create(ctx, makeJob("456"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := r.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != created.Title {
		t.Errorf("got.Title = %q; want %q", got.Title, created.Title)
	}
	if got.Status != "applied" {
		t.Errorf("got.Status = %q; want applied", got.Status)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	r, ctx := setupTest(t)
	_, err := r.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestList(t *testing.T) {
	r, ctx := setupTest(t)
	if _, err := r.Create(ctx, makeJob("1")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Create(ctx, makeJob("2")); err != nil {
		t.Fatal(err)
	}

	apps, err := r.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(apps) != 2 {
		t.Errorf("len(apps) = %d; want 2", len(apps))
	}
}

func TestListFiltered(t *testing.T) {
	r, ctx := setupTest(t)
	if _, err := r.Create(ctx, makeJob("1")); err != nil {
		t.Fatal(err)
	}

	// All should match "applied"
	apps, err := r.List(ctx, "applied")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(apps) != 1 {
		t.Errorf("len(apps) = %d; want 1", len(apps))
	}

	// None should match "rejected"
	apps, err = r.List(ctx, "rejected")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("len(apps) = %d; want 0", len(apps))
	}
}

func TestAddEvent(t *testing.T) {
	r, ctx := setupTest(t)
	app, err := r.Create(ctx, makeJob("789"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	evt, err := r.AddEvent(ctx, app.ID, "phone_screen", "Had a great conversation")
	if err != nil {
		t.Fatalf("AddEvent: %v", err)
	}

	if evt.Type != "phone_screen" {
		t.Errorf("evt.Type = %q; want phone_screen", evt.Type)
	}
	if evt.Notes != "Had a great conversation" {
		t.Errorf("evt.Notes = %q; want Had a great conversation", evt.Notes)
	}

	// Verify status updated
	updated, err := r.GetByID(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.Status != "phone_screen" {
		t.Errorf("updated.Status = %q; want phone_screen", updated.Status)
	}
}

func TestAddEventNotFound(t *testing.T) {
	r, ctx := setupTest(t)
	_, err := r.AddEvent(ctx, "nonexistent", "test", "")
	if err == nil {
		t.Fatal("expected error for nonexistent application")
	}
}

func TestGetEvents(t *testing.T) {
	r, ctx := setupTest(t)
	app, err := r.Create(ctx, makeJob("101"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Add a second event
	if _, err := r.AddEvent(ctx, app.ID, "tech_interview", "Went well"); err != nil {
		t.Fatal(err)
	}

	events, err := r.GetEvents(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("len(events) = %d; want 2", len(events))
	}
	if events[0].Type != "applied" {
		t.Errorf("events[0].Type = %q; want applied", events[0].Type)
	}
	if events[1].Type != "tech_interview" {
		t.Errorf("events[1].Type = %q; want tech_interview", events[1].Type)
	}
}

func TestFindByURL(t *testing.T) {
	r, ctx := setupTest(t)
	if _, err := r.Create(ctx, makeJob("202")); err != nil {
		t.Fatal(err)
	}

	// Should find by URL
	found, err := r.FindByURL(ctx, "https://example.com/jobs/202")
	if err != nil {
		t.Fatalf("FindByURL: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find application")
	}
	if found.Title != "Go Developer" {
		t.Errorf("found.Title = %q; want Go Developer", found.Title)
	}

	// Should NOT find non-existent URL
	notFound, err := r.FindByURL(ctx, "https://example.com/jobs/nonexistent")
	if err != nil {
		t.Fatalf("FindByURL: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-existent URL")
	}
}

func TestCountByStatus(t *testing.T) {
	r, ctx := setupTest(t)
	if _, err := r.Create(ctx, makeJob("1")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Create(ctx, makeJob("2")); err != nil {
		t.Fatal(err)
	}

	counts, err := r.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("CountByStatus: %v", err)
	}
	if counts["applied"] != 2 {
		t.Errorf("counts[applied] = %d; want 2", counts["applied"])
	}
}
