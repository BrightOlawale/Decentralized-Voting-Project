package types

import "time"

// VoteRequest represents a vote submission request
type VoteRequest struct {
	NIN             string `json:"nin" binding:"required"`
	FingerprintData string `json:"fingerprint_data" binding:"required"`
	CandidateID     string `json:"candidate_id" binding:"required"`
	PollingUnitID   string `json:"polling_unit_id" binding:"required"`
	EncryptedVote   string `json:"encrypted_vote"`
	Signature       string `json:"signature"`
}

// VoterRegistrationRequest represents a voter registration request
type VoterRegistrationRequest struct {
	NIN             string    `json:"nin" binding:"required"`
	FirstName       string    `json:"first_name" binding:"required"`
	LastName        string    `json:"last_name" binding:"required"`
	DateOfBirth     time.Time `json:"date_of_birth" binding:"required"`
	Gender          string    `json:"gender" binding:"required"`
	PollingUnitID   string    `json:"polling_unit_id" binding:"required"`
	FingerprintData string    `json:"fingerprint_data" binding:"required"`
}

// VoterRegistrationResponse represents the response after voter registration
type VoterRegistrationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	VoterID int64  `json:"voter_id,omitempty"`
}

// VoteResponse represents the response after vote submission
type VoteResponse struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	VoteID          string `json:"vote_id,omitempty"`
	TransactionHash string `json:"transaction_hash,omitempty"`
	QueuePosition   int    `json:"queue_position,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// VoterStatus represents voter verification status
type VoterStatus struct {
	HasVoted    bool   `json:"has_voted"`
	VoteID      string `json:"vote_id,omitempty"`
	Timestamp   int64  `json:"timestamp,omitempty"`
	PollingUnit string `json:"polling_unit,omitempty"`
}

// VoterInfo represents voter information
type VoterInfo struct {
	ID            int64  `json:"id"`
	NIN           string `json:"nin"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	DateOfBirth   string `json:"date_of_birth"`
	Gender        string `json:"gender"`
	PollingUnitID string `json:"polling_unit_id"`
	IsActive      bool   `json:"is_active"`
	RegisteredAt  string `json:"registered_at"`
}

// ElectionInfo represents election information
type ElectionInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	StartTime  int64    `json:"start_time"`
	EndTime    int64    `json:"end_time"`
	IsActive   bool     `json:"is_active"`
	Candidates []string `json:"candidates"`
	TotalVotes int64    `json:"total_votes"`
}

// SystemStatus represents the current system status
type SystemStatus struct {
	ServerStatus      string `json:"server_status"`
	BlockchainStatus  string `json:"blockchain_status"`
	DatabaseStatus    string `json:"database_status"`
	PendingVotes      int    `json:"pending_votes"`
	LastBlockNumber   uint64 `json:"last_block_number"`
	CurrentElectionID string `json:"current_election_id"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID            int64  `json:"id"`
	Action        string `json:"action"`
	UserID        string `json:"user_id"`
	PollingUnitID string `json:"polling_unit_id"`
	Details       string `json:"details"`
	Timestamp     int64  `json:"timestamp"`
	IPAddress     string `json:"ip_address"`
}
