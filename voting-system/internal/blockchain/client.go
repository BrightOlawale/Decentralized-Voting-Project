package blockchain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
)

// VoteData represents a vote to be cast
type VoteData struct {
	VerificationHash string
	EncryptedVote    string
	PollingUnitID    string
	CandidateID      string
}

// ElectionData represents election information
type ElectionData struct {
	ID         *big.Int
	Name       string
	StartTime  *big.Int
	EndTime    *big.Int
	IsActive   bool
	Candidates []string
	TotalVotes *big.Int
}

// VoteInfo represents detailed vote information
type VoteInfo struct {
	VerificationHash string
	EncryptedVote    string
	Timestamp        *big.Int
	PollingUnitID    string
	ElectionID       *big.Int
	IsValid          bool
}

// BlockchainClient handles all blockchain interactions
type BlockchainClient struct {
	client          *ethclient.Client
	contract        *SecureVotingSystem
	contractAddress common.Address
	privateKey      *ecdsa.PrivateKey
	auth            *bind.TransactOpts
	callOpts        *bind.CallOpts
	chainID         *big.Int
}

// NewBlockchainClient creates a new blockchain client
func NewBlockchainClient(nodeURL, contractAddress, privateKeyHex string) (*BlockchainClient, error) {
	// Connect to Ethereum node
	client, err := ethclient.Dial(nodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %v", err)
	}

	// Get chain ID
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %v", err)
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	// Create auth transactor
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth transactor: %v", err)
	}

	// Set gas parameters
	auth.GasLimit = uint64(3000000)         // 3M gas limit
	auth.GasPrice = big.NewInt(20000000000) // 20 Gwei

	// Parse contract address
	contractAddr := common.HexToAddress(contractAddress)

	// Create contract instance
	contract, err := NewSecureVotingSystem(contractAddr, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract instance: %v", err)
	}

	return &BlockchainClient{
		client:          client,
		contract:        contract,
		contractAddress: contractAddr,
		privateKey:      privateKey,
		auth:            auth,
		callOpts:        &bind.CallOpts{},
		chainID:         chainID,
	}, nil
}

// Close closes the blockchain client connection
func (bc *BlockchainClient) Close() {
	if bc.client != nil {
		bc.client.Close()
	}
}

// CastVote records a vote on the blockchain
func (bc *BlockchainClient) CastVote(voteData VoteData) (*types.Transaction, error) {
	// Convert verification hash and encrypted vote to bytes32
	verificationHash := [32]byte{}
	copy(verificationHash[:], crypto.Keccak256([]byte(voteData.VerificationHash)))

	encryptedVote := [32]byte{}
	copy(encryptedVote[:], crypto.Keccak256([]byte(voteData.EncryptedVote)))

	log.Printf("Casting vote - PollingUnit: %s, Candidate: %s",
		voteData.PollingUnitID, voteData.CandidateID)

	// Call the smart contract
	tx, err := bc.contract.CastVote(
		bc.auth,
		verificationHash,
		encryptedVote,
		voteData.PollingUnitID,
		voteData.CandidateID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to cast vote: %v", err)
	}

	log.Printf("Vote cast successfully. Transaction hash: %s", tx.Hash().Hex())
	return tx, nil
}

// HasVoterVoted checks if a voter has already voted
func (bc *BlockchainClient) HasVoterVoted(verificationHash string) (bool, error) {
	// Convert to bytes32
	hash := [32]byte{}
	copy(hash[:], crypto.Keccak256([]byte(verificationHash)))

	hasVoted, err := bc.contract.HasVoterVoted(bc.callOpts, hash)
	if err != nil {
		return false, fmt.Errorf("failed to check voter status: %v", err)
	}

	return hasVoted, nil
}

// GetCurrentElectionID returns the current active election ID
func (bc *BlockchainClient) GetCurrentElectionID() (*big.Int, error) {
	electionID, err := bc.contract.GetCurrentElectionId(bc.callOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get current election ID: %v", err)
	}
	return electionID, nil
}

// GetElectionDetails retrieves detailed information about an election
func (bc *BlockchainClient) GetElectionDetails(electionID *big.Int) (*ElectionData, error) {
	result, err := bc.contract.GetElectionDetails(
		bc.callOpts, electionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election details: %v", err)
	}

	return &ElectionData{
		ID:         electionID,
		Name:       result.Name,
		StartTime:  result.StartTime,
		EndTime:    result.EndTime,
		IsActive:   result.IsActive,
		Candidates: result.Candidates,
		TotalVotes: result.TotalVotes,
	}, nil
}

// GetElectionResults gets vote count for a specific candidate in an election
func (bc *BlockchainClient) GetElectionResults(electionID *big.Int, candidateID string) (*big.Int, error) {
	voteCount, err := bc.contract.GetElectionResults(bc.callOpts, electionID, candidateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election results: %v", err)
	}
	return voteCount, nil
}

// GetVoteDetails retrieves detailed information about a specific vote
func (bc *BlockchainClient) GetVoteDetails(voteID *big.Int) (*VoteInfo, error) {
	result, err := bc.contract.GetVoteDetails(
		bc.callOpts, voteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vote details: %v", err)
	}

	return &VoteInfo{
		VerificationHash: common.Bytes2Hex(result.VerificationHash[:]),
		EncryptedVote:    common.Bytes2Hex(result.EncryptedVote[:]),
		Timestamp:        result.Timestamp,
		PollingUnitID:    result.PollingUnitId,
		ElectionID:       result.ElectionId,
		IsValid:          result.IsValid,
	}, nil
}

