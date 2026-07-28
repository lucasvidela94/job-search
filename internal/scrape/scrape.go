// Package scrape orchestrates job scraping across multiple portals,
// deduplicates against previously seen jobs, and persists new entries.
package scrape

import (
	"context"
	"fmt"
	"time"

	"github.com/lucasvidela94/jobsearch/internal/portal"
	"github.com/lucasvidela94/jobsearch/internal/store"
)

// Config defines a scrape run.
type Config struct {
	Portals  []string
	Query    string
	Location string
	Days     int
	Remote   string
	Limit    int
}

// Result summarizes a scrape run.
type Result struct {
	NewJobs  []portal.JobPosting
	Existing int
	Errors   []ScrapeError
}

// ScrapeError records a non-fatal error from a single portal.
type ScrapeError struct {
	Portal string
	Error  string
}

// Run executes a scrape across the configured portals, deduplicates against
// the store's seen jobs, and persists any new entries.
func Run(ctx context.Context, cfg Config, st *store.Store) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	seen, err := st.LoadSeenJobs()
	if err != nil {
		return nil, fmt.Errorf("load seen jobs: %w", err)
	}

	portals := resolvePortals(cfg.Portals)

	result := &Result{
		NewJobs: make([]portal.JobPosting, 0),
	}

	for name, p := range portals {
		jobs, scrapeErr := scrapePortal(ctx, p, cfg)
		if scrapeErr != nil {
			result.Errors = append(result.Errors, ScrapeError{
				Portal: name,
				Error:  scrapeErr.Error(),
			})
			continue
		}

		for _, job := range jobs {
			if _, alreadySeen := seen[job.URL]; alreadySeen {
				result.Existing++
				continue
			}

			result.NewJobs = append(result.NewJobs, job)
			seen[job.URL] = toSeenEntry(job)
		}
	}

	if len(result.NewJobs) > 0 {
		if err := st.SaveSeenJobs(seen); err != nil {
			return nil, fmt.Errorf("save seen jobs: %w", err)
		}
	}

	return result, nil
}

func resolvePortals(names []string) map[string]portal.Portal {
	if len(names) == 0 {
		return portal.All()
	}
	for _, n := range names {
		if n == "all" {
			return portal.All()
		}
	}

	res := make(map[string]portal.Portal, len(names))
	for _, n := range names {
		p, ok := portal.Get(n)
		if ok {
			res[n] = p
		}
	}
	return res
}

func scrapePortal(ctx context.Context, p portal.Portal, cfg Config) ([]portal.JobPosting, error) {
	params := portal.SearchParams{
		Query:    cfg.Query,
		Location: cfg.Location,
		Days:     cfg.Days,
		Remote:   cfg.Remote,
		Page:     1,
		Limit:    cfg.Limit,
	}

	res, err := p.Search(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%s search: %w", p.Name(), err)
	}
	if res == nil {
		return nil, nil
	}
	return res.Results, nil
}

func toSeenEntry(job portal.JobPosting) store.SeenEntry {
	return store.SeenEntry{
		Title:     job.Title,
		Company:   job.Company,
		URL:       job.URL,
		FirstSeen: time.Now().Format(time.RFC3339),
		Status:    "new",
		Source:    job.Source,
	}
}
