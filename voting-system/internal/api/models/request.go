package models

// VoteRequest represents a vote submission request
type VoteRequest struct {
	NIN             string `json:"nin" binding:"required,len=11" example:"12345678901"`
	BVN             string `json:"bvn" binding:"required,len=10" example:"1234567890"`
	FingerprintData string `json:"fingerprint_data" binding:"required" example:"base64_encoded_fingerprint"`
	CandidateID     string `json:"candidate_id" binding:"required" example:"CANDIDATE_001"`
	PollingUnitID   string `json:"polling_unit_id" binding:"required" example:"PU001"`
	EncryptedVote   string `json:"encrypted_vote,omitempty" example:"encrypted_vote_data"`
	Signature       string `json:"signature,omitempty" example:"digital_signature"`
	Timestamp       int64  `json:"timestamp" example:"1640995200"`
}

// VoterVerificationRequest represents voter verification request
type VoterVerificationRequest struct {
	NIN             string `json:"nin" binding:"required,len=11"`
	BVN             string `json:"bvn" binding:"required,len=10"`
	FingerprintData string `json:"fingerprint_data" binding:"required"`
}

// LoginRequest represents authentication login request
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"password123"`
	Terminal string `json:"terminal,omitempty" example:"TERMINAL_001"`
}

// CreateElectionRequest represents election creation request
type CreateElectionRequest struct {
	Name        string   `json:"name" binding:"required" example:"2024 Presidential Election"`
	Description string   `json:"description" example:"National presidential election"`
	StartTime   int64    `json:"start_time" binding:"required" example:"1640995200"`
	EndTime     int64    `json:"end_time" binding:"required" example:"1641081600"`
	Candidates  []string `json:"candidates" binding:"required,min=2" example:"CANDIDATE_001,CANDIDATE_002"`
}

// UpdateElectionRequest represents election update request
type UpdateElectionRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	StartTime   int64  `json:"start_time,omitempty"`
	EndTime     int64  `json:"end_time,omitempty"`
}

// TerminalRegistrationRequest represents terminal registration
type TerminalRegistrationRequest struct {
	TerminalID    string `json:"terminal_id" binding:"required" example:"TERMINAL_001"`
	Name          string `json:"name" binding:"required" example:"Primary School Terminal"`
	Location      string `json:"location" binding:"required" example:"Lagos State"`
	PollingUnitID string `json:"polling_unit_id" binding:"required" example:"PU001"`
	EthAddress    string `json:"eth_address" binding:"required" example:"0x1234...abcd"`
	PublicKey     string `json:"public_key" binding:"required" example:"-----BEGIN PUBLIC KEY-----"`
}

// TerminalAuthorizationRequest represents terminal authorization
type TerminalAuthorizationRequest struct {
	Authorize bool   `json:"authorize" example:"true"`
	Reason    string `json:"reason" example:"Terminal verified and approved"`
}

// UserCreateRequest represents user creation request
type UserCreateRequest struct {
	Username    string   `json:"username" binding:"required" example:"operator1"`
	Email       string   `json:"email" binding:"required,email" example:"operator@example.com"`
	Password    string   `json:"password" binding:"required,min=8" example:"securepass123"`
	FirstName   string   `json:"first_name" binding:"required" example:"John"`
	LastName    string   `json:"last_name" binding:"required" example:"Doe"`
	Role        string   `json:"role" binding:"required,oneof=admin operator viewer" example:"operator"`
	Permissions []string `json:"permissions,omitempty" example:"vote.read,election.read"`
}

// UserUpdateRequest represents user update request
type UserUpdateRequest struct {
	Email       string   `json:"email,omitempty"`
	FirstName   string   `json:"first_name,omitempty"`
	LastName    string   `json:"last_name,omitempty"`
	Role        string   `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// PasswordResetRequest represents password reset request
type PasswordResetRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// VoteInvalidationRequest represents vote invalidation request
type VoteInvalidationRequest struct {
	Reason string `json:"reason" binding:"required" example:"Suspected fraud"`
}

// SystemConfigRequest represents system configuration update
type SystemConfigRequest struct {
	MaintenanceMode bool                   `json:"maintenance_mode,omitempty"`
	Settings        map[string]interface{} `json:"settings,omitempty"`
}

// AuditLogFilterRequest represents audit log filtering
type AuditLogFilterRequest struct {
	StartTime     int64  `json:"start_time,omitempty" example:"1640995200"`
	EndTime       int64  `json:"end_time,omitempty" example:"1641081600"`
	Action        string `json:"action,omitempty" example:"vote_cast"`
	UserID        string `json:"user_id,omitempty" example:"voter_hash_123"`
	PollingUnitID string `json:"polling_unit_id,omitempty" example:"PU001"`
	Limit         int    `json:"limit,omitempty" example:"50"`
	Offset        int    `json:"offset,omitempty" example:"0"`
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page     int    `json:"page" example:"1"`
	PageSize int    `json:"page_size" example:"20"`
	SortBy   string `json:"sort_by,omitempty" example:"created_at"`
	SortDir  string `json:"sort_dir,omitempty" example:"desc"`
}
