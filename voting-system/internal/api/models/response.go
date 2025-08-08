package models

// BaseResponse represents the base API response structure
type BaseResponse struct {
	Success   bool        `json:"success" example:"true"`
	Message   string      `json:"message,omitempty" example:"Operation completed successfully"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp" example:"1640995200"`
	RequestID string      `json:"request_id,omitempty" example:"req_123456"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Code    string `json:"code" example:"INVALID_REQUEST"`
	Message string `json:"message" example:"Invalid request parameters"`
	Details string `json:"details,omitempty" example:"Field 'nin' is required"`
}

// VoteResponse represents vote submission response
type VoteResponse struct {
	Success         bool   `json:"success" example:"true"`
	Message         string `json:"message" example:"Vote cast successfully"`
	VoteID          string `json:"vote_id,omitempty" example:"12345"`
	TransactionHash string `json:"transaction_hash,omitempty" example:"0xabcd1234..."`
	QueuePosition   int    `json:"queue_position,omitempty" example:"3"`
	Status          string `json:"status" example:"confirmed"`
}

// VoterStatusResponse represents voter status response
type VoterStatusResponse struct {
	HasVoted    bool   `json:"has_voted" example:"false"`
	VoteID      string `json:"vote_id,omitempty" example:"12345"`
	Timestamp   int64  `json:"timestamp,omitempty" example:"1640995200"`
	PollingUnit string `json:"polling_unit,omitempty" example:"PU001"`
	ElectionID  string `json:"election_id,omitempty" example:"1"`
	IsEligible  bool   `json:"is_eligible" example:"true"`
	VerifiedAt  int64  `json:"verified_at" example:"1640995200"`
}

// ElectionResponse represents election information
type ElectionResponse struct {
	ID          string              `json:"id" example:"1"`
	Name        string              `json:"name" example:"2024 Presidential Election"`
	Description string              `json:"description" example:"National presidential election"`
	StartTime   int64               `json:"start_time" example:"1640995200"`
	EndTime     int64               `json:"end_time" example:"1641081600"`
	IsActive    bool                `json:"is_active" example:"true"`
	Status      string              `json:"status" example:"active"`
	Candidates  []CandidateInfo     `json:"candidates"`
	TotalVotes  int64               `json:"total_votes" example:"12500"`
	Results     []CandidateResult   `json:"results,omitempty"`
	Statistics  *ElectionStatistics `json:"statistics,omitempty"`
	CreatedAt   int64               `json:"created_at" example:"1640995200"`
}

// CandidateInfo represents candidate information
type CandidateInfo struct {
	ID       string `json:"id" example:"CANDIDATE_001"`
	Name     string `json:"name" example:"John Doe"`
	Party    string `json:"party,omitempty" example:"Democratic Party"`
	Bio      string `json:"bio,omitempty" example:"Experienced politician"`
	ImageURL string `json:"image_url,omitempty" example:"https://example.com/image.jpg"`
}

// CandidateResult represents election results for a candidate
type CandidateResult struct {
	CandidateID string  `json:"candidate_id" example:"CANDIDATE_001"`
	Name        string  `json:"name" example:"John Doe"`
	VoteCount   int64   `json:"vote_count" example:"5500"`
	Percentage  float64 `json:"percentage" example:"44.0"`
	Rank        int     `json:"rank" example:"1"`
}

// ElectionStatistics represents detailed election statistics
type ElectionStatistics struct {
	TotalVotes     int64   `json:"total_votes" example:"12500"`
	ValidVotes     int64   `json:"valid_votes" example:"12450"`
	InvalidVotes   int64   `json:"invalid_votes" example:"50"`
	TurnoutRate    float64 `json:"turnout_rate" example:"75.5"`
	Duration       int64   `json:"duration" example:"86400"`
	IsCompleted    bool    `json:"is_completed" example:"false"`
	ValidityRate   float64 `json:"validity_rate" example:"99.6"`
	PeakVotingHour int     `json:"peak_voting_hour" example:"14"`
}

// SystemStatusResponse represents system status
type SystemStatusResponse struct {
	ServerStatus      string                 `json:"server_status" example:"running"`
	BlockchainStatus  string                 `json:"blockchain_status" example:"connected"`
	DatabaseStatus    string                 `json:"database_status" example:"connected"`
	PendingVotes      int                    `json:"pending_votes" example:"5"`
	LastBlockNumber   uint64                 `json:"last_block_number" example:"12345"`
	CurrentElectionID string                 `json:"current_election_id" example:"1"`
	Uptime            int64                  `json:"uptime" example:"86400"`
	Version           string                 `json:"version" example:"1.0.0"`
	Environment       string                 `json:"environment" example:"production"`
	Services          map[string]interface{} `json:"services"`
}

// TerminalResponse represents terminal information
type TerminalResponse struct {
	ID            string `json:"id" example:"TERMINAL_001"`
	Name          string `json:"name" example:"Primary School Terminal"`
	Location      string `json:"location" example:"Lagos State"`
	PollingUnitID string `json:"polling_unit_id" example:"PU001"`
	Status        string `json:"status" example:"online"`
	Authorized    bool   `json:"authorized" example:"true"`
	LastHeartbeat *int64 `json:"last_heartbeat,omitempty" example:"1640995200"`
	VotesToday    int    `json:"votes_today" example:"25"`
	Uptime        string `json:"uptime" example:"2h 15m"`
	CreatedAt     int64  `json:"created_at" example:"1640995200"`
}

// UserResponse represents user information
type UserResponse struct {
	ID          string   `json:"id" example:"123"`
	Username    string   `json:"username" example:"operator1"`
	Email       string   `json:"email" example:"operator@example.com"`
	FirstName   string   `json:"first_name" example:"John"`
	LastName    string   `json:"last_name" example:"Doe"`
	Role        string   `json:"role" example:"operator"`
	Permissions []string `json:"permissions" example:"vote.read,election.read"`
	IsActive    bool     `json:"is_active" example:"true"`
	LastLogin   *int64   `json:"last_login,omitempty" example:"1640995200"`
	CreatedAt   int64    `json:"created_at" example:"1640995200"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Success      bool          `json:"success" example:"true"`
	Token        string        `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string        `json:"refresh_token,omitempty" example:"refresh_token_here"`
	ExpiresIn    int64         `json:"expires_in" example:"3600"`
	User         *UserResponse `json:"user,omitempty"`
}

// AuditLogResponse represents audit log entry
type AuditLogResponse struct {
	ID            int64  `json:"id" example:"12345"`
	Action        string `json:"action" example:"vote_cast"`
	UserID        string `json:"user_id" example:"voter_hash_123"`
	PollingUnitID string `json:"polling_unit_id" example:"PU001"`
	Details       string `json:"details" example:"Vote cast for candidate CANDIDATE_001"`
	IPAddress     string `json:"ip_address" example:"192.168.1.100"`
	Timestamp     int64  `json:"timestamp" example:"1640995200"`
	Severity      string `json:"severity" example:"info"`
}

// PaginatedResponse represents paginated response
type PaginatedResponse struct {
	Data       interface{}    `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
}

