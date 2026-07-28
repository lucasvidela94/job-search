package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/lucasvidela94/jobsearch/internal/applications"
	"github.com/lucasvidela94/jobsearch/internal/cli"
	"github.com/lucasvidela94/jobsearch/internal/config"
	"github.com/lucasvidela94/jobsearch/internal/db"
	"github.com/lucasvidela94/jobsearch/internal/output"
	_ "github.com/lucasvidela94/jobsearch/internal/portal/freehire" // registers freehire portal
	_ "github.com/lucasvidela94/jobsearch/internal/portal/linkedin" // registers linkedin portal
	"github.com/lucasvidela94/jobsearch/internal/store"
	"github.com/lucasvidela94/jobsearch/internal/update"
)

// version is set by goreleaser at build time.
var version = "dev"

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return version
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "update" {
		if err := runUpdate(); err != nil {
			log.Fatalf("update failed: %s", err)
		}
		return
	}

	climode, args := detectCLI(os.Args[1:])
	if climode {
		cfg, err := config.Load()
		if err != nil {
			output.WriteError(os.Stderr, err, "CONFIG_ERROR")
			os.Exit(1)
		}
		st := store.New(cfg.StoreDir())

		database, err := db.Open(cfg.StoreDir())
		if err != nil {
			output.WriteError(os.Stderr, err, "DB_ERROR")
			os.Exit(1)
		}
		defer database.Close()

		deps := &cli.Deps{
			Config:  cfg,
			Store:   st,
			DB:      database,
			Apps:    applications.NewRepository(database),
			Version: resolveVersion(),
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
			Ctx:     context.Background(),
		}

		// Auto-migrate legacy JSON tracker
		oldEntries, err := st.LoadTracker()
		if err != nil {
			fmt.Fprintf(deps.Stderr, "WARNING: could not read legacy tracker for migration: %s\n", err)
		} else if len(oldEntries) > 0 {
			type legacyEntry = struct {
				Company string `json:"company"`
				Role    string `json:"role"`
				URL     string `json:"url,omitempty"`
				Date    string `json:"date,omitempty"`
				Status  string `json:"status,omitempty"`
			}
			entries := make([]applications.TrackerEntry, len(oldEntries))
			for i, e := range oldEntries {
				entries[i] = applications.TrackerEntry(e)
			}
			n, err := deps.Apps.MigrateFromTracker(context.Background(), entries)
			if err != nil {
				fmt.Fprintf(deps.Stderr, "WARNING: tracker migration incomplete: %s\n", err)
			}
			if n > 0 {
				fmt.Fprintf(deps.Stderr, "Migrated %d legacy tracker entries to SQLite.\n", n)
			}
		}
		if err := cli.Run(args, deps); err != nil {
			output.WriteError(os.Stderr, err, "CLI_ERROR")
			os.Exit(1)
		}
		return
	}

	transport := flag.String("transport", "stdio", "transport type: stdio or http")
	port := flag.String("port", "8080", "HTTP listen port (used with --transport http)")
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(resolveVersion())
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		output.WriteError(os.Stderr, err, "CONFIG_ERROR")
		os.Exit(1)
	}

	_ = cfg
	_ = transport
	_ = port
	// MCP mode: future --transport stdio|http (v2.0)
	fmt.Fprintf(os.Stderr, "MCP server mode not yet implemented. Use CLI mode instead.\n")
	os.Exit(1)
}

func runUpdate() error {
	u := update.NewSelfUpdater("lucasvidela94/jobsearch")
	result, err := u.Run(resolveVersion())
	if err != nil {
		return err
	}
	fmt.Println(result.Message)
	if result.Updated {
		fmt.Println("Restart your terminal or AI agent to use the new version.")
	}
	return nil
}

// detectCLI checks whether the invocation looks like a CLI subcommand.
// A bare positional argument (not starting with "-") activates CLI mode.
// The --cli flag also forces CLI mode and is stripped from the arg slice.
func detectCLI(rawArgs []string) (bool, []string) {
	var filtered []string
	cliMode := false
	for _, a := range rawArgs {
		if a == "--cli" {
			cliMode = true
			continue
		}
		filtered = append(filtered, a)
	}
	if !cliMode && len(filtered) > 0 && !strings.HasPrefix(filtered[0], "-") {
		cliMode = true
	}
	return cliMode, filtered
}
