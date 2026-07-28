package linkedin

import (
	"regexp"
	"strconv"
	"strings"
)

// --- Shared types ---

// JobCard represents a job listing in search results.
type JobCard struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Company    string `json:"company"`
	CompanyURL string `json:"company_url,omitempty"`
	Location   string `json:"location,omitempty"`
	Date       string `json:"date,omitempty"`
	URL        string `json:"url"`
}

// JobDetail represents a full job description.
type JobDetail struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Company        string `json:"company"`
	CompanyURL     string `json:"company_url,omitempty"`
	Location       string `json:"location,omitempty"`
	URL            string `json:"url"`
	Description    string `json:"description,omitempty"`
	Seniority      string `json:"seniority,omitempty"`
	EmploymentType string `json:"employment_type,omitempty"`
	JobFunction    string `json:"job_function,omitempty"`
	Industries     string `json:"industries,omitempty"`
	ApplyURL       string `json:"apply_url,omitempty"`
}

// --- HTML entity decoding ---

func numericEntity(cp int) string {
	if cp >= 0 && cp <= 0x10ffff {
		return string(rune(cp))
	}
	return ""
}

func decodeHTMLEntities(text string) string {
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&apos;", "'")
	// Numeric decimal: &#233;
	reDec := regexp.MustCompile(`&#(\d+);`)
	text = reDec.ReplaceAllStringFunc(text, func(m string) string {
		num := reDec.FindStringSubmatch(m)[1]
		if n, err := strconv.Atoi(num); err == nil {
			return numericEntity(n)
		}
		return m
	})
	// Numeric hex: &#xE9;
	reHex := regexp.MustCompile(`&#[xX]([0-9a-fA-F]+);`)
	text = reHex.ReplaceAllStringFunc(text, func(m string) string {
		hex := reHex.FindStringSubmatch(m)[1]
		if n, err := strconv.ParseInt(hex, 16, 32); err == nil {
			return numericEntity(int(n))
		}
		return m
	})
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	return text
}

func stripTags(html string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(html, " ")
	// Collapse whitespace
	ws := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(ws.ReplaceAllString(text, " "))
}

func clean(html string) string {
	return decodeHTMLEntities(stripTags(html))
}

// extractDivContent extracts the inner HTML of a <div> identified by a CSS class name,
// correctly handling nested <div> elements by tracking tag depth.
func extractDivContent(html, className string) string {
	escaped := regexp.QuoteMeta(className)
	openRe := regexp.MustCompile(`(?i)<div[^>]*class="[^"]*` + escaped + `[^"]*"[^>]*>`)
	open := openRe.FindStringIndex(html)
	if open == nil {
		return ""
	}

	i := open[1] // position after the opening tag
	depth := 1

	for depth > 0 && i < len(html) {
		nextOpen := indexOf(html, "<div", i)
		nextClose := indexOf(html, "</div>", i)

		if nextClose == -1 {
			return ""
		}

		if nextOpen != -1 && nextOpen < nextClose {
			depth++
			i = nextOpen + 4
		} else {
			depth--
			i = nextClose + 6
		}
	}

	return html[open[1] : i-6]
}

func indexOf(s, substr string, start int) int {
	idx := strings.Index(s[start:], substr)
	if idx == -1 {
		return -1
	}
	return start + idx
}

// --- Search parsing ---

// parseJobCards parses the search response HTML into job cards.
// Splits on the job-posting URN so one malformed card doesn't break the rest.
func parseJobCards(html string) []JobCard {
	var results []JobCard
	chunks := strings.Split(html, `data-entity-urn="urn:li:jobPosting:`)

	for _, chunk := range chunks[1:] { // skip the first chunk (before any split)
		idMatch := regexp.MustCompile(`^(\d+)`).FindStringSubmatch(chunk)
		if len(idMatch) < 2 {
			continue
		}
		id := idMatch[1]

		// Full link
		linkRe := regexp.MustCompile(`(?i)class="base-card__full-link[^"]*"[^>]*href="([^"]+)"`)
		linkMatch := linkRe.FindStringSubmatch(chunk)
		url := ""
		if len(linkMatch) >= 2 {
			url = decodeHTMLEntities(linkMatch[1])
			if idx := strings.Index(url, "?"); idx != -1 {
				url = url[:idx]
			}
		}

		// Title from h3 or sr-only span
		title := ""
		h3Re := regexp.MustCompile(`(?i)class="base-search-card__title"[^>]*>([\s\S]*?)</h3>`)
		h3Match := h3Re.FindStringSubmatch(chunk)
		if len(h3Match) >= 2 {
			title = clean(h3Match[1])
		}
		if title == "" {
			srRe := regexp.MustCompile(`(?i)class="sr-only"[^>]*>([\s\S]*?)</span>`)
			srMatch := srRe.FindStringSubmatch(chunk)
			if len(srMatch) >= 2 {
				title = clean(srMatch[1])
			}
		}
		if title == "" {
			continue
		}

		// Company
		company := ""
		companyURL := ""
		subRe := regexp.MustCompile(`(?i)class="base-search-card__subtitle"[^>]*>([\s\S]*?)</h4>`)
		subMatch := subRe.FindStringSubmatch(chunk)
		if len(subMatch) >= 2 {
			aRe := regexp.MustCompile(`href="([^"]+)"`)
			aMatch := aRe.FindStringSubmatch(subMatch[1])
			if len(aMatch) >= 2 {
				companyURL = decodeHTMLEntities(aMatch[1])
				if idx := strings.Index(companyURL, "?"); idx != -1 {
					companyURL = companyURL[:idx]
				}
			}
			company = clean(subMatch[1])
		}

		// Location
		locRe := regexp.MustCompile(`(?i)class="job-search-card__location"[^>]*>([\s\S]*?)</span>`)
		locMatch := locRe.FindStringSubmatch(chunk)
		location := ""
		if len(locMatch) >= 2 {
			location = clean(locMatch[1])
		}

		// Date
		dateRe := regexp.MustCompile(`(?i)class="job-search-card__listdate[^"]*"[^>]*datetime="([^"]+)"`)
		dateMatch := dateRe.FindStringSubmatch(chunk)
		date := ""
		if len(dateMatch) >= 2 {
			date = dateMatch[1]
		}

		linkURL := url
		if linkURL == "" {
			linkURL = "https://www.linkedin.com/jobs/view/" + id
		}

		results = append(results, JobCard{
			ID:         id,
			Title:      title,
			Company:    company,
			CompanyURL: companyURL,
			Location:   location,
			Date:       date,
			URL:        linkURL,
		})
	}

	return results
}

