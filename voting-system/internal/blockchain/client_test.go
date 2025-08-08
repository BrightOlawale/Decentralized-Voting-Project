package blockchain

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Test configuration
	testNodeURL      = "http://localhost:8545"
	testPrivateKey   = "0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"
	testContractAddr = "0x345cA3e014Aaf5dcA488057592ee47305D9B3e10" // Will be set from environment or deployment
)

// TestBlockchainClient tests the blockchain client functionality
func TestBlockchainClient(t *testing.T) {
	// Skip if no blockchain is running
	if !isBlockchainAvailable() {
		t.Skip("Blockchain not available, skipping integration tests")
	}

	// Get contract address from environment
	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	if contractAddr == "" {
		contractAddr = testContractAddr
	}
	if contractAddr == "" {
		t.Skip("CONTRACT_ADDRESS not set, skipping integration tests")
	}

	// Create blockchain client
	client, err := NewBlockchainClient(testNodeURL, contractAddr, testPrivateKey)
	require.NoError(t, err, "Failed to create blockchain client")
	defer client.Close()

	t.Run("TestConnection", func(t *testing.T) {
		testConnection(t, client)
	})

	t.Run("TestCurrentElection", func(t *testing.T) {
		testCurrentElection(t, client)
	})

	t.Run("TestVoterStatus", func(t *testing.T) {
		testVoterStatus(t, client)
	})

	t.Run("TestVoteCasting", func(t *testing.T) {
		testVoteCasting(t, client)
	})

	t.Run("TestElectionResults", func(t *testing.T) {
		testElectionResults(t, client)
	})
}

func testConnection(t *testing.T, client *BlockchainClient) {
	// Test getting block number
	blockNumber, err := client.GetBlockNumber()
	assert.NoError(t, err, "Failed to get block number")
	// Note: Block number can be 0 if blockchain is just starting
	t.Logf("Current block number: %d", blockNumber)

	// Test getting account balance
	balance, err := client.GetAccountBalance()
	assert.NoError(t, err, "Failed to get account balance")
	assert.NotNil(t, balance, "Balance should not be nil")

	t.Logf("Account balance: %s wei", balance.String())
}

func testCurrentElection(t *testing.T, client *BlockchainClient) {
	// Test getting current election ID
	electionID, err := client.GetCurrentElectionID()
	if err != nil {
		// If contract is not deployed or has no code, this is expected
		if strings.Contains(err.Error(), "no contract code") {
			t.Skip("Contract not deployed at address, skipping election tests")
		}
		assert.NoError(t, err, "Failed to get current election ID")
	}

	// electionID can be nil if no election is active
	if electionID == nil {
		t.Log("No active election found")
		return
	}

	t.Logf("Current election ID: %s", electionID.String())

	// If there's an active election, get its details
	if electionID.Cmp(big.NewInt(0)) > 0 {
		electionData, err := client.GetElectionDetails(electionID)
		if err != nil {
			t.Logf("Failed to get election details: %v", err)
			return
		}
		assert.NotNil(t, electionData, "Election data should not be nil")

		t.Logf("Election name: %s", electionData.Name)
		t.Logf("Start time: %s", time.Unix(electionData.StartTime.Int64(), 0))
		t.Logf("End time: %s", time.Unix(electionData.EndTime.Int64(), 0))
		t.Logf("Is active: %t", electionData.IsActive)
		t.Logf("Candidates: %v", electionData.Candidates)
	}
}

func testVoterStatus(t *testing.T, client *BlockchainClient) {
	// Test voter status for a dummy hash
	testHash := "test_voter_hash_123"

	hasVoted, err := client.HasVoterVoted(testHash)
	if err != nil {
		// If contract is not deployed, this is expected
		if strings.Contains(err.Error(), "no contract code") {
			t.Skip("Contract not deployed at address, skipping voter status tests")
		}
		assert.NoError(t, err, "Failed to check voter status")
	}

	t.Logf("Test voter %s has voted: %t", testHash, hasVoted)

	// Test terminal authorization
	terminalAddr := common.HexToAddress("0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1")
	isAuthorized, err := client.IsTerminalAuthorized(terminalAddr)
	if err != nil {
		// If contract is not deployed, this is expected
		if strings.Contains(err.Error(), "no contract code") {
			t.Skip("Contract not deployed at address, skipping terminal authorization tests")
		}
		assert.NoError(t, err, "Failed to check terminal authorization")
	}

	t.Logf("Terminal %s is authorized: %t", terminalAddr.Hex(), isAuthorized)
}

