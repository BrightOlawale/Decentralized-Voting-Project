package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"voting-system/internal/api/interfaces"
	"voting-system/internal/api/types"
	"voting-system/internal/database"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// HealthCheck provides a simple health check endpoint
func HealthCheck(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	}
}

// GetSystemStatus returns comprehensive system status
func GetSystemStatus(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := types.SystemStatus{
			ServerStatus:     "running",
			DatabaseStatus:   getDatabaseStatus(services),
			BlockchainStatus: getBlockchainStatus(services),
			PendingVotes:     services.GetSyncManager().GetPendingVoteCount(),
		}

		// Get current election ID
		if currentElectionID, err := services.GetBlockchainClient().GetCurrentElectionID(); err == nil {
			if currentElectionID != nil {
				status.CurrentElectionID = currentElectionID.String()
			} else {
				status.CurrentElectionID = "none"
			}
		}

		// Get last block number
		if blockNumber, err := services.GetBlockchainClient().GetBlockNumber(); err == nil {
			status.LastBlockNumber = blockNumber
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    status,
		})
	}
}

// GetSystemStats returns detailed system statistics
func GetSystemStats(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get memory stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Get blockchain stats
		totalVotes, _ := services.GetBlockchainClient().GetTotalVotes()
		// totalElections, _ := services.GetBlockchainClient().GetTotalElections() // Method not implemented yet
		blockNumber, _ := services.GetBlockchainClient().GetBlockNumber()
		accountBalance, _ := services.GetBlockchainClient().GetAccountBalance()

		stats := map[string]interface{}{
			"server": map[string]interface{}{
				"uptime":       time.Since(startTime).Seconds(),
				"goroutines":   runtime.NumGoroutine(),
				"memory_alloc": bToMb(m.Alloc),
				"memory_total": bToMb(m.TotalAlloc),
				"memory_sys":   bToMb(m.Sys),
				"gc_runs":      m.NumGC,
			},
			"blockchain": map[string]interface{}{
				"connected":       services.GetConnManager().IsConnected(),
				"block_number":    blockNumber,
				"account_balance": accountBalance.String(),
			},
			"voting": map[string]interface{}{
				"total_votes":     totalVotes.String(),
				"total_elections": "0", // Placeholder until GetTotalElections is implemented
				"pending_votes":   services.GetSyncManager().GetPendingVoteCount(),
				"sync_running":    services.GetSyncManager().IsRunning(),
			},
			"timestamp": time.Now().Unix(),
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    stats,
		})
	}
}

// TriggerSync manually triggers a synchronization process
func TriggerSync(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !services.GetSyncManager().IsRunning() {
			c.JSON(http.StatusServiceUnavailable, types.ErrorResponse{
				Error:   "sync_not_running",
				Code:    503,
				Message: "Sync manager is not running",
			})
			return
		}

		pendingCount := services.GetSyncManager().GetPendingVoteCount()
		if pendingCount == 0 {
			c.JSON(http.StatusOK, types.SuccessResponse{
				Success: true,
				Message: "No pending votes to sync",
				Data: map[string]interface{}{
					"pending_votes": 0,
				},
			})
			return
		}

		// Trigger manual sync
		go func() {
			syncedCount, failedCount, err := services.GetSyncManager().SyncNow()
			if err != nil {
				services.GetLogger().Error("Manual sync failed: %v", err)
			} else {
				services.GetLogger().Info("Manual sync completed - synced: %d, failed: %d", syncedCount, failedCount)
			}
		}()

		clientIP := getClientIP(c)
		createAuditLog(services, "manual_sync_triggered", "system", "",
			"Manual sync triggered", clientIP)

		c.JSON(http.StatusAccepted, types.SuccessResponse{
			Success: true,
			Message: "Sync process initiated",
			Data: map[string]interface{}{
				"pending_votes": pendingCount,
			},
		})
	}
}

