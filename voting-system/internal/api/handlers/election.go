package handlers

import (
	"fmt"
	"math/big"
	"net/http"
	"time"
	"voting-system/internal/api/interfaces"
	"voting-system/internal/api/types"
	"voting-system/internal/database"

	"github.com/gin-gonic/gin"
)

// GetCurrentElection returns information about the current active election
func GetCurrentElection(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentElectionID, err := services.GetBlockchainClient().GetCurrentElectionID()
		if err != nil {
			services.GetLogger().Error("Error getting current election: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "blockchain_error",
				Code:    500,
				Message: "Failed to get current election",
			})
			return
		}

		if currentElectionID == nil || currentElectionID.Cmp(big.NewInt(0)) == 0 {
			c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "no_active_election",
				Code:    404,
				Message: "No active election found",
			})
			return
		}

		// Get election details
		electionData, err := services.GetBlockchainClient().GetElectionDetails(currentElectionID)
		if err != nil {
			services.GetLogger().Error("Error getting election details: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "blockchain_error",
				Code:    500,
				Message: "Failed to get election details",
			})
			return
		}

		election := types.ElectionInfo{
			ID:         currentElectionID.String(),
			Name:       electionData.Name,
			StartTime:  electionData.StartTime.Int64(),
			EndTime:    electionData.EndTime.Int64(),
			IsActive:   electionData.IsActive,
			Candidates: electionData.Candidates,
			TotalVotes: electionData.TotalVotes.Int64(),
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    election,
		})
	}
}

// GetElectionDetails returns detailed information about a specific election
func GetElectionDetails(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		electionIDStr := c.Param("id")
		electionID, ok := new(big.Int).SetString(electionIDStr, 10)
		if !ok {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_election_id",
				Code:    400,
				Message: "Invalid election ID format",
			})
			return
		}

		electionData, err := services.GetBlockchainClient().GetElectionDetails(electionID)
		if err != nil {
			services.GetLogger().Error("Error getting election details: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "blockchain_error",
				Code:    500,
				Message: "Failed to get election details",
			})
			return
		}

		election := types.ElectionInfo{
			ID:         electionID.String(),
			Name:       electionData.Name,
			StartTime:  electionData.StartTime.Int64(),
			EndTime:    electionData.EndTime.Int64(),
			IsActive:   electionData.IsActive,
			Candidates: electionData.Candidates,
			TotalVotes: electionData.TotalVotes.Int64(),
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    election,
		})
	}
}

// GetElectionResults gets the complete results for an election
func GetElectionResults(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		electionID := c.Param("id")
		if electionID == "" {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "missing_parameter",
				Code:    400,
				Message: "Election ID is required",
			})
			return
		}

		// Parse election ID
		var id int64
		if _, err := fmt.Sscanf(electionID, "%d", &id); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_election_id",
				Code:    400,
				Message: "Invalid election ID format",
			})
			return
		}

		// Prefer on-chain results if connected
		if services.GetConnManager().IsConnected() {
			bcID := new(big.Int).SetInt64(id)
			// Aggregate per-candidate counts from chain
			agg, err := services.GetBlockchainClient().GetCandidateResults(bcID)
			if err == nil {
				resp := map[string]interface{}{
					"election_id": id,
					"results":     map[string]string{},
				}
				for k, v := range agg {
					resp["results"].(map[string]string)[k] = v.String()
				}
				c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Data: resp, Message: "Election results retrieved successfully"})
				return
			}
			services.GetLogger().Warning("Blockchain results unavailable, falling back to DB: %v", err)
		}

		// Fallback to database cache
		election, err := services.ElectionRepository().GetElectionByID(id)
		if err != nil {
			services.GetLogger().Error("Error getting election: %v", err)
			c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "election_not_found",
				Code:    404,
				Message: "Election not found",
			})
			return
		}

		results, err := services.VoteRepository().GetElectionResults(id)
		if err != nil {
			services.GetLogger().Error("Error getting election results: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "results_error",
				Code:    500,
				Message: "Failed to get election results",
			})
			return
		}

		results["election"] = map[string]interface{}{
			"id":          election.ID,
			"name":        election.Name,
			"description": election.Description,
			"start_time":  election.StartTime.Unix(),
			"end_time":    election.EndTime.Unix(),
			"is_active":   election.IsActive,
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    results,
			Message: "Election results retrieved successfully",
		})
	}
}

// GetElectionCandidates returns candidate IDs for a given election (public)
func GetElectionCandidates(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		electionIDStr := c.Param("id")
		if electionIDStr == "" {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "missing_parameter", Code: 400, Message: "Election ID is required"})
			return
		}

		// Try DB cache first by blockchain_id -> local id -> candidates list
		if e, err := services.ElectionRepository().GetElectionByBlockchainID(electionIDStr); err == nil && e != nil {
			if list, err2 := services.CandidateRepository().ListByElection(e.ID); err2 == nil && len(list) > 0 {
				cands := make([]string, 0, len(list))
				for _, x := range list {
					cands = append(cands, x.CandidateID)
				}
				c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Data: map[string]interface{}{
					"election_id": electionIDStr,
					"candidates":  cands,
				}})
				return
			}
		}

		// Fallback to blockchain details
		bid, ok := new(big.Int).SetString(electionIDStr, 10)
		if !ok {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid_election_id", Code: 400, Message: "Invalid election ID format"})
			return
		}
		details, err := services.GetBlockchainClient().GetElectionDetails(bid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "blockchain_error", Code: 500, Message: "Failed to fetch election"})
			return
		}
		c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Data: map[string]interface{}{
			"election_id": electionIDStr,
			"candidates":  details.Candidates,
		}})
	}
}

