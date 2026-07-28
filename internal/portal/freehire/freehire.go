package freehire

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/lucasvidela94/jobsearch/internal/httputil"
	"github.com/lucasvidela94/jobsearch/internal/portal"
)

const (
	defaultBaseURL = "https://freehire.me"
	searchPath     = "/api/v1/agent/jobs/search"
	detailPath     = "/api/v1/jobs"
)

type Freehire struct {
	client  *httputil.Client
	baseURL string
}

func New() *Freehire {
	baseURL := os.Getenv("FREEHIRE_API_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Freehire{
		client:  httputil.NewDefaultClient(),
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (f *Freehire) Name() string { return "freehire" }

func (f *Freehire) Search(ctx context.Context, params portal.SearchParams) (*portal.SearchResult, error) {
	u := f.buildSearchURL(params)
	body, err := f.client.FetchHTML(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("freehire search: %w", err)
	}
	if body == "" {
		return &portal.SearchResult{Results: nil, Page: params.Page}, nil
	}

	jobs, total, err := parseSearchResponse(body)
	if err != nil {
		return nil, fmt.Errorf("freehire search: %w", err)
	}

	return &portal.SearchResult{
		Results: jobs,
		Total:   total,
		Page:    params.Page,
	}, nil
}

func (f *Freehire) Detail(ctx context.Context, id string) (*portal.JobPosting, error) {
	u := f.baseURL + detailPath + "/" + url.PathEscape(id)
	body, err := f.client.FetchHTML(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("freehire detail: %w", err)
	}
	if body == "" {
		return nil, fmt.Errorf("freehire detail: job %s not found", id)
	}

	var resp freehireResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("freehire detail: parse response: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("freehire detail: api error: %s", resp.Error)
	}

	var job freehireJob
	if err := json.Unmarshal(resp.Data, &job); err != nil {
		return nil, fmt.Errorf("freehire detail: parse job: %w", err)
	}

	p := parseJob(job)
	return &p, nil
}

func (f *Freehire) buildSearchURL(params portal.SearchParams) string {
	q := url.Values{}
	if params.Query != "" {
		q.Set("q", params.Query)
	}
	if params.Days > 0 {
		q.Set("posted_within_days", strconv.Itoa(params.Days))
	}
	if params.Remote != "" {
		q.Set("work_mode", params.Remote)
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 10
	}
	q.Set("limit", strconv.Itoa(limit))

	offset := 0
	if params.Page > 1 {
		offset = (params.Page - 1) * limit
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}

	q.Set("include_description", "true")
	q.Set("description_format", "text")

	return f.baseURL + searchPath + "?" + q.Encode()
}

type freehireResponse struct {
	Data  json.RawMessage `json:"data"`
	Meta  *meta           `json:"meta,omitempty"`
	Error string          `json:"error,omitempty"`
}

type meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type freehireJob struct {
	PublicSlug  string      `json:"public_slug"`
	URL         string      `json:"url"`
	Title       string      `json:"title"`
	Company     string      `json:"company"`
	CompanySlug string      `json:"company_slug"`
	Location    string      `json:"location"`
	Description string      `json:"description"`
	Skills      []string    `json:"skills"`
	WorkMode    string      `json:"work_mode"`
	Regions     []string    `json:"regions"`
	Countries   []string    `json:"countries"`
	Cities      []string    `json:"cities"`
	PostedAt    string      `json:"posted_at"`
	Enrichment  *enrichment `json:"enrichment,omitempty"`
}

type enrichment struct {
	Seniority      string `json:"seniority"`
	Category       string `json:"category"`
	EmploymentType string `json:"employment_type"`
	SalaryMin      int    `json:"salary_min"`
	SalaryMax      int    `json:"salary_max"`
	SalaryCurrency string `json:"salary_currency"`
}

func parseSearchResponse(data string) ([]portal.JobPosting, int, error) {
	var resp freehireResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, 0, fmt.Errorf("parse response: %w", err)
	}
	if resp.Error != "" {
		return nil, 0, fmt.Errorf("api error: %s", resp.Error)
	}

	var jobs []freehireJob
	if err := json.Unmarshal(resp.Data, &jobs); err != nil {
		return nil, 0, fmt.Errorf("parse jobs: %w", err)
	}

	total := 0
	if resp.Meta != nil {
		total = resp.Meta.Total
	}

	result := make([]portal.JobPosting, 0, len(jobs))
	for _, j := range jobs {
		result = append(result, parseJob(j))
	}
	return result, total, nil
}

func parseJob(j freehireJob) portal.JobPosting {
	p := portal.JobPosting{
		ID:          j.PublicSlug,
		Title:       j.Title,
		Company:     j.Company,
		Location:    j.Location,
		Description: j.Description,
		Date:        j.PostedAt,
		URL:         j.URL,
		Source:      "freehire",
	}

	if j.Enrichment != nil {
		p.Seniority = j.Enrichment.Seniority
		p.Employment = j.Enrichment.EmploymentType
		p.Salary = formatSalary(j.Enrichment.SalaryMin, j.Enrichment.SalaryMax, j.Enrichment.SalaryCurrency)
	}

	return p
}

func formatSalary(min, max int, currency string) string {
	var parts []string
	if min > 0 {
		parts = append(parts, strconv.Itoa(min))
	}
	if max > 0 {
		parts = append(parts, strconv.Itoa(max))
	}
	if len(parts) == 0 {
		return ""
	}
	s := strings.Join(parts, " - ")
	if currency != "" {
		s = currency + " " + s
	}
	return s
}
