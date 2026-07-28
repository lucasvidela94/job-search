// Package rank scores job postings against a user profile.
package rank

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/lucasvidela94/jobsearch/internal/portal"
	"github.com/lucasvidela94/jobsearch/internal/profile"
)

// Config controls ranking behavior.
type Config struct {
	TopN int
}

// ScoredJob pairs a job posting with its relevance score and reasoning.
type ScoredJob struct {
	Job     portal.JobPosting
	Score   float64
	Reasons []string
}

// ScoreJobs ranks job postings by relevance to the given profile.
func ScoreJobs(jobs []portal.JobPosting, p *profile.Profile, cfg Config) []ScoredJob {
	if cfg.TopN <= 0 {
		cfg.TopN = 5
	}
	if p == nil || len(jobs) == 0 {
		return nil
	}

	scored := make([]ScoredJob, 0, len(jobs))
	for _, j := range jobs {
		var score float64
		var reasons []string

		score += skillMatch(j, p.Skills, &reasons)
		score += seniorityMatch(j, p.Seniority, &reasons)
		score += locationMatch(j, p.Locations, p.Remote, &reasons)
		score += titleKeywordMatch(j, p.Title, &reasons)

		score = math.Round(score*100) / 100
		scored = append(scored, ScoredJob{Job: j, Score: score, Reasons: reasons})
	}

	slices.SortStableFunc(scored, func(a, b ScoredJob) int {
		if a.Score > b.Score {
			return -1
		}
		if a.Score < b.Score {
			return 1
		}
		return 0
	})

	if len(scored) > cfg.TopN {
		scored = scored[:cfg.TopN]
	}
	return scored
}

func skillMatch(j portal.JobPosting, skills []string, reasons *[]string) float64 {
	if len(skills) == 0 {
		return 0
	}

	haystack := strings.ToLower(j.Title + " " + j.Description)
	pointsPer := 40.0 / float64(len(skills))
	var matched float64

	for _, s := range skills {
		if strings.Contains(haystack, strings.ToLower(s)) {
			matched += pointsPer
		}
	}

	if matched > 0 {
		*reasons = append(*reasons, "skills:"+fmtScore(matched)+"/40")
	}
	return matched
}

func seniorityMatch(j portal.JobPosting, seniority string, reasons *[]string) float64 {
	if seniority == "" {
		return 0
	}

	jobSeniority := strings.ToLower(strings.TrimSpace(j.Seniority))
	profileSeniority := strings.ToLower(strings.TrimSpace(seniority))

	if jobSeniority == "" || jobSeniority == profileSeniority {
		*reasons = append(*reasons, "seniority:20/20")
		return 20
	}
	return 0
}

func locationMatch(j portal.JobPosting, locations []string, remote string, reasons *[]string) float64 {
	var score float64

	jobLoc := strings.ToLower(j.Location)
	profileRemote := strings.ToLower(strings.TrimSpace(remote))

	switch {
	case profileRemote == "remote" && isRemote(jobLoc):
		score += 20
	case len(locations) > 0:
		for _, loc := range locations {
			if strings.Contains(jobLoc, strings.ToLower(loc)) {
				score += 15
				break
			}
		}
	case profileRemote == "any":
		score += 10
	}

	if score > 0 {
		*reasons = append(*reasons, "location:"+fmtScore(score)+"/20")
	}
	return score
}

func isRemote(location string) bool {
	return strings.Contains(location, "remote") || location == ""
}

func titleKeywordMatch(j portal.JobPosting, title string, reasons *[]string) float64 {
	if title == "" {
		return 0
	}

	words := strings.Fields(title)
	if len(words) == 0 {
		return 0
	}

	pointsPer := 20.0 / float64(len(words))
	jobTitle := strings.ToLower(j.Title)
	var matched float64

	for _, w := range words {
		if strings.Contains(jobTitle, strings.ToLower(w)) {
			matched += pointsPer
		}
	}

	if matched > 0 {
		*reasons = append(*reasons, "title:"+fmtScore(matched)+"/20")
	}
	return matched
}

func fmtScore(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimSuffix(s, ".")
	return s
}
