// Package applications manages job application lifecycle.
package applications

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lucasvidela94/jobsearch/internal/db"
	"github.com/lucasvidela94/jobsearch/internal/portal"
)

// Application represents a job application with its latest status.
type Application struct {
	ID        string `json:"id"`
	JobID     string `json:"job_id"`
	Title     string `json:"title"`
	Company   string `json:"company"`
	URL       string `json:"url"`
	Source    string `json:"source"`
	ApplyURL  string `json:"apply_url,omitempty"`
	AppliedAt string `json:"applied_at"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Event represents a single event in an application's timeline.
type Event struct {
	ID            string `json:"id"`
	ApplicationID string `json:"application_id"`
	Type          string `json:"type"`
	Notes         string `json:"notes,omitempty"`
	Timestamp     string `json:"timestamp"`
}

// Repository provides CRUD operations for applications and events.
type Repository struct {
	db *db.DB
}

// NewRepository creates a new Repository backed by the given DB.
func NewRepository(d *db.DB) *Repository {
	return &Repository{db: d}
}

// Create registers a new application for a job posting.
// The first event is automatically created with type "applied".
func (r *Repository) Create(ctx context.Context, job portal.JobPosting) (*Application, error) {
	appID := uuid.New().String()
	eventID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO applications (id, job_id, title, company, url, source, apply_url, applied_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		appID, job.ID, job.Title, job.Company, job.URL, job.Source, job.ApplyURL, now, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert application: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO events (id, application_id, type, notes, timestamp)
		 VALUES (?, ?, 'applied', ?, ?)`,
		eventID, appID, "", now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &Application{
		ID:        appID,
		JobID:     job.ID,
		Title:     job.Title,
		Company:   job.Company,
		URL:       job.URL,
		Source:    job.Source,
		ApplyURL:  job.ApplyURL,
		AppliedAt: now,
		Status:    "applied",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetByID returns an application with its latest event type as Status.
func (r *Repository) GetByID(ctx context.Context, id string) (*Application, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT a.id, a.job_id, a.title, a.company, a.url, a.source, a.apply_url,
		       a.applied_at, a.created_at, a.updated_at,
		       COALESCE((SELECT e.type FROM events e WHERE e.application_id = a.id ORDER BY e.timestamp DESC LIMIT 1), 'unknown')
		FROM applications a
		WHERE a.id = ?`, id)

	var app Application
	err := row.Scan(&app.ID, &app.JobID, &app.Title, &app.Company, &app.URL,
		&app.Source, &app.ApplyURL, &app.AppliedAt, &app.CreatedAt, &app.UpdatedAt, &app.Status)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("application not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("scan application: %w", err)
	}
	return &app, nil
}

// List returns all applications, optionally filtered by status.
func (r *Repository) List(ctx context.Context, statusFilter string) ([]Application, error) {
	query := `
		SELECT a.id, a.job_id, a.title, a.company, a.url, a.source, a.apply_url,
		       a.applied_at, a.created_at, a.updated_at,
		       COALESCE((SELECT e.type FROM events e WHERE e.application_id = a.id ORDER BY e.timestamp DESC LIMIT 1), 'unknown')
		FROM applications a`
	args := []interface{}{}

	if statusFilter != "" {
		query += ` WHERE (SELECT e.type FROM events e WHERE e.application_id = a.id ORDER BY e.timestamp DESC LIMIT 1) = ?`
		args = append(args, statusFilter)
	}
	query += ` ORDER BY a.updated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []Application
	for rows.Next() {
		var app Application
		if err := rows.Scan(&app.ID, &app.JobID, &app.Title, &app.Company, &app.URL,
			&app.Source, &app.ApplyURL, &app.AppliedAt, &app.CreatedAt, &app.UpdatedAt, &app.Status); err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

// AddEvent adds an event to an application and updates its updated_at timestamp.
func (r *Repository) AddEvent(ctx context.Context, appID, eventType, notes string) (*Event, error) {
	eventID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx,
		`UPDATE applications SET updated_at = ? WHERE id = ?`, now, appID)
	if err != nil {
		return nil, fmt.Errorf("update application: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("application not found: %s", appID)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO events (id, application_id, type, notes, timestamp)
		 VALUES (?, ?, ?, ?, ?)`,
		eventID, appID, eventType, notes, now)
	if err != nil {
		return nil, fmt.Errorf("insert event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &Event{
		ID:            eventID,
		ApplicationID: appID,
		Type:          eventType,
		Notes:         notes,
		Timestamp:     now,
	}, nil
}

// GetEvents returns all events for an application, ordered by timestamp.
func (r *Repository) GetEvents(ctx context.Context, appID string) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, application_id, type, notes, timestamp
		 FROM events WHERE application_id = ?
		 ORDER BY timestamp ASC`, appID)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.ApplicationID, &e.Type, &e.Notes, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// FindByURL checks if a job URL already has an application.
func (r *Repository) FindByURL(ctx context.Context, url string) (*Application, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT a.id, a.job_id, a.title, a.company, a.url, a.source, a.apply_url,
		       a.applied_at, a.created_at, a.updated_at,
		       COALESCE((SELECT e.type FROM events e WHERE e.application_id = a.id ORDER BY e.timestamp DESC LIMIT 1), 'unknown')
		FROM applications a WHERE a.url = ?`, url)

	var app Application
	err := row.Scan(&app.ID, &app.JobID, &app.Title, &app.Company, &app.URL,
		&app.Source, &app.ApplyURL, &app.AppliedAt, &app.CreatedAt, &app.UpdatedAt, &app.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find by url: %w", err)
	}
	return &app, nil
}

// CountByStatus returns the number of applications grouped by latest event type.
func (r *Repository) CountByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT COALESCE(e.type, 'unknown') AS status, COUNT(*)
		FROM applications a
		LEFT JOIN events e ON e.id = (
			SELECT e2.id FROM events e2 WHERE e2.application_id = a.id ORDER BY e2.timestamp DESC LIMIT 1
		)
		GROUP BY status
		ORDER BY status`)
	if err != nil {
		return nil, fmt.Errorf("count by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}
