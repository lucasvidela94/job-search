package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/lucasvidela94/jobsearch/internal/applications"
	"github.com/lucasvidela94/jobsearch/internal/config"
	"github.com/lucasvidela94/jobsearch/internal/cover"
	"github.com/lucasvidela94/jobsearch/internal/db"
	"github.com/lucasvidela94/jobsearch/internal/portal"
	"github.com/lucasvidela94/jobsearch/internal/profile"
	"github.com/lucasvidela94/jobsearch/internal/rank"
	"github.com/lucasvidela94/jobsearch/internal/scrape"
	"github.com/lucasvidela94/jobsearch/internal/store"
)

// ToolSchema defines an MCP tool.
type ToolSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// TextContent is an MCP text content item.
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolResult wraps the result of a tool call.
type ToolResult struct {
	Content []TextContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolHandler implements mcp.Handler for jobsearch tools.
type ToolHandler struct {
	DB      *db.DB
	Apps    *applications.Repository
	Store   *store.Store
	Config  *config.Config
	Version string
	Portals []string
}

var _ Handler = (*ToolHandler)(nil)

// NewToolHandler creates a handler with all tool dependencies.
func NewToolHandler(cfg *config.Config, st *store.Store, database *db.DB, apps *applications.Repository, version string) *ToolHandler {
	names := portal.Names()
	sort.Strings(names)
	return &ToolHandler{
		DB:      database,
		Apps:    apps,
		Store:   st,
		Config:  cfg,
		Version: version,
		Portals: names,
	}
}

// HandleMethod dispatches MCP methods.
func (h *ToolHandler) HandleMethod(method string, params json.RawMessage) (any, *RPCError) {
	switch method {
	case "initialize":
		return h.handleInitialize(params)
	case "tools/list":
		return h.handleToolList(params)
	case "tools/call":
		return h.handleToolCall(params)
	default:
		return nil, &RPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", method)}
	}
}

// InitializeResult is the response to initialize.
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]any         `json:"capabilities"`
	ServerInfo      map[string]string      `json:"serverInfo"`
}

func (h *ToolHandler) handleInitialize(_ json.RawMessage) (any, *RPCError) {
	return InitializeResult{
		ProtocolVersion: "2025-03-26",
		Capabilities: map[string]any{
			"tools": map[string]any{},
		},
		ServerInfo: map[string]string{
			"name":    "jobsearch",
			"version": h.Version,
		},
	}, nil
}

func (h *ToolHandler) handleToolList(_ json.RawMessage) (any, *RPCError) {
	return map[string][]ToolSchema{
		"tools": h.tools(),
	}, nil
}

