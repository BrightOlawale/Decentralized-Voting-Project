package handlers

import (
	"fmt"
	"math/big"
	"net/http"
	"time"
	"voting-system/internal/api/interfaces"
	"voting-system/internal/api/types"

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

		// Get election from database
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

		// Get election results from database
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

		// Add election metadata to results
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
			Name       string   `json:"name" binding:"required"`
			StartTime  int64    `json:"start_time" binding:"required"`
			EndTime    int64    `json:"end_time" binding:"required"`
			Candidates []string `json:"candidates" binding:"required,min=2"`
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
		if req.StartTime <= now {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_start_time",
				Code:    400,
				Message: "Start time must be in the future",
			})
			return
		}

		if req.EndTime <= req.StartTime {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_end_time",
				Code:    400,
				Message: "End time must be after start time",
			})
			return
		}

		// Create election on blockchain
		// startTime := big.NewInt(req.StartTime)
		// endTime := big.NewInt(req.EndTime)

		// This would require owner permissions - implement based on your admin setup
		// For now, this is a placeholder showing the structure

		clientIP := getClientIP(c)
		createAuditLog(services, "election_create_attempt", "admin", "",
			"Election creation attempt: "+req.Name, clientIP)

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Election creation initiated",
			Data: map[string]interface{}{
				"name":       req.Name,
				"start_time": req.StartTime,
				"end_time":   req.EndTime,
				"candidates": req.Candidates,
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

		clientIP := getClientIP(c)
		createAuditLog(services, "election_start_attempt", "admin", "",
			"Election start attempt: "+electionIDStr, clientIP)

		// This would require owner permissions - implement based on your admin setup
		services.GetLogger().Info("Election start requested - election_id: %s", electionID.String())

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Election start initiated",
			Data: map[string]interface{}{
				"election_id": electionID.String(),
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

		// This would require owner permissions - implement based on your admin setup
		services.GetLogger().Info("Election end requested - election_id: %s", electionID.String())

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Message: "Election end initiated",
			Data: map[string]interface{}{
				"election_id": electionID.String(),
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