// --- Detail parsing ---

// parseJobDetail parses a single job posting detail page.
func parseJobDetail(html string, id string) *JobDetail {
	d := &JobDetail{ID: id, URL: "https://www.linkedin.com/jobs/view/" + id}

	// Title
	titleRe := regexp.MustCompile(`(?i)class="(?:top-card-layout__title|topcard__title)[^"]*"[^>]*>([\s\S]*?)</h[12]>`)
	titleMatch := titleRe.FindStringSubmatch(html)
	if len(titleMatch) >= 2 {
		d.Title = clean(titleMatch[1])
	}

	// Company
	orgRe := regexp.MustCompile(`(?i)class="topcard__org-name-link[^"]*"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	orgMatch := orgRe.FindStringSubmatch(html)
	if len(orgMatch) >= 3 {
		d.Company = clean(orgMatch[2])
		cu := decodeHTMLEntities(orgMatch[1])
		if idx := strings.Index(cu, "?"); idx != -1 {
			cu = cu[:idx]
		}
		d.CompanyURL = cu
	}

	// Location
	locRe := regexp.MustCompile(`(?i)class="topcard__flavor topcard__flavor--bullet"[^>]*>([\s\S]*?)</span>`)
	locMatch := locRe.FindStringSubmatch(html)
	if len(locMatch) >= 2 {
		d.Location = clean(locMatch[1])
	}

	// Description
	descHTML := extractDivContent(html, "show-more-less-html__markup")
	if descHTML == "" {
		descHTML = extractDivContent(html, "description__text")
	}
	if descHTML != "" {
		withBreaks := strings.NewReplacer(
			"<br>", "\n", "<br/>", "\n", "<br />", "\n",
		).Replace(descHTML)
		// Also add newlines after block-level closing tags
		blockRe := regexp.MustCompile(`</(p|li|ul|ol|div|h\d)>`)
		withBreaks = blockRe.ReplaceAllString(withBreaks, "\n")
		d.Description = decodeHTMLEntities(stripTags(withBreaks))
		// Collapse multiple newlines
		nlRe := regexp.MustCompile(`\n{3,}`)
		d.Description = strings.TrimSpace(nlRe.ReplaceAllString(d.Description, "\n\n"))
	}

	// Job criteria (seniority, employment type, job function, industries)
	criteriaRe := regexp.MustCompile(`(?i)class="description__job-criteria-subheader"[^>]*>([\s\S]*?)</h3>[\s\S]*?class="description__job-criteria-text[^"]*"[^>]*>([\s\S]*?)</span>`)
	criteriaMatches := criteriaRe.FindAllStringSubmatch(html, -1)
	criteria := make(map[string]string)
	for _, m := range criteriaMatches {
		if len(m) >= 3 {
			key := strings.ToLower(clean(m[1]))
			val := clean(m[2])
			criteria[key] = val
		}
	}
	d.Seniority = criteria["seniority level"]
	d.EmploymentType = criteria["employment type"]
	d.JobFunction = criteria["job function"]
	d.Industries = criteria["industries"]

	// Apply URL
	applyRe := regexp.MustCompile(`(?i)class="topcard__link[^"]*"[^>]*href="([^"]+)"`)
	applyMatch := applyRe.FindStringSubmatch(html)
	if len(applyMatch) >= 2 {
		au := decodeHTMLEntities(applyMatch[1])
		if idx := strings.Index(au, "?"); idx != -1 {
			au = au[:idx]
		}
		d.ApplyURL = au
	}

	return d
}

// --- URL helpers ---

// jobageToTPR converts job age in days to LinkedIn's f_TPR seconds value.
func jobageToTPR(days int) string {
	if days <= 0 || days >= 9999 {
		return ""
	}
	return "r" + strconv.Itoa(days*86400)
}

// workTypeFlag returns the LinkedIn workplace type filter flag.
func workTypeFlag(mode string) string {
	switch strings.ToLower(mode) {
	case "remote":
		return "2"
	case "hybrid":
		return "3"
	case "onsite", "on-site":
		return "1"
	default:
		return ""
	}
}
