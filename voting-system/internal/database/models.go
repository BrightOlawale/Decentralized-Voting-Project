package database

import "time"

// AuditLog represents an audit log entry
type AuditLog struct {
	ID            int64     `db:"id" json:"id"`
	Action        string    `db:"action" json:"action"`
	UserID        string    `db:"user_id" json:"user_id"`
	PollingUnitID string    `db:"polling_unit_id" json:"polling_unit_id"`
	Details       string    `db:"details" json:"details"`
	IPAddress     string    `db:"ip_address" json:"ip_address"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// Terminal represents a voting terminal
type Terminal struct {
	ID            string     `db:"id" json:"id"`
	Name          string     `db:"name" json:"name"`
	Location      string     `db:"location" json:"location"`
	PollingUnitID string     `db:"polling_unit_id" json:"polling_unit_id"`
	EthAddress    string     `db:"eth_address" json:"eth_address"`
	PublicKey     string     `db:"public_key" json:"public_key"`
	Status        string     `db:"status" json:"status"`
	Authorized    bool       `db:"authorized" json:"authorized"`
	LastHeartbeat *time.Time `db:"last_heartbeat" json:"last_heartbeat"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

// Voter represents a registered voter
type Voter struct {
	ID              int64     `db:"id" json:"id"`
	NIN             string    `db:"nin" json:"nin"`
	FirstName       string    `db:"first_name" json:"first_name"`
	LastName        string    `db:"last_name" json:"last_name"`
	DateOfBirth     time.Time `db:"date_of_birth" json:"date_of_birth"`
	Gender          string    `db:"gender" json:"gender"`
	PollingUnitID   string    `db:"polling_unit_id" json:"polling_unit_id"`
	FingerprintHash string    `db:"fingerprint_hash" json:"fingerprint_hash"`
	RegisteredAt    time.Time `db:"registered_at" json:"registered_at"`
	IsActive        bool      `db:"is_active" json:"is_active"`
}

// Election represents an election
type Election struct {
	ID           int64     `db:"id" json:"id"`
	BlockchainID string    `db:"blockchain_id" json:"blockchain_id"`
	Name         string    `db:"name" json:"name"`
	Description  string    `db:"description" json:"description"`
	StartTime    time.Time `db:"start_time" json:"start_time"`
	EndTime      time.Time `db:"end_time" json:"end_time"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// Vote represents a vote record
type Vote struct {
	ID               int64      `db:"id" json:"id"`
	BlockchainVoteID string     `db:"blockchain_vote_id" json:"blockchain_vote_id"`
	VerificationHash string     `db:"verification_hash" json:"verification_hash"`
	ElectionID       int64      `db:"election_id" json:"election_id"`
	PollingUnitID    string     `db:"polling_unit_id" json:"polling_unit_id"`
	CandidateID      string     `db:"candidate_id" json:"candidate_id"`
	EncryptedVote    string     `db:"encrypted_vote" json:"encrypted_vote"`
	TransactionHash  string     `db:"transaction_hash" json:"transaction_hash"`
	BlockNumber      int64      `db:"block_number" json:"block_number"`
	Status           string     `db:"status" json:"status"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	SyncedAt         *time.Time `db:"synced_at" json:"synced_at"`
}

// PollingUnit represents a polling unit
type PollingUnit struct {
	ID                    string    `db:"id" json:"id"`
	Name                  string    `db:"name" json:"name"`
	Location              string    `db:"location" json:"location"`
	Ward                  string    `db:"ward" json:"ward"`
	LGA                   string    `db:"lga" json:"lga"`
	State                 string    `db:"state" json:"state"`
	TotalRegisteredVoters int       `db:"total_registered_voters" json:"total_registered_voters"`
	IsActive              bool      `db:"is_active" json:"is_active"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
}

// SystemLog represents a system log entry
type SystemLog struct {
	ID        int64     `db:"id" json:"id"`
	Level     string    `db:"level" json:"level"`
	Message   string    `db:"message" json:"message"`
	Component string    `db:"component" json:"component"`
	Details   string    `db:"details" json:"details"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// User represents an API user
type User struct {
	ID           int64      `db:"id" json:"id"`
	Username     string     `db:"username" json:"username"`
	Email        string     `db:"email" json:"email"`
	PasswordHash string     `db:"password_hash" json:"-"` // Never include in JSON
	FirstName    string     `db:"first_name" json:"first_name"`
	LastName     string     `db:"last_name" json:"last_name"`
	Role         string     `db:"role" json:"role"`
	Permissions  string     `db:"permissions" json:"permissions"` // JSON string
	IsActive     bool       `db:"is_active" json:"is_active"`
	LastLogin    *time.Time `db:"last_login" json:"last_login"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

// Session represents a user session
type Session struct {
	ID        string    `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Data      string    `db:"data" json:"data"` // JSON data
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
