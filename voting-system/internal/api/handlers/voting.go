package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"voting-system/internal/api/interfaces"
	"voting-system/internal/api/types"
	"voting-system/internal/blockchain"
	"voting-system/internal/database"

	"github.com/gin-gonic/gin"
)

// Helper functions
func getClientIP(c *gin.Context) string {
	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return realIP
	}
	return c.ClientIP()
}

func createAuditLog(services interfaces.Services, action, verificationHash, pollingUnitID, details, clientIP string) {
	// Create audit log entry
	auditLog := &database.AuditLog{
		Action:        action,
		UserID:        verificationHash, // Using verification hash as user identifier
		PollingUnitID: pollingUnitID,
		Details:       details,
		IPAddress:     clientIP,
		CreatedAt:     time.Now(),
	}

	// Insert audit log
	err := services.AuditLogRepository().InsertAuditLog(auditLog)
	if err != nil {
		services.GetLogger().Error("Failed to create audit log: %v", err)
	}
}

// RegisterVoter handles voter registration requests
func RegisterVoter(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req types.VoterRegistrationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			services.GetLogger().Error("Invalid voter registration request: %v", err)
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_request",
				Code:    400,
				Message: "Invalid request format: " + err.Error(),
			})
			return
		}

		// Log the registration attempt
		clientIP := getClientIP(c)
		services.GetLogger().Info("Voter registration attempt - nin: %s, polling_unit: %s, ip: %s",
			req.NIN, req.PollingUnitID, clientIP)

		// Check if voter already exists
		existingVoter, err := services.VoterRepository().GetVoterByNIN(req.NIN)
		if err == nil && existingVoter != nil {
			services.GetLogger().Warning("Duplicate voter registration attempt - nin: %s", req.NIN)
			c.JSON(http.StatusConflict, types.ErrorResponse{
				Error:   "voter_exists",
				Code:    409,
				Message: "Voter with this NIN is already registered",
			})
			return
		}

		// Create fingerprint hash
		fingerprintHash := sha256.Sum256([]byte(req.FingerprintData))
		fingerprintHashStr := hex.EncodeToString(fingerprintHash[:])

		// Check if fingerprint is already registered
		existingByFingerprint, err := services.VoterRepository().GetVoterByFingerprint(fingerprintHashStr)
		if err == nil && existingByFingerprint != nil {
			services.GetLogger().Warning("Duplicate fingerprint registration attempt - nin: %s", req.NIN)
			c.JSON(http.StatusConflict, types.ErrorResponse{
				Error:   "fingerprint_exists",
				Code:    409,
				Message: "Fingerprint is already registered",
			})
			return
		}

		// Create voter record
		voter := &database.Voter{
			NIN:             req.NIN,
			FirstName:       req.FirstName,
			LastName:        req.LastName,
			DateOfBirth:     req.DateOfBirth,
			Gender:          req.Gender,
			PollingUnitID:   req.PollingUnitID,
			FingerprintHash: fingerprintHashStr,
			RegisteredAt:    time.Now(),
			IsActive:        true,
		}

		// Register voter in database
		err = services.VoterRepository().RegisterVoter(voter)
		if err != nil {
			services.GetLogger().Error("Failed to register voter: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "registration_failed",
				Code:    500,
				Message: "Failed to register voter",
			})
			return
		}

		// Create audit log
		createAuditLog(services, "voter_registered", req.NIN, req.PollingUnitID,
			fmt.Sprintf("Voter registered: %s %s", req.FirstName, req.LastName), clientIP)

		services.GetLogger().Info("Voter registered successfully - nin: %s, name: %s %s",
			req.NIN, req.FirstName, req.LastName)

		c.JSON(http.StatusCreated, types.VoterRegistrationResponse{
			Success: true,
			Message: "Voter registered successfully",
			VoterID: voter.ID,
		})
	}
}

