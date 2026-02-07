package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Workspace represents a project workspace.
type Workspace struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Settings    string `json:"settings"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkspaceRepository handles workspace database operations.
type WorkspaceRepository struct {
	db *DB
}

// NewWorkspaceRepository creates a new workspace repository.
func NewWorkspaceRepository(db *DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

// Create inserts a new workspace.
func (r *WorkspaceRepository) Create(ws *Workspace) error {
	now := time.Now()
	ws.CreatedAt = now
	ws.UpdatedAt = now

	query := `
		INSERT INTO workspaces (id, name, description, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		ws.ID,
		ws.Name,
		ws.Description,
		ws.Settings,
		ws.CreatedAt,
		ws.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	return nil
}

// GetByID retrieves a workspace by ID.
func (r *WorkspaceRepository) GetByID(id string) (*Workspace, error) {
	query := `SELECT * FROM workspaces WHERE id = ?`

	var ws Workspace
	err := r.db.QueryRow(query, id).Scan(
		&ws.ID,
		&ws.Name,
		&ws.Description,
		&ws.Settings,
		&ws.CreatedAt,
		&ws.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	return &ws, nil
}

// List retrieves all workspaces.
func (r *WorkspaceRepository) List() ([]*Workspace, error) {
	query := `SELECT * FROM workspaces ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []*Workspace
	for rows.Next() {
		var ws Workspace
		if err := rows.Scan(
			&ws.ID,
			&ws.Name,
			&ws.Description,
			&ws.Settings,
			&ws.CreatedAt,
			&ws.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan workspace: %w", err)
		}
		workspaces = append(workspaces, &ws)
	}

	return workspaces, nil
}

// Update updates an existing workspace.
func (r *WorkspaceRepository) Update(ws *Workspace) error {
	ws.UpdatedAt = time.Now()

	query := `
		UPDATE workspaces SET
			name = ?,
			description = ?,
			settings = ?,
			updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query,
		ws.Name,
		ws.Description,
		ws.Settings,
		ws.UpdatedAt,
		ws.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	return nil
}

// Delete removes a workspace by ID.
func (r *WorkspaceRepository) Delete(id string) error {
	query := `DELETE FROM workspaces WHERE id = ?`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	return nil
}