// tools returns all tool definitions.
func (h *ToolHandler) tools() []ToolSchema {
	return []ToolSchema{
		{
			Name:        "search",
			Description: "Search jobs across registered portals. Returns job postings with title, company, location, URL, and apply link.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":        map[string]any{"type": "string", "description": "Search keywords for job title, skills, or company"},
					"location":     map[string]any{"type": "string", "description": "City, region, or 'remote' for location filter"},
					"source":       map[string]any{"type": "string", "description": fmt.Sprintf("Portal: %s (default: all)", joinStrings(h.Portals))},
					"days":         map[string]any{"type": "number", "description": "Recency filter in days (1, 7, 14, 30)"},
					"remote":       map[string]any{"type": "string", "description": "Work type: remote, hybrid, onsite"},
					"limit":        map[string]any{"type": "number", "description": "Max results per portal (max 100)"},
					"applied_only": map[string]any{"type": "boolean", "description": "Only show jobs you've already applied to"},
				},
			},
		},
		{
			Name:        "detail",
			Description: "Get full job description for a specific posting ID.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string", "description": "Job posting ID from search results"},
					"source": map[string]any{"type": "string", "description": fmt.Sprintf("Portal: %s (default: linkedin)", joinStrings(h.Portals))},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "scrape",
			Description: "Multi-portal job scrape with dedup. Searches all portals, deduplicates against previously seen jobs, and saves new ones.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":    map[string]any{"type": "string", "description": "Search keywords"},
					"location": map[string]any{"type": "string", "description": "Location filter"},
					"days":     map[string]any{"type": "number", "description": "Recency in days (default: 7)"},
					"remote":   map[string]any{"type": "string", "description": "Work type: remote, hybrid, onsite"},
					"limit":    map[string]any{"type": "number", "description": "Max results per portal (default: 25)"},
				},
			},
		},
		{
			Name:        "rank",
			Description: "Score scraped jobs against your profile. Returns ranked list with fit scores.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"top":    map[string]any{"type": "number", "description": "Number of top results to return"},
					"min_score": map[string]any{"type": "number", "description": "Minimum fit score (0-100) to include"},
				},
			},
		},
		{
			Name:        "apply",
			Description: "Mark a job as applied, open the apply URL in browser, and optionally generate a cover letter.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string", "description": "Job posting ID from search results"},
					"source": map[string]any{"type": "string", "description": "Portal source for the job ID"},
					"cover":  map[string]any{"type": "boolean", "description": "Generate a cover letter (default: false)"},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "status",
			Description: "Show the full timeline of events for an application.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "Application ID (from apply or pipeline)"},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "pipeline",
			Description: "List all job applications with their latest status.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{"type": "string", "description": "Filter by status tag (e.g. applied, interview, rejected)"},
				},
			},
		},
		{
			Name:        "log_event",
			Description: "Add an event to an application's timeline (e.g. phone_screen, tech_interview, offer).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "string", "description": "Application ID"},
					"event": map[string]any{"type": "string", "description": "Event type (e.g. phone_screen, interview, offer, rejected)"},
					"notes": map[string]any{"type": "string", "description": "Optional notes about the event"},
				},
				"required": []string{"id", "event"},
			},
		},
		{
			Name:        "profile",
			Description: "View or update your candidate profile used for job matching and cover letters.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action":    map[string]any{"type": "string", "description": "Action: 'show' to view, 'edit' to update", "enum": []string{"show", "edit"}},
					"name":      map[string]any{"type": "string", "description": "Your full name"},
					"title":     map[string]any{"type": "string", "description": "Your professional title"},
					"skill":     map[string]any{"type": "string", "description": "Add a skill (repeatable per call)"},
					"seniority": map[string]any{"type": "string", "description": "Seniority level"},
					"remote":    map[string]any{"type": "string", "description": "Work mode: remote, hybrid, onsite"},
				},
				"required": []string{"action"},
			},
		},
	}
}

// CallToolRequest params for tools/call.
type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (h *ToolHandler) handleToolCall(rawParams json.RawMessage) (any, *RPCError) {
	var params callToolParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params: " + err.Error()}
	}

	ctx := context.Background()
	args := params.Arguments

	var result any
	var err error

	switch params.Name {
	case "search":
		result, err = h.callSearch(ctx, args)
	case "detail":
		result, err = h.callDetail(ctx, args)
	case "scrape":
		result, err = h.callScrape(ctx, args)
	case "rank":
		result, err = h.callRank(ctx, args)
	case "apply":
		result, err = h.callApply(ctx, args)
	case "status":
		result, err = h.callStatus(ctx, args)
	case "pipeline":
		result, err = h.callPipeline(ctx, args)
	case "log_event":
		result, err = h.callLogEvent(ctx, args)
	case "profile":
		result, err = h.callProfile(ctx, args)
	default:
		return nil, &RPCError{Code: -32602, Message: fmt.Sprintf("Unknown tool: %s", params.Name)}
	}

	if err != nil {
		return ToolResult{
			Content: []TextContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return ToolResult{
		Content: []TextContent{{Type: "text", Text: string(data)}},
	}, nil
}

// --- Tool implementations ---

func (h *ToolHandler) callSearch(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		Query       string `json:"query"`
		Location    string `json:"location"`
		Source      string `json:"source"`
		Days        int    `json:"days"`
		Remote      string `json:"remote"`
		Limit       int    `json:"limit"`
		AppliedOnly bool   `json:"applied_only"`
	}
	json.Unmarshal(args, &p)

	srcNames, _ := parseSource(p.Source)
	portals := resolveSources(srcNames)
	if len(portals) == 0 {
		portals = portal.All()
	}

	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 10
	}

	params := portal.SearchParams{
		Query:    p.Query,
		Location: p.Location,
		Days:     p.Days,
		Remote:   p.Remote,
		Limit:    p.Limit,
	}

	type enrichedJob struct {
		portal.JobPosting
		Applied bool `json:"applied"`
	}

	var results []enrichedJob
	for name, pr := range portals {
		res, err := pr.Search(ctx, params)
		if err != nil {
			continue
		}
		for _, j := range res.Results {
			applied := false
			if j.URL != "" {
				if existing, err := h.Apps.FindByURL(ctx, j.URL); err == nil && existing != nil {
					applied = true
				}
			}
			if p.AppliedOnly && !applied {
				continue
			}
			j.Source = name
			results = append(results, enrichedJob{JobPosting: j, Applied: applied})
		}
	}

	return results, nil
}

