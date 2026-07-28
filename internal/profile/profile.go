// Package profile defines the user's job-search profile for ranking.
package profile

// Profile represents the user's professional profile for job matching.
type Profile struct {
	Name       string   `json:"name"`
	Title      string   `json:"title"`
	Skills     []string `json:"skills"`
	Seniority  string   `json:"seniority"`
	Categories []string `json:"categories"`
	Locations  []string `json:"locations"`
	Remote     string   `json:"remote"`
	MinSalary  int      `json:"min_salary"`
	Currency   string   `json:"currency"`
}
