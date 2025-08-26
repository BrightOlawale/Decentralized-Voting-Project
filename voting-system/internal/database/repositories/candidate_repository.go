package repositories

import (
	"database/sql"
	"voting-system/internal/database"
)

type CandidateRepository struct {
	db *sql.DB
}

func NewCandidateRepository(db *sql.DB) *CandidateRepository {
	return &CandidateRepository{db: db}
}

func (r *CandidateRepository) Insert(electionID int64, candidateID, name, party string) error {
	query := `
        INSERT OR IGNORE INTO candidates (election_id, candidate_id, name, party)
        VALUES (?, ?, ?, ?)
    `
	_, err := r.db.Exec(query, electionID, candidateID, name, party)
	return err
}

func (r *CandidateRepository) ListByElection(electionID int64) ([]database.Candidate, error) {
	query := `
        SELECT id, election_id, candidate_id, name, party, created_at
        FROM candidates
        WHERE election_id = ?
        ORDER BY created_at ASC
    `
	rows, err := r.db.Query(query, electionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []database.Candidate
	for rows.Next() {
		var c database.Candidate
		if err := rows.Scan(&c.ID, &c.ElectionID, &c.CandidateID, &c.Name, &c.Party, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}
