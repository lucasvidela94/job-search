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
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lucasvidela94/jobsearch/internal/applications"
	"github.com/lucasvidela94/jobsearch/internal/config"
	"github.com/lucasvidela94/jobsearch/internal/cover"
	"github.com/lucasvidela94/jobsearch/internal/db"
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
	Config  *config.Config
	Store   *store.Store
	DB      *db.DB
	Apps    *applications.Repository
	Version string
	Stdout  io.Writer
	Stderr  io.Writer
	Ctx     context.Context
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
		fmt.Fprintln(deps.Stdout, deps.Version)
		return nil
	case "search":
		return cmdSearch(cmdArgs, deps)
	case "detail":
		return cmdDetail(cmdArgs, deps)
	case "scrape":
		return cmdScrape(cmdArgs, deps)
	case "rank":
		return cmdRank(cmdArgs, deps)
	case "apply":
		return cmdApply(cmdArgs, deps)
	case "status":
		return cmdStatus(cmdArgs, deps)
	case "pipeline":
		return cmdPipeline(cmdArgs, deps)
	case "log":
		return cmdLog(cmdArgs, deps)
	case "profile":
		return cmdProfile(cmdArgs, deps)
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
  jobsearch apply <id> [flags]     Mark a job as applied + open browser
  jobsearch status <id>            Show application timeline
  jobsearch pipeline [flags]       List all applications
  jobsearch log <id> [flags]       Add an event to an application
  jobsearch profile show           Display profile
  jobsearch profile edit [flags]   Update profile fields
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
	appliedOnly := fs.Bool("applied-only", false, "Show only applied-to jobs")
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
  --applied-only  Show only applied-to jobs
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

	// Enrich results with applied status
	type enrichedJob struct {
		portal.JobPosting
		Applied bool `json:"applied"`
	}
	enriched := make([]enrichedJob, 0, len(allResults))
	for _, j := range allResults {
		applied := false
		if j.URL != "" {
			existing, err := deps.Apps.FindByURL(deps.Ctx, j.URL)
			if err == nil && existing != nil {
				applied = true
			}
		}
		if *appliedOnly && !applied {
			continue
		}
		enriched = append(enriched, enrichedJob{JobPosting: j, Applied: applied})
	}

	if *appliedOnly && len(enriched) == 0 {
		fmt.Fprintln(deps.Stderr, "No applied-to jobs match your search criteria.")
		return nil
	}

	return output.WriteResult(deps.Stdout, enriched, parseFormat(*format))
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

	// Enrich with applied status
	type detailResult struct {
		portal.JobPosting
		Applied    bool   `json:"applied"`
		AppID      string `json:"app_id,omitempty"`
		AppStatus  string `json:"app_status,omitempty"`
	}
	result := detailResult{JobPosting: *job}
	if job.URL != "" {
		existing, err := deps.Apps.FindByURL(deps.Ctx, job.URL)
		if err == nil && existing != nil {
			result.Applied = true
			result.AppID = existing.ID
			result.AppStatus = existing.Status
		}
	}

	return output.WriteResult(deps.Stdout, result, parseFormat(*format))
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

// --- candidate-flow commands ---

func applyHelpText() string {
	return `Usage: jobsearch apply <id> [flags]

Open a job's apply URL in the browser and register it as applied.

Arguments:
  id    Job posting ID from search results

Flags:
  --source, -s  Portal source (default: linkedin)
  --no-open     Don't open browser, just mark as applied
  --cover       Generate cover letter for this application
  --format      Output format: json (default), table, plain
`
}