func (h *ToolHandler) callDetail(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		ID     string `json:"id"`
		Source string `json:"source"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	if p.Source == "" {
		p.Source = "linkedin"
	}

	pr, ok := portal.Get(p.Source)
	if !ok {
		return nil, fmt.Errorf("unknown source: %s", p.Source)
	}

	job, err := pr.Detail(ctx, p.ID)
	if err != nil {
		return nil, err
	}

	type detailResult struct {
		portal.JobPosting
		Applied   bool   `json:"applied"`
		AppID     string `json:"app_id,omitempty"`
		AppStatus string `json:"app_status,omitempty"`
	}
	result := detailResult{JobPosting: *job}
	if job.URL != "" {
		if existing, err := h.Apps.FindByURL(ctx, job.URL); err == nil && existing != nil {
			result.Applied = true
			result.AppID = existing.ID
			result.AppStatus = existing.Status
		}
	}
	return result, nil
}

func (h *ToolHandler) callScrape(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		Query    string `json:"query"`
		Location string `json:"location"`
		Days     int    `json:"days"`
		Remote   string `json:"remote"`
		Limit    int    `json:"limit"`
	}
	json.Unmarshal(args, &p)

	if p.Limit <= 0 {
		p.Limit = 25
	}
	if p.Days <= 0 {
		p.Days = 7
	}

	cfg := scrape.Config{
		Portals:  portal.Names(),
		Query:    p.Query,
		Location: p.Location,
		Days:     p.Days,
		Remote:   p.Remote,
		Limit:    p.Limit,
	}

	result, err := scrape.Run(ctx, cfg, h.Store)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (h *ToolHandler) callRank(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		Top      int `json:"top"`
		MinScore int `json:"min_score"`
	}
	json.Unmarshal(args, &p)

	prof, err := profile.Load()
	if err != nil {
		return nil, fmt.Errorf("load profile: %w", err)
	}

	entries, err := h.Store.LoadSeenJobs()
	if err != nil {
		return nil, fmt.Errorf("load seen jobs: %w", err)
	}

	jobs := make([]portal.JobPosting, 0, len(entries))
	for _, e := range entries {
		jobs = append(jobs, portal.JobPosting{
			ID:      e.URL,
			Title:   e.Title,
			Company: e.Company,
			URL:     e.URL,
			Source:  e.Source,
		})
	}

	cfgRank := rank.Config{TopN: p.Top}
	if cfgRank.TopN <= 0 {
		cfgRank.TopN = 50
	}

	scored := rank.ScoreJobs(jobs, prof, cfgRank)
	if p.MinScore > 0 {
		var filtered []rank.ScoredJob
		for _, s := range scored {
			if int(s.Score) >= p.MinScore {
				filtered = append(filtered, s)
			}
		}
		scored = filtered
	}

	return scored, nil
}

func (h *ToolHandler) callApply(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		ID     string `json:"id"`
		Source string `json:"source"`
		Cover  bool   `json:"cover"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	if p.ID == "" {
		return nil, fmt.Errorf("id is required")
	}

	// Try to find the job
	var job *portal.JobPosting
	if p.Source != "" {
		pr, ok := portal.Get(p.Source)
		if ok {
			j, err := pr.Detail(ctx, p.ID)
			if err == nil {
				job = j
			}
		}
	}
	if job == nil {
		job = &portal.JobPosting{ID: p.ID, Title: p.ID, Source: p.Source}
	}

	if job.URL != "" {
		existing, err := h.Apps.FindByURL(ctx, job.URL)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("already applied (application: %s)", existing.ID)
		}
	}

	app, err := h.Apps.Create(ctx, *job)
	if err != nil {
		return nil, fmt.Errorf("create application: %w", err)
	}

	type applyResult struct {
		Application *applications.Application `json:"application"`
		CoverLetter string                    `json:"cover_letter,omitempty"`
	}
	result := applyResult{Application: app}

	if p.Cover {
		prof, err := profile.Load()
		if err == nil {
			gen, err := cover.New(prof, *job)
			if err == nil {
				path, err := gen.RenderToFile(config.ConfigDir())
				if err == nil {
					result.CoverLetter = path
				}
			}
		}
	}

	return result, nil
}