// CastVote handles vote submission requests
func CastVote(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req types.VoteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			services.GetLogger().Error("Invalid vote request: %v", err)
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_request",
				Code:    400,
				Message: "Invalid request format: " + err.Error(),
			})
			return
		}

		// Log the vote attempt
		clientIP := getClientIP(c)
		services.GetLogger().Info("Vote submission attempt - polling_unit: %s, candidate: %s, ip: %s",
			req.PollingUnitID, req.CandidateID, clientIP)

		// Verify voter exists in database
		voter, err := services.VoterRepository().GetVoterByNIN(req.NIN)
		if err != nil {
			services.GetLogger().Warning("Voter not found - nin: %s", req.NIN)
			c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "voter_not_found",
				Code:    404,
				Message: "Voter not found or not registered",
			})
			return
		}

		// Verify fingerprint
		fingerprintHash := sha256.Sum256([]byte(req.FingerprintData))
		fingerprintHashStr := hex.EncodeToString(fingerprintHash[:])
		if fingerprintHashStr != voter.FingerprintHash {
			services.GetLogger().Warning("Invalid fingerprint - nin: %s", req.NIN)
			createAuditLog(services, "vote_rejected_invalid_fingerprint", req.NIN, req.PollingUnitID,
				"Invalid fingerprint provided", clientIP)
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Error:   "invalid_fingerprint",
				Code:    401,
				Message: "Invalid fingerprint",
			})
			return
		}

		// Verify polling unit matches
		if voter.PollingUnitID != req.PollingUnitID {
			services.GetLogger().Warning("Polling unit mismatch - nin: %s, expected: %s, got: %s",
				req.NIN, voter.PollingUnitID, req.PollingUnitID)
			createAuditLog(services, "vote_rejected_polling_unit_mismatch", req.NIN, req.PollingUnitID,
				fmt.Sprintf("Polling unit mismatch: expected %s, got %s", voter.PollingUnitID, req.PollingUnitID), clientIP)
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "polling_unit_mismatch",
				Code:    400,
				Message: "Voter is not registered in this polling unit",
			})
			return
		}

		// Create verification hash from NIN + Fingerprint
		verificationData := req.NIN + req.FingerprintData
		hash := sha256.Sum256([]byte(verificationData))
		verificationHash := hex.EncodeToString(hash[:])

		// Check if voter has already voted
		hasVoted, err := services.GetBlockchainClient().HasVoterVoted(verificationHash)
		if err != nil {
			services.GetLogger().Error("Error checking voter status: %v", err)

			// If blockchain is unavailable, add to sync queue
			if !services.GetConnManager().IsConnected() {
				voteData := blockchain.VoteData{
					VerificationHash: verificationHash,
					EncryptedVote:    req.EncryptedVote,
					PollingUnitID:    req.PollingUnitID,
					CandidateID:      req.CandidateID,
				}

				services.GetSyncManager().AddPendingVote(voteData)
				queuePosition := services.GetSyncManager().GetPendingVoteCount()

				// Create audit log
				createAuditLog(services, "vote_queued", verificationHash, req.PollingUnitID,
					fmt.Sprintf("Vote queued for candidate %s", req.CandidateID), clientIP)

				c.JSON(http.StatusAccepted, types.VoteResponse{
					Success:       true,
					Message:       "Vote queued for blockchain sync",
					QueuePosition: queuePosition,
				})
				return
			}

			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "blockchain_error",
				Code:    500,
				Message: "Failed to verify voter status",
			})
			return
		}

		if hasVoted {
			services.GetLogger().Warning("Duplicate vote attempt - hash: %s", verificationHash)
			createAuditLog(services, "vote_rejected_duplicate", verificationHash, req.PollingUnitID,
				"Voter has already cast a vote", clientIP)

			c.JSON(http.StatusConflict, types.ErrorResponse{
				Error:   "already_voted",
				Code:    409,
				Message: "Voter has already cast a vote in this election",
			})
			return
		}

		// Verify current election is active
		currentElectionID, err := services.GetBlockchainClient().GetCurrentElectionID()
		if err != nil {
			services.GetLogger().Error("Error getting current election: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "election_error",
				Code:    500,
				Message: "Failed to get current election status",
			})
			return
		}

		if currentElectionID == nil || currentElectionID.Cmp(big.NewInt(0)) == 0 {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "no_active_election",
				Code:    400,
				Message: "No active election found",
			})
			return
		}

		// Get election details to validate candidate
		electionData, err := services.GetBlockchainClient().GetElectionDetails(currentElectionID)
		if err != nil {
			services.GetLogger().Error("Error getting election details: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "election_error",
				Code:    500,
				Message: "Failed to get election details",
			})
			return
		}

		// Validate candidate ID
		validCandidate := false
		for _, candidate := range electionData.Candidates {
			if candidate == req.CandidateID {
				validCandidate = true
				break
			}
		}

		if !validCandidate {
			services.GetLogger().Warning("Invalid candidate ID: %s", req.CandidateID)
			createAuditLog(services, "vote_rejected_invalid_candidate", verificationHash, req.PollingUnitID,
				fmt.Sprintf("Invalid candidate ID: %s", req.CandidateID), clientIP)

			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "invalid_candidate",
				Code:    400,
				Message: "Invalid candidate ID",
			})
			return
		}

		// Prepare vote data
		voteData := blockchain.VoteData{
			VerificationHash: verificationHash,
			EncryptedVote:    req.EncryptedVote,
			PollingUnitID:    req.PollingUnitID,
			CandidateID:      req.CandidateID,
		}

		// Get current election from database
		currentElection, err := services.ElectionRepository().GetActiveElection()
		if err != nil {
			services.GetLogger().Error("Error getting current election from database: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "election_error",
				Code:    500,
				Message: "Failed to get current election from database",
			})
			return
		}

		// Store vote in database
		dbVote := &database.Vote{
			VerificationHash: verificationHash,
			ElectionID:       currentElection.ID,
			PollingUnitID:    req.PollingUnitID,
			CandidateID:      req.CandidateID,
			EncryptedVote:    req.EncryptedVote,
			Status:           "pending",
			CreatedAt:        time.Now(),
		}

		err = services.VoteRepository().InsertVote(dbVote)
		if err != nil {
			services.GetLogger().Error("Failed to store vote in database: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "database_error",
				Code:    500,
				Message: "Failed to store vote",
			})
			return
		}

		// Try to cast vote immediately if blockchain is connected
		if services.GetConnManager().IsConnected() {
			tx, err := services.GetBlockchainClient().CastVote(voteData)
			if err != nil {
				services.GetLogger().Error("Error casting vote: %v", err)

				// Add to sync queue as fallback
				services.GetSyncManager().AddPendingVote(voteData)
				queuePosition := services.GetSyncManager().GetPendingVoteCount()

				createAuditLog(services, "vote_queued_error", verificationHash, req.PollingUnitID,
					fmt.Sprintf("Vote queued due to error: %s", err.Error()), clientIP)

				c.JSON(http.StatusAccepted, types.VoteResponse{
					Success:       true,
					Message:       "Vote queued due to blockchain error",
					QueuePosition: queuePosition,
				})
				return
			}

			// Wait for transaction confirmation
			receipt, err := services.GetBlockchainClient().WaitForTransaction(tx)
			if err != nil {
				services.GetLogger().Error("Transaction failed: %v", err)

				// Add to sync queue for retry
				services.GetSyncManager().AddPendingVote(voteData)
				queuePosition := services.GetSyncManager().GetPendingVoteCount()

				c.JSON(http.StatusAccepted, types.VoteResponse{
					Success:       true,
					Message:       "Vote queued for retry",
					QueuePosition: queuePosition,
				})
				return
			}

			// Update vote in database with transaction details
			err = services.VoteRepository().UpdateVoteSync(verificationHash, receipt.TxHash.Hex(), receipt.BlockNumber.Int64())
			if err != nil {
				services.GetLogger().Error("Failed to update vote sync status: %v", err)
			}

			// Success - vote recorded on blockchain
			services.GetLogger().Info("Vote cast successfully - tx_hash: %s, gas_used: %d, polling_unit: %s",
				receipt.TxHash.Hex(), receipt.GasUsed, req.PollingUnitID)

			createAuditLog(services, "vote_cast_success", verificationHash, req.PollingUnitID,
				fmt.Sprintf("Vote cast for candidate %s, TX: %s", req.CandidateID, receipt.TxHash.Hex()), clientIP)

			c.JSON(http.StatusOK, types.VoteResponse{
				Success:         true,
				Message:         "Vote cast successfully",
				TransactionHash: receipt.TxHash.Hex(),
			})
		} else {
			// Blockchain offline - add to sync queue
			services.GetSyncManager().AddPendingVote(voteData)
			queuePosition := services.GetSyncManager().GetPendingVoteCount()

			createAuditLog(services, "vote_queued_offline", verificationHash, req.PollingUnitID,
				fmt.Sprintf("Vote queued (blockchain offline) for candidate %s", req.CandidateID), clientIP)

			c.JSON(http.StatusAccepted, types.VoteResponse{
				Success:       true,
				Message:       "Vote queued (blockchain offline)",
				QueuePosition: queuePosition,
			})
		}
	}
}