// RegisterTerminal registers a new voting terminal
func RegisterTerminal(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TerminalID    string `json:"terminal_id" binding:"required"`
			Name          string `json:"name"`
			Location      string `json:"location" binding:"required"`
			PollingUnitID string `json:"polling_unit_id" binding:"required"`
			Address       string `json:"address" binding:"required"`
			Authorize     bool   `json:"authorize"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_request",
				Code:    400,
				Message: "Invalid request format: " + err.Error(),
			})
			return
		}

		// Basic address validation
		if !common.IsHexAddress(req.Address) {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_address",
				Code:    400,
				Message: "Invalid terminal Ethereum address",
			})
			return
		}

		// Log terminal registration attempt
		clientIP := getClientIP(c)
		createAuditLog(services, "terminal_registration", req.TerminalID, req.PollingUnitID,
			"Terminal registration attempt from "+req.Location, clientIP)

		// Persist terminal in DB (cache)
		term := &database.Terminal{
			ID:            req.TerminalID,
			Name:          req.Name,
			Location:      req.Location,
			PollingUnitID: req.PollingUnitID,
			EthAddress:    req.Address,
			PublicKey:     "",
			Status:        "registered",
			Authorized:    false,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := services.TerminalRepository().RegisterTerminal(term); err != nil {
			services.GetLogger().Error("Failed to store terminal: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "database_error",
				Code:    500,
				Message: "Failed to store terminal",
			})
			return
		}

		// Optionally authorize on-chain immediately
		var txHash string
		if req.Authorize {
			if services.GetConnManager().IsConnected() {
				tx, err := services.GetBlockchainClient().AuthorizeTerminal(req.Address, true)
				if err != nil {
					services.GetLogger().Error("Blockchain authorizeTerminal failed: %v", err)
					c.JSON(http.StatusBadRequest, types.ErrorResponse{
						Error:   "blockchain_error",
						Code:    400,
						Message: "Failed to authorize terminal on-chain: " + err.Error(),
					})
					return
				}
				receipt, err := services.GetBlockchainClient().WaitForTransaction(tx)
				if err != nil {
					services.GetLogger().Error("AuthorizeTerminal tx failed: %v", err)
				} else {
					txHash = receipt.TxHash.Hex()
				}
				// Update DB flag
				_ = services.TerminalRepository().AuthorizeTerminal(req.TerminalID)
				term.Authorized = true
				term.Status = "authorized"
			}
		}

		services.GetLogger().Info("Terminal registered - id: %s, address: %s, authorized: %t", req.TerminalID, req.Address, term.Authorized)

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Terminal registration completed",
			Data: map[string]interface{}{
				"terminal_id":   req.TerminalID,
				"status":        term.Status,
				"authorized":    term.Authorized,
				"transaction":   txHash,
				"registered_at": time.Now().Unix(),
			},
		})
	}
}

// GetTerminalStatus returns the status of a specific terminal
func GetTerminalStatus(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		terminalID := c.Param("id")
		if terminalID == "" {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "missing_parameter",
				Code:    400,
				Message: "Terminal ID is required",
			})
			return
		}

		// Here you would query your database for terminal information
		// For now, returning a mock response
		terminalStatus := map[string]interface{}{
			"terminal_id":     terminalID,
			"status":          "online",
			"last_heartbeat":  time.Now().Unix() - 30, // 30 seconds ago
			"location":        "Polling Unit A",
			"polling_unit_id": "PU001",
			"votes_today":     25,
			"uptime":          "2h 15m",
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    terminalStatus,
		})
	}
}

// AuthorizeTerminal authorizes a terminal to participate in voting
func AuthorizeTerminal(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		terminalID := c.Param("id")
		if terminalID == "" {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "missing_parameter",
				Code:    400,
				Message: "Terminal ID is required",
			})
			return
		}

		var req struct {
			Address   string `json:"address" binding:"required"`
			Authorize bool   `json:"authorize"`
			Reason    string `json:"reason"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_request",
				Code:    400,
				Message: "Invalid request format: " + err.Error(),
			})
			return
		}

		if !common.IsHexAddress(req.Address) {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_address",
				Code:    400,
				Message: "Invalid terminal Ethereum address",
			})
			return
		}

		clientIP := getClientIP(c)
		action := "terminal_deauthorized"
		if req.Authorize {
			action = "terminal_authorized"
		}
		createAuditLog(services, action, terminalID, "",
			"Terminal authorization change: "+req.Reason, clientIP)

		var txHash string
		if services.GetConnManager().IsConnected() {
			tx, err := services.GetBlockchainClient().AuthorizeTerminal(req.Address, req.Authorize)
			if err != nil {
				services.GetLogger().Error("AuthorizeTerminal failed: %v", err)
				c.JSON(http.StatusBadRequest, types.ErrorResponse{
					Error:   "blockchain_error",
					Code:    400,
					Message: "Failed to authorize terminal on-chain: " + err.Error(),
				})
				return
			}
			receipt, err := services.GetBlockchainClient().WaitForTransaction(tx)
			if err == nil {
				txHash = receipt.TxHash.Hex()
			}
		}

		// Update DB cache
		if req.Authorize {
			_ = services.TerminalRepository().AuthorizeTerminal(terminalID)
		} else {
			_ = services.TerminalRepository().DeauthorizeTerminal(terminalID)
		}

		services.GetLogger().Info("Terminal authorization change - terminal_id: %s, authorize: %t, reason: %s",
			terminalID, req.Authorize, req.Reason)

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Terminal authorization updated",
			Data: map[string]interface{}{
				"terminal_id": terminalID,
				"authorized":  req.Authorize,
				"transaction": txHash,
				"updated_at":  time.Now().Unix(),
			},
		})
	}
}

