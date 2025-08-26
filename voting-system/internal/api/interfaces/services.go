package interfaces

import (
	"voting-system/internal/blockchain"
	"voting-system/internal/database/repositories"
	"voting-system/pkg/logger"
)

// Services defines the interface for API services
type Services interface {
	GetLogger() *logger.Logger
	GetBlockchainClient() *blockchain.BlockchainClient
	GetSyncManager() *blockchain.SyncManager
	GetConnManager() *blockchain.ConnectionManager
	AuthService() AuthServiceInterface
	VoterRepository() *repositories.VoterRepository
	ElectionRepository() *repositories.ElectionRepository
	VoteRepository() *repositories.VoteRepository
	AuditLogRepository() *repositories.AuditLogRepository
	TerminalRepository() *repositories.TerminalRepository
	UserRepository() *repositories.UserRepository
	CandidateRepository() *repositories.CandidateRepository
}