// GetVoterStatus checks if a voter has already voted
func GetVoterStatus(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		voterHash := c.Param("voter_hash")
		if voterHash == "" {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "missing_parameter",
				Code:    400,
				Message: "Voter hash is required",
			})
			return
		}

		// Check voter status on blockchain
		hasVoted, err := services.GetBlockchainClient().HasVoterVoted(voterHash)
		if err != nil {
			services.GetLogger().Error("Error checking voter status: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "blockchain_error",
				Code:    500,
				Message: "Failed to check voter status",
			})
			return
		}

		status := types.VoterStatus{
			HasVoted: hasVoted,
		}

		// If voter has voted, try to get additional details
		if hasVoted {
			// Note: Getting vote details would require storing vote ID mapping
			// This would need to be implemented based on your specific requirements
			status.Timestamp = time.Now().Unix() // Placeholder
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    status,
		})
	}
}

// VerifyVoter handles voter verification requests
func VerifyVoter(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		voterHash := c.Param("voter_hash")
		if voterHash == "" {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "missing_parameter",
				Code:    400,
				Message: "Voter hash is required",
			})
			return
		}

		// Log verification attempt
		clientIP := getClientIP(c)
		createAuditLog(services, "voter_verification", voterHash, "",
			"Voter verification request", clientIP)

		// Check if voter exists in system (implement based on your voter database)
		// This would involve checking against your voter registration database

		// For now, we'll check blockchain status
		hasVoted, err := services.GetBlockchainClient().HasVoterVoted(voterHash)
		if err != nil {
			services.GetLogger().Error("Error verifying voter: %v", err)
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "verification_error",
				Code:    500,
				Message: "Failed to verify voter",
			})
			return
		}

		verification := map[string]interface{}{
			"voter_hash":  voterHash,
			"is_eligible": true, // This should be checked against voter registration DB
			"has_voted":   hasVoted,
			"verified_at": time.Now().Unix(),
		}

		c.JSON(http.StatusOK, types.SuccessResponse{
			Success: true,
			Data:    verification,
			Message: "Voter verification completed",
		})
	}
}
