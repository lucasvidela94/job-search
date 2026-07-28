package linkedin

import (
	"testing"
)

func TestJobageToTPR(t *testing.T) {
	tests := []struct {
		days int
		want string
	}{
		{0, ""},
		{1, "r86400"},
		{7, "r604800"},
		{30, "r2592000"},
		{9999, ""},
		{-1, ""},
	}
	for _, tt := range tests {
		got := jobageToTPR(tt.days)
		if got != tt.want {
			t.Errorf("jobageToTPR(%d) = %q; want %q", tt.days, got, tt.want)
		}
	}
}

func TestWorkTypeFlag(t *testing.T) {
	tests := []struct {
		mode string
		want string
	}{
		{"remote", "2"},
		{"hybrid", "3"},
		{"onsite", "1"},
		{"on-site", "1"},
		{"", ""},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := workTypeFlag(tt.mode)
		if got != tt.want {
			t.Errorf("workTypeFlag(%q) = %q; want %q", tt.mode, got, tt.want)
		}
	}
}

func TestDecodeHTMLEntities(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"&amp;", "&"},
		{"&lt;3", "<3"},
		{"&quot;hello&quot;", `"hello"`},
		{"&#233;", "é"},
		{"&#xE9;", "é"},
		{"&nbsp;", " "},
		{"no entities here", "no entities here"},
		{"mixed &amp; &lt; stuff", "mixed & < stuff"},
	}
	for _, tt := range tests {
		got := decodeHTMLEntities(tt.input)
		if got != tt.want {
			t.Errorf("decodeHTMLEntities(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestStripTags(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<div><span>nested</span></div>", "nested"},
		{"<br/>", ""},
		{"text <b>with</b> tags", "text with tags"},
		{"no tags", "no tags"},
	}
	for _, tt := range tests {
		got := stripTags(tt.input)
		if got != tt.want {
			t.Errorf("stripTags(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestClean(t *testing.T) {
	// strip then decode: &amp;amp; → &amp; (only outer decoded), &amp;lt; → &lt;
	input := "<p>Hello &amp;amp; &amp;lt; World</p>"
	want := "Hello &amp; < World"
	got := clean(input)
	if got != want {
		t.Errorf("clean(%q) = %q; want %q", input, got, want)
	}
}

func TestExtractDivContent(t *testing.T) {
	html := `<div class="content"><div class="nested">inner</div></div>`
	got := extractDivContent(html, "content")
	if got != `<div class="nested">inner</div>` {
		t.Errorf("extractDivContent = %q; want %q", got, `<div class="nested">inner</div>`)
	}

	// No match
	got2 := extractDivContent(html, "missing")
	if got2 != "" {
		t.Errorf("extractDivContent(missing) = %q; want empty", got2)
	}
}

func TestParseJobCardsNoEntityURN(t *testing.T) {
	// No data-entity-urn in the HTML → returns 0 cards
	html := `<div>no entity urn here</div>`
	cards := parseJobCards(html)
	if len(cards) != 0 {
		t.Fatalf("expected 0 cards (no data-entity-urn), got %d", len(cards))
	}
}

func TestParseJobCardsWithEntitySplit(t *testing.T) {
	// Simulate the actual LinkedIn response structure with entity-urn delimiters
	html := `junk_before
data-entity-urn="urn:li:jobPosting:123456"
<a class="base-card__full-link" href="/jobs/view/123456?position=1"></a>
<h3 class="base-search-card__title">Software Engineer</h3>
<h4 class="base-search-card__subtitle"><a href="/company/acme?trk=public">Acme Inc</a></h4>
<span class="job-search-card__location">San Francisco, CA</span>
<time class="job-search-card__listdate" datetime="2025-07-01"></time>
<div class="separator">next</div>
data-entity-urn="urn:li:jobPosting:789012"
<a class="base-card__full-link" href="/jobs/view/789012"></a>
<h3 class="base-search-card__title">Senior Go Developer</h3>
<h4 class="base-search-card__subtitle">Some Corp</h4>
<span class="job-search-card__location">Remote</span>
<time class="job-search-card__listdate" datetime="2025-07-15"></time>
`

	cards := parseJobCards(html)
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}
	if cards[0].ID != "123456" {
		t.Errorf("card[0].ID = %q; want 123456", cards[0].ID)
	}
	if cards[0].Title != "Software Engineer" {
		t.Errorf("card[0].Title = %q; want Software Engineer", cards[0].Title)
	}
	if cards[0].Company != "Acme Inc" {
		t.Errorf("card[0].Company = %q; want Acme Inc", cards[0].Company)
	}
	if cards[0].Location != "San Francisco, CA" {
		t.Errorf("card[0].Location = %q; want San Francisco, CA", cards[0].Location)
	}
	if cards[0].Date != "2025-07-01" {
		t.Errorf("card[0].Date = %q; want 2025-07-01", cards[0].Date)
	}
	if cards[1].ID != "789012" {
		t.Errorf("card[1].ID = %q; want 789012", cards[1].ID)
	}
	if cards[1].Title != "Senior Go Developer" {
		t.Errorf("card[1].Title = %q; want Senior Go Developer", cards[1].Title)
	}
}

func TestParseJobDetail(t *testing.T) {
	html := `
<div class="topcard__content">
	<h1 class="topcard__title">Go Backend Engineer</h1>
	<a class="topcard__org-name-link" href="/company/startupxyz">StartupXYZ</a>
	<span class="topcard__flavor topcard__flavor--bullet">Berlin, Germany</span>
</div>
<div class="show-more-less-html__markup">
	<ul>
		<li>Build APIs</li>
		<li>Write tests</li>
	</ul>
	<p>We use Go, Postgres, and Kafka.</p>
</div>
<h3 class="description__job-criteria-subheader">Seniority Level</h3>
<span class="description__job-criteria-text">Mid-Senior level</span>
<h3 class="description__job-criteria-subheader">Employment Type</h3>
<span class="description__job-criteria-text">Full-time</span>
<h3 class="description__job-criteria-subheader">Job Function</h3>
<span class="description__job-criteria-text">Engineering</span>
<h3 class="description__job-criteria-subheader">Industries</h3>
<span class="description__job-criteria-text">Software Development</span>
<a class="topcard__link" href="https://startupxyz.com/careers/apply"></a>
`

	d := parseJobDetail(html, "555")
	if d.ID != "555" {
		t.Errorf("ID = %q; want 555", d.ID)
	}
	if d.Title != "Go Backend Engineer" {
		t.Errorf("Title = %q; want Go Backend Engineer", d.Title)
	}
	if d.Company != "StartupXYZ" {
		t.Errorf("Company = %q; want StartupXYZ", d.Company)
	}
	if d.Location != "Berlin, Germany" {
		t.Errorf("Location = %q; want Berlin, Germany", d.Location)
	}
	if d.Seniority != "Mid-Senior level" {
		t.Errorf("Seniority = %q; want Mid-Senior level", d.Seniority)
	}
	if d.EmploymentType != "Full-time" {
		t.Errorf("EmploymentType = %q; want Full-time", d.EmploymentType)
	}
	if d.JobFunction != "Engineering" {
		t.Errorf("JobFunction = %q; want Engineering", d.JobFunction)
	}
	if d.Industries != "Software Development" {
		t.Errorf("Industries = %q; want Software Development", d.Industries)
	}
	// Description should contain the cleaned text
	if d.Description == "" {
		t.Errorf("Description is empty; expected content")
	}
	if d.URL != "https://www.linkedin.com/jobs/view/555" {
		t.Errorf("URL = %q; want https://www.linkedin.com/jobs/view/555", d.URL)
	}
	if d.ApplyURL != "https://startupxyz.com/careers/apply" {
		t.Errorf("ApplyURL = %q; want https://startupxyz.com/careers/apply", d.ApplyURL)
	}
}

func TestParseJobDetailNoCriteria(t *testing.T) {
	// Minimal HTML without criteria sections
	html := `<div class="topcard__content"><h2 class="top-card-layout__title">Dev</h2></div>`
	d := parseJobDetail(html, "1")
	if d.Title != "Dev" {
		t.Errorf("Title = %q; want Dev", d.Title)
	}
	if d.Seniority != "" {
		t.Errorf("expected empty Seniority, got %q", d.Seniority)
	}
}