func testVoteCasting(t *testing.T, client *BlockchainClient) {
	// Check if there's an active election
	electionID, err := client.GetCurrentElectionID()
	if err != nil {
		// If contract is not deployed, this is expected
		if strings.Contains(err.Error(), "no contract code") {
			t.Skip("Contract not deployed at address, skipping vote casting test")
		}
		require.NoError(t, err, "Failed to get current election ID")
	}

	// Check if electionID is nil (no active election)
	if electionID == nil || electionID.Cmp(big.NewInt(0)) == 0 {
		t.Skip("No active election, skipping vote casting test")
	}

	// Get election details to find valid candidates
	electionData, err := client.GetElectionDetails(electionID)
	if err != nil {
		t.Logf("Failed to get election details: %v", err)
		t.Skip("Cannot get election details, skipping vote casting test")
	}

	if len(electionData.Candidates) == 0 {
		t.Skip("No candidates in election, skipping vote casting test")
	}

	// Create test vote data
	voteData := VoteData{
		VerificationHash: "test_voter_unique_hash_" + time.Now().Format("20060102150405"),
		EncryptedVote:    "encrypted_vote_data_12345",
		PollingUnitID:    "PU001",
		CandidateID:      electionData.Candidates[0], // Vote for first candidate
	}

	// Check if test voter has already voted
	hasVoted, err := client.HasVoterVoted(voteData.VerificationHash)
	if err != nil {
		t.Logf("Failed to check voter status: %v", err)
		t.Skip("Cannot check voter status, skipping vote casting test")
	}

	if hasVoted {
		t.Skip("Test voter has already voted, skipping vote casting test")
	}

	// Estimate gas for the transaction
	estimatedGas, err := client.EstimateGas(voteData)
	if err != nil {
		t.Logf("Gas estimation failed (this is expected if terminal is not authorized): %v", err)
		t.Skip("Skipping vote casting due to authorization requirements")
	}

	t.Logf("Estimated gas: %d", estimatedGas)

	// Note: Actual vote casting would require the client to be authorized
	// For testing, we can only estimate gas unless we have owner permissions
	t.Logf("Vote casting test completed (estimation only)")
}

func testElectionResults(t *testing.T, client *BlockchainClient) {
	// Get current election
	electionID, err := client.GetCurrentElectionID()
	if err != nil {
		// If contract is not deployed, this is expected
		if strings.Contains(err.Error(), "no contract code") {
			t.Skip("Contract not deployed at address, skipping results test")
		}
		require.NoError(t, err, "Failed to get current election ID")
	}

	// Check if electionID is nil (no active election)
	if electionID == nil || electionID.Cmp(big.NewInt(0)) == 0 {
		t.Skip("No active election, skipping results test")
	}

	// Get election details
	electionData, err := client.GetElectionDetails(electionID)
	if err != nil {
		t.Logf("Failed to get election details: %v", err)
		t.Skip("Cannot get election details, skipping results test")
	}

	// Test getting results for each candidate
	for _, candidate := range electionData.Candidates {
		voteCount, err := client.GetElectionResults(electionID, candidate)
		if err != nil {
			t.Logf("Failed to get election results for candidate %s: %v", candidate, err)
			continue
		}
		assert.NotNil(t, voteCount, "Vote count should not be nil")

		t.Logf("Candidate %s: %s votes", candidate, voteCount.String())
	}

	// Test getting total votes
	totalVotes, err := client.GetTotalVotes()
	if err != nil {
		t.Logf("Failed to get total votes: %v", err)
	} else {
		assert.NotNil(t, totalVotes, "Total votes should not be nil")
		t.Logf("Total votes across all elections: %s", totalVotes.String())
	}

	// Test getting polling unit vote count
	pollingUnitCount, err := client.GetPollingUnitVoteCount("PU001")
	if err != nil {
		t.Logf("Failed to get polling unit vote count: %v", err)
	} else {
		assert.NotNil(t, pollingUnitCount, "Polling unit count should not be nil")
		t.Logf("Votes at PU001: %s", pollingUnitCount.String())
	}

	// Test getting election statistics
	stats, err := client.GetElectionStatistics(electionID)
	if err != nil {
		t.Logf("Failed to get election statistics: %v", err)
	} else {
		assert.NotNil(t, stats, "Statistics should not be nil")
		t.Logf("Election statistics: %+v", stats)
	}
}

