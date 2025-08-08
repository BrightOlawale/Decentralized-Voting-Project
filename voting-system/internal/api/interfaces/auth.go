package interfaces

// Claims represents JWT token claims
type Claims struct {
	UserID      string   `json:"user_id"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	SessionID   string   `json:"session_id"`
	ExpiresAt   int64    `json:"expires_at"`
}

type AuthServiceInterface interface {
	ValidateToken(token string) (*Claims, error)
}
