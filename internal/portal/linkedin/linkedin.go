// Package linkedin implements job search on LinkedIn's public jobs-guest endpoints.
package linkedin

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/lucasvidela94/jobsearch/internal/httputil"
	"github.com/lucasvidela94/jobsearch/internal/portal"
)

const (
	searchURL = "https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search"
	detailURL = "https://www.linkedin.com/jobs-guest/jobs/api/jobPosting"
)

// LinkedIn implements portal.Portal for LinkedIn.
type LinkedIn struct {
	client *httputil.Client
}

// New creates a new LinkedIn portal.
func New() *LinkedIn {
	return &LinkedIn{
		client: httputil.NewDefaultClient(),
	}
}

// Name returns the portal name.
func (l *LinkedIn) Name() string { return "linkedin" }

// Search fetches job listings from LinkedIn's guest search endpoint.
func (l *LinkedIn) Search(ctx context.Context, params portal.SearchParams) (*portal.SearchResult, error) {
	u := buildSearchURL(params)
	html, err := l.client.FetchHTML(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("linkedin search: %w", err)
	}
	if html == "" {
		return &portal.SearchResult{Results: nil, Page: params.Page}, nil
	}

	cards := parseJobCards(html)
	if params.Limit > 0 && len(cards) > params.Limit {
		cards = cards[:params.Limit]
	}

	results := make([]portal.JobPosting, 0, len(cards))
	for _, c := range cards {
		results = append(results, portal.JobPosting{
			ID:         c.ID,
			Title:      c.Title,
			Company:    c.Company,
			CompanyURL: c.CompanyURL,
			Location:   c.Location,
			Date:       c.Date,
			URL:        c.URL,
			Source:     "linkedin",
		})
	}

	return &portal.SearchResult{
		Results: results,
		Page:    params.Page,
	}, nil
}

// Detail fetches a full job posting by ID.
func (l *LinkedIn) Detail(ctx context.Context, id string) (*portal.JobPosting, error) {
	u := detailURL + "/" + url.PathEscape(id)
	html, err := l.client.FetchHTML(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("linkedin detail: %w", err)
	}
	if html == "" {
		return nil, fmt.Errorf("linkedin detail: job %s not found", id)
	}

	jd := parseJobDetail(html, id)
	return &portal.JobPosting{
		ID:          jd.ID,
		Title:       jd.Title,
		Company:     jd.Company,
		CompanyURL:  jd.CompanyURL,
		Location:    jd.Location,
		URL:         jd.URL,
		Description: jd.Description,
		Source:      "linkedin",
		Seniority:   jd.Seniority,
		Employment:  jd.EmploymentType,
	}, nil
}

// buildSearchURL constructs the LinkedIn search URL from params.
func buildSearchURL(params portal.SearchParams) string {
	q := url.Values{}
	if params.Query != "" {
		q.Set("keywords", params.Query)
	}
	if params.Location != "" {
		q.Set("location", params.Location)
	}
	if tpr := jobageToTPR(params.Days); tpr != "" {
		q.Set("f_TPR", tpr)
	}
	if wt := workTypeFlag(params.Remote); wt != "" {
		q.Set("f_WT", wt)
	}
	start := 0
	if params.Page > 1 {
		start = (params.Page - 1) * 10
	}
	q.Set("start", strconv.Itoa(start))
	return searchURL + "?" + q.Encode()
}