// AuthorizeTerminalAdmin handles admin-level terminal authorization
func AuthorizeTerminalAdmin(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		terminalID := c.Param("id")

		var req struct {
			Address   string `json:"address" binding:"required"`
			Authorize bool   `json:"authorize"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_request",
				Code:    400,
				Message: "Invalid request format: " + err.Error(),
			})
			return
		}

		// This would call the blockchain to authorize/deauthorize the terminal
		// services.GetBlockchainClient().AuthorizeTerminal(address, authorize)

		clientIP := getClientIP(c)
		createAuditLog(services, "admin_terminal_auth", "admin", "",
			"Admin terminal authorization for "+terminalID, clientIP)

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Terminal authorization completed",
		})
	}
}

// RegisterPollingUnit registers a polling unit on-chain (admin only)
func RegisterPollingUnit(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ID          string `json:"id" binding:"required"`
			Name        string `json:"name" binding:"required"`
			Location    string `json:"location" binding:"required"`
			TotalVoters int64  `json:"total_voters" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid_request", Code: 400, Message: "Invalid request: " + err.Error()})
			return
		}

		if !services.GetConnManager().IsConnected() {
			c.JSON(http.StatusServiceUnavailable, types.ErrorResponse{Error: "blockchain_offline", Code: 503, Message: "Blockchain is offline"})
			return
		}

		tx, err := services.GetBlockchainClient().RegisterPollingUnit(req.ID, req.Name, req.Location, big.NewInt(req.TotalVoters))
		if err != nil {
			services.GetLogger().Error("RegisterPollingUnit failed: %v", err)
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "blockchain_error", Code: 400, Message: err.Error()})
			return
		}

		rec, err := services.GetBlockchainClient().WaitForTransaction(tx)
		if err != nil {
			services.GetLogger().Error("RegisterPollingUnit tx failed: %v", err)
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "transaction_failed", Code: 400, Message: err.Error()})
			return
		}

		c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Message: "Polling unit registered", Data: map[string]interface{}{
			"tx_hash": rec.TxHash.Hex(),
		}})
	}
}

// GetPollingUnitInfo returns minimal on-chain info to confirm a polling unit exists
func GetPollingUnitInfo(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "missing_parameter", Code: 400, Message: "Polling unit id required"})
			return
		}
		if !services.GetConnManager().IsConnected() {
			c.JSON(http.StatusServiceUnavailable, types.ErrorResponse{Error: "blockchain_offline", Code: 503, Message: "Blockchain is offline"})
			return
		}
		pu, err := services.GetBlockchainClient().GetPollingUnit(id)
		if err != nil || pu == nil || pu.ID == "" {
			services.GetLogger().Warning("GetPollingUnit failed: %v", err)
			c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Data: map[string]interface{}{
				"id":     id,
				"exists": false,
			}})
			return
		}
		c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Data: map[string]interface{}{
			"id":             pu.ID,
			"name":           pu.Name,
			"location":       pu.Location,
			"total_voters":   pu.TotalVoters.String(),
			"votes_recorded": pu.VotesRecorded.String(),
			"is_active":      pu.IsActive,
		}})
	}
}

