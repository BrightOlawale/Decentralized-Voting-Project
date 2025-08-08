package repositories

import (
	"database/sql"
	"time"
	"voting-system/internal/database"
)

type AuditLogRepository struct {
	db *sql.DB
}

func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// InsertAuditLog inserts a new audit log entry
func (r *AuditLogRepository) InsertAuditLog(log *database.AuditLog) error {
	query := `
        INSERT INTO audit_logs (action, user_id, polling_unit_id, details, ip_address)
        VALUES (?, ?, ?, ?, ?)
    `
	result, err := r.db.Exec(query, log.Action, log.UserID, log.PollingUnitID, log.Details, log.IPAddress)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	log.ID = id
	return nil
}

// GetAuditLogs retrieves audit logs with pagination and filtering
func (r *AuditLogRepository) GetAuditLogs(limit, offset int, action, pollingUnitID string, startTime, endTime *time.Time) ([]database.AuditLog, error) {
	query := `
        SELECT id, action, user_id, polling_unit_id, details, ip_address, created_at
        FROM audit_logs
        WHERE 1=1
    `
	args := []interface{}{}

	if action != "" {
		query += " AND action = ?"
		args = append(args, action)
	}

	if pollingUnitID != "" {
		query += " AND polling_unit_id = ?"
		args = append(args, pollingUnitID)
	}

	if startTime != nil {
		query += " AND created_at >= ?"
		args = append(args, startTime)
	}

	if endTime != nil {
		query += " AND created_at <= ?"
		args = append(args, endTime)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []database.AuditLog
	for rows.Next() {
		var log database.AuditLog
		err := rows.Scan(&log.ID, &log.Action, &log.UserID, &log.PollingUnitID,
			&log.Details, &log.IPAddress, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetAuditLogsByAction gets audit logs filtered by action
func (r *AuditLogRepository) GetAuditLogsByAction(action string, limit, offset int) ([]database.AuditLog, error) {
	query := `
        SELECT id, action, user_id, polling_unit_id, details, ip_address, created_at
        FROM audit_logs
        WHERE action = ?
        ORDER BY created_at DESC
        LIMIT ? OFFSET ?
    `

	rows, err := r.db.Query(query, action, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []database.AuditLog
	for rows.Next() {
		var log database.AuditLog
		err := rows.Scan(&log.ID, &log.Action, &log.UserID, &log.PollingUnitID,
			&log.Details, &log.IPAddress, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetAuditLogsByPollingUnit gets audit logs for a specific polling unit
func (r *AuditLogRepository) GetAuditLogsByPollingUnit(pollingUnitID string, limit, offset int) ([]database.AuditLog, error) {
	query := `
        SELECT id, action, user_id, polling_unit_id, details, ip_address, created_at
        FROM audit_logs
        WHERE polling_unit_id = ?
        ORDER BY created_at DESC
        LIMIT ? OFFSET ?
    `

	rows, err := r.db.Query(query, pollingUnitID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []database.AuditLog
	for rows.Next() {
		var log database.AuditLog
		err := rows.Scan(&log.ID, &log.Action, &log.UserID, &log.PollingUnitID,
			&log.Details, &log.IPAddress, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetAuditStatistics gets audit statistics
func (r *AuditLogRepository) GetAuditStatistics(startTime, endTime *time.Time) (map[string]interface{}, error) {
	// Get total audit logs
	var totalLogs int
	query := "SELECT COUNT(*) FROM audit_logs"
	args := []interface{}{}

	if startTime != nil && endTime != nil {
		query += " WHERE created_at BETWEEN ? AND ?"
		args = append(args, startTime, endTime)
	}

	err := r.db.QueryRow(query, args...).Scan(&totalLogs)
	if err != nil {
		return nil, err
	}

	// Get logs by action
	actionQuery := `
        SELECT action, COUNT(*) as count
        FROM audit_logs
    `
	if startTime != nil && endTime != nil {
		actionQuery += " WHERE created_at BETWEEN ? AND ?"
	}
	actionQuery += " GROUP BY action ORDER BY count DESC"

	rows, err := r.db.Query(actionQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type ActionCount struct {
		Action string `json:"action"`
		Count  int    `json:"count"`
	}

	var actionCounts []ActionCount
	for rows.Next() {
		var action string
		var count int
		err := rows.Scan(&action, &count)
		if err != nil {
			return nil, err
		}
		actionCounts = append(actionCounts, ActionCount{
			Action: action,
			Count:  count,
		})
	}

	// Get logs by polling unit
	pollingUnitQuery := `
        SELECT polling_unit_id, COUNT(*) as count
        FROM audit_logs
    `
	if startTime != nil && endTime != nil {
		pollingUnitQuery += " WHERE created_at BETWEEN ? AND ?"
	}
	pollingUnitQuery += " GROUP BY polling_unit_id ORDER BY count DESC"

	rows, err = r.db.Query(pollingUnitQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type PollingUnitCount struct {
		PollingUnitID string `json:"polling_unit_id"`
		Count         int    `json:"count"`
	}

	var pollingUnitCounts []PollingUnitCount
	for rows.Next() {
		var pollingUnitID string
		var count int
		err := rows.Scan(&pollingUnitID, &count)
		if err != nil {
			return nil, err
		}
		pollingUnitCounts = append(pollingUnitCounts, PollingUnitCount{
			PollingUnitID: pollingUnitID,
			Count:         count,
		})
	}

	return map[string]interface{}{
		"total_logs":          totalLogs,
		"action_counts":       actionCounts,
		"polling_unit_counts": pollingUnitCounts,
	}, nil
}

// GetRecentAuditLogs gets the most recent audit logs
func (r *AuditLogRepository) GetRecentAuditLogs(limit int) ([]database.AuditLog, error) {
	query := `
        SELECT id, action, user_id, polling_unit_id, details, ip_address, created_at
        FROM audit_logs
        ORDER BY created_at DESC
        LIMIT ?
    `

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []database.AuditLog
	for rows.Next() {
		var log database.AuditLog
		err := rows.Scan(&log.ID, &log.Action, &log.UserID, &log.PollingUnitID,
			&log.Details, &log.IPAddress, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}
