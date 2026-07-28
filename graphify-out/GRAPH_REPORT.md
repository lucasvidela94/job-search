# Graph Report - .  (2026-07-28)

## Corpus Check
- 158 files · ~102,421 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 937 nodes · 1412 edges · 60 communities (45 shown, 15 thin omitted)
- Extraction: 97% EXTRACTED · 3% INFERRED · 0% AMBIGUOUS · INFERRED: 36 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Shared CLI Core
- Jobdanmark Portal
- Jobnet Portal
- Security & Gitignore Guards
- Jobindex Portal
- LinkedIn Portal
- Candidate Profile & Applications
- Salary Conversion Tool
- Freehire Portal
- Jobbank Package Config
- Jobdanmark Package Config
- Jobindex Package Config
- Rank & Match Scoring
- Workflow Commands
- PDF Verification
- Freehire TSConfig
- Jobbank TSConfig
- Jobdanmark TSConfig
- Jobnet Package Config
- Jobindex TSConfig
- LinkedIn TSConfig
- Jobnet TSConfig
- LinkedIn Package Config
- Lint Skills Tool
- Cover Letter LaTeX
- Job Scraper Skill
- Upskill Analysis
- CLI Retry & Timeout Tests
- Framework Updates
- CI Pipeline
- Project Docs & Governance
- Search Query Normalization
- CLI Flag Validation Tests
- CLI Contract Tests
- RSS Parsing Tests
- Detail Formatting Tests
- Detail JSON-LD Tests
- Detail Parsing Tests
- RSS Fetch Tests
- Search Tests
- CLI Helper Utilities
- Readme Assets Tests
- Notion Sync Tool
- Outcome Follow-up Tests
- HTML Report Tool
- HTML Entity Decoding
- Salary Lookup Tool
- Gmail Sync Tests
- Onboarding Tests
- Job Application Forms
- Pip Mascot Branding
- Setup Scripts
- Skill Registry
- Changelog
- LinkedIn URL Reference
- Freehire URL Reference
- Jobbank URL Reference

## God Nodes (most connected - your core abstractions)
1. `match_score()` - 25 edges
2. `search_company()` - 21 edges
3. `run_guards()` - 20 edges
4. `DetectColumnTypeTests` - 18 edges
5. `parse_sheet()` - 18 edges
6. `VerificationError` - 17 edges
7. `compilerOptions` - 15 edges
8. `compilerOptions` - 15 edges
9. `compilerOptions` - 15 edges
10. `compilerOptions` - 15 edges

## Surprising Connections (you probably didn't know these)
- `AI Job Search` --references--> `PR template`  [INFERRED]
  README.md → .github/PULL_REQUEST_TEMPLATE.md
- `ParsePageCountTests` --uses--> `VerificationError`  [INFERRED]
  tests/test_verify_pdf.py → tools/verify_pdf.py
- `VerifyPdfTests` --uses--> `VerificationError`  [INFERRED]
  tests/test_verify_pdf.py → tools/verify_pdf.py
- `RunToolTests` --uses--> `VerificationError`  [INFERRED]
  tests/test_verify_pdf.py → tools/verify_pdf.py
- `AI Job Search` --references--> `Thin-pointer design`  [EXTRACTED]
  README.md → AGENTS.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Danish Job Portal Skills** — agents_skills_jobbank_search_skill, agents_skills_jobdanmark_search_skill, agents_skills_jobindex_search_skill, agents_skills_jobnet_search_skill, denmark_job_market [EXTRACTED 1.00]
- **Country-Agnostic Worked Examples** — agents_skills_freehire_search_skill, agents_skills_linkedin_search_skill, portal_skill_pattern [EXTRACTED 1.00]
- **Portal Skill Shared Conventions** — agents_skills_freehire_search_skill, agents_skills_jobbank_search_skill, agents_skills_jobdanmark_search_skill, agents_skills_jobindex_search_skill, agents_skills_jobnet_search_skill, agents_skills_linkedin_search_skill, fork_context, bun_runtime, cli_search_detail, json_error_stderr [EXTRACTED 1.00]

