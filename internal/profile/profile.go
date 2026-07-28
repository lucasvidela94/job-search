// Package profile defines the user's job-search profile for ranking.
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucasvidela94/jobsearch/internal/config"
)

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

// Load reads the profile from the config directory.
func Load() (*Profile, error) {
	path := filepath.Join(config.ConfigDir(), "profile.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}

	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	return &p, nil
}

// Save writes the profile to the config directory atomically.
func Save(p *Profile) error {
	dir := config.ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	path := filepath.Join(dir, "profile.json")
	tmpPath := path + ".tmp"

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profile: %w", err)
	}

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// SetSkill adds a skill to the profile, deduplicating.
func (p *Profile) SetSkill(skill string) {
	skill = strings.TrimSpace(skill)
	if skill == "" {
		return
	}
	for _, s := range p.Skills {
		if strings.EqualFold(s, skill) {
			return // already present
		}
	}
	p.Skills = append(p.Skills, skill)
}

// SetField sets a scalar field by JSON key name.
func (p *Profile) SetField(key, value string) error {
	value = strings.TrimSpace(value)
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "name":
		p.Name = value
	case "title":
		p.Title = value
	case "seniority":
		p.Seniority = value
	case "remote":
		p.Remote = value
	case "currency":
		p.Currency = value
	case "min_salary":
		if value == "" {
			p.MinSalary = 0
		} else {
			var n int
			if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
				return fmt.Errorf("invalid number for min_salary: %s", value)
			}
			p.MinSalary = n
		}
	default:
		return fmt.Errorf("unknown field: %s (valid: name, title, seniority, remote, min_salary, currency, skill)", key)
	}
	return nil
}