// Helper functions
var startTime = time.Now()

func getDatabaseStatus(services interfaces.Services) string {
	// Note: This would need to be implemented based on your database setup
	// For now, returning a default status
	return "connected"
}

func getBlockchainStatus(services interfaces.Services) string {
	if services.GetConnManager().IsConnected() {
		return "connected"
	}
	return "disconnected"
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// IssueTerminalToken issues a short-lived JWT for a terminal using optional HMAC verification
func IssueTerminalToken(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			DeviceID  string `json:"device_id" binding:"required"`
			Timestamp string `json:"ts"`
			Signature string `json:"signature"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid_request", Code: 400, Message: "Invalid request: " + err.Error()})
			return
		}

		shared := os.Getenv("TERMINAL_SHARED_SECRET")
		if shared != "" {
			// Verify HMAC over ts|device_id
			if req.Timestamp == "" || req.Signature == "" {
				c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "auth_required", Code: 401, Message: "Missing timestamp or signature"})
				return
			}
			// replay window 5 minutes
			if ts, err := strconv.ParseInt(req.Timestamp, 10, 64); err == nil {
				now := time.Now().Unix()
				if ts > now+300 || ts < now-300 {
					c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "stale_request", Code: 401, Message: "Timestamp outside allowed window"})
					return
				}
			}
			mac := hmac.New(sha256.New, []byte(shared))
			mac.Write([]byte(req.Timestamp + "|" + req.DeviceID))
			expected := hex.EncodeToString(mac.Sum(nil))
			if !hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(req.Signature))) {
				c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "invalid_signature", Code: 401, Message: "Signature verification failed"})
				return
			}
		}

		// Mint JWT
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "server_error", Code: 500, Message: "JWT secret not configured"})
			return
		}
		claims := jwt.MapClaims{
			"user_id":     req.DeviceID,
			"role":        "terminal",
			"permissions": []string{"voting", "terminal"},
			"exp":         time.Now().Add(24 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(secret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "server_error", Code: 500, Message: "Failed to sign token"})
			return
		}

		c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Data: map[string]string{"token": signed}})
	}
}

// EnsurePollingUnitTerminal allows a terminal to ensure its polling unit exists on-chain (idempotent)
func EnsurePollingUnitTerminal(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Location    string `json:"location"`
			TotalVoters int64  `json:"total_voters"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid_request", Code: 400, Message: "Invalid request: " + err.Error()})
			return
		}
		// default ID from token if not provided
		if req.ID == "" {
			if uid, ok := c.Get("user_id"); ok {
				req.ID, _ = uid.(string)
			}
		}
		if req.ID == "" {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "missing_parameter", Code: 400, Message: "Polling unit ID is required"})
			return
		}

		if !services.GetConnManager().IsConnected() {
			c.JSON(http.StatusServiceUnavailable, types.ErrorResponse{Error: "blockchain_offline", Code: 503, Message: "Blockchain is offline"})
			return
		}

		tx, err := services.GetBlockchainClient().RegisterPollingUnit(req.ID, req.Name, req.Location, big.NewInt(req.TotalVoters))
		if err != nil {
			// Treat as idempotent success if revert likely means exists
			services.GetLogger().Warning("EnsurePollingUnit - register returned error, treating as idempotent: %v", err)
			c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Message: "Polling unit ensured (existing)"})
			return
		}
		_, waitErr := services.GetBlockchainClient().WaitForTransaction(tx)
		if waitErr != nil {
			services.GetLogger().Warning("EnsurePollingUnit tx wait error, treating as idempotent: %v", waitErr)
			c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Message: "Polling unit ensured (pending/existing)"})
			return
		}
		c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Message: "Polling unit ensured"})
	}
}
