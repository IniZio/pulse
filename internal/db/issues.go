// Package db provides database operations for Pulse entities.
package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Issue represents a single issue/task.
type Issue struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    int        `json:"priority"`
	AssigneeID  string     `json:"assignee_id"`
	Estimate    int        `json:"estimate"`
	CycleID     string     `json:"cycle_id"`
	Labels      []string   `json:"labels"`
	ParentID    string     `json:"parent_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// IssueRepository handles issue database operations.
type IssueRepository struct {
	db *DB
}

// NewIssueRepository creates a new issue repository.
func NewIssueRepository(db *DB) *IssueRepository {
	return &IssueRepository{db: db}
}

// Create inserts a new issue.
func (r *IssueRepository) Create(issue *Issue) error {
	now := time.Now()
	issue.CreatedAt = now
	issue.UpdatedAt = now

	labelsJSON, _ := json.Marshal(issue.Labels)

	query := `
		INSERT INTO issues (id, workspace_id, title, description, status, priority, assignee_id, estimate, cycle_id, labels, parent_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		issue.ID,
		issue.WorkspaceID,
		issue.Title,
		issue.Description,
		issue.Status,
		issue.Priority,
		issue.AssigneeID,
		issue.Estimate,
		issue.CycleID,
		string(labelsJSON),
		issue.ParentID,
		issue.CreatedAt,
		issue.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	return nil
}

// GetByID retrieves an issue by ID.
func (r *IssueRepository) GetByID(id string) (*Issue, error) {
	query := `SELECT * FROM issues WHERE id = ?`

	var issue Issue
	var labelsJSON string

	err := r.db.QueryRow(query, id).Scan(
		&issue.ID,
		&issue.WorkspaceID,
		&issue.Title,
		&issue.Description,
		&issue.Status,
		&issue.Priority,
		&issue.AssigneeID,
		&issue.Estimate,
		&issue.CycleID,
		&labelsJSON,
		&issue.ParentID,
		&issue.CreatedAt,
		&issue.UpdatedAt,
		&issue.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	json.Unmarshal([]byte(labelsJSON), &issue.Labels)

	return &issue, nil
}

// List retrieves issues with optional filters.
func (r *IssueRepository) List(workspaceID, status string, limit, offset int) ([]*Issue, error) {
	query := `SELECT * FROM issues WHERE workspace_id = ?`
	args := []interface{}{workspaceID}

	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}

	query += ` ORDER BY priority ASC, created_at DESC`

	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	if offset > 0 {
		query += ` OFFSET ?`
		args = append(args, offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}
	defer rows.Close()

	var issues []*Issue
	for rows.Next() {
		var issue Issue
		var labelsJSON string

		err := rows.Scan(
			&issue.ID,
			&issue.WorkspaceID,
			&issue.Title,
			&issue.Description,
			&issue.Status,
			&issue.Priority,
			&issue.AssigneeID,
			&issue.Estimate,
			&issue.CycleID,
			&labelsJSON,
			&issue.ParentID,
			&issue.CreatedAt,
			&issue.UpdatedAt,
			&issue.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan issue: %w", err)
		}

		json.Unmarshal([]byte(labelsJSON), &issue.Labels)
		issues = append(issues, &issue)
	}

	return issues, nil
}

// Update updates an existing issue.
func (r *IssueRepository) Update(issue *Issue) error {
	issue.UpdatedAt = time.Now()

	labelsJSON, _ := json.Marshal(issue.Labels)

	query := `
		UPDATE issues SET
			title = ?,
			description = ?,
			status = ?,
			priority = ?,
			assignee_id = ?,
			estimate = ?,
			cycle_id = ?,
			labels = ?,
			parent_id = ?,
			updated_at = ?,
			completed_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query,
		issue.Title,
		issue.Description,
		issue.Status,
		issue.Priority,
		issue.AssigneeID,
		issue.Estimate,
		issue.CycleID,
		string(labelsJSON),
		issue.ParentID,
		issue.UpdatedAt,
		issue.CompletedAt,
		issue.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	return nil
}

// UpdateStatus updates only the status of an issue.
func (r *IssueRepository) UpdateStatus(id, status string) error {
	now := time.Now()

	var completedAt *time.Time
	if status == "done" {
		completedAt = &now
	}

	query := `
		UPDATE issues SET status = ?, updated_at = ?, completed_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query, status, now, completedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update issue status: %w", err)
	}

	return nil
}

// Delete removes an issue by ID.
func (r *IssueRepository) Delete(id string) error {
	query := `DELETE FROM issues WHERE id = ?`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete issue: %w", err)
	}

	return nil
}

// CountByStatus counts issues by status for a workspace.
func (r *IssueRepository) CountByStatus(workspaceID string) (map[string]int, error) {
	query := `SELECT status, COUNT(*) FROM issues WHERE workspace_id = ? GROUP BY status`

	rows, err := r.db.Query(query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to count issues: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result[status] = count
	}

	return result, nil
}

// CountByCycle counts issues by cycle for a workspace.
func (r *IssueRepository) CountByCycle(workspaceID, cycleID string) (total, completed int, err error) {
	query := `
		SELECT COUNT(*), SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END)
		FROM issues WHERE workspace_id = ? AND cycle_id = ?
	`

	err = r.db.QueryRow(query, workspaceID, cycleID).Scan(&total, &completed)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count cycle issues: %w", err)
	}

	return total, completed, nil
}
