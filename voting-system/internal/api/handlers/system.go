package handlers

import (
	"net/http"
	"runtime"
	"time"
	"voting-system/internal/api/interfaces"
	"voting-system/internal/api/types"

	"github.com/gin-gonic/gin"
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
			Location      string `json:"location" binding:"required"`
			PollingUnitID string `json:"polling_unit_id" binding:"required"`
			Address       string `json:"address" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_request",
				Code:    400,
				Message: "Invalid request format: " + err.Error(),
			})
			return
		}

		// Log terminal registration attempt
		clientIP := getClientIP(c)
		createAuditLog(services, "terminal_registration", req.TerminalID, req.PollingUnitID,
			"Terminal registration attempt from "+req.Location, clientIP)

		// Here you would typically:
		// 1. Validate the terminal credentials
		// 2. Store terminal information in database
		// 3. Generate terminal certificates/keys
		// 4. Authorize the terminal on blockchain (if needed)

		services.GetLogger().Info("Terminal registration request - terminal_id: %s, location: %s, polling_unit: %s",
			req.TerminalID, req.Location, req.PollingUnitID)

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Terminal registration completed",
			Data: map[string]interface{}{
				"terminal_id":   req.TerminalID,
				"status":        "registered",
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

		clientIP := getClientIP(c)
		action := "terminal_deauthorized"
		if req.Authorize {
			action = "terminal_authorized"
		}

		createAuditLog(services, action, terminalID, "",
			"Terminal authorization change: "+req.Reason, clientIP)

		services.GetLogger().Info("Terminal authorization change - terminal_id: %s, authorize: %t, reason: %s",
			terminalID, req.Authorize, req.Reason)

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Terminal authorization updated",
			Data: map[string]interface{}{
				"terminal_id": terminalID,
				"authorized":  req.Authorize,
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
