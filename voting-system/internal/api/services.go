package api

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"voting-system/internal/api/interfaces"
	"voting-system/internal/blockchain"
	"voting-system/internal/database/repositories"
	"voting-system/pkg/config"
	"voting-system/pkg/logger"

	"github.com/golang-jwt/jwt/v5"
)

// Services contains all the dependencies for API handlers
type Services struct {
	// Core dependencies
	DB               *sql.DB
	BlockchainClient *blockchain.BlockchainClient
	SyncManager      *blockchain.SyncManager
	EventMonitor     *blockchain.EventMonitor
	ConnManager      *blockchain.ConnectionManager
	Logger           *logger.Logger
	Config           *config.Config

	// Business logic services - commented out for now to avoid import cycles
	// VotingService     services.VotingServiceInterface
	// ElectionService   services.ElectionServiceInterface
	// AuditService      services.AuditServiceInterface
	// AuthService       services.AuthServiceInterface
	// TerminalService   services.TerminalServiceInterface
	// BlockchainService services.BlockchainServiceInterface

	// WebSocket hub - commented out for now
	// WSHub *websocket.Hub

	// Cache and session management - commented out for now
	// Cache   CacheInterface
	// Session SessionInterface

	// Auth service interface
	authService interfaces.AuthServiceInterface

	// Repositories
	voterRepository    *repositories.VoterRepository
	electionRepository *repositories.ElectionRepository
	voteRepository     *repositories.VoteRepository
	auditLogRepository *repositories.AuditLogRepository
	terminalRepository *repositories.TerminalRepository
	userRepository     *repositories.UserRepository
}

// CacheInterface defines caching operations
type CacheInterface interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{}, ttl int) error
	Delete(key string) error
	Exists(key string) bool
	Clear() error
}

// SessionInterface defines session management operations
type SessionInterface interface {
	Create(userID string, data map[string]interface{}) (string, error)
	Get(sessionID string) (map[string]interface{}, error)
	Update(sessionID string, data map[string]interface{}) error
	Delete(sessionID string) error
	Cleanup() error
}

// NewServices creates a new services container
func NewServices(
	db *sql.DB,
	blockchainClient *blockchain.BlockchainClient,
	syncManager *blockchain.SyncManager,
	eventMonitor *blockchain.EventMonitor,
	connManager *blockchain.ConnectionManager,
	logger *logger.Logger,
	config *config.Config,
) *Services {

	// Initialize WebSocket hub - commented out for now
	// wsHub := websocket.NewHub(logger)

	// Initialize cache (Redis or in-memory) - commented out for now
	// var cache CacheInterface
	// if config.Redis.Enabled {
	// 	cache = NewRedisCache(config.Redis)
	// } else {
	// 	cache = NewInMemoryCache()
	// }

	// Initialize session manager - commented out for now
	// var session SessionInterface
	// if config.Redis.Enabled {
	// 	session = NewRedisSession(config.Redis)
	// } else {
	// 	session = NewInMemorySession()
	// }

	// Create base services
	services := &Services{
		DB:               db,
		BlockchainClient: blockchainClient,
		SyncManager:      syncManager,
		EventMonitor:     eventMonitor,
		ConnManager:      connManager,
		Logger:           logger,
		Config:           config,
		// WSHub:            wsHub,
		// Cache:            cache,
		// Session:          session,
	}

	// Initialize business logic services - commented out for now
	// services.initializeBusinessServices()

	// Initialize auth service
	services.authService = services

	// Initialize repositories
	services.voterRepository = repositories.NewVoterRepository(db)
	services.electionRepository = repositories.NewElectionRepository(db)
	services.voteRepository = repositories.NewVoteRepository(db)
	services.auditLogRepository = repositories.NewAuditLogRepository(db)
	services.terminalRepository = repositories.NewTerminalRepository(db)
	services.userRepository = repositories.NewUserRepository(db)

	return services
}

// initializeBusinessServices initializes all business logic services
// func (s *Services) initializeBusinessServices() {
// 	s.VotingService = services.NewVotingService(
// 		s.DB,
// 		s.BlockchainClient,
// 		s.SyncManager,
// 		s.Logger,
// 		s.Cache,
// 	)

// 	s.ElectionService = services.NewElectionService(
// 		s.DB,
// 		s.BlockchainClient,
// 		s.Logger,
// 		s.Cache,
// 	)

