package repositories

import (
	"database/sql"
	"time"
	"voting-system/internal/database"
)

type VoteRepository struct {
	db *sql.DB
}

func NewVoteRepository(db *sql.DB) *VoteRepository {
	return &VoteRepository{db: db}
}

func (r *VoteRepository) InsertVote(vote *database.Vote) error {
	query := `
        INSERT INTO votes (verification_hash, election_id, polling_unit_id, candidate_id, 
                          encrypted_vote, status)
        VALUES (?, ?, ?, ?, ?, ?)
    `
	result, err := r.db.Exec(query, vote.VerificationHash, vote.ElectionID, vote.PollingUnitID,
		vote.CandidateID, vote.EncryptedVote, vote.Status)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	vote.ID = id
	return nil
}

func (r *VoteRepository) UpdateVoteSync(verificationHash, transactionHash string, blockNumber int64) error {
	query := `
        UPDATE votes 
        SET transaction_hash = ?, block_number = ?, status = 'synced', synced_at = CURRENT_TIMESTAMP
        WHERE verification_hash = ?
    `
	_, err := r.db.Exec(query, transactionHash, blockNumber, verificationHash)
	return err
}

func (r *VoteRepository) GetPendingVotes() ([]database.Vote, error) {
	query := `
        SELECT id, verification_hash, election_id, polling_unit_id, candidate_id, 
               encrypted_vote, status, created_at
        FROM votes
        WHERE status = 'pending'
        ORDER BY created_at ASC
    `

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []database.Vote
	for rows.Next() {
		var vote database.Vote
		err := rows.Scan(&vote.ID, &vote.VerificationHash, &vote.ElectionID,
			&vote.PollingUnitID, &vote.CandidateID, &vote.EncryptedVote,
			&vote.Status, &vote.CreatedAt)
		if err != nil {
			return nil, err
		}
		votes = append(votes, vote)
	}

	return votes, nil
}

func (r *VoteRepository) GetByVerificationHash(hash string) (*database.Vote, error) {
	query := `
        SELECT id, blockchain_vote_id, verification_hash, election_id, polling_unit_id, 
               candidate_id, encrypted_vote, transaction_hash, block_number, status, 
               created_at, synced_at
        FROM votes
        WHERE verification_hash = ?
    `

	var vote database.Vote
	err := r.db.QueryRow(query, hash).Scan(
		&vote.ID, &vote.BlockchainVoteID, &vote.VerificationHash, &vote.ElectionID,
		&vote.PollingUnitID, &vote.CandidateID, &vote.EncryptedVote,
		&vote.TransactionHash, &vote.BlockNumber, &vote.Status,
		&vote.CreatedAt, &vote.SyncedAt,
	)

	if err != nil {
		return nil, err
	}

	return &vote, nil
}

// GetElectionResults gets the complete results for an election
func (r *VoteRepository) GetElectionResults(electionID int64) (map[string]interface{}, error) {
	// Get total votes cast
	var totalVotes int
	err := r.db.QueryRow("SELECT COUNT(*) FROM votes WHERE election_id = ? AND status = 'synced'", electionID).Scan(&totalVotes)
	if err != nil {
		return nil, err
	}

	// Get results by candidate
	candidateQuery := `
        SELECT candidate_id, COUNT(*) as vote_count
        FROM votes 
        WHERE election_id = ? AND status = 'synced'
        GROUP BY candidate_id
        ORDER BY vote_count DESC
    `
	rows, err := r.db.Query(candidateQuery, electionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type CandidateResult struct {
		CandidateID string  `json:"candidate_id"`
		VoteCount   int     `json:"vote_count"`
		Percentage  float64 `json:"percentage"`
	}

	var candidateResults []CandidateResult
	for rows.Next() {
		var candidateID string
		var voteCount int
		err := rows.Scan(&candidateID, &voteCount)
		if err != nil {
			return nil, err
		}

		percentage := 0.0
		if totalVotes > 0 {
			percentage = float64(voteCount) / float64(totalVotes) * 100
		}

		candidateResults = append(candidateResults, CandidateResult{
			CandidateID: candidateID,
			VoteCount:   voteCount,
			Percentage:  percentage,
		})
	}

	// Get results by polling unit
	pollingUnitQuery := `
        SELECT polling_unit_id, COUNT(*) as vote_count
        FROM votes 
        WHERE election_id = ? AND status = 'synced'
        GROUP BY polling_unit_id
        ORDER BY vote_count DESC
    `
	rows, err = r.db.Query(pollingUnitQuery, electionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type PollingUnitResult struct {
		PollingUnitID string `json:"polling_unit_id"`
		VoteCount     int    `json:"vote_count"`
	}

	var pollingUnitResults []PollingUnitResult
	for rows.Next() {
		var pollingUnitID string
		var voteCount int
		err := rows.Scan(&pollingUnitID, &voteCount)
		if err != nil {
			return nil, err
		}
		pollingUnitResults = append(pollingUnitResults, PollingUnitResult{
			PollingUnitID: pollingUnitID,
			VoteCount:     voteCount,
		})
	}

	// Get voting timeline
	timelineQuery := `
        SELECT DATE(created_at) as vote_date, COUNT(*) as vote_count
        FROM votes 
        WHERE election_id = ? AND status = 'synced'
        GROUP BY DATE(created_at)
        ORDER BY vote_date
    `
	rows, err = r.db.Query(timelineQuery, electionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type TimelineEntry struct {
		Date      string `json:"date"`
		VoteCount int    `json:"vote_count"`
	}

	var timeline []TimelineEntry
	for rows.Next() {
		var date string
		var voteCount int
		err := rows.Scan(&date, &voteCount)
		if err != nil {
			return nil, err
		}
		timeline = append(timeline, TimelineEntry{
			Date:      date,
			VoteCount: voteCount,
		})
	}

	return map[string]interface{}{
		"total_votes":          totalVotes,
		"candidate_results":    candidateResults,
		"polling_unit_results": pollingUnitResults,
		"voting_timeline":      timeline,
	}, nil
}

// GetVotesByTimeRange gets votes within a specific time range
func (r *VoteRepository) GetVotesByTimeRange(electionID int64, startTime, endTime time.Time) ([]database.Vote, error) {
	query := `
        SELECT id, verification_hash, election_id, polling_unit_id, candidate_id, 
               encrypted_vote, transaction_hash, block_number, status, created_at, synced_at
        FROM votes
        WHERE election_id = ? AND created_at BETWEEN ? AND ?
        ORDER BY created_at DESC
    `

	rows, err := r.db.Query(query, electionID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []database.Vote
	for rows.Next() {
		var vote database.Vote
		err := rows.Scan(&vote.ID, &vote.VerificationHash, &vote.ElectionID,
			&vote.PollingUnitID, &vote.CandidateID, &vote.EncryptedVote,
			&vote.TransactionHash, &vote.BlockNumber, &vote.Status,
			&vote.CreatedAt, &vote.SyncedAt)
		if err != nil {
			return nil, err
		}
		votes = append(votes, vote)
	}

	return votes, nil
}

// GetVoteCountByStatus gets vote count by status for an election
func (r *VoteRepository) GetVoteCountByStatus(electionID int64) (map[string]int, error) {
	query := `
        SELECT status, COUNT(*) as count
        FROM votes
        WHERE election_id = ?
        GROUP BY status
    `

	rows, err := r.db.Query(query, electionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statusCounts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, err
		}
		statusCounts[status] = count
	}

	return statusCounts, nil
}
