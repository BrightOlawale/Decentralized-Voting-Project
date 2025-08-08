package repositories

import (
	"database/sql"
	"time"
	"voting-system/internal/database"
)

type TerminalRepository struct {
	db *sql.DB
}

func NewTerminalRepository(db *sql.DB) *TerminalRepository {
	return &TerminalRepository{db: db}
}

// RegisterTerminal registers a new terminal
func (r *TerminalRepository) RegisterTerminal(terminal *database.Terminal) error {
	query := `
        INSERT INTO terminals (id, name, location, polling_unit_id, eth_address, public_key, status)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `
	_, err := r.db.Exec(query, terminal.ID, terminal.Name, terminal.Location,
		terminal.PollingUnitID, terminal.EthAddress, terminal.PublicKey, terminal.Status)
	return err
}

// UpdateTerminalHeartbeat updates the last heartbeat timestamp for a terminal
func (r *TerminalRepository) UpdateTerminalHeartbeat(terminalID string) error {
	query := `
        UPDATE terminals 
        SET last_heartbeat = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
	_, err := r.db.Exec(query, terminalID)
	return err
}

// GetTerminal retrieves terminal information by ID
func (r *TerminalRepository) GetTerminal(terminalID string) (*database.Terminal, error) {
	query := `
        SELECT id, name, location, polling_unit_id, eth_address, public_key, 
               status, authorized, last_heartbeat, created_at, updated_at
        FROM terminals
        WHERE id = ?
    `

	var terminal database.Terminal
	err := r.db.QueryRow(query, terminalID).Scan(
		&terminal.ID, &terminal.Name, &terminal.Location, &terminal.PollingUnitID,
		&terminal.EthAddress, &terminal.PublicKey, &terminal.Status, &terminal.Authorized,
		&terminal.LastHeartbeat, &terminal.CreatedAt, &terminal.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &terminal, nil
}

// ListTerminals retrieves all terminals with optional filtering
func (r *TerminalRepository) ListTerminals(status, pollingUnitID string, limit, offset int) ([]database.Terminal, error) {
	query := `
        SELECT id, name, location, polling_unit_id, eth_address, public_key, 
               status, authorized, last_heartbeat, created_at, updated_at
        FROM terminals
        WHERE 1=1
    `
	args := []interface{}{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if pollingUnitID != "" {
		query += " AND polling_unit_id = ?"
		args = append(args, pollingUnitID)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terminals []database.Terminal
	for rows.Next() {
		var terminal database.Terminal
		err := rows.Scan(
			&terminal.ID, &terminal.Name, &terminal.Location, &terminal.PollingUnitID,
			&terminal.EthAddress, &terminal.PublicKey, &terminal.Status, &terminal.Authorized,
			&terminal.LastHeartbeat, &terminal.CreatedAt, &terminal.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		terminals = append(terminals, terminal)
	}

	return terminals, nil
}

// UpdateTerminalStatus updates the status of a terminal
func (r *TerminalRepository) UpdateTerminalStatus(terminalID, status string) error {
	query := `UPDATE terminals SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, status, terminalID)
	return err
}

// AuthorizeTerminal authorizes a terminal for voting
func (r *TerminalRepository) AuthorizeTerminal(terminalID string) error {
	query := `UPDATE terminals SET authorized = true, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, terminalID)
	return err
}

// DeauthorizeTerminal deauthorizes a terminal
func (r *TerminalRepository) DeauthorizeTerminal(terminalID string) error {
	query := `UPDATE terminals SET authorized = false, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, terminalID)
	return err
}

// GetTerminalsByPollingUnit gets all terminals for a specific polling unit
func (r *TerminalRepository) GetTerminalsByPollingUnit(pollingUnitID string) ([]database.Terminal, error) {
	query := `
        SELECT id, name, location, polling_unit_id, eth_address, public_key, 
               status, authorized, last_heartbeat, created_at, updated_at
        FROM terminals
        WHERE polling_unit_id = ?
        ORDER BY created_at DESC
    `

	rows, err := r.db.Query(query, pollingUnitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terminals []database.Terminal
	for rows.Next() {
		var terminal database.Terminal
		err := rows.Scan(
			&terminal.ID, &terminal.Name, &terminal.Location, &terminal.PollingUnitID,
			&terminal.EthAddress, &terminal.PublicKey, &terminal.Status, &terminal.Authorized,
			&terminal.LastHeartbeat, &terminal.CreatedAt, &terminal.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		terminals = append(terminals, terminal)
	}

	return terminals, nil
}

// GetOfflineTerminals gets terminals that haven't sent a heartbeat recently
func (r *TerminalRepository) GetOfflineTerminals(timeoutMinutes int) ([]database.Terminal, error) {
	query := `
        SELECT id, name, location, polling_unit_id, eth_address, public_key, 
               status, authorized, last_heartbeat, created_at, updated_at
        FROM terminals
        WHERE last_heartbeat < ? OR last_heartbeat IS NULL
        ORDER BY last_heartbeat ASC
    `

	timeout := time.Now().Add(-time.Duration(timeoutMinutes) * time.Minute)
	rows, err := r.db.Query(query, timeout)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terminals []database.Terminal
	for rows.Next() {
		var terminal database.Terminal
		err := rows.Scan(
			&terminal.ID, &terminal.Name, &terminal.Location, &terminal.PollingUnitID,
			&terminal.EthAddress, &terminal.PublicKey, &terminal.Status, &terminal.Authorized,
			&terminal.LastHeartbeat, &terminal.CreatedAt, &terminal.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		terminals = append(terminals, terminal)
	}

	return terminals, nil
}
