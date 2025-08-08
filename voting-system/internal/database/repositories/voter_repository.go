package repositories

import (
	"database/sql"
	"voting-system/internal/database"
)

type VoterRepository struct {
	db *sql.DB
}

func NewVoterRepository(db *sql.DB) *VoterRepository {
	return &VoterRepository{db: db}
}

// RegisterVoter registers a new voter
func (r *VoterRepository) RegisterVoter(voter *database.Voter) error {
	query := `
        INSERT INTO voters (nin, first_name, last_name, date_of_birth, gender, 
                           polling_unit_id, fingerprint_hash)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `
	_, err := r.db.Exec(query, voter.NIN, voter.FirstName, voter.LastName,
		voter.DateOfBirth, voter.Gender, voter.PollingUnitID, voter.FingerprintHash)
	return err
}

// GetVoterByNIN retrieves voter information by NIN
func (r *VoterRepository) GetVoterByNIN(nin string) (*database.Voter, error) {
	query := `
        SELECT id, nin, first_name, last_name, date_of_birth, gender,
               polling_unit_id, fingerprint_hash, registered_at, is_active
        FROM voters
        WHERE nin = ? AND is_active = true
    `

	var voter database.Voter
	err := r.db.QueryRow(query, nin).Scan(
		&voter.ID, &voter.NIN, &voter.FirstName, &voter.LastName,
		&voter.DateOfBirth, &voter.Gender, &voter.PollingUnitID, &voter.FingerprintHash,
		&voter.RegisteredAt, &voter.IsActive,
	)

	if err != nil {
		return nil, err
	}

	return &voter, nil
}

// GetVoterByFingerprint retrieves voter by fingerprint hash
func (r *VoterRepository) GetVoterByFingerprint(fingerprintHash string) (*database.Voter, error) {
	query := `
        SELECT id, nin, first_name, last_name, date_of_birth, gender,
               polling_unit_id, fingerprint_hash, registered_at, is_active
        FROM voters
        WHERE fingerprint_hash = ? AND is_active = true
    `

	var voter database.Voter
	err := r.db.QueryRow(query, fingerprintHash).Scan(
		&voter.ID, &voter.NIN, &voter.FirstName, &voter.LastName,
		&voter.DateOfBirth, &voter.Gender, &voter.PollingUnitID, &voter.FingerprintHash,
		&voter.RegisteredAt, &voter.IsActive,
	)

	if err != nil {
		return nil, err
	}

	return &voter, nil
}

// GetVotersByPollingUnit retrieves all voters in a polling unit
func (r *VoterRepository) GetVotersByPollingUnit(pollingUnitID string) ([]*database.Voter, error) {
	query := `
        SELECT id, nin, first_name, last_name, date_of_birth, gender,
               polling_unit_id, fingerprint_hash, registered_at, is_active
        FROM voters
        WHERE polling_unit_id = ? AND is_active = true
        ORDER BY last_name, first_name
    `

	rows, err := r.db.Query(query, pollingUnitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var voters []*database.Voter
	for rows.Next() {
		var voter database.Voter
		err := rows.Scan(
			&voter.ID, &voter.NIN, &voter.FirstName, &voter.LastName,
			&voter.DateOfBirth, &voter.Gender, &voter.PollingUnitID, &voter.FingerprintHash,
			&voter.RegisteredAt, &voter.IsActive,
		)
		if err != nil {
			return nil, err
		}
		voters = append(voters, &voter)
	}

	return voters, nil
}

// UpdateVoter updates voter information
func (r *VoterRepository) UpdateVoter(voter *database.Voter) error {
	query := `
        UPDATE voters 
        SET first_name = ?, last_name = ?, date_of_birth = ?, gender = ?,
            polling_unit_id = ?, fingerprint_hash = ?, is_active = ?
        WHERE id = ?
    `
	_, err := r.db.Exec(query, voter.FirstName, voter.LastName, voter.DateOfBirth,
		voter.Gender, voter.PollingUnitID, voter.FingerprintHash, voter.IsActive, voter.ID)
	return err
}

// DeactivateVoter deactivates a voter
func (r *VoterRepository) DeactivateVoter(voterID int64) error {
	query := `UPDATE voters SET is_active = false WHERE id = ?`
	_, err := r.db.Exec(query, voterID)
	return err
}
