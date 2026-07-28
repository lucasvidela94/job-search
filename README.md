# jobsearch

Multi-source job search toolkit. CLI for searching, scraping, ranking, and applying to jobs — works with any AI agent.

```bash
curl -fsSL https://get.jobsearch.dev/install.sh | bash
```

## Status

**Active development.** Porting from TypeScript/Bun to Go.

| Phase | Status | Description |
|-------|--------|-------------|
| A — Skeleton + Core | ✅ Done | CLI skeleton, config, output formatters, state store, self-update |
| B — LinkedIn Portal | 🔜 Next | Search + detail via LinkedIn jobs-guest API |
| C — Multi-Portal + Scrape | ⏳ | Freehire, scrape orchestrator, rank scorer |
| D — Distribution | ⏳ | Install script, agent config injection, Homebrew |

## Usage

```bash
jobsearch help
jobsearch version

# Search jobs (coming in Phase B)
jobsearch linkedin search -q "golang" -l "Berlin" --format table

# Multi-portal scrape (coming in Phase C)
jobsearch scrape

# Rank results (coming in Phase C)
jobsearch rank --top 5

# Config setup (coming in Phase D)
jobsearch setup --agent opencode --markets latam,us

# Self-update
jobsearch update
```

## Architecture

```
cmd/jobsearch/main.go    — Entry point, detectCLI, dispatch
internal/
  cli/                   — CLI command implementations
  output/                — JSON/table/plain formatters
  config/                — Env-var + JSON config loader
  store/                 — Atomic JSON file I/O (seen_jobs, tracker)
  portal/                — Portal interface + types
  update/                — Self-updater from GitHub releases
  agent/                 — AI agent config injection
  scrape/                — Multi-portal scrape orchestrator
  rank/                  — Match scoring engine
```

## Install

```bash
# One-liner (coming in Phase D)
curl -fsSL https://get.jobsearch.dev/install.sh | bash

# Or build from source
go install github.com/lucasvidela94/jobsearch@latest
```

## Configuration

Set environment variables or run `jobsearch setup`:

| Variable | Description |
|----------|-------------|
| `JOBSEARCH_AGENT` | Target AI agent (opencode, claude, cursor, windsurf) |
| `JOBSEARCH_MARKETS` | Comma-separated markets (latam, us, europe) |
| `LINKEDIN_COOKIE` | LinkedIn session cookie for authenticated search |

Config file: `~/.config/jobsearch/config.json`

## Development

```bash
make test        # Run all tests
make test-race   # Race detector
make fmt         # Format code
make vet         # Vet code
make build       # Build binary
make run ARGS="help"
```

## License

MIT
# job-search