func (h *ToolHandler) callStatus(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}

	app, err := h.Apps.GetByID(ctx, p.ID)
	if err != nil {
		return nil, err
	}

	events, err := h.Apps.GetEvents(ctx, p.ID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"application": app,
		"events":      events,
	}, nil
}

func (h *ToolHandler) callPipeline(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		Status string `json:"status"`
	}
	json.Unmarshal(args, &p)

	apps, err := h.Apps.List(ctx, p.Status)
	if err != nil {
		return nil, err
	}
	if apps == nil {
		apps = []applications.Application{}
	}
	return apps, nil
}

func (h *ToolHandler) callLogEvent(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		ID    string `json:"id"`
		Event string `json:"event"`
		Notes string `json:"notes"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	if p.ID == "" {
		return nil, fmt.Errorf("id is required")
	}
	if p.Event == "" {
		return nil, fmt.Errorf("event is required")
	}

	event, err := h.Apps.AddEvent(ctx, p.ID, p.Event, p.Notes)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (h *ToolHandler) callProfile(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		Action    string `json:"action"`
		Name      string `json:"name"`
		Title     string `json:"title"`
		Skill     string `json:"skill"`
		Seniority string `json:"seniority"`
		Remote    string `json:"remote"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}

	switch p.Action {
	case "show":
		prof, err := profile.Load()
		if err != nil {
			return nil, fmt.Errorf("no profile found. Use profile edit to create one.")
		}
		return prof, nil

	case "edit":
		prof, err := profile.Load()
		if err != nil {
			prof = &profile.Profile{}
		}
		if p.Name != "" {
			prof.Name = p.Name
		}
		if p.Title != "" {
			prof.Title = p.Title
		}
		if p.Skill != "" {
			prof.SetSkill(p.Skill)
		}
		if p.Seniority != "" {
			prof.Seniority = p.Seniority
		}
		if p.Remote != "" {
			prof.Remote = p.Remote
		}
		if err := profile.Save(prof); err != nil {
			return nil, fmt.Errorf("save profile: %w", err)
		}
		return prof, nil

	default:
		return nil, fmt.Errorf("unknown action: %s (use 'show' or 'edit')", p.Action)
	}
}

// -- helpers (duplicated from cli.go to avoid circular import) --

func parseSource(raw string) ([]string, error) {
	if raw == "" || raw == "all" {
		return []string{"all"}, nil
	}
	return []string{raw}, nil
}

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

func joinStrings(s []string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) == 1 {
		return s[0]
	}
	return strings.Join(s, ", ")
}