// GetTotalVotes returns the total number of votes cast across all elections
func (bc *BlockchainClient) GetTotalVotes() (*big.Int, error) {
	totalVotes, err := bc.contract.GetTotalVotes(bc.callOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get total votes: %v", err)
	}
	return totalVotes, nil
}

// GetPollingUnitVoteCount returns the number of votes recorded at a specific polling unit
func (bc *BlockchainClient) GetPollingUnitVoteCount(pollingUnitID string) (*big.Int, error) {
	voteCount, err := bc.contract.GetPollingUnitVoteCount(bc.callOpts, pollingUnitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get polling unit vote count: %v", err)
	}
	return voteCount, nil
}

// IsTerminalAuthorized checks if a terminal address is authorized
func (bc *BlockchainClient) IsTerminalAuthorized(terminalAddress common.Address) (bool, error) {
	isAuthorized, err := bc.contract.IsTerminalAuthorized(bc.callOpts, terminalAddress)
	if err != nil {
		return false, fmt.Errorf("failed to check terminal authorization: %v", err)
	}
	return isAuthorized, nil
}

// WaitForTransaction waits for a transaction to be mined and returns the receipt
func (bc *BlockchainClient) WaitForTransaction(tx *types.Transaction) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	receipt, err := bind.WaitMined(ctx, bc.client, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction: %v", err)
	}

	if receipt.Status == types.ReceiptStatusFailed {
		return receipt, fmt.Errorf("transaction failed")
	}

	return receipt, nil
}

// GetTransactionStatus checks the status of a transaction
func (bc *BlockchainClient) GetTransactionStatus(txHash common.Hash) (*types.Receipt, error) {
	receipt, err := bc.client.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %v", err)
	}
	return receipt, nil
}

// GetBlockNumber returns the latest block number
func (bc *BlockchainClient) GetBlockNumber() (uint64, error) {
	blockNumber, err := bc.client.BlockNumber(context.Background())
	if err != nil {
		return 0, fmt.Errorf("failed to get block number: %v", err)
	}
	return blockNumber, nil
}

// EstimateGas estimates the gas required for a vote transaction
func (bc *BlockchainClient) EstimateGas(voteData VoteData) (uint64, error) {
	// Convert to bytes32
	verificationHash := [32]byte{}
	copy(verificationHash[:], crypto.Keccak256([]byte(voteData.VerificationHash)))

	encryptedVote := [32]byte{}
	copy(encryptedVote[:], crypto.Keccak256([]byte(voteData.EncryptedVote)))

	// Create a copy of auth for estimation (don't modify original)
	authCopy := *bc.auth
	authCopy.NoSend = true

	// Estimate gas
	_, err := bc.contract.CastVote(
		&authCopy,
		verificationHash,
		encryptedVote,
		voteData.PollingUnitID,
		voteData.CandidateID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate gas: %v", err)
	}

	return authCopy.GasLimit, nil
}

// GetElectionStatistics returns comprehensive statistics for an election
func (bc *BlockchainClient) GetElectionStatistics(electionID *big.Int) (map[string]interface{}, error) {
	result, err := bc.contract.GetElectionStatistics(
		bc.callOpts, electionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get election statistics: %v", err)
	}

	return map[string]interface{}{
		"total_votes":   result.TotalVotes,
		"valid_votes":   result.ValidVotes,
		"invalid_votes": result.InvalidVotes,
		"duration":      result.Duration,
		"is_completed":  result.IsCompleted,
	}, nil
}

// GetVotesByTimeRange retrieves vote IDs within a specific time range
func (bc *BlockchainClient) GetVotesByTimeRange(startTime, endTime *big.Int) ([]*big.Int, error) {
	voteIDs, err := bc.contract.GetVotesByTimeRange(bc.callOpts, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get votes by time range: %v", err)
	}
	return voteIDs, nil
}

// SubscribeToVoteEvents subscribes to vote casting events
func (bc *BlockchainClient) SubscribeToVoteEvents(ch chan<- *SecureVotingSystemVoteCast) (event.Subscription, error) {
	opts := &bind.WatchOpts{
		Start:   nil, // Start from latest block
		Context: context.Background(),
	}

	sub, err := bc.contract.WatchVoteCast(opts, ch, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to vote events: %v", err)
	}

	return sub, nil
}

// Helper function to convert string to bytes32
func stringToBytes32(s string) [32]byte {
	var result [32]byte
	copy(result[:], []byte(s))
	return result
}

// Helper function to convert bytes32 to string
func bytes32ToString(b [32]byte) string {
	return string(b[:])
}

// GetAccountBalance returns the balance of the client's account
func (bc *BlockchainClient) GetAccountBalance() (*big.Int, error) {
	address := crypto.PubkeyToAddress(bc.privateKey.PublicKey)
	balance, err := bc.client.BalanceAt(context.Background(), address, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get account balance: %v", err)
	}
	return balance, nil
}

// UpdateGasPrice updates the gas price for transactions
func (bc *BlockchainClient) UpdateGasPrice() error {
	gasPrice, err := bc.client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get suggested gas price: %v", err)
	}

	// Add 10% buffer to suggested gas price
	buffer := new(big.Int).Div(gasPrice, big.NewInt(10))
	bc.auth.GasPrice = new(big.Int).Add(gasPrice, buffer)

	log.Printf("Updated gas price to: %s wei", bc.auth.GasPrice.String())
	return nil
}
