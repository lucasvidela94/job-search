package setup

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"

	"github.com/lucasvidela94/jobsearch/internal/config"
	"github.com/lucasvidela94/jobsearch/internal/portal"
)

type Generator struct {
	BinaryPath string
}

func New(binaryPath string) *Generator {
	return &Generator{BinaryPath: binaryPath}
}

type Result struct {
	Files []GeneratedFile
}

type GeneratedFile struct {
	Path  string
	Label string
}

type agentConfig struct {
	BinaryPath string    `json:"binary_path"`
	Version    string    `json:"version"`
	Portals    []string  `json:"portals"`
	Commands   []command `json:"commands"`
}

type command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

func resolveVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return "dev"
}

func (g *Generator) Run() (*Result, error) {
	names := portal.Names()
	sort.Strings(names)

	cfg := agentConfig{
		BinaryPath: g.BinaryPath,
		Version:    resolveVersion(),
		Portals:    names,
		Commands:   buildCommands(names),
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(cfg); err != nil {
		return nil, err
	}

	dir := config.ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	configPath := filepath.Join(dir, "agent-config.json")
	if err := os.WriteFile(configPath, buf.Bytes(), 0600); err != nil {
		return nil, err
	}

	return &Result{
		Files: []GeneratedFile{
			{Path: configPath, Label: "Agent configuration"},
		},
	}, nil
}

func buildCommands(portals []string) []command {
	return []command{
		{
			Name:        "search",
			Description: "Search jobs across one or more portals",
			Example:     `jobsearch search --query golang --source linkedin --location remote --limit 10`,
		},
		{
			Name:        "detail",
			Description: "Get full job description for a specific posting",
			Example:     `jobsearch detail <job-id> --source linkedin`,
		},
		{
			Name:        "scrape",
			Description: "Multi-portal job scrape with dedup against previously seen jobs",
			Example:     `jobsearch scrape --query "software engineer" --days 7 --limit 25`,
		},
		{
			Name:        "rank",
			Description: "Score scraped jobs against your profile",
			Example:     `jobsearch rank --top 5 --profile ~/.config/jobsearch/profile.json`,
		},
		{
			Name:        "setup",
			Description: "Generate agent configuration for AI coding tools",
			Example:     `jobsearch setup`,
		},
	}
}
