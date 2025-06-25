package blockchain

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// SyncManager handles blockchain synchronization operations
type SyncManager struct {
	client         *BlockchainClient
	syncInterval   time.Duration
	retryInterval  time.Duration
	maxRetries     int
	isRunning      bool
	stopChan       chan struct{}
	pendingVotes   []VoteData
	mutex          sync.RWMutex
	onVoteSuccess  func(voteData VoteData, txHash string)
	onVoteFailed   func(voteData VoteData, err error)
	onSyncComplete func(syncedCount int, failedCount int)
}

// NewSyncManager creates a new blockchain sync manager
func NewSyncManager(client *BlockchainClient, syncInterval time.Duration) *SyncManager {
	return &SyncManager{
		client:        client,
		syncInterval:  syncInterval,
		retryInterval: 30 * time.Second,
		maxRetries:    3,
		isRunning:     false,
		stopChan:      make(chan struct{}),
		pendingVotes:  make([]VoteData, 0),
	}
}

// SetCallbacks sets callback functions for sync events
func (sm *SyncManager) SetCallbacks(
	onSuccess func(VoteData, string),
	onFailed func(VoteData, error),
	onComplete func(int, int),
) {
	sm.onVoteSuccess = onSuccess
	sm.onVoteFailed = onFailed
	sm.onSyncComplete = onComplete
}

// AddPendingVote adds a vote to the pending queue for sync
func (sm *SyncManager) AddPendingVote(voteData VoteData) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.pendingVotes = append(sm.pendingVotes, voteData)
	log.Printf("Added vote to pending queue. Total pending: %d", len(sm.pendingVotes))
}

// GetPendingVoteCount returns the number of pending votes
func (sm *SyncManager) GetPendingVoteCount() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return len(sm.pendingVotes)
}

// Start begins the sync process
func (sm *SyncManager) Start() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.isRunning {
		return fmt.Errorf("sync manager is already running")
	}

	sm.isRunning = true
	go sm.syncLoop()

	log.Printf("Blockchain sync manager started with interval: %v", sm.syncInterval)
	return nil
}

// Stop stops the sync process
func (sm *SyncManager) Stop() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if !sm.isRunning {
		return
	}

	close(sm.stopChan)
	sm.isRunning = false

	log.Println("Blockchain sync manager stopped")
}

// IsRunning returns whether the sync manager is currently running
func (sm *SyncManager) IsRunning() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.isRunning
}

// SyncNow performs an immediate sync operation
func (sm *SyncManager) SyncNow() (int, int, error) {
	return sm.performSync()
}

// syncLoop is the main sync loop that runs periodically
func (sm *SyncManager) syncLoop() {
	ticker := time.NewTicker(sm.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if sm.GetPendingVoteCount() > 0 {
				syncedCount, failedCount, err := sm.performSync()
				if err != nil {
					log.Printf("Sync error: %v", err)
				}

				if sm.onSyncComplete != nil {
					sm.onSyncComplete(syncedCount, failedCount)
				}
			}

		case <-sm.stopChan:
			log.Println("Sync loop stopped")
			return
		}
	}
}

// performSync performs the actual synchronization of pending votes
func (sm *SyncManager) performSync() (int, int, error) {
	sm.mutex.Lock()
	pendingVotes := make([]VoteData, len(sm.pendingVotes))
	copy(pendingVotes, sm.pendingVotes)
	sm.mutex.Unlock()

	if len(pendingVotes) == 0 {
		return 0, 0, nil
	}

	log.Printf("Starting sync of %d pending votes", len(pendingVotes))

	var syncedCount, failedCount int
	var successfulIndices []int

	for i, voteData := range pendingVotes {
		success := sm.syncSingleVote(voteData, 0)
		if success {
			syncedCount++
			successfulIndices = append(successfulIndices, i)
		} else {
			failedCount++
		}
	}

	// Remove successfully synced votes from pending queue
	if len(successfulIndices) > 0 {
		sm.removeSyncedVotes(successfulIndices)
	}

	log.Printf("Sync completed. Synced: %d, Failed: %d, Remaining: %d",
		syncedCount, failedCount, sm.GetPendingVoteCount())

	return syncedCount, failedCount, nil
}