## Communities (60 total, 15 thin omitted)

### Community 0 - "Shared CLI Core"
Cohesion: 0.08
Nodes (35): ALIAS, commaList(), Flags, FlagValue, main(), parseFlags(), parseIntFlag(), stringFlag() (+27 more)

### Community 1 - "Jobdanmark Portal"
Cohesion: 0.06
Nodes (26): autocomplete, AutocompleteGroup, AutocompleteItem, categories, Category, cleanText(), detail, DetailResult (+18 more)

### Community 2 - "Jobnet Portal"
Cohesion: 0.08
Nodes (20): detail, DetailApiResponse, formatDetailPlain(), outputPlain(), Occupation, OccupationAlias, occupations, buildSearchParams() (+12 more)

### Community 3 - "Security & Gitignore Guards"
Cohesion: 0.11
Nodes (15): CleanTreeTests, GitignoreGuardTests, GitignoreNegationTests, GuardRepoFixture, ManifestGuardTests, PermissionGuardTests, CompletedProcess, Path (+7 more)

### Community 4 - "Jobindex Portal"
Cohesion: 0.10
Nodes (22): buildUrl(), decodeHtmlEntities(), detail, DetailResult, extractIdFromUrl(), numericEntity(), parseDetailPage(), stripTags() (+14 more)

### Community 5 - "LinkedIn Portal"
Cohesion: 0.13
Nodes (23): Flags, main(), parseFlags(), DetailOpts, normalizeId(), runDetail(), buildUrl(), renderTable() (+15 more)

### Community 6 - "Candidate Profile & Applications"
Cohesion: 0.10
Nodes (34): Candidate Profile Document, Behavioral Profile Document, Writing Style Guide Document, Job Evaluation Framework Document, CV Templates and Tailoring Guide, Cover Letter Templates and Tailoring Guide, Interview Preparation Guide, Application Form Fields Guide (+26 more)

### Community 7 - "Salary Conversion Tool"
Cohesion: 0.13
Nodes (11): DetectColumnTypeTests, FakeWorksheet, detect_column_type(), header_matches(), main(), parse_sheet(), Return True when a header contains a meaningful pattern match.      Patterns mat, Remove count/index words from a header to derive a category name. (+3 more)

### Community 8 - "Freehire Portal"
Cohesion: 0.13
Nodes (14): detail, search, extractCdata(), extractJobIdFromUrl(), extractLink(), fetchWithUA(), findJobPosting(), ParsedDescription (+6 more)

### Community 9 - "Jobbank Package Config"
Cohesion: 0.08
Nodes (25): bin, jobbank, dependencies, @bunli/core, @bunli/utils, node-html-parser, zod, description (+17 more)

### Community 10 - "Jobdanmark Package Config"
Cohesion: 0.08
Nodes (25): bin, jobdanmark, dependencies, @bunli/core, @bunli/utils, node-html-parser, zod, description (+17 more)

### Community 11 - "Jobindex Package Config"
Cohesion: 0.08
Nodes (25): bin, jobindex, dependencies, @bunli/core, @bunli/utils, node-html-parser, zod, description (+17 more)

### Community 12 - "Rank & Match Scoring"
Cohesion: 0.11
Nodes (8): match_score(), Compute a match score between 0 and 100 for ranking results., MatchScoreTests, TestMatchScoreAnglicize, TestMatchScoreExactMatch, TestMatchScoreNoOverlap, TestMatchScoreShortQuery, TestMatchScoreSubstring

### Community 13 - "Workflow Commands"
Cohesion: 0.17
Nodes (25): Active Template Managed Block - Wires Custom Templates into /apply, Application Archive - documents/applications/<company>_<role>/, ATS Keyword and Parseability Verification - PDF Text Layer Audit, /add-portal - Generate a Job-Portal Search Skill, /add-template - Register a Custom CV or Cover Letter Template, /apply - Drafter-Reviewer Job Application Workflow, /expand - Competency Expansion from Documents and Online Presence, /gmail-sync - Sync Application Status from Gmail (+17 more)

