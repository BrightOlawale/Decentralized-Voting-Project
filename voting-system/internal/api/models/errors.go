package models

// Error codes
const (
	// General errors
	ErrCodeInvalidRequest     = "INVALID_REQUEST"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"
	ErrCodeNotFound           = "NOT_FOUND"
	ErrCodeConflict           = "CONFLICT"
	ErrCodeInternalError      = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"

	// Voting specific errors
	ErrCodeVoterNotFound     = "VOTER_NOT_FOUND"
	ErrCodeAlreadyVoted      = "ALREADY_VOTED"
	ErrCodeInvalidCandidate  = "INVALID_CANDIDATE"
	ErrCodeElectionNotActive = "ELECTION_NOT_ACTIVE"
	ErrCodeInvalidVote       = "INVALID_VOTE"

	// Blockchain errors
	ErrCodeBlockchainError   = "BLOCKCHAIN_ERROR"
	ErrCodeTransactionFailed = "TRANSACTION_FAILED"
	ErrCodeContractError     = "CONTRACT_ERROR"
	ErrCodeInsufficientFunds = "INSUFFICIENT_FUNDS"

	// Authentication errors
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired       = "TOKEN_EXPIRED"
	ErrCodeInvalidToken       = "INVALID_TOKEN"
	ErrCodeAccountLocked      = "ACCOUNT_LOCKED"

	// Terminal errors
	ErrCodeTerminalNotFound     = "TERMINAL_NOT_FOUND"
	ErrCodeTerminalUnauthorized = "TERMINAL_UNAUTHORIZED"
	ErrCodeTerminalOffline      = "TERMINAL_OFFLINE"
)

// APIError represents a structured API error
type APIError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	Fields     map[string]string      `json:"fields,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	StatusCode int                    `json:"-"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// NewAPIError creates a new API error
func NewAPIError(code, message string, statusCode int) *APIError {
	return &APIError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// WithDetails adds details to the error
func (e *APIError) WithDetails(details string) *APIError {
	e.Details = details
	return e
}

// WithField adds a field error
func (e *APIError) WithField(field, message string) *APIError {
	if e.Fields == nil {
		e.Fields = make(map[string]string)
	}
	e.Fields[field] = message
	return e
}

// WithMetadata adds metadata to the error
func (e *APIError) WithMetadata(key string, value interface{}) *APIError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}
