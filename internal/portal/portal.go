// Package portal defines the interface for job search sources.
package portal

import "context"

// SearchParams holds search criteria.
type SearchParams struct {
	Query    string
	Location string
	Days     int    // recency in days, 0 = any
	Remote   string // "remote", "hybrid", "onsite", ""
	Page     int    // 1-indexed
	Limit    int    // max results
}

// JobPosting represents a single job listing.
type JobPosting struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	CompanyURL  string `json:"company_url,omitempty"`
	Location    string `json:"location,omitempty"`
	Date        string `json:"date,omitempty"`    // ISO date
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`            // e.g. "linkedin"
	Seniority   string `json:"seniority,omitempty"`
	Employment  string `json:"employment_type,omitempty"`
	Salary      string `json:"salary,omitempty"`
}

// SearchResult contains search results.
type SearchResult struct {
	Results []JobPosting `json:"results"`
	Total   int          `json:"total,omitempty"`
	Page    int          `json:"page"`
}

// Portal is the interface all job sources must implement.
type Portal interface {
	Name() string
	Search(ctx context.Context, params SearchParams) (*SearchResult, error)
	Detail(ctx context.Context, id string) (*JobPosting, error)
}