// 	s.AuditService = services.NewAuditService(
// 		s.DB,
// 		s.BlockchainClient,
// 		s.Logger,
// 	)

// 	s.AuthService = services.NewAuthService(
// 		s.DB,
// 		s.Config.Security,
// 		s.Session,
// 		s.Logger,
// 	)

// 	s.TerminalService = services.NewTerminalService(
// 		s.DB,
// 		s.BlockchainClient,
// 		s.Logger,
// 		s.Cache,
// 	)

// 	s.BlockchainService = services.NewBlockchainService(
// 		s.BlockchainClient,
// 		s.SyncManager,
// 		s.EventMonitor,
// 		s.ConnManager,
// 		s.Logger,
// 	)
// }

// Start starts all background services
func (s *Services) Start() error {
	s.Logger.Info("Starting API services...")

	// Start WebSocket hub - commented out for now
	// go s.WSHub.Run()

	// Start blockchain services
	if err := s.SyncManager.Start(); err != nil {
		s.Logger.Error("Failed to start sync manager: %v", err)
		return err
	}

	if err := s.EventMonitor.Start(); err != nil {
		s.Logger.Error("Failed to start event monitor: %v", err)
		return err
	}

	if err := s.ConnManager.Start(); err != nil {
		s.Logger.Error("Failed to start connection manager: %v", err)
		return err
	}

	// Set up event callbacks
	s.setupEventCallbacks()

	s.Logger.Info("All API services started successfully")
	return nil
}

// Stop stops all background services
func (s *Services) Stop() {
	s.Logger.Info("Stopping API services...")

	// Stop blockchain services
	s.SyncManager.Stop()
	s.EventMonitor.Stop()
	s.ConnManager.Stop()

	// Stop WebSocket hub - commented out for now
	// s.WSHub.Stop()

	// Cleanup sessions - commented out for now
	// if err := s.Session.Cleanup(); err != nil {
	// 	s.Logger.Error("Error during session cleanup: %v", err)
	// }

	s.Logger.Info("All API services stopped")
}

// setupEventCallbacks configures callbacks for blockchain events
func (s *Services) setupEventCallbacks() {
	// Sync manager callbacks
	s.SyncManager.SetCallbacks(
		func(voteData blockchain.VoteData, txHash string) {
			// Vote sync success - broadcast to WebSocket clients
			// s.WSHub.BroadcastVoteSync(voteData, txHash)
			s.Logger.Info("Vote synced successfully: %s", txHash)
		},
		func(voteData blockchain.VoteData, err error) {
			// Vote sync failed
			s.Logger.Error("Vote sync failed: %v", err)
		},
		func(syncedCount, failedCount int) {
			// Sync cycle complete - broadcast system status update
			// s.WSHub.BroadcastSystemStatus(map[string]interface{}{
			// 	"pending_votes": s.SyncManager.GetPendingVoteCount(),
			// 	"last_sync": map[string]interface{}{
			// 		"synced": syncedCount,
			// 		"failed": failedCount,
			// 	},
			// })
			s.Logger.Info("Sync cycle completed - synced: %d, failed: %d", syncedCount, failedCount)
		},
	)

	// Event monitor callbacks
	s.EventMonitor.SetVoteCastCallback(func(event *blockchain.SecureVotingSystemVoteCast) {
		// Broadcast vote cast event to WebSocket clients
		// s.WSHub.BroadcastVoteCast(event)
		s.Logger.Info("Vote cast event received")
	})

	// Connection manager callbacks
	s.ConnManager.SetCallbacks(
		func() {
			// Blockchain disconnected
			// s.WSHub.BroadcastSystemStatus(map[string]interface{}{
			// 	"blockchain_status": "disconnected",
			// })
			s.Logger.Warning("Blockchain connection lost")
		},
		func() {
			// Blockchain reconnected
			// s.WSHub.BroadcastSystemStatus(map[string]interface{}{
			// 	"blockchain_status": "connected",
			// })
			s.Logger.Info("Blockchain connection restored")
		},
	)
}

// GetService returns a specific service by name
func (s *Services) GetService(name string) interface{} {
	switch name {
	case "voting":
		// return s.VotingService
		return nil
	case "election":
		// return s.ElectionService
		return nil
	case "audit":
		// return s.AuditService
		return nil
	case "auth":
		// return s.AuthService
		return nil
	case "terminal":
		// return s.TerminalService
		return nil
	case "blockchain":
		// return s.BlockchainService
		return nil
	default:
		return nil
	}
}

