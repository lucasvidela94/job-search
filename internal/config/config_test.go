package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDir(t *testing.T) {
	t.Run("env var override", func(t *testing.T) {
		os.Setenv("JOBSEARCH_CONFIG_DIR", "/tmp/jobsearch-test")
		defer os.Unsetenv("JOBSEARCH_CONFIG_DIR")
		if got := ConfigDir(); got != "/tmp/jobsearch-test" {
			t.Errorf("ConfigDir = %q, want /tmp/jobsearch-test", got)
		}
	})

	t.Run("default with XDG", func(t *testing.T) {
		os.Unsetenv("JOBSEARCH_CONFIG_DIR")
		os.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
		defer os.Unsetenv("XDG_CONFIG_HOME")
		got := ConfigDir()
		want := filepath.Join("/tmp/xdg", "jobsearch")
		if got != want {
			t.Errorf("ConfigDir = %q, want %q", got, want)
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Agent != "opencode" {
		t.Errorf("default agent = %q, want opencode", cfg.Agent)
	}
	if len(cfg.Markets) != 2 {
		t.Errorf("expected 2 default markets, got %d", len(cfg.Markets))
	}
	if !cfg.Portals["linkedin"].Enabled {
		t.Error("linkedin should be enabled by default")
	}
	if !cfg.Portals["freehire"].Enabled {
		t.Error("freehire should be enabled by default")
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	os.Setenv("JOBSEARCH_AGENT", "claude")
	defer os.Unsetenv("JOBSEARCH_AGENT")
	os.Setenv("JOBSEARCH_MARKETS", "latam,us,europe")
	defer os.Unsetenv("JOBSEARCH_MARKETS")
	os.Setenv("LINKEDIN_COOKIE", "test-cookie")
	defer os.Unsetenv("LINKEDIN_COOKIE")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent != "claude" {
		t.Errorf("agent = %q, want claude", cfg.Agent)
	}
	if len(cfg.Markets) != 3 {
		t.Errorf("expected 3 markets, got %d", len(cfg.Markets))
	}
	if cfg.Portals["linkedin"].Cookie != "test-cookie" {
		t.Errorf("linkedin cookie = %q, want test-cookie", cfg.Portals["linkedin"].Cookie)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("JOBSEARCH_CONFIG_DIR", dir)
	defer os.Unsetenv("JOBSEARCH_CONFIG_DIR")

	cfg := DefaultConfig()
	cfg.Agent = "cursor"
	cfg.Markets = []string{"latam"}

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Agent != "cursor" {
		t.Errorf("loaded agent = %q, want cursor", loaded.Agent)
	}
	if len(loaded.Markets) != 1 || loaded.Markets[0] != "latam" {
		t.Errorf("loaded markets = %v, want [latam]", loaded.Markets)
	}
}
