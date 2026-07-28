package freehire

import (
	"net/url"
	"testing"

	"github.com/lucasvidela94/jobsearch/internal/portal"
)

func TestBuildSearchURL_Defaults(t *testing.T) {
	f := &Freehire{baseURL: defaultBaseURL}
	raw := f.buildSearchURL(portal.SearchParams{})
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()

	if q.Get("include_description") != "true" {
		t.Errorf("include_description = %q; want true", q.Get("include_description"))
	}
	if q.Get("description_format") != "text" {
		t.Errorf("description_format = %q; want text", q.Get("description_format"))
	}
	if q.Get("limit") != "10" {
		t.Errorf("limit = %q; want 10", q.Get("limit"))
	}
	if q.Get("offset") != "" {
		t.Errorf("offset = %q; want empty", q.Get("offset"))
	}
}

func TestBuildSearchURL_FullParams(t *testing.T) {
	f := &Freehire{baseURL: defaultBaseURL}
	raw := f.buildSearchURL(portal.SearchParams{
		Query:  "golang kubernetes",
		Days:   7,
		Remote: "remote",
		Limit:  20,
		Page:   3,
	})
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()

	if q.Get("q") != "golang kubernetes" {
		t.Errorf("q = %q; want golang kubernetes", q.Get("q"))
	}
	if q.Get("posted_within_days") != "7" {
		t.Errorf("posted_within_days = %q; want 7", q.Get("posted_within_days"))
	}
	if q.Get("work_mode") != "remote" {
		t.Errorf("work_mode = %q; want remote", q.Get("work_mode"))
	}
	if q.Get("limit") != "20" {
		t.Errorf("limit = %q; want 20", q.Get("limit"))
	}
	if q.Get("offset") != "40" {
		t.Errorf("offset = %q; want 40", q.Get("offset"))
	}
}

func TestBuildSearchURL_Page1NoOffset(t *testing.T) {
	f := &Freehire{baseURL: defaultBaseURL}
	raw := f.buildSearchURL(portal.SearchParams{
		Query: "go",
		Page:  1,
		Limit: 10,
	})
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("offset") != "" {
		t.Errorf("offset = %q; want empty for page 1", u.Query().Get("offset"))
	}
}

func TestParseSearchResponse(t *testing.T) {
	resp := `{
		"data": [
			{
				"public_slug": "senior-go-dev-abc123",
				"url": "https://freehire.me/jobs/senior-go-dev-abc123",
				"title": "Senior Go Developer",
				"company": "TechCorp",
				"company_slug": "techcorp",
				"location": "Remote",
				"description": "We are looking for a senior Go developer with experience in Kubernetes.",
				"skills": ["go", "kubernetes", "postgresql"],
				"work_mode": "remote",
				"regions": ["eu"],
				"countries": ["de"],
				"cities": [],
				"posted_at": "2026-07-06T00:00:00Z",
				"enrichment": {
					"seniority": "senior",
					"category": "backend",
					"employment_type": "full_time",
					"salary_min": 90000,
					"salary_max": 120000,
					"salary_currency": "USD"
				}
			}
		],
		"meta": {
			"total": 1,
			"limit": 10,
			"offset": 0
		}
	}`

	jobs, total, err := parseSearchResponse(resp)
	if err != nil {
		t.Fatal(err)
	}

	if total != 1 {
		t.Errorf("total = %d; want 1", total)
	}

	if len(jobs) != 1 {
		t.Fatalf("got %d jobs; want 1", len(jobs))
	}

	job := jobs[0]
	if job.ID != "senior-go-dev-abc123" {
		t.Errorf("ID = %q; want senior-go-dev-abc123", job.ID)
	}
	if job.Title != "Senior Go Developer" {
		t.Errorf("Title = %q; want Senior Go Developer", job.Title)
	}
	if job.Company != "TechCorp" {
		t.Errorf("Company = %q; want TechCorp", job.Company)
	}
	if job.Location != "Remote" {
		t.Errorf("Location = %q; want Remote", job.Location)
	}
	if job.Date != "2026-07-06T00:00:00Z" {
		t.Errorf("Date = %q; want 2026-07-06T00:00:00Z", job.Date)
	}
	if job.URL != "https://freehire.me/jobs/senior-go-dev-abc123" {
		t.Errorf("URL = %q; want https://freehire.me/jobs/senior-go-dev-abc123", job.URL)
	}
	if job.Source != "freehire" {
		t.Errorf("Source = %q; want freehire", job.Source)
	}
}

func TestParseSearchResponse_NoResults(t *testing.T) {
	resp := `{"data": [], "meta": {"total": 0, "limit": 10, "offset": 0}}`
	jobs, total, err := parseSearchResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Errorf("total = %d; want 0", total)
	}
	if len(jobs) != 0 {
		t.Errorf("got %d jobs; want 0", len(jobs))
	}
}

func TestParseJobEnrichment(t *testing.T) {
	job := freehireJob{
		PublicSlug:  "test-slug",
		Title:       "Senior Engineer",
		Company:     "Acme",
		URL:         "https://freehire.me/jobs/test-slug",
		PostedAt:    "2026-07-06T00:00:00Z",
		Description: "A great job.",
		Enrichment: &enrichment{
			Seniority:      "senior",
			EmploymentType: "full_time",
			SalaryMin:      90000,
			SalaryMax:      120000,
			SalaryCurrency: "USD",
		},
	}

	p := parseJob(job)
	if p.Seniority != "senior" {
		t.Errorf("Seniority = %q; want senior", p.Seniority)
	}
	if p.Employment != "full_time" {
		t.Errorf("Employment = %q; want full_time", p.Employment)
	}
	if p.Salary != "USD 90000 - 120000" {
		t.Errorf("Salary = %q; want USD 90000 - 120000", p.Salary)
	}
}

func TestParseJobNoEnrichment(t *testing.T) {
	job := freehireJob{
		PublicSlug: "test-slug",
		Title:      "Junior Dev",
		Company:    "Acme",
		URL:        "https://freehire.me/jobs/test-slug",
		PostedAt:   "2026-07-06T00:00:00Z",
	}

	p := parseJob(job)
	if p.Seniority != "" {
		t.Errorf("expected empty Seniority, got %q", p.Seniority)
	}
	if p.Employment != "" {
		t.Errorf("expected empty Employment, got %q", p.Employment)
	}
	if p.Salary != "" {
		t.Errorf("expected empty Salary, got %q", p.Salary)
	}
}

func TestFormatSalary(t *testing.T) {
	tests := []struct {
		min      int
		max      int
		currency string
		want     string
	}{
		{0, 0, "", ""},
		{90000, 0, "", "90000"},
		{0, 120000, "", "120000"},
		{90000, 120000, "", "90000 - 120000"},
		{90000, 120000, "USD", "USD 90000 - 120000"},
		{50000, 0, "EUR", "EUR 50000"},
	}
	for _, tt := range tests {
		got := formatSalary(tt.min, tt.max, tt.currency)
		if got != tt.want {
			t.Errorf("formatSalary(%d, %d, %q) = %q; want %q", tt.min, tt.max, tt.currency, got, tt.want)
		}
	}
}

func TestParseSearchResponse_APIError(t *testing.T) {
	resp := `{"data": null, "error": "rate limit exceeded"}`
	_, _, err := parseSearchResponse(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