// Interface implementation methods
func (s *Services) GetLogger() *logger.Logger {
	return s.Logger
}

func (s *Services) GetBlockchainClient() *blockchain.BlockchainClient {
	return s.BlockchainClient
}

func (s *Services) GetSyncManager() *blockchain.SyncManager {
	return s.SyncManager
}

func (s *Services) GetConnManager() *blockchain.ConnectionManager {
	return s.ConnManager
}

func (s *Services) AuthService() interfaces.AuthServiceInterface {
	return s.authService
}

func (s *Services) VoterRepository() *repositories.VoterRepository {
	return s.voterRepository
}

func (s *Services) ElectionRepository() *repositories.ElectionRepository {
	return s.electionRepository
}

func (s *Services) VoteRepository() *repositories.VoteRepository {
	return s.voteRepository
}

func (s *Services) AuditLogRepository() *repositories.AuditLogRepository {
	return s.auditLogRepository
}

func (s *Services) TerminalRepository() *repositories.TerminalRepository {
	return s.terminalRepository
}

func (s *Services) UserRepository() *repositories.UserRepository {
	return s.userRepository
}

// IsHealthy checks if all critical services are healthy
func (s *Services) IsHealthy() bool {
	// Check database connection
	if err := s.DB.Ping(); err != nil {
		s.Logger.Error("Database health check failed: %v", err)
		return false
	}

	// Check if critical services are running
	if !s.SyncManager.IsRunning() {
		s.Logger.Warning("Sync manager is not running")
		return false
	}

	// Check blockchain connection (optional for health)
	if !s.ConnManager.IsConnected() {
		s.Logger.Warning("Blockchain is disconnected (system can run in offline mode)")
		// Don't fail health check for blockchain disconnection
	}

	return true
}

// GetStats returns current service statistics
func (s *Services) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"sync_manager": map[string]interface{}{
			"running":       s.SyncManager.IsRunning(),
			"pending_votes": s.SyncManager.GetPendingVoteCount(),
		},
		"blockchain": map[string]interface{}{
			"connected": s.ConnManager.IsConnected(),
		},
		"websocket": map[string]interface{}{
			"active_connections": 0, // s.WSHub.GetConnectionCount()
		},
	}

	// Add blockchain stats if connected
	if s.ConnManager.IsConnected() {
		if blockNumber, err := s.BlockchainClient.GetBlockNumber(); err == nil {
			stats["blockchain"].(map[string]interface{})["last_block"] = blockNumber
		}
		if totalVotes, err := s.BlockchainClient.GetTotalVotes(); err == nil {
			stats["blockchain"].(map[string]interface{})["total_votes"] = totalVotes.String()
		}
	}

	return stats
}

// ValidateToken implements the AuthServiceInterface
func (s *Services) ValidateToken(token string) (*interfaces.Claims, error) {
	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	// Parse and validate JWT token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get secret key from config
		secretKey := s.Config.Security.JWTSecret
		if secretKey == "" {
			return nil, errors.New("JWT secret key not configured")
		}

		return []byte(secretKey), nil
	})

	if err != nil {
		s.Logger.Error("Token parsing failed: %v", err)
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	// Check if token is valid
	if !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}

	// Extract claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Validate required claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, errors.New("missing user_id claim")
	}

	role, ok := claims["role"].(string)
	if !ok {
		return nil, errors.New("missing role claim")
	}

	// Check expiration
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("missing expiration claim")
	}

	if time.Now().Unix() > int64(exp) {
		return nil, errors.New("token has expired")
	}

	// Extract permissions (optional)
	var permissions []string
	if perms, ok := claims["permissions"].([]interface{}); ok {
		for _, perm := range perms {
			if permStr, ok := perm.(string); ok {
				permissions = append(permissions, permStr)
			}
		}
	}

	// Extract session ID (optional)
	sessionID := ""
	if sid, ok := claims["session_id"].(string); ok {
		sessionID = sid
	}

	// Create and return claims
	result := &interfaces.Claims{
		UserID:      userID,
		Role:        role,
		Permissions: permissions,
		SessionID:   sessionID,
		ExpiresAt:   int64(exp),
	}

	s.Logger.Info("Token validated successfully for user: %s, role: %s", userID, role)
	return result, nil
}
