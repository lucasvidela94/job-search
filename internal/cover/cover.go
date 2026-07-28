// Package cover generates cover letters from templates and profile data.
package cover

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/lucasvidela94/jobsearch/internal/config"
	"github.com/lucasvidela94/jobsearch/internal/portal"
	"github.com/lucasvidela94/jobsearch/internal/profile"
)

//go:embed cover.txt
var defaultTemplate string

// Data is the template data model for cover letters.
type Data struct {
	Name                 string
	Title                string
	Email                string
	Phone                string
	Location             string
	Date                 string
	HiringManager        string
	HiringManagerSalutation string
	Company              string
	CompanyAddress       string
	JobTitle             string
	JobCompany           string
	SkillList            string
	Body                 string
}

// Generator creates cover letters.
type Generator struct {
	p       *profile.Profile
	job     portal.JobPosting
	tmplStr string // resolved template content
}

// New creates a Generator for the given profile and job.
func New(p *profile.Profile, job portal.JobPosting) (*Generator, error) {
	g := &Generator{p: p, job: job}

	tmpl, err := resolveTemplate()
	if err != nil {
		return nil, fmt.Errorf("resolve template: %w", err)
	}
	g.tmplStr = tmpl

	return g, nil
}

// Render generates the cover letter text.
func (g *Generator) Render() (string, error) {
	tmpl, err := template.New("cover").Parse(g.tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	skillList := ""
	if len(g.p.Skills) > 0 {
		skillList = strings.Join(g.p.Skills, ", ")
	}

	hiringManager := "Hiring Manager"
	salutation := "Hiring Manager"

	body := fmt.Sprintf(
		"I have been working with %s for the past several years, "+
			"building and maintaining production systems using modern tools and practices. "+
			"My experience includes designing APIs, managing deployments, and collaborating "+
			"across cross-functional teams to deliver results.",
		skillList,
	)

	data := Data{
		Name:                    g.p.Name,
		Title:                   g.p.Title,
		Email:                   "",
		Phone:                   "",
		Location:                "",
		Date:                    time.Now().Format("January 2, 2006"),
		HiringManager:           hiringManager,
		HiringManagerSalutation: salutation,
		Company:                 g.job.Company,
		CompanyAddress:          "",
		JobTitle:                g.job.Title,
		JobCompany:              g.job.Company,
		SkillList:               skillList,
		Body:                    body,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

// RenderToFile generates the cover letter and writes it to a file.
func (g *Generator) RenderToFile(dir string) (string, error) {
	content, err := g.Render()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	path := filepath.Join(dir, fmt.Sprintf("cover_%s.txt", g.job.ID))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write cover letter: %w", err)
	}
	return path, nil
}

// resolveTemplate finds the template: user override or built-in default.
func resolveTemplate() (string, error) {
	userPath := filepath.Join(config.ConfigDir(), "templates", "cover.txt")
	if data, err := os.ReadFile(userPath); err == nil {
		return string(data), nil
	}

	// Fall back to embedded default
	if defaultTemplate == "" {
		return "", fmt.Errorf("embedded template is empty")
	}
	return defaultTemplate, nil
}
