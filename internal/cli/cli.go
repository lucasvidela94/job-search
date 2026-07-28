// Package cli dispatches jobsearch subcommands.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucasvidela94/jobsearch/internal/config"
	"github.com/lucasvidela94/jobsearch/internal/output"
	"github.com/lucasvidela94/jobsearch/internal/portal"
	"github.com/lucasvidela94/jobsearch/internal/profile"
	"github.com/lucasvidela94/jobsearch/internal/rank"
	"github.com/lucasvidela94/jobsearch/internal/scrape"
	"github.com/lucasvidela94/jobsearch/internal/setup"
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
	fs := flag.NewFlagSet("scrape", flag.ContinueOnError)
	sourceRaw := fs.String("source", "all", "Portal(s): linkedin, freehire, all")
	sourceShort := fs.String("s", "all", "Portal(s) (shorthand)")
	query := fs.String("query", "", "Search keywords")
	queryShort := fs.String("q", "", "Search keywords (shorthand)")
	location := fs.String("location", "", "Location filter")
	locationShort := fs.String("l", "", "Location filter (shorthand)")
	days := fs.Int("days", 7, "Recency in days (default: 7)")
	remote := fs.String("remote", "", "Work type: remote, hybrid, onsite")
	limit := fs.Int("limit", 25, "Max results per portal (default: 25)")
	format := fs.String("format", "json", "Output format: json, table, plain")

	fs.Usage = func() {
		portals := strings.Join(portal.Names(), ", ")
		fmt.Fprint(flag.CommandLine.Output(), `Usage: jobsearch scrape [flags]

Multi-portal job scrape with dedup. Searches all portals, deduplicates against
previously seen jobs, and saves new ones to the store.

Flags:
  --source, -s    Portal(s): `+portals+`, all
  --query, -q     Search keywords
  --location, -l  Location filter
  --days          Recency in days (default: 7)
  --remote        Work type: remote, hybrid, onsite
  --limit         Max results per portal (default: 25)
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

	q := *query
	if q == "" {
		q = *queryShort
	}
	loc := *location
	if loc == "" {
		loc = *locationShort
	}
	d := *days
	if d < 0 {
		d = 7
	}
	lim := *limit
	if lim <= 0 || lim > 100 {
		lim = 25
	}

	srcNames, err := parseSource(*sourceRaw)
	if err != nil {
		return err
	}
	if *sourceRaw == "all" && *sourceShort != "all" {
		srcNames, err = parseSource(*sourceShort)
		if err != nil {
			return err
		}
	}

	cfg := scrape.Config{
		Portals:  srcNames,
		Query:    q,
		Location: loc,
		Days:     d,
		Remote:   *remote,
		Limit:    lim,
	}

	result, err := scrape.Run(deps.Ctx, cfg, deps.Store)
	if err != nil {
		return fmt.Errorf("scrape failed: %w", err)
	}

	// Report errors on stderr
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(deps.Stderr, "WARNING: %s: %s\n", e.Portal, e.Error)
		}
	}

	// Build a concise output
	type scrapeOutput struct {
		NewJobs   int              `json:"new_jobs"`
		Existing  int              `json:"existing"`
		Errors    int              `json:"errors"`
		Results   []portal.JobPosting `json:"results,omitempty"`
	}

	out := scrapeOutput{
		NewJobs:  len(result.NewJobs),
		Existing: result.Existing,
		Errors:   len(result.Errors),
		Results:  result.NewJobs,
	}
	return output.WriteResult(deps.Stdout, out, parseFormat(*format))
}

func cmdRank(args []string, deps *Deps) error {
	fs := flag.NewFlagSet("rank", flag.ContinueOnError)
	topN := fs.Int("top", 5, "Number of top results to show")
	format := fs.String("format", "table", "Output format: json, table, plain")
	profilePath := fs.String("profile", "", "Path to profile JSON (default: ~/.config/jobsearch/profile.json)")

	fs.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), `Usage: jobsearch rank [flags]

Score scraped jobs against your profile. Reads the profile from
~/.config/jobsearch/profile.json and scores all scraped jobs in the store.

Flags:
  --top <N>       Number of top results (default: 5)
  --profile       Path to profile JSON (default: ~/.config/jobsearch/profile.json)
  --format        Output format: json, table, plain (default: table)
  --help          Show this help
`)
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Load profile
	pp := *profilePath
	if pp == "" {
		pp = config.ConfigDir() + "/profile.json"
	}
	p, err := loadProfile(pp)
	if err != nil {
		return fmt.Errorf("load profile: %w\n\nCreate a profile at %s or specify --profile.\nExample profile:\n"+
			`{"title":"Senior Go Developer","skills":["go","kubernetes","postgresql"],"seniority":"senior","remote":"remote"}`, err, pp)
	}

	// Load jobs from store
	seen, err := deps.Store.LoadSeenJobs()
	if err != nil {
		return fmt.Errorf("load seen jobs: %w", err)
	}

	if len(seen) == 0 {
		return fmt.Errorf("no scraped jobs found. Run 'jobsearch scrape' first")
	}

	// Convert seen entries to job postings
	jobs := make([]portal.JobPosting, 0, len(seen))
	for _, entry := range seen {
		jobs = append(jobs, portal.JobPosting{
			ID:      entry.URL, // use URL as ID
			Title:   entry.Title,
			Company: entry.Company,
			URL:     entry.URL,
			Source:  entry.Source,
		})
	}

	n := *topN
	if n <= 0 {
		n = 5
	}

	scored := rank.ScoreJobs(jobs, p, rank.Config{TopN: n})
	if len(scored) == 0 {
		return fmt.Errorf("no matching jobs found")
	}

	return output.WriteResult(deps.Stdout, scored, parseFormat(*format))
}

// loadProfile reads a profile from a JSON file.
func loadProfile(path string) (*profile.Profile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var p profile.Profile
	if err := json.NewDecoder(f).Decode(&p); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	return &p, nil
}

func cmdSetup(args []string, deps *Deps) error {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(deps.Stdout, setupHelpText())
		return nil
	}

	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate binary: %w", err)
	}

	gen := setup.New(binaryPath)
	result, err := gen.Run()
	if err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	for _, f := range result.Files {
		fmt.Fprintf(deps.Stdout, "Created %s: %s\n", f.Label, f.Path)
	}

	fmt.Fprintln(deps.Stdout)
	fmt.Fprintln(deps.Stdout, "Setup complete!")
	fmt.Fprintln(deps.Stdout, "Add this to your AI agent's context to enable job search capabilities:")
	fmt.Fprintf(deps.Stdout, "  Reference: %s\n", filepath.Join(config.ConfigDir(), "agent-config.json"))

	return nil
}

func setupHelpText() string {
	return `Usage: jobsearch setup

Generate agent configuration for AI coding tools.

Creates agent-config.json in the jobsearch config directory
(~/.config/jobsearch/) with the binary path, available portals,
and command references for AI agents.

No flags required. Run it after installation or update.
`
}
