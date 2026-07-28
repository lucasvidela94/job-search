package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"

	"github.com/lucasvidela94/jobsearch/internal/config"
	"github.com/lucasvidela94/jobsearch/internal/portal"
)

// Generator creates agent configuration files.
type Generator struct {
	BinaryPath string
}

// New creates a Generator.
func New(binaryPath string) *Generator {
	return &Generator{BinaryPath: binaryPath}
}

// Result summarizes generated files.
type Result struct {
	Files []GeneratedFile
}

// GeneratedFile represents a file created by setup.
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

// mcpServerConfig is the jobsearch entry in an MCP client's config.
type mcpServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// opencodeConfig mirrors the MCP server section of opencode.json.
type opencodeConfig struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers,omitempty"`
}

func resolveVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return "dev"
}

// Run generates all config files.
func (g *Generator) Run() (*Result, error) {
	names := portal.Names()
	sort.Strings(names)

	cfg := agentConfig{
		BinaryPath: g.BinaryPath,
		Version:    resolveVersion(),
		Portals:    names,
		Commands:   buildCommands(names),
	}

	var result Result

	// 1. Agent config reference
	dir := config.ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(cfg); err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}

	configPath := filepath.Join(dir, "agent-config.json")
	if err := os.WriteFile(configPath, buf.Bytes(), 0600); err != nil {
		return nil, fmt.Errorf("write agent config: %w", err)
	}
	result.Files = append(result.Files, GeneratedFile{
		Path: configPath, Label: "Agent command reference",
	})

	// 2. Register MCP server in OpenCode config
	mcpPath, err := registerMCPServer(g.BinaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not register MCP server: %s\n", err)
	} else if mcpPath != "" {
		result.Files = append(result.Files, GeneratedFile{
			Path: mcpPath, Label: "OpenCode MCP server entry",
		})
	}

	return &result, nil
}

// registerMCPServer adds jobsearch as an MCP tool in opencode.json.
// Returns the path that was modified, or empty if nothing was done.
func registerMCPServer(binaryPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}

	opencodeDir := filepath.Join(home, ".config", "opencode")
	opencodePath := filepath.Join(opencodeDir, "opencode.json")

	mcpEntry := mcpServerConfig{
		Command: binaryPath,
		Args:    []string{"--transport", "stdio"},
	}

	// Try to read existing config
	var cfg opencodeConfig
	if data, err := os.ReadFile(opencodePath); err == nil {
		json.Unmarshal(data, &cfg)
	}

	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]mcpServerConfig)
	}
	cfg.MCPServers["jobsearch"] = mcpEntry

	if err := os.MkdirAll(opencodeDir, 0755); err != nil {
		return "", fmt.Errorf("create opencode dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode: %w", err)
	}

	if err := os.WriteFile(opencodePath+".tmp", data, 0600); err != nil {
		return "", fmt.Errorf("write temp: %w", err)
	}
	if err := os.Rename(opencodePath+".tmp", opencodePath); err != nil {
		os.Remove(opencodePath + ".tmp")
		return "", fmt.Errorf("rename: %w", err)
	}

	return opencodePath, nil
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
			Example:     `jobsearch rank --top 5`,
		},
		{
			Name:        "apply",
			Description: "Mark a job as applied and open its apply URL",
			Example:     `jobsearch apply <id> --source linkedin --cover`,
		},
		{
			Name:        "status",
			Description: "Show application timeline",
			Example:     `jobsearch status <application-id>`,
		},
		{
			Name:        "pipeline",
			Description: "List all job applications",
			Example:     `jobsearch pipeline`,
		},
		{
			Name:        "log",
			Description: "Add an event to an application",
			Example:     `jobsearch log <id> --event interview --notes "Technical round passed"`,
		},
		{
			Name:        "profile",
			Description: "View or update your candidate profile",
			Example:     `jobsearch profile edit --name "Your Name" --title "Senior Go Developer" --skill Go`,
		},
		{
			Name:        "setup",
			Description: "Generate agent and MCP configuration",
			Example:     `jobsearch setup`,
		},
	}
}
