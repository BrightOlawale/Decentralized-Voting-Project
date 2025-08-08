package handlers

import (
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
	"voting-system/internal/api/interfaces"
	"voting-system/internal/api/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

// GetAuditLogs retrieves audit logs with filtering and pagination
func GetAuditLogs(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse query parameters
		limit := 50 // default limit
		offset := 0
		action := c.Query("action")
		pollingUnitID := c.Query("polling_unit_id")
		startTimeStr := c.Query("start_time")
		endTimeStr := c.Query("end_time")

		// Parse limit and offset
		if limitStr := c.Query("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
				limit = l
			}
		}
		if offsetStr := c.Query("offset"); offsetStr != "" {
			if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
				offset = o
			}
		}

		// Parse time range
		var startTime, endTime *time.Time
		if startTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
				startTime = &t
			}
		}
		if endTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
				endTime = &t
			}
		}

		// Get audit logs
		logs, err := services.AuditLogRepository().GetAuditLogs(limit, offset, action, pollingUnitID, startTime, endTime)
		if err != nil {
			services.GetLogger().Error("Error getting audit logs: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "audit_error",
				Code:    500,
				Message: "Failed to retrieve audit logs",
			})
			return
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data: map[string]interface{}{
				"logs":   logs,
				"limit":  limit,
				"offset": offset,
				"total":  len(logs),
			},
			Message: "Audit logs retrieved successfully",
		})
	}
}

// GetVotesByTimeRange gets votes within a specific time range for auditing
func GetVotesByTimeRange(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		timeRange := c.Param("time_range")
		if timeRange == "" {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "missing_parameter",
				Code:    400,
				Message: "Time range parameter is required",
			})
			return
		}

		// Parse time range (e.g., "24h", "7d", "30d")
		var duration time.Duration
		var err error
		switch timeRange {
		case "24h":
			duration = 24 * time.Hour
		case "7d":
			duration = 7 * 24 * time.Hour
		case "30d":
			duration = 30 * 24 * time.Hour
		default:
			duration, err = time.ParseDuration(timeRange)
			if err != nil {
				c.JSON(http.StatusBadRequest, types.ErrorResponse{
					Error:   "invalid_time_range",
					Code:    400,
					Message: "Invalid time range format. Use 24h, 7d, 30d, or duration string",
				})
				return
			}
		}

		endTime := time.Now()
		startTime := endTime.Add(-duration)

		// Get current election
		currentElection, err := services.ElectionRepository().GetActiveElection()
		if err != nil {
			services.GetLogger().Error("Error getting current election: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "election_error",
				Code:    500,
				Message: "Failed to get current election",
			})
			return
		}

		// Get votes in time range
		votes, err := services.VoteRepository().GetVotesByTimeRange(currentElection.ID, startTime, endTime)
		if err != nil {
			services.GetLogger().Error("Error getting votes by time range: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "votes_error",
				Code:    500,
				Message: "Failed to get votes",
			})
			return
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data: map[string]interface{}{
				"votes":      votes,
				"start_time": startTime.Unix(),
				"end_time":   endTime.Unix(),
				"count":      len(votes),
			},
			Message: "Votes retrieved successfully",
		})
	}
}

// GetFullAuditLogs gets comprehensive audit logs for admin
func GetFullAuditLogs(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse query parameters
		limit := 100 // default limit for admin
		offset := 0
		action := c.Query("action")
		pollingUnitID := c.Query("polling_unit_id")
		startTimeStr := c.Query("start_time")
		endTimeStr := c.Query("end_time")

		// Parse limit and offset
		if limitStr := c.Query("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 5000 {
				limit = l
			}
		}
		if offsetStr := c.Query("offset"); offsetStr != "" {
			if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
				offset = o
			}
		}

		// Parse time range
		var startTime, endTime *time.Time
		if startTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
				startTime = &t
			}
		}
		if endTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
				endTime = &t
			}
		}

		// Get audit logs
		logs, err := services.AuditLogRepository().GetAuditLogs(limit, offset, action, pollingUnitID, startTime, endTime)
		if err != nil {
			services.GetLogger().Error("Error getting full audit logs: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "audit_error",
				Code:    500,
				Message: "Failed to retrieve audit logs",
			})
			return
		}

		// Get audit statistics
		statistics, err := services.AuditLogRepository().GetAuditStatistics(startTime, endTime)
		if err != nil {
			services.GetLogger().Error("Error getting audit statistics: %v", err)
		}

		// Get recent audit logs for summary
		recentLogs, err := services.AuditLogRepository().GetRecentAuditLogs(10)
		if err != nil {
			services.GetLogger().Error("Error getting recent audit logs: %v", err)
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data: map[string]interface{}{
				"logs":       logs,
				"statistics": statistics,
				"recent":     recentLogs,
				"limit":      limit,
				"offset":     offset,
				"total":      len(logs),
			},
			Message: "Full audit logs retrieved successfully",
		})
	}
}