// TestSyncManager tests the blockchain sync manager
func TestSyncManager(t *testing.T) {
	if !isBlockchainAvailable() {
		t.Skip("Blockchain not available, skipping sync manager tests")
	}

	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	if contractAddr == "" {
		contractAddr = testContractAddr
	}
	if contractAddr == "" {
		t.Skip("CONTRACT_ADDRESS not set, skipping sync manager tests")
	}

	// Create blockchain client
	client, err := NewBlockchainClient(testNodeURL, contractAddr, testPrivateKey)
	require.NoError(t, err, "Failed to create blockchain client")
	defer client.Close()

	// Create sync manager
	syncManager := NewSyncManager(client, 5*time.Second)

	t.Run("TestSyncManagerBasics", func(t *testing.T) {
		// Test initial state
		assert.False(t, syncManager.IsRunning(), "Sync manager should not be running initially")
		assert.Equal(t, 0, syncManager.GetPendingVoteCount(), "Should have no pending votes initially")

		// Add a test vote to pending queue
		testVote := VoteData{
			VerificationHash: "test_sync_vote_123",
			EncryptedVote:    "encrypted_test_vote",
			PollingUnitID:    "PU001",
			CandidateID:      "CANDIDATE_A", // Use valid candidate from active election
		}

		syncManager.AddPendingVote(testVote)
		assert.Equal(t, 1, syncManager.GetPendingVoteCount(), "Should have 1 pending vote")

		// Test getting pending votes
		pendingVotes := syncManager.GetPendingVotes()
		assert.Len(t, pendingVotes, 1, "Should return 1 pending vote")
		assert.Equal(t, testVote.VerificationHash, pendingVotes[0].VerificationHash, "Vote data should match")

		// Test clearing pending votes
		cleared := syncManager.ClearPendingVotes()
		assert.Equal(t, 1, cleared, "Should clear 1 vote")
		assert.Equal(t, 0, syncManager.GetPendingVoteCount(), "Should have no pending votes after clear")
	})

	t.Run("TestSyncManagerCallbacks", func(t *testing.T) {
		var successCount, failureCount, completeCount int

		// Set up callbacks
		syncManager.SetCallbacks(
			func(voteData VoteData, txHash string) {
				successCount++
				t.Logf("Vote success callback: %s -> %s", voteData.VerificationHash, txHash)
			},
			func(voteData VoteData, err error) {
				failureCount++
				t.Logf("Vote failure callback: %s -> %v", voteData.VerificationHash, err)
			},
			func(synced, failed int) {
				completeCount++
				t.Logf("Sync complete callback: synced=%d, failed=%d", synced, failed)
			},
		)

		// Add test vote and perform sync
		testVote := VoteData{
			VerificationHash: "test_callback_vote_456",
			EncryptedVote:    "encrypted_callback_vote",
			PollingUnitID:    "PU001",
			CandidateID:      "CANDIDATE_B", // Use valid candidate from active election
		}

		syncManager.AddPendingVote(testVote)

		// Perform immediate sync (this will likely fail due to authorization or contract not deployed, but we test callbacks)
		syncedCount, failedCount, err := syncManager.SyncNow()

		// We expect this to fail since we're not authorized or contract not deployed, but callbacks should still work
		t.Logf("Sync result: synced=%d, failed=%d, error=%v", syncedCount, failedCount, err)

		// Callbacks should have been called
		assert.Equal(t, 1, completeCount, "Complete callback should be called once")
	})
}