// syncSingleVote attempts to sync a single vote with retry logic
func (sm *SyncManager) syncSingleVote(voteData VoteData, retryCount int) bool {
	// Check if voter has already voted (to prevent double submission)
	hasVoted, err := sm.client.HasVoterVoted(voteData.VerificationHash)
	if err != nil {
		log.Printf("Error checking voter status: %v", err)
		if retryCount < sm.maxRetries {
			time.Sleep(sm.retryInterval)
			return sm.syncSingleVote(voteData, retryCount+1)
		}
		return false
	}

	if hasVoted {
		log.Printf("Voter has already voted, removing from queue: %s", voteData.VerificationHash)
		if sm.onVoteSuccess != nil {
			sm.onVoteSuccess(voteData, "already_voted")
		}
		return true // Remove from queue since vote is already recorded
	}

	// Attempt to cast the vote
	tx, err := sm.client.CastVote(voteData)
	if err != nil {
		log.Printf("Failed to cast vote (attempt %d/%d): %v", retryCount+1, sm.maxRetries+1, err)

		if retryCount < sm.maxRetries {
			time.Sleep(sm.retryInterval)
			return sm.syncSingleVote(voteData, retryCount+1)
		}

		if sm.onVoteFailed != nil {
			sm.onVoteFailed(voteData, err)
		}
		return false
	}

	// Wait for transaction to be mined
	receipt, err := sm.client.WaitForTransaction(tx)
	if err != nil {
		log.Printf("Transaction failed or timed out: %v", err)

		if retryCount < sm.maxRetries {
			time.Sleep(sm.retryInterval)
			return sm.syncSingleVote(voteData, retryCount+1)
		}

		if sm.onVoteFailed != nil {
			sm.onVoteFailed(voteData, err)
		}
		return false
	}

	log.Printf("Vote synced successfully. TX: %s, Gas used: %d",
		receipt.TxHash.Hex(), receipt.GasUsed)

	if sm.onVoteSuccess != nil {
		sm.onVoteSuccess(voteData, receipt.TxHash.Hex())
	}

	return true
}

// removeSyncedVotes removes successfully synced votes from the pending queue
func (sm *SyncManager) removeSyncedVotes(successfulIndices []int) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Remove indices in reverse order to maintain index validity
	for i := len(successfulIndices) - 1; i >= 0; i-- {
		index := successfulIndices[i]
		if index < len(sm.pendingVotes) {
			sm.pendingVotes = append(sm.pendingVotes[:index], sm.pendingVotes[index+1:]...)
		}
	}
}

// ClearPendingVotes clears all pending votes (use with caution)
func (sm *SyncManager) ClearPendingVotes() int {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	count := len(sm.pendingVotes)
	sm.pendingVotes = make([]VoteData, 0)

	log.Printf("Cleared %d pending votes", count)
	return count
}

// GetPendingVotes returns a copy of the pending votes slice
func (sm *SyncManager) GetPendingVotes() []VoteData {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	pendingCopy := make([]VoteData, len(sm.pendingVotes))
	copy(pendingCopy, sm.pendingVotes)

	return pendingCopy
}

// EventMonitor monitors blockchain events
type EventMonitor struct {
	client     *BlockchainClient
	isRunning  bool
	stopChan   chan struct{}
	onVoteCast func(*SecureVotingSystemVoteCast)
	mutex      sync.RWMutex
}

// NewEventMonitor creates a new blockchain event monitor
func NewEventMonitor(client *BlockchainClient) *EventMonitor {
	return &EventMonitor{
		client:    client,
		isRunning: false,
		stopChan:  make(chan struct{}),
	}
}

// SetVoteCastCallback sets the callback for vote cast events
func (em *EventMonitor) SetVoteCastCallback(callback func(*SecureVotingSystemVoteCast)) {
	em.onVoteCast = callback
}

