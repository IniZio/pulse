package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pulse/pm/internal/db"
)

// Server represents the Pulse web server
type Server struct {
	addr             string
	mux              *http.ServeMux
	server           *http.Server
	db               *db.DB
	workspaceRepo    *db.WorkspaceRepository
	issueRepo        *db.IssueRepository
	cycleRepo        *db.CycleRepository
}

// NewServer creates a new Pulse server
func NewServer(addr, dataDir string) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	database, err := db.New(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := database.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	s := &Server{
		addr:             addr,
		mux:              http.NewServeMux(),
		db:               database,
		workspaceRepo:    db.NewWorkspaceRepository(database),
		issueRepo:        db.NewIssueRepository(database),
		cycleRepo:        db.NewCycleRepository(database),
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) registerRoutes() {
	// API routes
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/workspaces", s.handleWorkspaces)
	s.mux.HandleFunc("/api/workspaces/", s.handleWorkspace)
	s.mux.HandleFunc("/api/issues", s.handleIssues)
	s.mux.HandleFunc("/api/issues/", s.handleIssue)
	s.mux.HandleFunc("/api/cycles", s.handleCycles)
	s.mux.HandleFunc("/api/cycles/", s.handleCycle)
	s.mux.HandleFunc("/api/metrics", s.handleMetrics)
	s.mux.HandleFunc("/api/search", s.handleSearch)

	// Web UI
	s.mux.HandleFunc("/", s.handleWebUI)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
		"database":  s.db.Path(),
	})
}