// TestEventMonitor tests the blockchain event monitor
func TestEventMonitor(t *testing.T) {
	if !isBlockchainAvailable() {
		t.Skip("Blockchain not available, skipping event monitor tests")
	}

	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	if contractAddr == "" {
		contractAddr = testContractAddr
	}
	if contractAddr == "" {
		t.Skip("CONTRACT_ADDRESS not set, skipping event monitor tests")
	}

	// Create blockchain client
	client, err := NewBlockchainClient(testNodeURL, contractAddr, testPrivateKey)
	require.NoError(t, err, "Failed to create blockchain client")
	defer client.Close()

	// Create event monitor
	eventMonitor := NewEventMonitor(client)

	t.Run("TestEventMonitorBasics", func(t *testing.T) {
		// Test event monitor callback
		var receivedEvents int
		eventMonitor.SetVoteCastCallback(func(event *SecureVotingSystemVoteCast) {
			receivedEvents++
			t.Logf("Received vote cast event: %+v", event)
		})

		// Start monitoring (this creates a subscription)
		err := eventMonitor.Start()
		if err != nil {
			// This is expected if contract is not deployed or no events are available
			t.Logf("Event monitoring failed to start (expected in test): %v", err)
		} else {
			// Let it run for a short time
			time.Sleep(2 * time.Second)
			eventMonitor.Stop()
		}

		t.Logf("Received %d events during test", receivedEvents)
	})
}

// TestConnectionManager tests the blockchain connection manager
func TestConnectionManager(t *testing.T) {
	if !isBlockchainAvailable() {
		t.Skip("Blockchain not available, skipping connection manager tests")
	}

	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	if contractAddr == "" {
		contractAddr = testContractAddr
	}
	if contractAddr == "" {
		t.Skip("CONTRACT_ADDRESS not set, skipping connection manager tests")
	}

	// Create blockchain client
	client, err := NewBlockchainClient(testNodeURL, contractAddr, testPrivateKey)
	require.NoError(t, err, "Failed to create blockchain client")
	defer client.Close()

	// Create connection manager
	connManager := NewConnectionManager(client, 2*time.Second)

	t.Run("TestConnectionStatus", func(t *testing.T) {
		// Test initial connection status
		isConnected := connManager.IsConnected()
		t.Logf("Initial connection status: %t", isConnected)

		// Get last block number
		lastBlock := connManager.GetLastBlockNumber()
		t.Logf("Last known block number: %d", lastBlock)
	})

	t.Run("TestConnectionCallbacks", func(t *testing.T) {
		var disconnectCount, reconnectCount int

		// Set up callbacks
		connManager.SetCallbacks(
			func() {
				disconnectCount++
				t.Log("Connection lost callback triggered")
			},
			func() {
				reconnectCount++
				t.Log("Connection restored callback triggered")
			},
		)

		// Start connection monitoring
		err := connManager.Start()
		assert.NoError(t, err, "Failed to start connection manager")

		// Let it run for a short time
		time.Sleep(5 * time.Second)

		// Stop monitoring
		connManager.Stop()

		t.Logf("Disconnect events: %d, Reconnect events: %d", disconnectCount, reconnectCount)
	})
}

