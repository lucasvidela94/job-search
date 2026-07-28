// Package cli dispatches jobsearch subcommands.
package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/lucasvidela94/jobsearch/internal/config"
	"github.com/lucasvidela94/jobsearch/internal/store"
)

// Deps holds the dependencies each command needs.
type Deps struct {
	Config *config.Config
	Store  *store.Store
	Stdout io.Writer
	Stderr io.Writer
}

// Run dispatches the CLI command.
// The first positional arg determines the command.
func Run(args []string, deps *Deps) error {
	if len(args) == 0 {
		return errors.New("usage: jobsearch <command> [args...]\n\n" + helpText())
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "help", "-h", "--help":
		fmt.Fprint(deps.Stdout, helpText())
		return nil
	case "version", "--version":
		fmt.Fprintln(deps.Stdout, "dev") // goreleaser sets main.version
		return nil
	case "search":
		return cmdSearch(cmdArgs, deps)
	case "detail":
		return cmdDetail(cmdArgs, deps)
	case "scrape":
		return cmdScrape(cmdArgs, deps)
	case "rank":
		return cmdRank(cmdArgs, deps)
	case "setup":
		return cmdSetup(cmdArgs, deps)
	default:
		return fmt.Errorf("unknown command: %s\n\nAvailable commands:\n  search  detail  scrape  rank  setup  update  version  help\n\nUse 'jobsearch help' for full usage.", cmd)
	}
}

func helpText() string {
	return `jobsearch — multi-source job search toolkit

Usage:
  jobsearch search [flags]         Search jobs across portals
  jobsearch detail <id> [flags]    Get full job description
  jobsearch scrape [flags]         Multi-portal job scrape with dedup
  jobsearch rank [flags]           Score scraped jobs against profile
  jobsearch setup [flags]          First-time configuration
  jobsearch update                 Self-update the binary
  jobsearch version                Print version
  jobsearch help                   Show this help

Flags:
  --source, -s    Portal(s): linkedin, freehire, all (default: all)
  --query, -q     Keywords
  --location, -l  Location
  --days          Recency filter (1, 7, 14, 30)
  --format        Output format: json (default), table, plain
  --top <N>       Shortlist size for rank (default 5)

Use 'jobsearch <command> --help' for command-specific help.
`
}

func cmdSearch(args []string, deps *Deps) error {
	return fmt.Errorf("search: not yet implemented — coming in Phase B")
}

func cmdDetail(args []string, deps *Deps) error {
	return fmt.Errorf("detail: not yet implemented — coming in Phase B")
}

func cmdScrape(args []string, deps *Deps) error {
	return fmt.Errorf("scrape: not yet implemented — coming in Phase C")
}

func cmdRank(args []string, deps *Deps) error {
	return fmt.Errorf("rank: not yet implemented — coming in Phase C")
}

func cmdSetup(args []string, deps *Deps) error {
	return fmt.Errorf("setup: not yet implemented — coming in Phase D")
}
