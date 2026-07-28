// Package config loads and stores jobsearch configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PortalConfig holds per-portal settings.
type PortalConfig struct {
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"api_key,omitempty"`
	Cookie  string `json:"cookie,omitempty"`
}

// Config holds all jobsearch configuration.
type Config struct {
	Agent       string                  `json:"agent,omitempty"`
	Markets     []string                `json:"markets,omitempty"`
	Portals     map[string]PortalConfig `json:"portals,omitempty"`
	ProfilePath string                  `json:"profile_path,omitempty"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Agent: "opencode",
		Markets: []string{"latam", "us"},
		Portals: map[string]PortalConfig{
			"linkedin": {Enabled: true},
			"freehire": {Enabled: true},
		},
	}
}

// ConfigDir returns the jobsearch configuration directory.
// Uses JOBSEARCH_CONFIG_DIR env var, defaults to ~/.config/jobsearch/.
func ConfigDir() string {
	if d := os.Getenv("JOBSEARCH_CONFIG_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "jobsearch")
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "jobsearch")
}

// StoreDir returns the state storage directory (config dir + /state).
func (c *Config) StoreDir() string {
	return filepath.Join(ConfigDir(), "state")
}

// ConfigPath returns the path to the JSON config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

// Load reads configuration from environment variables and JSON file.
// Env vars take precedence over file values.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from config file
	cfgPath := ConfigPath()
	if data, err := os.ReadFile(cfgPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", cfgPath, err)
		}
	}

	// Environment overrides
	if v := os.Getenv("JOBSEARCH_AGENT"); v != "" {
		cfg.Agent = v
	}
	if v := os.Getenv("JOBSEARCH_MARKETS"); v != "" {
		cfg.Markets = splitAndTrim(v, ",")
	}
	if v := os.Getenv("LINKEDIN_COOKIE"); v != "" {
		if cfg.Portals == nil {
			cfg.Portals = make(map[string]PortalConfig)
		}
		p := cfg.Portals["linkedin"]
		p.Cookie = v
		cfg.Portals["linkedin"] = p
	}

	return cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	path := ConfigPath()
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