### Community 14 - "PDF Verification"
Cohesion: 0.18
Nodes (12): Exception, ParsePageCountTests, RunToolTests, VerifyPdfTests, build_parser(), main(), normalize_text(), parse_page_count() (+4 more)

### Community 15 - "Freehire TSConfig"
Cohesion: 0.08
Nodes (23): compilerOptions, lib, module, moduleResolution, noFallthroughCasesInSwitch, noImplicitAny, noUnusedLocals, noUnusedParameters (+15 more)

### Community 16 - "Jobbank TSConfig"
Cohesion: 0.08
Nodes (23): compilerOptions, lib, module, moduleResolution, noFallthroughCasesInSwitch, noImplicitAny, noUnusedLocals, noUnusedParameters (+15 more)

### Community 17 - "Jobdanmark TSConfig"
Cohesion: 0.08
Nodes (23): compilerOptions, lib, module, moduleResolution, noFallthroughCasesInSwitch, noImplicitAny, noUnusedLocals, noUnusedParameters (+15 more)

### Community 18 - "Jobnet Package Config"
Cohesion: 0.08
Nodes (23): bin, jobnet, dependencies, @bunli/core, @bunli/utils, zod, description, devDependencies (+15 more)

### Community 19 - "Jobindex TSConfig"
Cohesion: 0.08
Nodes (23): compilerOptions, lib, module, moduleResolution, noFallthroughCasesInSwitch, noImplicitAny, noUnusedLocals, noUnusedParameters (+15 more)

### Community 20 - "LinkedIn TSConfig"
Cohesion: 0.13
Nodes (18): freehire-search CLI README, freehire-search SKILL.md, freehire-search URL Reference, jobbank-search CLI README, jobbank-search SKILL.md, jobbank-search URL Reference, jobdanmark-search CLI README, jobdanmark-search SKILL.md (+10 more)

### Community 21 - "Jobnet TSConfig"
Cohesion: 0.11
Nodes (17): bin, freehire-search, dependencies, description, devDependencies, @types/bun, typescript, @types/bun (+9 more)

### Community 22 - "LinkedIn Package Config"
Cohesion: 0.11
Nodes (17): bin, linkedin-search, dependencies, description, devDependencies, @types/bun, typescript, @types/bun (+9 more)

### Community 23 - "Lint Skills Tool"
Cohesion: 0.18
Nodes (17): Thin-pointer design, Changelog, Candidate profile, Contribution guidelines, Documents folder structure, Funding, CI pipeline, PR template (+9 more)

### Community 24 - "Cover Letter LaTeX"
Cohesion: 0.30
Nodes (7): Search for a company by name. Returns matching entries sorted by relevance., search_company(), _entry(), _make_data(), TestSearchCompanyBasicMatch, TestSearchCompanyCityFilter, TestSearchCompanyScoreThreshold

### Community 25 - "Job Scraper Skill"
Cohesion: 0.20
Nodes (13): collect_validation_issues(), fail_data_error(), load_data(), main(), print_validation_report(), Validate the salary data shape before lookups use it.      Preserves historical, Load and JSON-parse salary_data.json; exit with a helpful message if missing/inv, Load, parse, and validate salary_data.json for lookups. (+5 more)

### Community 26 - "Upskill Analysis"
Cohesion: 0.17
Nodes (11): anglicize(), extract_core_words(), match_score_optimized(), normalize(), Normalize string for robust fuzzy matching., Convert Danish/Nordic characters to anglicized equivalents., Extract meaningful words from a company name, ignoring noise., Compute a match score between 0 and 100 using precalculated query values. (+3 more)

### Community 27 - "CLI Retry & Timeout Tests"
Cohesion: 0.21
Nodes (3): Category-shape and duplicate-name checks (reuses assert_invalid_data)., ValidateDataShapeTests, ValidateDataTests