// Start begins monitoring blockchain events
func (em *EventMonitor) Start() error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	if em.isRunning {
		return fmt.Errorf("event monitor is already running")
	}

	em.isRunning = true
	go em.monitorEvents()

	log.Println("Blockchain event monitor started")
	return nil
}

// Stop stops the event monitor
func (em *EventMonitor) Stop() {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	if !em.isRunning {
		return
	}

	close(em.stopChan)
	em.isRunning = false

	log.Println("Blockchain event monitor stopped")
}

// monitorEvents monitors blockchain events in a separate goroutine
func (em *EventMonitor) monitorEvents() {
	// Create a channel for vote cast events
	voteCastChan := make(chan *SecureVotingSystemVoteCast)

	// Subscribe to vote cast events
	sub, err := em.client.SubscribeToVoteEvents(voteCastChan)
	if err != nil {
		log.Printf("Failed to subscribe to vote events: %v", err)
		return
	}
	defer sub.Unsubscribe()

	for {
		select {
		case event := <-voteCastChan:
			if em.onVoteCast != nil {
				em.onVoteCast(event)
			}
			log.Printf("Vote cast event received: %s", event.Raw.TxHash.Hex())

		case err := <-sub.Err():
			log.Printf("Event subscription error: %v", err)
			// Attempt to reconnect after a delay
			time.Sleep(10 * time.Second)
			return // Exit and let the calling code decide whether to restart

		case <-em.stopChan:
			log.Println("Event monitor stopped")
			return
		}
	}
}

// ConnectionManager manages blockchain connection health
type ConnectionManager struct {
	client          *BlockchainClient
	checkInterval   time.Duration
	isRunning       bool
	stopChan        chan struct{}
	lastBlockNumber uint64
	onDisconnected  func()
	onReconnected   func()
	mutex           sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(client *BlockchainClient, checkInterval time.Duration) *ConnectionManager {
	return &ConnectionManager{
		client:        client,
		checkInterval: checkInterval,
		isRunning:     false,
		stopChan:      make(chan struct{}),
	}
}

// SetCallbacks sets the disconnect/reconnect callbacks
func (cm *ConnectionManager) SetCallbacks(onDisconnected, onReconnected func()) {
	cm.onDisconnected = onDisconnected
	cm.onReconnected = onReconnected
}

// Start begins connection monitoring
func (cm *ConnectionManager) Start() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if cm.isRunning {
		return fmt.Errorf("connection manager is already running")
	}

	cm.isRunning = true
	go cm.monitorConnection()

	log.Printf("Connection manager started with check interval: %v", cm.checkInterval)
	return nil
}

// Stop stops the connection monitor
func (cm *ConnectionManager) Stop() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if !cm.isRunning {
		return
	}

	close(cm.stopChan)
	cm.isRunning = false

	log.Println("Connection manager stopped")
}

// monitorConnection monitors the blockchain connection health
func (cm *ConnectionManager) monitorConnection() {
	ticker := time.NewTicker(cm.checkInterval)
	defer ticker.Stop()

	isConnected := true

	for {
		select {
		case <-ticker.C:
			blockNumber, err := cm.client.GetBlockNumber()
			if err != nil {
				if isConnected {
					log.Printf("Blockchain connection lost: %v", err)
					isConnected = false
					if cm.onDisconnected != nil {
						cm.onDisconnected()
					}
				}
			} else {
				if !isConnected {
					log.Printf("Blockchain connection restored. Current block: %d", blockNumber)
					isConnected = true
					if cm.onReconnected != nil {
						cm.onReconnected()
					}
				}
				cm.lastBlockNumber = blockNumber
			}

		case <-cm.stopChan:
			log.Println("Connection monitor stopped")
			return
		}
	}
}

// IsConnected returns the current connection status
func (cm *ConnectionManager) IsConnected() bool {
	_, err := cm.client.GetBlockNumber()
	return err == nil
}

// GetLastBlockNumber returns the last known block number
func (cm *ConnectionManager) GetLastBlockNumber() uint64 {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.lastBlockNumber
}
