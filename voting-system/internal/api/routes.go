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

	// Web interface routes
	setupWebRoutes(router, services)

	// Static file serving
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")
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

		// Voter registration (public endpoint)
		public.POST("/voter/register", handlers.RegisterVoter(services))

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
	authenticated := rg.Group("/")
	authenticated.Use(middlewares.AuthRequired(services))
	{
		// Voting endpoints
		voting := authenticated.Group("/voting")
		{
			voting.POST("/cast", handlers.CastVote(services))
			voting.GET("/status/:voter_hash", handlers.GetVoterStatus(services))
			voting.POST("/verify", handlers.VerifyVoter(services))
		}

		// Election endpoints
		election := authenticated.Group("/election")
		{
			election.GET("/:id/statistics", handlers.GetElectionStatistics(services))
			// election.GET("/:id/audit", handlers.GetElectionAudit(services))
		}

		// Terminal endpoints
		terminal := authenticated.Group("/terminal")
		{
			terminal.POST("/register", handlers.RegisterTerminal(services))
			terminal.GET("/:id/status", handlers.GetTerminalStatus(services))
			// terminal.POST("/:id/heartbeat", handlers.TerminalHeartbeat(services))
			// terminal.GET("/:id/config", handlers.GetTerminalConfig(services))
		}

		// // User profile
		// user := authenticated.Group("/user")
		// {
		// 	user.GET("/profile", handlers.GetUserProfile(services))
		// 	user.PUT("/profile", handlers.UpdateUserProfile(services))
		// 	user.POST("/logout", handlers.Logout(services))
		// }

		// Audit endpoints
		audit := authenticated.Group("/audit")
		{
			audit.GET("/logs", handlers.GetAuditLogs(services))
			audit.GET("/votes/:time_range", handlers.GetVotesByTimeRange(services))
		}
	}
}

// setupAdminRoutes configures admin-only routes
func setupAdminRoutes(rg *gin.RouterGroup, services interfaces.Services) {
	admin := rg.Group("/admin")
	admin.Use(middlewares.AuthRequired(services))
	admin.Use(middlewares.AdminRequired(services))
	{
		// Dashboard
		admin.GET("/dashboard", handlers.AdminDashboard(services))
		// admin.GET("/stats", handlers.GetAdminStats(services))

		// Election management
		elections := admin.Group("/elections")
		{
			elections.POST("/", handlers.CreateElection(services))
			// elections.PUT("/:id", handlers.UpdateElection(services))
			elections.POST("/:id/start", handlers.StartElection(services))
			elections.POST("/:id/end", handlers.EndElection(services))
			// elections.DELETE("/:id", handlers.DeleteElection(services))
			// elections.GET("/", handlers.ListElections(services))
		}

		// Terminal management
		terminals := admin.Group("/terminals")
		{
			// terminals.GET("/", handlers.ListTerminals(services))
			terminals.POST("/:id/authorize", handlers.AuthorizeTerminal(services))
			// terminals.POST("/:id/deauthorize", handlers.DeauthorizeTerminal(services))
			// terminals.DELETE("/:id", handlers.RemoveTerminal(services))
			// terminals.GET("/:id/logs", handlers.GetTerminalLogs(services))
		}

		// Vote management
		votes := admin.Group("/votes")
		{
			// votes.GET("/", handlers.ListVotes(services))
			// votes.GET("/:id", handlers.GetVoteDetails(services))
			votes.POST("/:id/invalidate", handlers.InvalidateVote(services))
			// votes.GET("/export", handlers.ExportVotes(services))
		}

		// System management
		system := admin.Group("/system")
		{
			system.POST("/sync", handlers.TriggerSync(services))
			// system.POST("/backup", handlers.CreateBackup(services))
			// system.GET("/config", handlers.GetSystemConfig(services))
			// system.PUT("/config", handlers.UpdateSystemConfig(services))
			// system.POST("/maintenance", handlers.MaintenanceMode(services))
		}

		// Blockchain management
		blockchain := admin.Group("/blockchain")
		{
			// blockchain.GET("/status", handlers.GetBlockchainStatus(services))
			// blockchain.GET("/transactions", handlers.ListTransactions(services))
			blockchain.GET("/contracts", handlers.GetContractInfo(services))
			// blockchain.POST("/redeploy", handlers.RedeployContract(services))
		}

		// // User management
		// users := admin.Group("/users")
		// {
		// 	users.GET("/", handlers.ListUsers(services))
		// 	users.POST("/", handlers.CreateUser(services))
		// 	users.PUT("/:id", handlers.UpdateUser(services))
		// 	users.DELETE("/:id", handlers.DeleteUser(services))
		// 	users.POST("/:id/reset-password", handlers.ResetUserPassword(services))
		// }

		// Audit and reporting
		reports := admin.Group("/reports")
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
		authenticated := ws.Group("/")
		authenticated.Use(middlewares.WSAuthRequired(services))
		{
			authenticated.GET("/votes", handlers.VoteUpdatesWebSocket(services))
			// authenticated.GET("/terminal/:id", handlers.TerminalWebSocket(services))
		}

		// Admin WebSocket endpoints
		admin := ws.Group("/admin")
		admin.Use(middlewares.WSAuthRequired(services))
		// admin.Use(middlewares.WSAdminRequired(services))
		{
			// admin.GET("/dashboard", handlers.AdminDashboardWebSocket(services))
			// admin.GET("/system", handlers.SystemMonitoringWebSocket(services))
			// admin.GET("/audit", handlers.AuditWebSocket(services))
		}
	}
}

// setupWebRoutes configures web interface routes
func setupWebRoutes(router *gin.Engine, services interfaces.Services) {
	web := router.Group("/web")
	web.Use(middlewares.WebAuth(services))
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