// InvalidateVote invalidates a specific vote (Admin only)
func InvalidateVote(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		voteIDStr := c.Param("id")
		voteID, ok := new(big.Int).SetString(voteIDStr, 10)
		if !ok {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_vote_id",
				Code:    400,
				Message: "Invalid vote ID format",
			})
			return
		}

		var req struct {
			Reason string `json:"reason" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_request",
				Code:    400,
				Message: "Reason is required",
			})
			return
		}

		clientIP := getClientIP(c)
		createAuditLog(services, "vote_invalidation_attempt", "admin", "",
			"Vote invalidation attempt: "+voteIDStr+" - "+req.Reason, clientIP)

		// This would require owner permissions on the blockchain
		services.GetLogger().Info("Vote invalidation requested - vote_id: %s, reason: %s",
			voteID.String(), req.Reason)

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Vote invalidation initiated",
			Data: map[string]interface{}{
				"vote_id": voteID.String(),
				"reason":  req.Reason,
			},
		})
	}
}

// GetBlockInfo returns information about a specific blockchain block
func GetBlockInfo(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		blockNumberStr := c.Param("number")
		blockNumber, err := strconv.ParseUint(blockNumberStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_block_number",
				Code:    400,
				Message: "Invalid block number format",
			})
			return
		}

		// Get current block number to validate request
		currentBlock, err := services.GetBlockchainClient().GetBlockNumber()
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "blockchain_error",
				Code:    500,
				Message: "Failed to get current block number",
			})
			return
		}

		if blockNumber > currentBlock {
			c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "block_not_found",
				Code:    404,
				Message: "Block number exceeds current blockchain height",
			})
			return
		}

		// In a real implementation, you would fetch actual block data
		// For now, returning mock data
		blockInfo := map[string]interface{}{
			"block_number": blockNumber,
			"hash":         "0x" + "abcd1234567890",
			"parent_hash":  "0x" + "1234567890abcd",
			"timestamp":    time.Now().Unix() - int64((currentBlock-blockNumber)*15),
			"gas_used":     "8000000",
			"gas_limit":    "8000000",
			"tx_count":     25,
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    blockInfo,
		})
	}
}

// GetTransactionInfo returns information about a specific transaction
func GetTransactionInfo(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		txHash := c.Param("hash")
		if len(txHash) != 66 || !strings.HasPrefix(txHash, "0x") {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_tx_hash",
				Code:    400,
				Message: "Invalid transaction hash format",
			})
			return
		}

		// Parse the hash
		hash := common.HexToHash(txHash)

		// Get transaction status
		receipt, err := services.GetBlockchainClient().GetTransactionStatus(hash)
		if err != nil {
			c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "transaction_not_found",
				Code:    404,
				Message: "Transaction not found",
			})
			return
		}

		txInfo := map[string]interface{}{
			"hash":         receipt.TxHash.Hex(),
			"block_number": receipt.BlockNumber.Uint64(),
			"block_hash":   receipt.BlockHash.Hex(),
			"gas_used":     receipt.GasUsed,
			"status":       receipt.Status,
			"success":      receipt.Status == 1,
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    txInfo,
		})
	}
}

// GetContractInfo returns information about the voting contract
func GetContractInfo(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get contract statistics
		totalVotes, _ := services.GetBlockchainClient().GetTotalVotes()
		// totalElections, _ := services.GetBlockchainClient().GetTotalElections() // Method not implemented yet
		currentElectionID, _ := services.GetBlockchainClient().GetCurrentElectionID()

		contractInfo := map[string]interface{}{
			"address":          "0x1234567890abcdef", // Placeholder - would be from config
			"total_votes":      totalVotes.String(),
			"total_elections":  "0", // Placeholder until GetTotalElections is implemented
			"current_election": "0",
			"deployed_at":      "2024-01-01T00:00:00Z", // Would be actual deployment time
			"version":          "1.0.0",
			"compiler_version": "0.8.19",
		}

		if currentElectionID != nil {
			contractInfo["current_election"] = currentElectionID.String()
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    contractInfo,
		})
	}
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