func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		workspaces, err := s.workspaceRepo.List()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to list workspaces: %v", err), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, workspaces)

	case http.MethodPost:
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Settings    string `json:"settings"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		ws := &db.Workspace{
			ID:          fmt.Sprintf("ws_%d", time.Now().UnixNano()),
			Name:        req.Name,
			Description: req.Description,
			Settings:    req.Settings,
		}

		if err := s.workspaceRepo.Create(ws); err != nil {
			http.Error(w, fmt.Sprintf("failed to create workspace: %v", err), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, ws)
	}
}

func (s *Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	id := filepath.Base(r.URL.Path)

	ws, err := s.workspaceRepo.GetByID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get workspace: %v", err), http.StatusInternalServerError)
		return
	}
	if ws == nil {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, ws)

	case http.MethodPut:
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if name, ok := req["name"].(string); ok {
			ws.Name = name
		}
		if desc, ok := req["description"].(string); ok {
			ws.Description = desc
		}
		if settings, ok := req["settings"].(string); ok {
			ws.Settings = settings
		}

		if err := s.workspaceRepo.Update(ws); err != nil {
			http.Error(w, fmt.Sprintf("failed to update workspace: %v", err), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, ws)

	case http.MethodDelete:
		if err := s.workspaceRepo.Delete(id); err != nil {
			http.Error(w, fmt.Sprintf("failed to delete workspace: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleIssues(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		workspaceID := r.URL.Query().Get("workspace_id")
		status := r.URL.Query().Get("status")

		var limit, offset int
		fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)
		fmt.Sscanf(r.URL.Query().Get("offset"), "%d", &offset)

		issues, err := s.issueRepo.List(workspaceID, status, limit, offset)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to list issues: %v", err), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, issues)

	case http.MethodPost:
		var req struct {
			WorkspaceID string   `json:"workspace_id"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Status      string   `json:"status"`
			Priority    int      `json:"priority"`
			AssigneeID  string   `json:"assignee_id"`
			Labels      []string `json:"labels"`
			Estimate    int      `json:"estimate"`
			CycleID     string   `json:"cycle_id"`
			ParentID    string   `json:"parent_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Verify workspace exists
		ws, err := s.workspaceRepo.GetByID(req.WorkspaceID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to verify workspace: %v", err), http.StatusInternalServerError)
			return
		}
		if ws == nil {
			http.Error(w, "workspace not found", http.StatusNotFound)
			return
		}

		if req.Status == "" {
			req.Status = "backlog"
		}

		issue := &db.Issue{
			ID:          fmt.Sprintf("issue_%d", time.Now().UnixNano()),
			WorkspaceID: req.WorkspaceID,
			Title:       req.Title,
			Description: req.Description,
			Status:      req.Status,
			Priority:    req.Priority,
			AssigneeID:  req.AssigneeID,
			Labels:      req.Labels,
			Estimate:    req.Estimate,
			CycleID:     req.CycleID,
			ParentID:    req.ParentID,
		}

		if err := s.issueRepo.Create(issue); err != nil {
			http.Error(w, fmt.Sprintf("failed to create issue: %v", err), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, issue)
	}
}

func (s *Server) handleIssue(w http.ResponseWriter, r *http.Request) {
	id := filepath.Base(r.URL.Path)

	issue, err := s.issueRepo.GetByID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get issue: %v", err), http.StatusInternalServerError)
		return
	}
	if issue == nil {
		http.Error(w, "issue not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, issue)

	case http.MethodPut:
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if title, ok := req["title"].(string); ok {
			issue.Title = title
		}
		if desc, ok := req["description"].(string); ok {
			issue.Description = desc
		}
		if status, ok := req["status"].(string); ok {
			issue.Status = status
		}
		if priority, ok := req["priority"].(float64); ok {
			issue.Priority = int(priority)
		}
		if assignee, ok := req["assignee_id"].(string); ok {
			issue.AssigneeID = assignee
		}
		if estimate, ok := req["estimate"].(float64); ok {
			issue.Estimate = int(estimate)
		}
		if cycleID, ok := req["cycle_id"].(string); ok {
			issue.CycleID = cycleID
		}
		if parentID, ok := req["parent_id"].(string); ok {
			issue.ParentID = parentID
		}
		if labels, ok := req["labels"].([]interface{}); ok {
			issue.Labels = make([]string, len(labels))
			for i, l := range labels {
				issue.Labels[i] = l.(string)
			}
		}

		if err := s.issueRepo.Update(issue); err != nil {
			http.Error(w, fmt.Sprintf("failed to update issue: %v", err), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, issue)

	case http.MethodDelete:
		if err := s.issueRepo.Delete(id); err != nil {
			http.Error(w, fmt.Sprintf("failed to delete issue: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodPatch:
		// Handle status-only updates
		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if err := s.issueRepo.UpdateStatus(id, req.Status); err != nil {
			http.Error(w, fmt.Sprintf("failed to update status: %v", err), http.StatusInternalServerError)
			return
		}

		issue, _ := s.issueRepo.GetByID(id)
		jsonResponse(w, issue)
	}
}

func (s *Server) handleCycles(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace_id")

	switch r.Method {
	case http.MethodGet:
		cycles, err := s.cycleRepo.List(workspaceID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to list cycles: %v", err), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, cycles)

	case http.MethodPost:
		var req struct {
			WorkspaceID string  `json:"workspace_id"`
			Name        string  `json:"name"`
			StartDate   *string `json:"start_date"`
			EndDate     *string `json:"end_date"`
			Status      string  `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		cycle := &db.Cycle{
			ID:          fmt.Sprintf("cycle_%d", time.Now().UnixNano()),
			WorkspaceID: req.WorkspaceID,
			Name:        req.Name,
			Status:      req.Status,
		}

		if req.StartDate != nil {
			t, _ := time.Parse(time.RFC3339, *req.StartDate)
			cycle.StartDate = &t
		}
		if req.EndDate != nil {
			t, _ := time.Parse(time.RFC3339, *req.EndDate)
			cycle.EndDate = &t
		}

		if err := s.cycleRepo.Create(cycle); err != nil {
			http.Error(w, fmt.Sprintf("failed to create cycle: %v", err), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, cycle)
	}
}