func cmdApply(args []string, deps *Deps) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	source := fs.String("source", "", "Portal source (optional)")
	sourceShort := fs.String("s", "", "Portal source (shorthand)")
	noOpen := fs.Bool("no-open", false, "Don't open browser")
	withCover := fs.Bool("cover", false, "Generate cover letter")
	format := fs.String("format", "json", "Output format: json, table, plain")

	fs.Usage = func() { fmt.Fprint(flag.CommandLine.Output(), applyHelpText()) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() == 0 {
		return errors.New("job ID is required.\n\n" + applyHelpText())
	}
	id := fs.Arg(0)

	// Resolve source
	src := *source
	if src == "" {
		src = *sourceShort
	}

	// Try to find the job via detail (if we have the source) or assume it exists
	var job *portal.JobPosting
	if src != "" {
		srcNames, err := parseSource(src)
		if err != nil {
			return err
		}
		p, ok := portal.Get(srcNames[0])
		if !ok {
			return fmt.Errorf("unknown source: %s", src)
		}
		job, err = p.Detail(deps.Ctx, id)
		if err != nil {
			// Non-fatal — create application with available data
			job = &portal.JobPosting{ID: id, Title: id, Source: srcNames[0]}
		}
	} else {
		job = &portal.JobPosting{ID: id, Title: id}
	}

	// Check if already applied
	existing, err := deps.Apps.FindByURL(deps.Ctx, job.URL)
	if err != nil {
		return fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("already applied to this job (application: %s)", existing.ID)
	}

	if job.ApplyURL == "" {
		return fmt.Errorf("no apply URL available for job %s", id)
	}

	// Open browser (unless --no-open)
	if !*noOpen {
		if err := openBrowser(job.ApplyURL); err != nil {
			fmt.Fprintf(deps.Stderr, "WARNING: could not open browser: %s\n", err)
		}
	}

	// Create application
	app, err := deps.Apps.Create(deps.Ctx, *job)
	if err != nil {
		return fmt.Errorf("create application: %w", err)
	}

	type applyResult struct {
		Application *applications.Application `json:"application"`
		CoverLetter string                    `json:"cover_letter,omitempty"`
		BrowserOpen bool                      `json:"browser_opened"`
	}

	result := applyResult{
		Application: app,
		BrowserOpen: !*noOpen,
	}

	// Generate cover letter
	if *withCover {
		p, err := profile.Load()
		if err != nil {
			fmt.Fprintf(deps.Stderr, "WARNING: could not load profile for cover letter: %s\n", err)
		} else {
			gen, err := cover.New(p, *job)
			if err != nil {
				fmt.Fprintf(deps.Stderr, "WARNING: could not create cover letter: %s\n", err)
			} else {
				path, err := gen.RenderToFile(config.ConfigDir())
				if err != nil {
					fmt.Fprintf(deps.Stderr, "WARNING: could not write cover letter: %s\n", err)
				} else {
					result.CoverLetter = path
				}
			}
		}
	}

	return output.WriteResult(deps.Stdout, result, parseFormat(*format))
}