### Community 28 - "Framework Updates"
Cohesion: 0.14
Nodes (13): compilerOptions, allowImportingTsExtensions, module, moduleResolution, noEmit, skipLibCheck, strict, target (+5 more)

### Community 29 - "CI Pipeline"
Cohesion: 0.14
Nodes (13): compilerOptions, allowImportingTsExtensions, module, moduleResolution, noEmit, skipLibCheck, strict, target (+5 more)

### Community 30 - "Project Docs & Governance"
Cohesion: 0.14
Nodes (8): HtmlReportCommandFileTests, HtmlReportGitignoreTests, HtmlReportLintIntegrationTests, Tests for the /html-report command and its gitignore rule.  Mirrors the pattern, Structural checks on the command file itself., lint_skills.py rejects command files that don't start with '# /<name>'., reports/ must be gitignored — it holds personal generated output., lint_skills.py must pass after the command is added.

### Community 31 - "Search Query Normalization"
Cohesion: 0.29
Nodes (5): LinterRepoFixture, CompletedProcess, Path, run_linter(), SettingsShapeTests

### Community 33 - "CLI Contract Tests"
Cohesion: 0.28
Nodes (3): CLI_PATH, CLIResult, runCLI()

### Community 34 - "RSS Parsing Tests"
Cohesion: 0.32
Nodes (3): CLI_PATH, CLIResult, runCLI()

### Community 35 - "Detail Formatting Tests"
Cohesion: 0.32
Nodes (3): CLI_PATH, CLIResult, runCLI()

### Community 36 - "Detail JSON-LD Tests"
Cohesion: 0.32
Nodes (3): CLI_PATH, CLIResult, runCLI()

### Community 38 - "RSS Fetch Tests"
Cohesion: 0.33
Nodes (3): CLI_PATH, CLIResult, runCLI()

### Community 39 - "Search Tests"
Cohesion: 0.33
Nodes (3): CLI_PATH, CLIResult, runCLI()

### Community 40 - "CLI Helper Utilities"
Cohesion: 0.43
Nodes (3): format_entry(), Format a single company entry for display., FormatEntryTests

### Community 41 - "Readme Assets Tests"
Cohesion: 0.57
Nodes (6): get_base_commit(), has_non_trivial_changes(), main(), parse_frontmatter(), Path, run_git()

### Community 42 - "Notion Sync Tool"
Cohesion: 0.62
Nodes (6): check_command(), check_settings(), check_skill(), main(), Path, rel()

### Community 45 - "HTML Entity Decoding"
Cohesion: 0.70
Nodes (4): get_framework_version_from_text(), main(), parse_semver(), run_git()

## Knowledge Gaps
- **241 isolated node(s):** `name`, `version`, `description`, `type`, `main` (+236 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **15 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `match_score()` connect `Rank & Match Scoring` to `Job Scraper Skill`, `Upskill Analysis`?**
  _High betweenness centrality (0.003) - this node is a cross-community bridge._
- **Why does `ValidateDataTests` connect `CLI Retry & Timeout Tests` to `Upskill Analysis`?**
  _High betweenness centrality (0.002) - this node is a cross-community bridge._
- **Why does `search_company()` connect `Cover Letter LaTeX` to `Job Scraper Skill`, `Upskill Analysis`, `Salary Lookup Tool`?**
  _High betweenness centrality (0.002) - this node is a cross-community bridge._
- **What connects `name`, `version`, `description` to the rest of the system?**
  _241 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Shared CLI Core` be split into smaller, more focused modules?**
  _Cohesion score 0.08295625942684766 - nodes in this community are weakly interconnected._
- **Should `Jobdanmark Portal` be split into smaller, more focused modules?**
  _Cohesion score 0.06431372549019608 - nodes in this community are weakly interconnected._
- **Should `Jobnet Portal` be split into smaller, more focused modules?**
  _Cohesion score 0.08048780487804878 - nodes in this community are weakly interconnected._