// PaginationInfo represents pagination information
type PaginationInfo struct {
	CurrentPage  int   `json:"current_page" example:"1"`
	PageSize     int   `json:"page_size" example:"20"`
	TotalPages   int   `json:"total_pages" example:"5"`
	TotalRecords int64 `json:"total_records" example:"100"`
	HasNext      bool  `json:"has_next" example:"true"`
	HasPrevious  bool  `json:"has_previous" example:"false"`
}

// BlockchainInfoResponse represents blockchain information
type BlockchainInfoResponse struct {
	NetworkID       int64  `json:"network_id" example:"1337"`
	ChainID         int64  `json:"chain_id" example:"1337"`
	BlockNumber     uint64 `json:"block_number" example:"12345"`
	ContractAddress string `json:"contract_address" example:"0x1234...abcd"`
	GasPrice        string `json:"gas_price" example:"20000000000"`
	IsConnected     bool   `json:"is_connected" example:"true"`
	LastSyncTime    int64  `json:"last_sync_time" example:"1640995200"`
}

// TransactionResponse represents blockchain transaction
type TransactionResponse struct {
	Hash        string `json:"hash" example:"0xabcd1234..."`
	BlockNumber uint64 `json:"block_number" example:"12345"`
	BlockHash   string `json:"block_hash" example:"0x5678efgh..."`
	GasUsed     uint64 `json:"gas_used" example:"21000"`
	Status      uint64 `json:"status" example:"1"`
	Success     bool   `json:"success" example:"true"`
	Timestamp   int64  `json:"timestamp" example:"1640995200"`
}

// WebSocketMessage represents WebSocket message structure
type WebSocketMessage struct {
	Type      string      `json:"type" example:"vote_cast"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp" example:"1640995200"`
	Channel   string      `json:"channel,omitempty" example:"election_1"`
}

// HealthCheckResponse represents health check response
type HealthCheckResponse struct {
	Status    string                 `json:"status" example:"healthy"`
	Timestamp int64                  `json:"timestamp" example:"1640995200"`
	Version   string                 `json:"version" example:"1.0.0"`
	Uptime    int64                  `json:"uptime" example:"86400"`
	Checks    map[string]HealthCheck `json:"checks"`
}

// HealthCheck represents individual health check
type HealthCheck struct {
	Status  string `json:"status" example:"healthy"`
	Message string `json:"message,omitempty" example:"Service is running normally"`
	Latency string `json:"latency,omitempty" example:"5ms"`
}
