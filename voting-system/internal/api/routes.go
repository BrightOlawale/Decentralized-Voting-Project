package api

import (
	"voting-system/internal/api/handlers"
	"voting-system/internal/api/interfaces"
	"voting-system/internal/api/middlewares"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes with proper middleware
func SetupRoutes(router *gin.Engine, services interfaces.Services) {
	// Global middleware
	router.Use(middlewares.Recovery())
	router.Use(middlewares.CORS())
	router.Use(middlewares.Security())
	router.Use(middlewares.RequestLogging(services.GetLogger()))
	router.Use(middlewares.RateLimit())

	// Health check (no auth required)
	router.GET("/health", handlers.HealthCheck(services))
	router.GET("/ping", handlers.HealthCheck(services))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		setupPublicRoutes(v1, services)
		setupAuthenticatedRoutes(v1, services)
		setupAdminRoutes(v1, services)
		setupWebSocketRoutes(v1, services)
	}

	// Serve built admin webapp (Vite) from /webapp/admin
	// keep assets
	router.Static("/webapp/admin/assets", "./webapp/admin/dist/assets")
	// serve index for root
	router.GET("/webapp/admin", func(c *gin.Context) { c.File("webapp/admin/dist/index.html") })
	// REMOVE this line (conflicts with assets):
	// router.GET("/webapp/admin/*path", func(c *gin.Context) { c.File("webapp/admin/dist/index.html") })
}

// setupPublicRoutes configures routes that don't require authentication
func setupPublicRoutes(rg *gin.RouterGroup, services interfaces.Services) {
	public := rg.Group("/public")
	{
		// System information
		public.GET("/status", handlers.GetSystemStatus(services))
		public.GET("/election/current", handlers.GetCurrentElection(services))
		public.GET("/election/:id", handlers.GetElectionDetails(services))
		public.GET("/election/:id/results", handlers.GetElectionResults(services))
		public.GET("/election/:id/candidates", handlers.GetElectionCandidates(services))

		// Polling Unit
		public.GET("/polling-unit/:id", handlers.GetPollingUnitInfo(services))

		// Voter registration (public endpoint)
		public.POST("/voter/register", handlers.RegisterVoter(services))

		// Terminal token issuance (HMAC optional)
		public.POST("/token/terminal", handlers.IssueTerminalToken(services))

		// Authentication
		// auth := public.Group("/auth")
		// {
		// 	auth.POST("/login", handlers.Login(services))
		// 	auth.POST("/refresh", handlers.RefreshToken(services))
		// }
	}
}

// setupAuthenticatedRoutes configures routes that require authentication
func setupAuthenticatedRoutes(rg *gin.RouterGroup, services interfaces.Services) {
	// Voting endpoints (no auth)
	voting := rg.Group("/voting")
	{
		voting.POST("/cast", handlers.CastVote(services))
		voting.GET("/status/:voter_hash", handlers.GetVoterStatus(services))
		voting.POST("/verify", handlers.VerifyVoter(services))
	}

	// Election endpoints (no auth)
	election := rg.Group("/election")
	{
		election.GET("/:id/statistics", handlers.GetElectionStatistics(services))
		// election.GET("/:id/audit", handlers.GetElectionAudit(services))
	}

	// Terminal endpoints (no auth)
	terminal := rg.Group("/terminal")
	{
		terminal.POST("/register", handlers.RegisterTerminal(services))
		terminal.GET("/:id/status", handlers.GetTerminalStatus(services))
		terminal.POST("/polling-unit/ensure", handlers.EnsurePollingUnitTerminal(services))
		// terminal.POST("/:id/heartbeat", handlers.TerminalHeartbeat(services))
		// terminal.GET("/:id/config", handlers.GetTerminalConfig(services))
	}

	// Audit endpoints (no auth)
	audit := rg.Group("/audit")
	{
		audit.GET("/logs", handlers.GetAuditLogs(services))
		audit.GET("/votes/:time_range", handlers.GetVotesByTimeRange(services))
	}
}