// GetElectionStatistics gets detailed statistics for an election
func GetElectionStatistics(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		electionID := c.Param("id")
		if electionID == "" {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "missing_parameter",
				Code:    400,
				Message: "Election ID is required",
			})
			return
		}

		// Parse election ID
		var id int64
		if _, err := fmt.Sscanf(electionID, "%d", &id); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_election_id",
				Code:    400,
				Message: "Invalid election ID format",
			})
			return
		}

		// Get election statistics from database
		statistics, err := services.ElectionRepository().GetElectionStatistics(id)
		if err != nil {
			services.GetLogger().Error("Error getting election statistics: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "statistics_error",
				Code:    500,
				Message: "Failed to get election statistics",
			})
			return
		}

		// Get vote status counts
		statusCounts, err := services.VoteRepository().GetVoteCountByStatus(id)
		if err != nil {
			services.GetLogger().Error("Error getting vote status counts: %v", err)
		} else {
			statistics["vote_status_counts"] = statusCounts
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    statistics,
			Message: "Election statistics retrieved successfully",
		})
	}
}

// CreateElection creates a new election (Admin only)
func CreateElection(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string   `json:"name" binding:"required"`
			StartTime   int64    `json:"start_time" binding:"required"`
			EndTime     int64    `json:"end_time" binding:"required"`
			Candidates  []string `json:"candidates" binding:"required,min=1"`
			Description string   `json:"description"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_request",
				Code:    400,
				Message: "Invalid request format: " + err.Error(),
			})
			return
		}

		// Validate times
		now := time.Now().Unix()
		if req.StartTime <= now || req.EndTime <= req.StartTime {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_time_window",
				Code:    400,
				Message: "Start time must be future and end time after start",
			})
			return
		}

		// Create on blockchain (owner account configured in blockchain client)
		start := big.NewInt(req.StartTime)
		end := big.NewInt(req.EndTime)
		tx, err := services.GetBlockchainClient().CreateElection(req.Name, start, end, req.Candidates)
		if err != nil {
			services.GetLogger().Error("CreateElection on-chain failed: %v", err)
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "blockchain_error", Code: 400, Message: err.Error()})
			return
		}
		receipt, err := services.GetBlockchainClient().WaitForTransaction(tx)
		if err != nil {
			services.GetLogger().Error("CreateElection tx failed: %v", err)
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "transaction_failed", Code: 400, Message: err.Error()})
			return
		}

		// Infer new election ID by reading total elections
		total, err := services.GetBlockchainClient().GetTotalElections()
		if err != nil {
			services.GetLogger().Warning("Could not fetch total elections: %v", err)
		}

		// Cache election in DB
		e := &database.Election{
			BlockchainID: total.String(),
			Name:         req.Name,
			Description:  req.Description,
			StartTime:    time.Unix(req.StartTime, 0),
			EndTime:      time.Unix(req.EndTime, 0),
			IsActive:     false,
			CreatedAt:    time.Now(),
		}
		if err := services.ElectionRepository().CreateElection(e); err != nil {
			services.GetLogger().Warning("Failed to cache election in DB: %v", err)
		} else {
			// Persist initial candidates in DB cache
			for _, cid := range req.Candidates {
				_ = services.CandidateRepository().Insert(e.ID, cid, "", "")
			}
		}

		c.JSON(http.StatusCreated, types.SuccessResponse{
			Success: true,
			Message: "Election created",
			Data: map[string]interface{}{
				"blockchain_id": total.String(),
				"tx_hash":       receipt.TxHash.Hex(),
			},
		})
	}
}

// StartElection starts a specific election (Admin only)
func StartElection(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		electionIDStr := c.Param("id")
		electionID, ok := new(big.Int).SetString(electionIDStr, 10)
		if !ok {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_election_id",
				Code:    400,
				Message: "Invalid election ID format",
			})
			return
		}

		tx, err := services.GetBlockchainClient().StartElection(electionID)
		if err != nil {
			services.GetLogger().Error("StartElection on-chain failed: %v", err)
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "blockchain_error", Code: 400, Message: err.Error()})
			return
		}
		receipt, err := services.GetBlockchainClient().WaitForTransaction(tx)
		if err != nil {
			services.GetLogger().Error("StartElection tx failed: %v", err)
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "transaction_failed", Code: 400, Message: err.Error()})
			return
		}

		// Update DB cache if present
		if e, err := services.ElectionRepository().GetElectionByBlockchainID(electionID.String()); err == nil {
			_ = services.ElectionRepository().UpdateElectionStatus(e.ID, true)
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Election started",
			Data: map[string]interface{}{
				"election_id": electionID.String(),
				"tx_hash":     receipt.TxHash.Hex(),
			},
		})
	}
}

// EndElection ends the current active election (Admin only)
func EndElection(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		electionIDStr := c.Param("id")
		electionID, ok := new(big.Int).SetString(electionIDStr, 10)
		if !ok {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_election_id",
				Code:    400,
				Message: "Invalid election ID format",
			})
			return
		}

		clientIP := getClientIP(c)
		createAuditLog(services, "election_end_attempt", "admin", "",
			"Election end attempt: "+electionIDStr, clientIP)

		// End the current active election on-chain (owner permissions required)
		tx, err := services.GetBlockchainClient().EndElection()
		if err != nil {
			services.GetLogger().Error("EndElection on-chain failed: %v", err)
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "blockchain_error", Code: 400, Message: err.Error()})
			return
		}
		receipt, err := services.GetBlockchainClient().WaitForTransaction(tx)
		if err != nil {
			services.GetLogger().Error("EndElection tx failed: %v", err)
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "transaction_failed", Code: 400, Message: err.Error()})
			return
		}

		// Update DB cache if present
		if e, err := services.ElectionRepository().GetElectionByBlockchainID(electionID.String()); err == nil {
			_ = services.ElectionRepository().UpdateElectionStatus(e.ID, false)
		}

		services.GetLogger().Info("Election ended - election_id: %s, tx: %s", electionID.String(), receipt.TxHash.Hex())

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Election ended",
			Data: map[string]interface{}{
				"election_id": electionID.String(),
				"tx_hash":     receipt.TxHash.Hex(),
			},
		})
	}
}

// RegisterCandidates registers one or more candidates for an election (Admin only)
func RegisterCandidates(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		electionIDStr := c.Param("id")
		electionID, ok := new(big.Int).SetString(electionIDStr, 10)
		if !ok {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid_election_id", Code: 400, Message: "Invalid election ID"})
			return
		}

		var req struct {
			Candidates []string `json:"candidates" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid_request", Code: 400, Message: err.Error()})
			return
		}

		if !services.GetConnManager().IsConnected() {
			c.JSON(http.StatusServiceUnavailable, types.ErrorResponse{Error: "blockchain_offline", Code: 503, Message: "Blockchain is offline"})
			return
		}

		var receiptHash string
		var err error
		if len(req.Candidates) == 1 {
			tx, e := services.GetBlockchainClient().RegisterCandidate(electionID, req.Candidates[0])
			if e != nil {
				services.GetLogger().Error("RegisterCandidate failed: %v", e)
				c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "blockchain_error", Code: 400, Message: e.Error()})
				return
			}
			rec, e := services.GetBlockchainClient().WaitForTransaction(tx)
			if e == nil {
				receiptHash = rec.TxHash.Hex()
			}
			err = e
		} else {
			tx, e := services.GetBlockchainClient().RegisterCandidates(electionID, req.Candidates)
			if e != nil {
				services.GetLogger().Error("RegisterCandidates failed: %v", e)
				c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "blockchain_error", Code: 400, Message: e.Error()})
				return
			}
			rec, e := services.GetBlockchainClient().WaitForTransaction(tx)
			if e == nil {
				receiptHash = rec.TxHash.Hex()
			}
			err = e
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "transaction_failed", Code: 400, Message: err.Error()})
			return
		}

		// Persist candidates to DB cache as well
		// Resolve local election row by blockchain_id
		elect, _ := services.ElectionRepository().GetElectionByBlockchainID(electionID.String())
		if elect != nil {
			for _, cid := range req.Candidates {
				_ = services.CandidateRepository().Insert(elect.ID, cid, "", "")
			}
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Candidates registered",
			Data: map[string]interface{}{
				"election_id": electionID.String(),
				"tx_hash":     receiptHash,
				"count":       len(req.Candidates),
			},
		})
	}
}

// Helper function to calculate percentage
func calculatePercentage(votes, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(votes) / float64(total) * 100
}

// ListElections returns a paginated list of elections from DB cache
func ListElections(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 50
		offset := 0
		if v := c.Query("limit"); v != "" {
			fmt.Sscanf(v, "%d", &limit)
		}
		if v := c.Query("offset"); v != "" {
			fmt.Sscanf(v, "%d", &offset)
		}
		list, err := services.ElectionRepository().ListElections(limit, offset)
		if err != nil {
			services.GetLogger().Error("ListElections err: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "db_error", Code: 500, Message: "Failed to list elections"})
			return
		}
		c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Data: list})
	}
}

// DeleteElection deletes an election and related entities from DB (does not touch chain)
func DeleteElection(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ID int64 `json:"id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.ID <= 0 {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid_request", Code: 400, Message: "id required"})
			return
		}
		if err := services.ElectionRepository().DeleteElectionCascade(req.ID); err != nil {
			services.GetLogger().Error("DeleteElection err: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "db_error", Code: 500, Message: "Failed to delete election"})
			return
		}
		c.JSON(http.StatusOK, types.SuccessResponse{Success: true, Message: "Election deleted"})
	}
}
