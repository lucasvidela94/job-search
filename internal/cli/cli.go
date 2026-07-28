// Package cli dispatches jobsearch subcommands.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/lucasvidela94/jobsearch/internal/config"
	"github.com/lucasvidela94/jobsearch/internal/output"
	"github.com/lucasvidela94/jobsearch/internal/portal"
	"github.com/lucasvidela94/jobsearch/internal/store"
)

// Deps holds the dependencies each command needs.
type Deps struct {
	Config *config.Config
	Store  *store.Store
	Stdout io.Writer
	Stderr io.Writer
	Ctx    context.Context
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

Use 'jobsearch <command> --help' for command-specific help.
`
}

// parseFormat validates and returns the output format, defaulting to JSON.
func parseFormat(s string) output.Format {
	switch s {
	case "table":
		return output.FormatTable
	case "plain":
		return output.FormatPlain
	default:
		return output.FormatJSON
	}
}

// parseSource resolves the --source flag value to a set of portal names.
func parseSource(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "all" {
		return []string{"all"}, nil
	}
	parts := strings.Split(raw, ",")
	var names []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := portal.Get(p); !ok {
			return nil, fmt.Errorf("unknown source: %s (available: %s)", p, strings.Join(portal.Names(), ", "))
		}
		names = append(names, p)
	}
	if len(names) == 0 {
		return []string{"all"}, nil
	}
	return names, nil
}

// resolveSources returns the map of portals to query.
func resolveSources(names []string) map[string]portal.Portal {
	if len(names) == 1 && names[0] == "all" {
		return portal.All()
	}
	m := make(map[string]portal.Portal, len(names))
	for _, name := range names {
		if p, ok := portal.Get(name); ok {
			m[name] = p
		}
	}
	return m
}

func cmdSearch(args []string, deps *Deps) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	sourceRaw := fs.String("source", "all", "Portal source(s): linkedin, freehire, all")
	sourceShort := fs.String("s", "all", "Portal source(s) (shorthand)")
	query := fs.String("query", "", "Search keywords")
	queryShort := fs.String("q", "", "Search keywords (shorthand)")
	location := fs.String("location", "", "Location filter")
	locationShort := fs.String("l", "", "Location filter (shorthand)")
	days := fs.Int("days", 0, "Recency in days (1, 7, 14, 30)")
	remote := fs.String("remote", "", "Work type: remote, hybrid, onsite")
	limit := fs.Int("limit", 10, "Max results per portal")
	format := fs.String("format", "json", "Output format: json, table, plain")

	fs.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `Usage: jobsearch search [flags]

Search jobs across registered portals.

Flags:
  --source, -s    Portal(s): `+strings.Join(portal.Names(), ", ")+`, all (default: all)
  --query, -q     Search keywords
  --location, -l  Location filter (optional)
  --days          Recency filter in days (1, 7, 14, 30; default: 0 = any)
  --remote        Work type: remote, hybrid, onsite (default: all)
  --limit         Max results per portal (default: 10)
  --format        Output format: json (default), table, plain
  --help          Show this help
`)
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Resolve query/location from short flags if long flags are empty
	q := *query
	if q == "" {
		q = *queryShort
	}
	loc := *location
	if loc == "" {
		loc = *locationShort
	}

	// Days validation
	d := *days
	if d < 0 {
		d = 0
	}

	// Limit validation
	lim := *limit
	if lim <= 0 || lim > 100 {
		lim = 10
	}

	// Resolve sources
	srcNames, err := parseSource(*sourceRaw)
	if err != nil {
		return err
	}
	// Check short flag if long is the default
	if *sourceRaw == "all" && *sourceShort != "all" {
		srcNames, err = parseSource(*sourceShort)
		if err != nil {
			return err
		}
	}
	portals := resolveSources(srcNames)
	if len(portals) == 0 {
		return fmt.Errorf("no matching portals found")
	}

	params := portal.SearchParams{
		Query:    q,
		Location: loc,
		Days:     d,
		Remote:   *remote,
		Page:     1,
		Limit:    lim,
	}

	type portalResult struct {
		name    string
		results []portal.JobPosting
		err     error
	}
	var portalRes []portalResult

	for name, p := range portals {
		res, err := p.Search(deps.Ctx, params)
		if err != nil {
			portalRes = append(portalRes, portalResult{name: name, err: err})
			continue
		}
		portalRes = append(portalRes, portalResult{name: name, results: res.Results})
	}

	// Compile errors and aggregate results
	var errs []string
	var allResults []portal.JobPosting
	for _, pr := range portalRes {
		if pr.err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", pr.name, pr.err))
			continue
		}
		allResults = append(allResults, pr.results...)
	}

	if len(allResults) == 0 && len(errs) > 0 {
		return fmt.Errorf("all portals failed: %s", strings.Join(errs, "; "))
	}
	if len(errs) > 0 {
		fmt.Fprintf(deps.Stderr, "WARNING: some portals returned errors:\n  %s\n", strings.Join(errs, "\n  "))
	}

	return output.WriteResult(deps.Stdout, allResults, parseFormat(*format))
}

func cmdDetail(args []string, deps *Deps) error {
	fs := flag.NewFlagSet("detail", flag.ContinueOnError)
	source := fs.String("source", "linkedin", "Portal source")
	sourceShort := fs.String("s", "linkedin", "Portal source (shorthand)")
	format := fs.String("format", "json", "Output format: json, table, plain")
	_ = fs.Bool("help", false, "Show this help")

	fs.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `Usage: jobsearch detail <id> [flags]

Get full job description for a specific posting.

Arguments:
  id    Job posting ID

Flags:
  --source, -s  Portal source (default: linkedin)
  --format      Output format: json (default), table, plain
  --help        Show this help
`)
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("job ID is required.\n\nUsage: jobsearch detail <job-id> [flags]")
	}
	id := fs.Arg(0)

	// Resolve source
	src := *source
	if src == "linkedin" && *sourceShort != "linkedin" {
		src = *sourceShort
	}
	// Parse source to validate
	srcNames, err := parseSource(src)
	if err != nil {
		return err
	}
	if len(srcNames) == 0 || srcNames[0] == "all" {
		return fmt.Errorf("detail requires a specific source (got %q)", src)
	}
	p, ok := portal.Get(srcNames[0])
	if !ok {
		return fmt.Errorf("unknown source: %s (available: %s)", srcNames[0], strings.Join(portal.Names(), ", "))
	}

	job, err := p.Detail(deps.Ctx, id)
	if err != nil {
		return fmt.Errorf("%s detail: %w", srcNames[0], err)
	}

	return output.WriteResult(deps.Stdout, job, parseFormat(*format))
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