// setupAdminRoutes configures admin-only routes
func setupAdminRoutes(rg *gin.RouterGroup, services interfaces.Services) {
	// Admin routes (no auth)
	{
		// Dashboard
		rg.GET("/admin/dashboard", handlers.AdminDashboard(services))
		// rg.GET("/admin/stats", handlers.GetAdminStats(services))

		// Election management
		elections := rg.Group("/admin/elections")
		{
			elections.POST("/", handlers.CreateElection(services))
			// elections.PUT("/:id", handlers.UpdateElection(services))
			elections.POST("/:id/start", handlers.StartElection(services))
			elections.POST("/:id/end", handlers.EndElection(services))
			// New: register candidates
			elections.POST("/:id/candidates", handlers.RegisterCandidates(services))
			// New: list and delete (DB only)
			elections.GET("/", handlers.ListElections(services))
			elections.DELETE("/", handlers.DeleteElection(services))
			// elections.DELETE("/:id", handlers.DeleteElection(services))
			// elections.GET("/", handlers.ListElections(services))
		}

		// Terminal management
		terminals := rg.Group("/admin/terminals")
		{
			// terminals.GET("/", handlers.ListTerminals(services))
			terminals.POST("/:id/authorize", handlers.AuthorizeTerminal(services))
			// terminals.POST("/:id/deauthorize", handlers.DeauthorizeTerminal(services))
			// terminals.DELETE("/:id", handlers.RemoveTerminal(services))
			// terminals.GET("/:id/logs", handlers.GetTerminalLogs(services))
		}

		// Vote management
		votes := rg.Group("/admin/votes")
		{
			// votes.GET("/", handlers.ListVotes(services))
			// votes.GET("/:id", handlers.GetVoteDetails(services))
			votes.POST("/:id/invalidate", handlers.InvalidateVote(services))
			// votes.GET("/export", handlers.ExportVotes(services))
		}

		// System management
		system := rg.Group("/admin/system")
		{
			system.POST("/sync", handlers.TriggerSync(services))
			// Register polling unit on-chain
			system.POST("/polling-unit", handlers.RegisterPollingUnit(services))
			// system.POST("/backup", handlers.CreateBackup(services))
			// system.GET("/config", handlers.GetSystemConfig(services))
			// system.PUT("/config", handlers.UpdateSystemConfig(services))
			// system.POST("/maintenance", handlers.MaintenanceMode(services))
		}

		// Blockchain management
		blockchain := rg.Group("/admin/blockchain")
		{
			// blockchain.GET("/status", handlers.GetBlockchainStatus(services))
			// blockchain.GET("/transactions", handlers.ListTransactions(services))
			blockchain.GET("/contracts", handlers.GetContractInfo(services))
			// blockchain.POST("/redeploy", handlers.RedeployContract(services))
		}

		// // User management
		// users := rg.Group("/admin/users")
		// {
		// 	users.GET("/", handlers.ListUsers(services))
		// 	users.POST("/", handlers.CreateUser(services))
		// 	users.PUT("/:id", handlers.UpdateUser(services))
		// 	users.DELETE("/:id", handlers.DeleteUser(services))
		// 	users.POST("/:id/reset-password", handlers.ResetUserPassword(services))
		// }

		// Audit and reporting
		reports := rg.Group("/admin/reports")
		{
			reports.GET("/audit/full", handlers.GetFullAuditLogs(services))
			// reports.GET("/election/:id/report", handlers.GetElectionReport(services))
			// reports.GET("/system/performance", handlers.GetPerformanceReport(services))
			// reports.GET("/security/events", handlers.GetSecurityEvents(services))
		}
	}
}

// setupWebSocketRoutes configures WebSocket endpoints
func setupWebSocketRoutes(rg *gin.RouterGroup, services interfaces.Services) {
	ws := rg.Group("/ws")
	{
		// Public WebSocket endpoints
		// ws.GET("/status", handlers.SystemStatusWebSocket(services))
		// ws.GET("/election/:id/results", handlers.ElectionResultsWebSocket(services))

		// Authenticated WebSocket endpoints
		ws.GET("/votes", handlers.VoteUpdatesWebSocket(services))

		// Admin WebSocket endpoints
		// admin := ws.Group("/admin")
		// admin.Use(middlewares.WSAdminRequired(services))
		// {
		//     admin.GET("/dashboard", handlers.AdminDashboardWebSocket(services))
		//     admin.GET("/system", handlers.SystemMonitoringWebSocket(services))
		//     admin.GET("/audit", handlers.AuditWebSocket(services))
		// }
	}
}

// setupWebRoutes configures web interface routes
func setupWebRoutes(router *gin.Engine, services interfaces.Services) {
	// Keep login but allow all web pages without auth
	router.GET("/web/login", handlers.WebLoginPage(services))
	router.POST("/web/login", handlers.WebLoginSubmit(services))

	web := router.Group("/web")
	{
		web.GET("/", handlers.WebDashboard(services))
		web.GET("/voting", handlers.WebVotingInterface(services))
		web.GET("/results", handlers.WebResultsPage(services))
		// web.GET("/terminal", handlers.WebTerminalInterface(services))

		// Admin web interface
		// admin := web.Group("/admin")
		// admin.Use(middlewares.WebAdminRequired(services))
		{
			// admin.GET("/", handlers.WebAdminPanel(services))
			// admin.GET("/elections", handlers.WebElectionManagement(services))
			// admin.GET("/terminals", handlers.WebTerminalManagement(services))
			// admin.GET("/users", handlers.WebUserManagement(services))
			// admin.GET("/reports", handlers.WebReports(services))
		}
	}
}