func (s *Server) handleCycle(w http.ResponseWriter, r *http.Request) {
	id := filepath.Base(r.URL.Path)

	cycle, err := s.cycleRepo.GetByID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get cycle: %v", err), http.StatusInternalServerError)
		return
	}
	if cycle == nil {
		http.Error(w, "cycle not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, cycle)

	case http.MethodPut:
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if name, ok := req["name"].(string); ok {
			cycle.Name = name
		}
		if status, ok := req["status"].(string); ok {
			cycle.Status = status
		}

		if err := s.cycleRepo.Update(cycle); err != nil {
			http.Error(w, fmt.Sprintf("failed to update cycle: %v", err), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, cycle)

	case http.MethodDelete:
		if err := s.cycleRepo.Delete(id); err != nil {
			http.Error(w, fmt.Sprintf("failed to delete cycle: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		workspaceID = "default"
	}

	// Get issue counts by status
	statusCounts, err := s.issueRepo.CountByStatus(workspaceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to count issues: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate velocity metrics
	var totalPoints, completedPoints int
	issues, _ := s.issueRepo.List(workspaceID, "", 0, 0)
	for _, issue := range issues {
		totalPoints += issue.Estimate
		if issue.Status == "done" {
			completedPoints += issue.Estimate
		}
	}

	var bugs int
	for _, issue := range issues {
		for _, label := range issue.Labels {
			if label == "bug" {
				bugs++
				break
			}
		}
	}

	totalIssues := 0
	for _, count := range statusCounts {
		totalIssues += count
	}

	completionRate := 0.0
	if totalIssues > 0 {
		completionRate = float64(statusCounts["done"]) / float64(totalIssues) * 100
	}

	metrics := map[string]interface{}{
		"workspace_id":      workspaceID,
		"total_issues":     totalIssues,
		"backlog_count":    statusCounts["backlog"],
		"todo_count":       statusCounts["todo"],
		"in_progress_count": statusCounts["in_progress"],
		"done_count":       statusCounts["done"],
		"total_points":     totalPoints,
		"completed_points": completedPoints,
		"completion_rate":   completionRate,
		"bug_count":        bugs,
	}

	jsonResponse(w, metrics)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		workspaceID = "default"
	}

	// Parse filters from query
	statusFilter := ""
	labelFilter := ""
	assigneeFilter := ""

	// Handle filter prefixes: status:, label:, assignee:
	if query != "" {
		// Check for status: filter
		if strings.HasPrefix(query, "status:") {
			statusFilter = strings.TrimPrefix(query, "status:")
			query = ""
		} else if strings.HasPrefix(query, "label:") {
			labelFilter = strings.TrimPrefix(query, "label:")
			query = ""
		} else if strings.HasPrefix(query, "assignee:") {
			assigneeFilter = strings.TrimPrefix(query, "assignee:")
			query = ""
		}
	}

	// Also check individual query params
	if statusFilter == "" {
		statusFilter = r.URL.Query().Get("status")
	}
	if labelFilter == "" {
		labelFilter = r.URL.Query().Get("label")
	}
	if assigneeFilter == "" {
		assigneeFilter = r.URL.Query().Get("assignee")
	}

	// Get all issues for workspace
	issues, err := s.issueRepo.List(workspaceID, "", 0, 0)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to search issues: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter issues
	var results []interface{}
	for _, issue := range issues {
		matches := true

		// Text search
		if query != "" {
			if !contains(issue.Title, query) && !contains(issue.Description, query) {
				matches = false
			}
		}

		// Status filter
		if statusFilter != "" && issue.Status != statusFilter {
			matches = false
		}

		// Label filter
		if labelFilter != "" {
			found := false
			for _, l := range issue.Labels {
				if contains(l, labelFilter) {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		// Assignee filter
		if assigneeFilter != "" && issue.AssigneeID != assigneeFilter {
			matches = false
		}

		if matches {
			results = append(results, map[string]interface{}{
				"type":      "issue",
				"id":        issue.ID,
				"title":     issue.Title,
				"status":    issue.Status,
				"labels":    issue.Labels,
				"estimate":  issue.Estimate,
				"workspace": workspaceID,
			})
		}
	}

	jsonResponse(w, results)
}

func (s *Server) handleWebUI(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, webUIHTML())
}

// webUIHTML returns the web UI HTML page
func webUIHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>Pulse - Project Management</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0F1117; color: #ECEFF1; }
        .app { display: flex; height: 100vh; }
        .sidebar { width: 240px; background: #161B22; border-right: 1px solid #30363D; padding: 16px; }
        .logo { font-size: 20px; font-weight: 700; color: #A371F7; margin-bottom: 24px; display: flex; align-items: center; gap: 8px; }
        .nav-item { display: flex; align-items: center; padding: 8px 12px; border-radius: 6px; color: #8B949E; cursor: pointer; margin-bottom: 4px; }
        .nav-item:hover, .nav-item.active { background: #21262D; color: #ECEFF1; }
        .main { flex: 1; overflow: auto; }
        .header { padding: 16px 24px; border-bottom: 1px solid #30363D; display: flex; justify-content: space-between; align-items: center; }
        .header h1 { font-size: 18px; font-weight: 600; }
        .btn { background: #238636; color: white; border: none; padding: 8px 16px; border-radius: 6px; cursor: pointer; font-size: 14px; }
        .btn:hover { background: #2EA043; }
        .btn-secondary { background: #21262D; color: #ECEFF1; border: 1px solid #30363D; }
        .btn-danger { background: #F85149; color: white; border: none; }
        .btn-danger:hover { background: #DA3633; }
        .board { display: flex; padding: 24px; gap: 16px; overflow-x: auto; }
        .column { min-width: 280px; background: #0D1117; border-radius: 8px; padding: 12px; }
        .column-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
        .column-title { font-weight: 600; font-size: 14px; display: flex; align-items: center; gap: 8px; }
        .column-count { background: #30363D; padding: 2px 8px; border-radius: 10px; font-size: 12px; color: #8B949E; }
        .issue { background: #161B22; border: 1px solid #30363D; border-radius: 6px; padding: 12px; margin-bottom: 8px; cursor: pointer; }
        .issue:hover { border-color: #58A6FF; }
        .issue-id { font-size: 11px; color: #6E7681; font-family: monospace; margin-bottom: 4px; }
        .issue-title { font-size: 14px; margin-bottom: 8px; }
        .issue-labels { display: flex; gap: 4px; flex-wrap: wrap; }
        .label { font-size: 11px; padding: 2px 6px; border-radius: 4px; background: #30363D; }
        .label.bug { background: #F85149; color: white; }
        .label.feature { background: #A371F7; color: white; }
        .priority { width: 4px; height: 20px; border-radius: 2px; display: inline-block; margin-right: 8px; }
        .priority.urgent { background: #F85149; }
        .priority.high { background: #F0883E; }
        .priority.medium { background: #F0C420; }
        .priority.low { background: #3FB950; }
        .search { background: #0D1117; border: 1px solid #30363D; padding: 8px 12px; border-radius: 6px; color: #ECEFF1; width: 240px; }
        .search:focus { outline: none; border-color: #58A6FF; }
        .modal { display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.7); align-items: center; justify-content: center; z-index: 1000; }
        .modal.active { display: flex; }
        .modal-content { background: #161B22; border-radius: 8px; padding: 24px; width: 600px; max-width: 95%; max-height: 90vh; overflow-y: auto; }
        .modal-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; }
        .modal-title { font-size: 18px; font-weight: 600; }
        .close-btn { background: none; border: none; color: #8B949E; cursor: pointer; font-size: 20px; }
        .form-group { margin-bottom: 16px; }
        .form-group label { display: block; margin-bottom: 8px; font-size: 14px; color: #8B949E; }
        .form-group input, .form-group select, .form-group textarea { width: 100%; background: #0D1117; border: 1px solid #30363D; border-radius: 6px; padding: 8px 12px; color: #ECEFF1; font-size: 14px; }
        .form-group input:focus, .form-group select:focus, .form-group textarea:focus { outline: none; border-color: #58A6FF; }
        .form-actions { display: flex; justify-content: space-between; gap: 8px; margin-top: 24px; }
        .form-actions-left { display: flex; gap: 8px; }
        .form-actions-right { display: flex; gap: 8px; }
        .metrics { display: flex; gap: 24px; padding: 16px 24px; background: #161B22; border-bottom: 1px solid #30363D; }
        .metric { text-align: center; }
        .metric-value { font-size: 24px; font-weight: 600; color: #A371F7; }
        .metric-label { font-size: 12px; color: #8B949E; margin-top: 4px; }
        .issue-meta { display: flex; gap: 16px; margin-top: 16px; padding-top: 16px; border-top: 1px solid #30363D; }
        .meta-item { display: flex; flex-direction: column; gap: 4px; }
        .meta-label { font-size: 12px; color: #8B949E; }
        .meta-value { font-size: 14px; color: #ECEFF1; }
        .status-badge { display: inline-block; padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: 500; }
        .status-backlog { background: #30363D; color: #8B949E; }
        .status-todo { background: #F59E0B20; color: #F59E0B; }
        .status-in_progress { background: #3B82F620; color: #3B82F6; }
        .status-done { background: #10B98120; color: #10B981; }
        .description-text { color: #8B949E; font-size: 14px; line-height: 1.6; margin-top: 8px; }
    </style>
</head>
<body>
    <div class="app">
        <div class="sidebar">
            <div class="logo">Pulse</div>
            <div class="nav-item active" onclick="showBoard()">Board</div>
            <div class="nav-item" onclick="showMetrics()">Analytics</div>
            <div class="nav-item" onclick="showCycles()">Cycles</div>
            <div class="nav-item">Labels</div>
            <div class="nav-item">Settings</div>
        </div>
        <div class="main">
            <div class="header">
                <h1 id="pageTitle">Project Board</h1>
                <input type="text" class="search" placeholder="Search issues..." id="search" oninput="handleSearch(this.value)">
                <button class="btn" id="createBtn" onclick="openCreateModal()">+ New Issue</button>
            </div>
            <div class="metrics" id="metricsBar" style="display: none;">
                <div class="metric">
                    <div class="metric-value" id="totalIssues">0</div>
                    <div class="metric-label">Total</div>
                </div>
                <div class="metric">
                    <div class="metric-value" id="inProgress">0</div>
                    <div class="metric-label">In Progress</div>
                </div>
                <div class="metric">
                    <div class="metric-value" id="completed">0</div>
                    <div class="metric-label">Completed</div>
                </div>
                <div class="metric">
                    <div class="metric-value" id="velocity">0</div>
                    <div class="metric-label">Velocity</div>
                </div>
            </div>
            <div class="board" id="board"></div>
            <div id="metricsView" style="display: none; padding: 24px;">
                <h2>Analytics</h2>
                <p>Velocity metrics coming soon...</p>
            </div>
        </div>
    </div>

    <!-- Create Issue Modal -->
    <div class="modal" id="createModal">
        <div class="modal-content">
            <div class="modal-header">
                <h2 class="modal-title">Create Issue</h2>
                <button class="close-btn" onclick="closeCreateModal()">&times;</button>
            </div>
            <form onsubmit="handleCreate(event)">
                <div class="form-group">
                    <label>Title</label>
                    <input type="text" id="issueTitle" required placeholder="Enter issue title">
                </div>
                <div class="form-group">
                    <label>Description</label>
                    <textarea id="issueDescription" rows="3" placeholder="Enter description"></textarea>
                </div>
                <div class="form-group">
                    <label>Priority</label>
                    <select id="issuePriority">
                        <option value="4">Low</option>
                        <option value="3" selected>Medium</option>
                        <option value="2">High</option>
                        <option value="1">Urgent</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Estimate (story points)</label>
                    <input type="number" id="issueEstimate" min="0" value="0">
                </div>
                <div class="form-group">
                    <label>Labels</label>
                    <input type="text" id="issueLabels" placeholder="feature, bug (comma-separated)">
                </div>
                <div class="form-actions">
                    <button type="button" class="btn btn-secondary" onclick="closeCreateModal()">Cancel</button>
                    <button type="submit" class="btn">Create Issue</button>
                </div>
            </form>
        </div>
    </div>

    <!-- Issue Detail Modal -->
    <div class="modal" id="detailModal">
        <div class="modal-content">
            <div class="modal-header">
                <div>
                    <div class="issue-id" id="detailId">PLS-000</div>
                    <h2 class="modal-title" id="detailTitle">Issue Title</h2>
                </div>
                <button class="close-btn" onclick="closeDetailModal()">&times;</button>
            </div>

            <div class="issue-meta">
                <div class="meta-item">
                    <span class="meta-label">Status</span>
                    <span class="status-badge" id="detailStatus">backlog</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">Priority</span>
                    <span class="meta-value" id="detailPriority">Medium</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">Estimate</span>
                    <span class="meta-value" id="detailEstimate">0 pts</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">Created</span>
                    <span class="meta-value" id="detailCreated">-</span>
                </div>
            </div>

            <div class="form-group" style="margin-top: 16px;">
                <label>Description</label>
                <p class="description-text" id="detailDescription">No description provided.</p>
            </div>

            <div class="form-group">
                <label>Labels</label>
                <div id="detailLabels"></div>
            </div>

            <div class="form-group">
                <label>Change Status</label>
                <select id="detailStatusSelect" onchange="updateDetailStatus()">
                    <option value="backlog">Backlog</option>
                    <option value="todo">To Do</option>
                    <option value="in_progress">In Progress</option>
                    <option value="done">Done</option>
                </select>
            </div>

            <div class="form-actions">
                <button class="btn btn-danger" onclick="deleteIssue()">Delete</button>
                <div>
                    <button class="btn btn-secondary" onclick="closeDetailModal()">Close</button>
                </div>
            </div>
        </div>
    </div>

    <script>
        var issues = [];
        var columns = ['backlog', 'todo', 'in_progress', 'done'];
        var currentView = 'board';
        var workspaceID = 'default';

        function getColumnColor(col) {
            var colors = { backlog: '#6B7280', todo: '#F59E0B', in_progress: '#3B82F6', done: '#10B981' };
            return colors[col] || '#6B7280';
        }

        function getPriorityClass(priority) {
            var classes = ['', 'urgent', 'high', 'medium', 'low'];
            return classes[priority] || '';
        }

        function renderIssue(issue) {
            var priorityClass = getPriorityClass(issue.priority);
            var labelsHtml = '';
            if (issue.labels) {
                for (var i = 0; i < issue.labels.length; i++) {
                    labelsHtml += '<span class="label ' + issue.labels[i] + '">' + issue.labels[i] + '</span>';
                }
            }
            var pointsHtml = issue.estimate > 0 ? '<span style="color: #8B949E; font-size: 12px; margin-left: 8px;">' + issue.estimate + ' pts</span>' : '';
            var shortId = 'PLS-' + issue.id.substring(issue.id.lastIndexOf('_') + 1).slice(-4);
            return '<div class="issue" onclick="editIssue(\'' + issue.id + '\')">' +
                '<div class="issue-id">' + shortId + '</div>' +
                '<div style="display: flex; align-items: flex-start;">' +
                '<div class="priority ' + priorityClass + '"></div>' +
                '<div>' +
                '<div class="issue-title">' + issue.title + pointsHtml + '</div>' +
                '<div class="issue-labels">' + labelsHtml + '</div>' +
                '</div></div></div>';
        }

        function renderBoard() {
            var board = document.getElementById('board');
            board.innerHTML = '';
            for (var i = 0; i < columns.length; i++) {
                var col = columns[i];
                var colIssues = [];
                for (var j = 0; j < issues.length; j++) {
                    if (issues[j].status === col) colIssues.push(issues[j]);
                }
                var colDiv = document.createElement('div');
                colDiv.className = 'column';
                var issuesHtml = '';
                for (var k = 0; k < colIssues.length; k++) {
                    issuesHtml += renderIssue(colIssues[k]);
                }
                colDiv.innerHTML = '<div class="column-header">' +
                    '<div class="column-title">' +
                    '<span style="color:' + getColumnColor(col) + '">●</span>' +
                    col.replace(/_/g, ' ') +
                    '<span class="column-count">' + colIssues.length + '</span>' +
                    '</div></div>' + issuesHtml;
                board.appendChild(colDiv);
            }
        }

        function loadIssues() {
            var xhr = new XMLHttpRequest();
            xhr.open('GET', '/api/issues?workspace_id=' + workspaceID, true);
            xhr.onreadystatechange = function() {
                if (xhr.readyState === 4 && xhr.status === 200) {
                    issues = JSON.parse(xhr.responseText);
                    renderBoard();
                    updateMetrics();
                }
            };
            xhr.send();
        }

        function updateMetrics() {
            var counts = { backlog: 0, todo: 0, in_progress: 0, done: 0 };
            var velocity = 0;
            for (var i = 0; i < issues.length; i++) {
                var status = issues[i].status;
                if (counts.hasOwnProperty(status)) {
                    counts[status]++;
                }
                if (issues[i].status === 'done' && issues[i].estimate) {
                    velocity += issues[i].estimate;
                }
            }
            document.getElementById('totalIssues').textContent = issues.length;
            document.getElementById('inProgress').textContent = counts.in_progress;
            document.getElementById('completed').textContent = counts.done;
            document.getElementById('velocity').textContent = velocity;
        }

        function openCreateModal() {
            document.getElementById('createModal').classList.add('active');
            document.getElementById('issueTitle').focus();
        }

        function closeCreateModal() {
            document.getElementById('createModal').classList.remove('active');
            document.getElementById('issueTitle').value = '';
            document.getElementById('issueDescription').value = '';
            document.getElementById('issueEstimate').value = '0';
            document.getElementById('issueLabels').value = '';
        }

        function handleCreate(event) {
            event.preventDefault();
            var title = document.getElementById('issueTitle').value;
            var description = document.getElementById('issueDescription').value;
            var priority = parseInt(document.getElementById('issuePriority').value);
            var estimate = parseInt(document.getElementById('issueEstimate').value) || 0;
            var labelsStr = document.getElementById('issueLabels').value;
            var labels = labelsStr ? labelsStr.split(',').map(function(l) { return l.trim(); }).filter(function(l) { return l; }) : [];

            var xhr = new XMLHttpRequest();
            xhr.open('POST', '/api/issues', true);
            xhr.setRequestHeader('Content-Type', 'application/json');
            xhr.onreadystatechange = function() {
                if (xhr.readyState === 4) {
                    if (xhr.status === 200) {
                        closeCreateModal();
                        loadIssues();
                    } else {
                        alert('Failed to create issue');
                    }
                }
            };
            xhr.send(JSON.stringify({
                workspace_id: workspaceID,
                title: title,
                description: description,
                status: 'backlog',
                priority: priority,
                estimate: estimate,
                labels: labels
            }));
        }

        var currentDetailId = null;

        function openDetailModal(id) {
            var issue = null;
            for (var i = 0; i < issues.length; i++) {
                if (issues[i].id === id) { issue = issues[i]; break; }
            }
            if (!issue) return;

            currentDetailId = id;

            // Generate short ID (PLS-XXX format)
            var shortId = 'PLS-' + id.substring(id.lastIndexOf('_') + 1).slice(-4);

            document.getElementById('detailId').textContent = shortId;
            document.getElementById('detailTitle').textContent = issue.title;
            document.getElementById('detailStatus').textContent = issue.status.replace(/_/g, ' ');
            document.getElementById('detailStatus').className = 'status-badge status-' + issue.status;

            var priorityNames = ['', 'Urgent', 'High', 'Medium', 'Low'];
            document.getElementById('detailPriority').textContent = priorityNames[issue.priority] || 'Medium';

            document.getElementById('detailEstimate').textContent = (issue.estimate || 0) + ' pts';

            var created = new Date(issue.created_at);
            document.getElementElementById('detailCreated').textContent = created.toLocaleDateString();

            document.getElementById('detailDescription').textContent = issue.description || 'No description provided.';

            // Labels
            var labelsHtml = '';
            if (issue.labels && issue.labels.length > 0) {
                for (var j = 0; j < issue.labels.length; j++) {
                    labelsHtml += '<span class="label ' + issue.labels[j] + '">' + issue.labels[j] + '</span>';
                }
            } else {
                labelsHtml = '<span style="color: #6E7681;">No labels</span>';
            }
            document.getElementById('detailLabels').innerHTML = labelsHtml;

            // Status select
            document.getElementById('detailStatusSelect').value = issue.status;

            document.getElementById('detailModal').classList.add('active');
        }

        function closeDetailModal() {
            document.getElementById('detailModal').classList.remove('active');
            currentDetailId = null;
        }

        function updateDetailStatus() {
            var newStatus = document.getElementById('detailStatusSelect').value;
            if (!currentDetailId) return;

            var xhr = new XMLHttpRequest();
            xhr.open('PATCH', '/api/issues/' + currentDetailId, true);
            xhr.setRequestHeader('Content-Type', 'application/json');
            xhr.onreadystatechange = function() {
                if (xhr.readyState === 4) {
                    if (xhr.status === 200) {
                        closeDetailModal();
                        loadIssues();
                    } else {
                        alert('Failed to update issue');
                    }
                }
            };
            xhr.send(JSON.stringify({ status: newStatus }));
        }

        function deleteIssue() {
            if (!currentDetailId) return;
            if (!confirm('Are you sure you want to delete this issue?')) return;

            var xhr = new XMLHttpRequest();
            xhr.open('DELETE', '/api/issues/' + currentDetailId, true);
            xhr.onreadystatechange = function() {
                if (xhr.readyState === 4) {
                    if (xhr.status === 204 || xhr.status === 200) {
                        closeDetailModal();
                        loadIssues();
                    } else {
                        alert('Failed to delete issue');
                    }
                }
            };
            xhr.send();
        }

        function editIssue(id) {
            openDetailModal(id);
        }

        function handleSearch(query) {
            if (!query) {
                renderBoard();
                return;
            }
            var xhr = new XMLHttpRequest();
            xhr.open('GET', '/api/search?q=' + encodeURIComponent(query) + '&workspace_id=' + workspaceID, true);
            xhr.onreadystatechange = function() {
                if (xhr.readyState === 4 && xhr.status === 200) {
                    var results = JSON.parse(xhr.responseText);
                    var board = document.getElementById('board');
                    if (results.length === 0) {
                        board.innerHTML = '<p style="color: #8B949E; padding: 24px;">No issues found matching "' + query + '"</p>';
                        return;
                    }
                    // Render results as a filtered board
                    board.innerHTML = '';
                    var columnsHtml = '';
                    for (var i = 0; i < columns.length; i++) {
                        var col = columns[i];
                        var colResults = [];
                        for (var j = 0; j < results.length; j++) {
                            if (results[j].status === col) colResults.push(results[j]);
                        }
                        columnsHtml += '<div class="column">' +
                            '<div class="column-header">' +
                            '<div class="column-title">' +
                            '<span style="color:' + getColumnColor(col) + '">●</span>' +
                            col.replace(/_/g, ' ') +
                            '<span class="column-count">' + colResults.length + '</span>' +
                            '</div></div>';
                        for (var k = 0; k < colResults.length; k++) {
                            var result = colResults[k];
                            var labelsHtml = '';
                            if (result.labels) {
                                for (var l = 0; l < result.labels.length; l++) {
                                    labelsHtml += '<span class="label ' + result.labels[l] + '">' + result.labels[l] + '</span>';
                                }
                            }
                            var pointsHtml = result.estimate > 0 ? '<span style="color: #8B949E; font-size: 12px; margin-left: 8px;">' + result.estimate + ' pts</span>' : '';
                            columnsHtml += '<div class="issue" onclick="editIssue(\'' + result.id + '\')">' +
                                '<div class="issue-title">' + result.title + pointsHtml + '</div>' +
                                '<div class="issue-labels">' + labelsHtml + '</div>' +
                                '</div>';
                        }
                        columnsHtml += '</div>';
                    }
                    board.innerHTML = columnsHtml;
                }
            };
            xhr.send();
        }

        function showBoard() {
            currentView = 'board';
            document.getElementById('board').style.display = 'flex';
            document.getElementById('metricsBar').style.display = 'none';
            document.getElementById('metricsView').style.display = 'none';
            document.getElementById('pageTitle').textContent = 'Project Board';
            document.querySelectorAll('.nav-item').forEach(function(el, i) {
                el.classList.toggle('active', i === 0);
            });
            renderBoard();
        }

        function showMetrics() {
            currentView = 'board';
            document.getElementById('board').style.display = 'none';
            document.getElementById('metricsBar').style.display = 'flex';
            document.getElementById('metricsView').style.display = 'block';
            document.getElementById('pageTitle').textContent = 'Analytics';
            document.querySelectorAll('.nav-item').forEach(function(el, i) {
                el.classList.toggle('active', i === 1);
            });

            var xhr = new XMLHttpRequest();
            xhr.open('GET', '/api/metrics?workspace_id=' + workspaceID, true);
            xhr.onreadystatechange = function() {
                if (xhr.readyState === 4 && xhr.status === 200) {
                    var metrics = JSON.parse(xhr.responseText);
                    var html = '<h2 style="margin-bottom: 16px;">Velocity Metrics</h2>';
                    html += '<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px;">';
                    html += '<div style="background: #161B22; padding: 16px; border-radius: 8px;"><h3 style="color: #8B949E; font-size: 12px;">Completion Rate</h3><p style="font-size: 32px; font-weight: 600;">' + metrics.completion_rate.toFixed(1) + '%</p></div>';
                    html += '<div style="background: #161B22; padding: 16px; border-radius: 8px;"><h3 style="color: #8B949E; font-size: 12px;">Total Points</h3><p style="font-size: 32px; font-weight: 600;">' + metrics.total_points + '</p></div>';
                    html += '<div style="background: #161B22; padding: 16px; border-radius: 8px;"><h3 style="color: #8B949E; font-size: 12px;">Completed Points</h3><p style="font-size: 32px; font-weight: 600;">' + metrics.completed_points + '</p></div>';
                    html += '<div style="background: #161B22; padding: 16px; border-radius: 8px;"><h3 style="color: #8B949E; font-size: 12px;">Bugs</h3><p style="font-size: 32px; font-weight: 600; color: #F85149;">' + metrics.bug_count + '</p></div>';
                    html += '</div>';
                    document.getElementById('metricsView').innerHTML = html;
                }
            };
            xhr.send();
        }

        function showCycles() {
            document.getElementById('board').style.display = 'none';
            document.getElementById('metricsBar').style.display = 'none';
            document.getElementById('metricsView').style.display = 'block';
            document.getElementById('pageTitle').textContent = 'Cycles';
            document.querySelectorAll('.nav-item').forEach(function(el, i) {
                el.classList.toggle('active', i === 2);
            });
            document.getElementById('metricsView').innerHTML = '<h2 style="margin-bottom: 16px;">Cycles</h2><p style="color: #8B949E;">Cycle management coming soon...</p>';
        }

        // Keyboard shortcuts
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') {
                closeCreateModal();
            }
            if (e.key === 'c' && !e.ctrlKey && !e.metaKey && document.activeElement.tagName !== 'INPUT' && document.activeElement.tagName !== 'TEXTAREA') {
                e.preventDefault();
                openCreateModal();
            }
        });

        loadIssues();
    </script>
</body>
</html>`
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	s.server = &http.Server{
		Addr:    s.addr,
		Handler: s.mux,
	}

	go func() {
		fmt.Printf("Pulse server starting on %s\n", s.addr)
		fmt.Printf("Database: %s\n", s.db.Path())
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	<-ctx.Done()
	return s.server.Shutdown(ctx)
}

// Close closes the server and database connection
func (s *Server) Close() error {
	return s.db.Close()
}

// Helper functions
func defaultColumns() []Column {
	return []Column{
		{ID: "backlog", Name: "Backlog", Order: 0, Color: "#6B7280"},
		{ID: "todo", Name: "To Do", Order: 1, Color: "#F59E0B"},
		{ID: "in_progress", Name: "In Progress", Order: 2, Color: "#3B82F6"},
		{ID: "done", Name: "Done", Order: 3, Color: "#10B981"},
	}
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Column represents a workflow column
type Column struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order int    `json:"order"`
	Color string `json:"color"`
}