// openBrowser opens a URL in the default browser.
func openBrowser(url string) error {
	commands := []string{"xdg-open", "open", "x-www-browser"}
	for _, cmd := range commands {
		if _, err := os.Stat("/usr/bin/" + cmd); err == nil {
			return runCommand(cmd, url)
		}
	}
	// Try without path
	for _, cmd := range commands {
		if err := runCommand(cmd, url); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no browser command found (tried: xdg-open, open, x-www-browser)")
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}

func statusHelpText() string {
	return `Usage: jobsearch status <id>

Show the full timeline of events for an application.

Arguments:
  id    Application ID (from apply or pipeline output)

Flags:
  --format  Output format: json (default), table, plain
`
}

func cmdStatus(args []string, deps *Deps) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	format := fs.String("format", "json", "Output format: json, table, plain")
	fs.Usage = func() { fmt.Fprint(flag.CommandLine.Output(), statusHelpText()) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() == 0 {
		return errors.New("application ID is required.\n\n" + statusHelpText())
	}
	id := fs.Arg(0)

	app, err := deps.Apps.GetByID(deps.Ctx, id)
	if err != nil {
		return fmt.Errorf("get application: %w", err)
	}

	type statusResult struct {
		Application *applications.Application `json:"application"`
		Events      []applications.Event      `json:"events"`
	}

	events, err := deps.Apps.GetEvents(deps.Ctx, id)
	if err != nil {
		return fmt.Errorf("get events: %w", err)
	}

	return output.WriteResult(deps.Stdout, statusResult{Application: app, Events: events}, parseFormat(*format))
}

func pipelineHelpText() string {
	return `Usage: jobsearch pipeline [flags]

List all applications with their latest status.

Flags:
  --status      Filter by status tag (e.g. "applied", "interview", "rejected")
  --format      Output format: json (default), table, plain
`
}

func cmdPipeline(args []string, deps *Deps) error {
	fs := flag.NewFlagSet("pipeline", flag.ContinueOnError)
	statusFilter := fs.String("status", "", "Filter by status tag")
	format := fs.String("format", "json", "Output format: json, table, plain")
	fs.Usage = func() { fmt.Fprint(flag.CommandLine.Output(), pipelineHelpText()) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	apps, err := deps.Apps.List(deps.Ctx, *statusFilter)
	if err != nil {
		return fmt.Errorf("list applications: %w", err)
	}

	if len(apps) == 0 {
		fmt.Fprintln(deps.Stderr, "No applications found.")
		if *statusFilter != "" {
			fmt.Fprintf(deps.Stderr, "Try without --status filter, or run 'jobsearch apply <id>' first.\n")
		} else {
			fmt.Fprintln(deps.Stderr, "Run 'jobsearch apply <id>' to start tracking applications.")
		}
		return nil
	}

	return output.WriteResult(deps.Stdout, apps, parseFormat(*format))
}

func logHelpText() string {
	return `Usage: jobsearch log <id> --event <type> [--notes <text>]

Add an event to an application's timeline.

Arguments:
  id    Application ID

Flags:
  --event, -e   Event type (e.g. "phone_screen", "tech_interview", "offer")
  --notes, -n   Optional notes about the event
  --format      Output format: json (default), table, plain
`
}

func cmdLog(args []string, deps *Deps) error {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	eventType := fs.String("event", "", "Event type (e.g. phone_screen, interview, offer)")
	eventShort := fs.String("e", "", "Event type (shorthand)")
	notes := fs.String("notes", "", "Event notes")
	notesShort := fs.String("n", "", "Event notes (shorthand)")
	format := fs.String("format", "json", "Output format: json, table, plain")
	fs.Usage = func() { fmt.Fprint(flag.CommandLine.Output(), logHelpText()) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() == 0 {
		return errors.New("application ID is required.\n\n" + logHelpText())
	}
	id := fs.Arg(0)

	evt := *eventType
	if evt == "" {
		evt = *eventShort
	}
	if evt == "" {
		return errors.New("--event flag is required.\n\n" + logHelpText())
	}

	nt := *notes
	if nt == "" {
		nt = *notesShort
	}

	event, err := deps.Apps.AddEvent(deps.Ctx, id, evt, nt)
	if err != nil {
		return fmt.Errorf("add event: %w", err)
	}

	return output.WriteResult(deps.Stdout, event, parseFormat(*format))
}

func profileHelpText() string {
	return `Usage: jobsearch profile <show|edit>

Manage your candidate profile.

Subcommands:
  show              Display current profile
  edit [flags]      Update profile fields

Profile edit flags:
  --name <text>       Set your name
  --title <text>      Set your professional title
  --skill <name>      Add a skill (repeatable)
  --seniority <lvl>   Set seniority level
  --remote <mode>     Set work mode: remote, hybrid, onsite
  --min-salary <n>    Set minimum salary
  --currency <code>   Set salary currency
`
}

func cmdProfile(args []string, deps *Deps) error {
	if len(args) == 0 {
		return errors.New("usage: jobsearch profile <show|edit>\n\n" + profileHelpText())
	}

	subcmd := args[0]
	subArgs := args[1:]

	switch subcmd {
	case "show":
		return cmdProfileShow(subArgs, deps)
	case "edit":
		return cmdProfileEdit(subArgs, deps)
	default:
		return fmt.Errorf("unknown profile subcommand: %s (use 'show' or 'edit')", subcmd)
	}
}

func cmdProfileShow(args []string, deps *Deps) error {
	fs := flag.NewFlagSet("profile show", flag.ContinueOnError)
	format := fs.String("format", "table", "Output format: json, table, plain")
	fs.Usage = func() { fmt.Fprint(flag.CommandLine.Output(), "Usage: jobsearch profile show\n\nDisplay your candidate profile.\n") }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	p, err := profile.Load()
	if err != nil {
		return fmt.Errorf("load profile: %w\n\nCreate a profile first:\n  jobsearch profile edit --name \"Your Name\" --title \"Your Title\" --skill Go", err)
	}

	return output.WriteResult(deps.Stdout, p, parseFormat(*format))
}

func cmdProfileEdit(args []string, deps *Deps) error {
	fs := flag.NewFlagSet("profile edit", flag.ContinueOnError)
	name := fs.String("name", "", "Set name")
	title := fs.String("title", "", "Set professional title")
	skill := fs.String("skill", "", "Add a skill (repeatable)")
	seniority := fs.String("seniority", "", "Set seniority level")
	remote := fs.String("remote", "", "Set work mode: remote, hybrid, onsite")
	minSalary := fs.Int("min-salary", 0, "Set minimum salary")
	currency := fs.String("currency", "", "Set salary currency")
	fs.Usage = func() { fmt.Fprint(flag.CommandLine.Output(), "Usage: jobsearch profile edit [flags]\n\nUpdate your candidate profile.\n") }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Load existing or create new
	p, err := profile.Load()
	if err != nil {
		p = &profile.Profile{}
	}

	changed := false
	if *name != "" {
		p.Name = *name
		changed = true
	}
	if *title != "" {
		p.Title = *title
		changed = true
	}
	if *skill != "" {
		p.SetSkill(*skill)
		changed = true
	}
	if *seniority != "" {
		p.Seniority = *seniority
		changed = true
	}
	if *remote != "" {
		p.Remote = *remote
		changed = true
	}
	if *minSalary > 0 {
		p.MinSalary = *minSalary
		changed = true
	}
	if *currency != "" {
		p.Currency = *currency
		changed = true
	}

	if !changed {
		return fmt.Errorf("no changes specified.\n\n" +
			"Use flags like: --name 'Your Name' --title 'Senior Go Developer' --skill Go\n" +
			"See 'jobsearch profile edit --help' for all options.")
	}

	if err := profile.Save(p); err != nil {
		return fmt.Errorf("save profile: %w", err)
	}

	fmt.Fprintf(deps.Stdout, "Profile updated.\n")
	return output.WriteResult(deps.Stdout, p, output.FormatTable)
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
