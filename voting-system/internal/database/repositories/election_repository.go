package repositories

import (
	"database/sql"
	"voting-system/internal/database"
)

type ElectionRepository struct {
	db *sql.DB
}

func NewElectionRepository(db *sql.DB) *ElectionRepository {
	return &ElectionRepository{db: db}
}

// CreateElection creates a new election record
func (r *ElectionRepository) CreateElection(election *database.Election) error {
	query := `
        INSERT INTO elections (blockchain_id, name, description, start_time, end_time)
        VALUES (?, ?, ?, ?, ?)
    `
	result, err := r.db.Exec(query, election.BlockchainID, election.Name, election.Description,
		election.StartTime, election.EndTime)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	election.ID = id
	return nil
}

// GetActiveElection retrieves the currently active election
func (r *ElectionRepository) GetActiveElection() (*database.Election, error) {
	query := `
        SELECT id, blockchain_id, name, description, start_time, end_time, is_active, created_at
        FROM elections
        WHERE is_active = true
        LIMIT 1
    `

	var election database.Election
	err := r.db.QueryRow(query).Scan(
		&election.ID, &election.BlockchainID, &election.Name, &election.Description,
		&election.StartTime, &election.EndTime, &election.IsActive, &election.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &election, nil
}

// GetElectionByID retrieves an election by ID
func (r *ElectionRepository) GetElectionByID(electionID int64) (*database.Election, error) {
	query := `
        SELECT id, blockchain_id, name, description, start_time, end_time, is_active, created_at
        FROM elections
        WHERE id = ?
    `

	var election database.Election
	err := r.db.QueryRow(query, electionID).Scan(
		&election.ID, &election.BlockchainID, &election.Name, &election.Description,
		&election.StartTime, &election.EndTime, &election.IsActive, &election.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &election, nil
}

// GetElectionByBlockchainID retrieves an election by blockchain ID
func (r *ElectionRepository) GetElectionByBlockchainID(blockchainID string) (*database.Election, error) {
	query := `
        SELECT id, blockchain_id, name, description, start_time, end_time, is_active, created_at
        FROM elections
        WHERE blockchain_id = ?
    `

	var election database.Election
	err := r.db.QueryRow(query, blockchainID).Scan(
		&election.ID, &election.BlockchainID, &election.Name, &election.Description,
		&election.StartTime, &election.EndTime, &election.IsActive, &election.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &election, nil
}

// ListElections retrieves all elections with pagination
func (r *ElectionRepository) ListElections(limit, offset int) ([]database.Election, error) {
	query := `
        SELECT id, blockchain_id, name, description, start_time, end_time, is_active, created_at
        FROM elections
        ORDER BY created_at DESC
        LIMIT ? OFFSET ?
    `

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var elections []database.Election
	for rows.Next() {
		var election database.Election
		err := rows.Scan(
			&election.ID, &election.BlockchainID, &election.Name, &election.Description,
			&election.StartTime, &election.EndTime, &election.IsActive, &election.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		elections = append(elections, election)
	}

	return elections, nil
}

// UpdateElectionStatus updates the active status of an election
func (r *ElectionRepository) UpdateElectionStatus(electionID int64, isActive bool) error {
	query := `UPDATE elections SET is_active = ? WHERE id = ?`
	_, err := r.db.Exec(query, isActive, electionID)
	return err
}

// DeleteElectionCascade deletes an election and related records
func (r *ElectionRepository) DeleteElectionCascade(electionID int64) error {
	// Delete votes for this election
	if _, err := r.db.Exec(`DELETE FROM votes WHERE election_id = ?`, electionID); err != nil {
		return err
	}
	// Delete candidates for this election
	if _, err := r.db.Exec(`DELETE FROM candidates WHERE election_id = ?`, electionID); err != nil {
		return err
	}
	// Finally delete the election itself
	if _, err := r.db.Exec(`DELETE FROM elections WHERE id = ?`, electionID); err != nil {
		return err
	}
	return nil
}

// GetElectionStatistics gets voting statistics for an election
func (r *ElectionRepository) GetElectionStatistics(electionID int64) (map[string]interface{}, error) {
	// Get total votes
	var totalVotes int
	err := r.db.QueryRow("SELECT COUNT(*) FROM votes WHERE election_id = ?", electionID).Scan(&totalVotes)
	if err != nil {
		return nil, err
	}

	// Get votes by polling unit
	pollingUnitQuery := `
        SELECT polling_unit_id, COUNT(*) as vote_count
        FROM votes 
        WHERE election_id = ? 
        GROUP BY polling_unit_id
    `
	rows, err := r.db.Query(pollingUnitQuery, electionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pollingUnitStats := make(map[string]int)
	for rows.Next() {
		var pollingUnitID string
		var voteCount int
		err := rows.Scan(&pollingUnitID, &voteCount)
		if err != nil {
			return nil, err
		}
		pollingUnitStats[pollingUnitID] = voteCount
	}

	// Get votes by candidate
	candidateQuery := `
        SELECT candidate_id, COUNT(*) as vote_count
        FROM votes 
        WHERE election_id = ? 
        GROUP BY candidate_id
    `
	rows, err = r.db.Query(candidateQuery, electionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidateStats := make(map[string]int)
	for rows.Next() {
		var candidateID string
		var voteCount int
		err := rows.Scan(&candidateID, &voteCount)
		if err != nil {
			return nil, err
		}
		candidateStats[candidateID] = voteCount
	}

	return map[string]interface{}{
		"total_votes":        totalVotes,
		"polling_unit_stats": pollingUnitStats,
		"candidate_stats":    candidateStats,
	}, nil
}