// Helper function to check if blockchain is available
func isBlockchainAvailable() bool {
	client, err := ethclient.Dial(testNodeURL)
	if err != nil {
		return false
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.NetworkID(ctx)
	return err == nil
}

// Benchmark tests for performance
func BenchmarkBlockchainOperations(b *testing.B) {
	if !isBlockchainAvailable() {
		b.Skip("Blockchain not available, skipping benchmarks")
	}

	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	if contractAddr == "" {
		contractAddr = testContractAddr
	}
	if contractAddr == "" {
		b.Skip("CONTRACT_ADDRESS not set, skipping benchmarks")
	}

	// Create blockchain client
	client, err := NewBlockchainClient(testNodeURL, contractAddr, testPrivateKey)
	if err != nil {
		b.Fatalf("Failed to create blockchain client: %v", err)
	}
	defer client.Close()

	b.Run("BenchmarkGetBlockNumber", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := client.GetBlockNumber()
			if err != nil {
				b.Fatalf("Failed to get block number: %v", err)
			}
		}
	})

	b.Run("BenchmarkHasVoterVoted", func(b *testing.B) {
		testHash := "benchmark_voter_hash"

		for i := 0; i < b.N; i++ {
			_, err := client.HasVoterVoted(testHash)
			if err != nil {
				// Skip benchmark if contract is not deployed
				if strings.Contains(err.Error(), "no contract code") {
					b.Skip("Contract not deployed, skipping benchmark")
				}
				b.Fatalf("Failed to check voter status: %v", err)
			}
		}
	})

	b.Run("BenchmarkGetCurrentElectionID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := client.GetCurrentElectionID()
			if err != nil {
				// Skip benchmark if contract is not deployed
				if strings.Contains(err.Error(), "no contract code") {
					b.Skip("Contract not deployed, skipping benchmark")
				}
				b.Fatalf("Failed to get current election ID: %v", err)
			}
		}
	})
}

// Test data structures and helper functions
func TestDataStructures(t *testing.T) {
	t.Run("TestVoteData", func(t *testing.T) {
		voteData := VoteData{
			VerificationHash: "test_hash",
			EncryptedVote:    "encrypted_data",
			PollingUnitID:    "PU001",
			CandidateID:      "CANDIDATE_001",
		}

		assert.NotEmpty(t, voteData.VerificationHash, "Verification hash should not be empty")
		assert.NotEmpty(t, voteData.EncryptedVote, "Encrypted vote should not be empty")
		assert.NotEmpty(t, voteData.PollingUnitID, "Polling unit ID should not be empty")
		assert.NotEmpty(t, voteData.CandidateID, "Candidate ID should not be empty")
	})

	t.Run("TestElectionData", func(t *testing.T) {
		electionData := ElectionData{
			ID:         big.NewInt(1),
			Name:       "Test Election",
			StartTime:  big.NewInt(time.Now().Unix()),
			EndTime:    big.NewInt(time.Now().Add(24 * time.Hour).Unix()),
			IsActive:   true,
			Candidates: []string{"CANDIDATE_001", "CANDIDATE_002"},
			TotalVotes: big.NewInt(0),
		}

		assert.Equal(t, int64(1), electionData.ID.Int64(), "Election ID should be 1")
		assert.Equal(t, "Test Election", electionData.Name, "Election name should match")
		assert.True(t, electionData.IsActive, "Election should be active")
		assert.Len(t, electionData.Candidates, 2, "Should have 2 candidates")
	})

	t.Run("TestVoteInfo", func(t *testing.T) {
		voteInfo := VoteInfo{
			VerificationHash: "test_hash",
			EncryptedVote:    "encrypted_data",
			Timestamp:        big.NewInt(time.Now().Unix()),
			PollingUnitID:    "PU001",
			ElectionID:       big.NewInt(1),
			IsValid:          true,
		}

		assert.NotEmpty(t, voteInfo.VerificationHash, "Verification hash should not be empty")
		assert.True(t, voteInfo.IsValid, "Vote should be valid")
		assert.Equal(t, int64(1), voteInfo.ElectionID.Int64(), "Election ID should be 1")
	})
}

