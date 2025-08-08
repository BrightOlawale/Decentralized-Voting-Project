package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"voting-system/internal/api"
	"voting-system/internal/api/middlewares"
	"voting-system/internal/blockchain"
	"voting-system/internal/database"
	"voting-system/pkg/config"
	"voting-system/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Load configuration
	cfg, err := config.LoadConfig("configs/server.yaml")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Initialize logger
	logger := logger.NewLogger(cfg.Logging.Level, cfg.Logging.File)
	logger.Info("Starting Voting System Central Server...")

	// Initialize database
	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run database migrations
	if err := database.RunMigrations(db); err != nil {
		logger.Fatal("Failed to run migrations: %v", err)
	}
	logger.Info("Database initialized successfully")

	// Initialize blockchain client
	blockchainClient, err := blockchain.NewBlockchainClient(
		cfg.Blockchain.NetworkURL,
		cfg.Blockchain.ContractAddress,
		cfg.Blockchain.PrivateKey,
	)
	if err != nil {
		logger.Fatal("Failed to initialize blockchain client: %v", err)
	}
	defer blockchainClient.Close()
	logger.Info("Blockchain client initialized successfully")

	// Verify blockchain connection
	if err := verifyBlockchainConnection(blockchainClient, logger); err != nil {
		logger.Fatal("Blockchain connection verification failed: %v", err)
	}

	// Initialize sync manager
	syncManager := blockchain.NewSyncManager(blockchainClient, 30*time.Second)
	setupSyncCallbacks(syncManager, logger)

	// Initialize event monitor
	eventMonitor := blockchain.NewEventMonitor(blockchainClient)
	setupEventCallbacks(eventMonitor, logger)

	// Initialize connection manager
	connManager := blockchain.NewConnectionManager(blockchainClient, 10*time.Second)
	setupConnectionCallbacks(connManager, logger)

	// Create services
	services := api.NewServices(
		db,
		blockchainClient,
		syncManager,
		eventMonitor,
		connManager,
		logger,
		cfg,
	)

	// Initialize Gin router
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(logger.GinLogger())
	router.Use(middlewares.CORS())

	// Setup API routes
	api.SetupRoutes(router, services)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start background services
	logger.Info("Starting background services...")
	if err := syncManager.Start(); err != nil {
		logger.Error("Failed to start sync manager:", err)
	}
	if err := eventMonitor.Start(); err != nil {
		logger.Error("Failed to start event monitor:", err)
	}
	if err := connManager.Start(); err != nil {
		logger.Error("Failed to start connection manager:", err)
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Stop background services
	syncManager.Stop()
	eventMonitor.Stop()
	connManager.Stop()

	// Shutdown server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	logger.Info("Server shutdown completed")
}

func verifyBlockchainConnection(client *blockchain.BlockchainClient, logger *logger.Logger) error {
	// Check blockchain connection
	blockNumber, err := client.GetBlockNumber()
	if err != nil {
		return fmt.Errorf("failed to get block number: %v", err)
	}
	logger.Info("Connected to blockchain at block: %d", blockNumber)

	// Check contract deployment
	electionID, err := client.GetCurrentElectionID()
	if err != nil {
		return fmt.Errorf("failed to get current election ID: %v", err)
	}
	logger.Info("Current election ID: %s", electionID)

	// Check account balance
	balance, err := client.GetAccountBalance()
	if err != nil {
		return fmt.Errorf("failed to get account balance: %v", err)
	}
	logger.Info("Account balance: %s wei", balance.String())

	return nil
}

func setupSyncCallbacks(syncManager *blockchain.SyncManager, logger *logger.Logger) {
	syncManager.SetCallbacks(
		// On vote success
		func(voteData blockchain.VoteData, txHash string) {
			logger.Info("Vote synced successfully - hash: %s, tx: %s",
				voteData.VerificationHash, txHash)
		},
		// On vote failed
		func(voteData blockchain.VoteData, err error) {
			logger.Error("Vote sync failed - hash: %s, error: %v",
				voteData.VerificationHash, err.Error())
		},
		// On sync complete
		func(syncedCount, failedCount int) {
			if syncedCount > 0 || failedCount > 0 {
				logger.Info("Sync cycle completed - synced: %d, failed: %d",
					syncedCount, failedCount)
			}
		},
	)
}

func setupEventCallbacks(eventMonitor *blockchain.EventMonitor, logger *logger.Logger) {
	eventMonitor.SetVoteCastCallback(func(event *blockchain.SecureVotingSystemVoteCast) {
		logger.Info("Vote cast event received - electionId: %s, pollingUnit: %s, voteId: %s, txHash: %s",
			event.ElectionId.String(), event.PollingUnitId.String(), event.VoteId.String(), event.Raw.TxHash.Hex())
	})
}

func setupConnectionCallbacks(connManager *blockchain.ConnectionManager, logger *logger.Logger) {
	connManager.SetCallbacks(
		// On disconnected
		func() {
			logger.Error("Blockchain connection lost - entering offline mode")
		},
		// On reconnected
		func() {
			logger.Info("Blockchain connection restored - resuming online mode")
		},
	)
}
