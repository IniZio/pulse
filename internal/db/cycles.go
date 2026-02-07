package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Cycle represents a sprint/cycle.
type Cycle struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	Name        string     `json:"name"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
	Status      string     `json:"status"` // upcoming, active, completed
	CreatedAt   time.Time  `json:"created_at"`
}

// CycleRepository handles cycle database operations.
type CycleRepository struct {
	db *DB
}

// NewCycleRepository creates a new cycle repository.
func NewCycleRepository(db *DB) *CycleRepository {
	return &CycleRepository{db: db}
}

// Create inserts a new cycle.
func (r *CycleRepository) Create(cycle *Cycle) error {
	now := time.Now()
	cycle.CreatedAt = now

	query := `
		INSERT INTO cycles (id, workspace_id, name, start_date, end_date, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		cycle.ID,
		cycle.WorkspaceID,
		cycle.Name,
		cycle.StartDate,
		cycle.EndDate,
		cycle.Status,
		cycle.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create cycle: %w", err)
	}

	return nil
}

// GetByID retrieves a cycle by ID.
func (r *CycleRepository) GetByID(id string) (*Cycle, error) {
	query := `SELECT * FROM cycles WHERE id = ?`

	var cycle Cycle
	var startDate, endDate sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&cycle.ID,
		&cycle.WorkspaceID,
		&cycle.Name,
		&startDate,
		&endDate,
		&cycle.Status,
		&cycle.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cycle: %w", err)
	}

	if startDate.Valid {
		cycle.StartDate = &startDate.Time
	}
	if endDate.Valid {
		cycle.EndDate = &endDate.Time
	}

	return &cycle, nil
}

// List retrieves all cycles for a workspace.
func (r *CycleRepository) List(workspaceID string) ([]*Cycle, error) {
	query := `SELECT * FROM cycles WHERE workspace_id = ? ORDER BY created_at DESC`

	rows, err := r.db.Query(query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cycles: %w", err)
	}
	defer rows.Close()

	var cycles []*Cycle
	for rows.Next() {
		var cycle Cycle
		var startDate, endDate sql.NullTime

		err := rows.Scan(
			&cycle.ID,
			&cycle.WorkspaceID,
			&cycle.Name,
			&startDate,
			&endDate,
			&cycle.Status,
			&cycle.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cycle: %w", err)
		}

		if startDate.Valid {
			cycle.StartDate = &startDate.Time
		}
		if endDate.Valid {
			cycle.EndDate = &endDate.Time
		}

		cycles = append(cycles, &cycle)
	}

	return cycles, nil
}

// Update updates an existing cycle.
func (r *CycleRepository) Update(cycle *Cycle) error {
	query := `
		UPDATE cycles SET
			name = ?,
			start_date = ?,
			end_date = ?,
			status = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query,
		cycle.Name,
		cycle.StartDate,
		cycle.EndDate,
		cycle.Status,
		cycle.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update cycle: %w", err)
	}

	return nil
}

// Delete removes a cycle by ID.
func (r *CycleRepository) Delete(id string) error {
	query := `DELETE FROM cycles WHERE id = ?`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete cycle: %w", err)
	}

	return nil
}

// GetActive retrieves the active cycle for a workspace.
func (r *CycleRepository) GetActive(workspaceID string) (*Cycle, error) {
	query := `SELECT * FROM cycles WHERE workspace_id = ? AND status = 'active' LIMIT 1`

	var cycle Cycle
	var startDate, endDate sql.NullTime

	err := r.db.QueryRow(query, workspaceID).Scan(
		&cycle.ID,
		&cycle.WorkspaceID,
		&cycle.Name,
		&startDate,
		&endDate,
		&cycle.Status,
		&cycle.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active cycle: %w", err)
	}

	if startDate.Valid {
		cycle.StartDate = &startDate.Time
	}
	if endDate.Valid {
		cycle.EndDate = &endDate.Time
	}

	return &cycle, nil
}

// GetUpcoming retrieves upcoming cycles for a workspace.
func (r *CycleRepository) GetUpcoming(workspaceID string) ([]*Cycle, error) {
	query := `SELECT * FROM cycles WHERE workspace_id = ? AND status = 'upcoming' ORDER BY created_at ASC`

	rows, err := r.db.Query(query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming cycles: %w", err)
	}
	defer rows.Close()

	var cycles []*Cycle
	for rows.Next() {
		var cycle Cycle
		var startDate, endDate sql.NullTime

		err := rows.Scan(
			&cycle.ID,
			&cycle.WorkspaceID,
			&cycle.Name,
			&startDate,
			&endDate,
			&cycle.Status,
			&cycle.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cycle: %w", err)
		}

		if startDate.Valid {
			cycle.StartDate = &startDate.Time
		}
		if endDate.Valid {
			cycle.EndDate = &endDate.Time
		}

		cycles = append(cycles, &cycle)
	}

	return cycles, nil
}