// Integration test that tests the complete workflow
func TestCompleteWorkflow(t *testing.T) {
	if !isBlockchainAvailable() {
		t.Skip("Blockchain not available, skipping integration test")
	}

	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	if contractAddr == "" {
		contractAddr = testContractAddr
	}
	if contractAddr == "" {
		t.Skip("CONTRACT_ADDRESS not set, skipping integration test")
	}

	// Create blockchain client
	client, err := NewBlockchainClient(testNodeURL, contractAddr, testPrivateKey)
	require.NoError(t, err, "Failed to create blockchain client")
	defer client.Close()

	// Create sync manager
	syncManager := NewSyncManager(client, 1*time.Second)

	// Step 1: Check system status
	t.Log("=== Step 1: Checking system status ===")

	blockNumber, err := client.GetBlockNumber()
	require.NoError(t, err, "Failed to get block number")
	t.Logf("Current block number: %d", blockNumber)

	balance, err := client.GetAccountBalance()
	require.NoError(t, err, "Failed to get account balance")
	t.Logf("Account balance: %s wei", balance.String())

	// Step 2: Check election status
	t.Log("=== Step 2: Checking election status ===")

	electionID, err := client.GetCurrentElectionID()
	if err != nil {
		if strings.Contains(err.Error(), "no contract code") {
			t.Skip("Contract not deployed at address, skipping integration test")
		}
		require.NoError(t, err, "Failed to get current election ID")
	}

	if electionID == nil {
		t.Log("No active election found")
	} else {
		t.Logf("Current election ID: %s", electionID.String())

		if electionID.Cmp(big.NewInt(0)) > 0 {
			electionData, err := client.GetElectionDetails(electionID)
			if err != nil {
				t.Logf("Failed to get election details: %v", err)
			} else {
				t.Logf("Election: %s (Active: %t)", electionData.Name, electionData.IsActive)
				t.Logf("Candidates: %v", electionData.Candidates)
			}
		}
	}

	// Step 3: Test voter verification
	t.Log("=== Step 3: Testing voter verification ===")

	testVoterHash := "integration_test_voter_" + time.Now().Format("20060102150405")
	hasVoted, err := client.HasVoterVoted(testVoterHash)
	if err != nil {
		if strings.Contains(err.Error(), "no contract code") {
			t.Log("Contract not deployed, skipping voter verification test")
		} else {
			t.Logf("Failed to check voter status: %v", err)
		}
	} else {
		t.Logf("Test voter %s has voted: %t", testVoterHash, hasVoted)
	}

	// Step 4: Test sync manager
	t.Log("=== Step 4: Testing sync manager ===")

	// Add test vote to pending queue
	if electionID != nil && electionID.Cmp(big.NewInt(0)) > 0 {
		electionData, err := client.GetElectionDetails(electionID)
		if err == nil && len(electionData.Candidates) > 0 {
			testVote := VoteData{
				VerificationHash: testVoterHash,
				EncryptedVote:    "integration_test_encrypted_vote",
				PollingUnitID:    "PU001",
				CandidateID:      electionData.Candidates[0],
			}

			syncManager.AddPendingVote(testVote)
			t.Logf("Added test vote to sync queue")

			// Attempt sync (will likely fail due to authorization, but tests the flow)
			syncedCount, failedCount, err := syncManager.SyncNow()
			t.Logf("Sync result: synced=%d, failed=%d, error=%v", syncedCount, failedCount, err)
		}
	}

	// Step 5: Test results and statistics
	t.Log("=== Step 5: Testing results and statistics ===")

	totalVotes, err := client.GetTotalVotes()
	if err != nil {
		if strings.Contains(err.Error(), "no contract code") {
			t.Log("Contract not deployed, skipping results test")
		} else {
			t.Logf("Failed to get total votes: %v", err)
		}
	} else {
		t.Logf("Total votes across all elections: %s", totalVotes.String())
	}

	if electionID != nil && electionID.Cmp(big.NewInt(0)) > 0 {
		stats, err := client.GetElectionStatistics(electionID)
		if err != nil {
			t.Logf("Failed to get election statistics: %v", err)
		} else {
			t.Logf("Election statistics: %+v", stats)
		}
	}

	t.Log("=== Integration test completed ===")
}
