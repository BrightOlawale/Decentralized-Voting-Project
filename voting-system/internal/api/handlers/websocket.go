package handlers

import (
	"encoding/json"
	"net/http"
	"time"
	"voting-system/internal/api/interfaces"
	"voting-system/internal/api/types"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for development
		// In production, implement proper origin checking
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// WebSocketHandler handles general system status WebSocket connections
func WebSocketHandler(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			services.GetLogger().Error("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		clientIP := getClientIP(c)
		services.GetLogger().Info("WebSocket connection established - client_ip: %s", clientIP)

		// Create a channel for this client
		clientChan := make(chan WebSocketMessage, 100)

		// Add client to broadcast list (implement a proper client manager)
		go handleWebSocketClient(conn, clientChan, services)

		// Send initial status
		initialStatus := WebSocketMessage{
			Type:      "system_status",
			Data:      getSystemStatus(services),
			Timestamp: time.Now().Unix(),
		}

		select {
		case clientChan <- initialStatus:
		default:
			// Channel full, close connection
			return
		}

		// Send periodic status updates
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				statusMsg := WebSocketMessage{
					Type:      "system_status",
					Data:      getSystemStatus(services),
					Timestamp: time.Now().Unix(),
				}

				select {
				case clientChan <- statusMsg:
				default:
					// Channel full, client might be slow
					services.GetLogger().Warning("WebSocket client channel full")
					return
				}

			case <-c.Request.Context().Done():
				services.GetLogger().Info("WebSocket client disconnected")
				return
			}
		}
	}
}

// VoteUpdatesWebSocket handles real-time vote update WebSocket connections
func VoteUpdatesWebSocket(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			services.GetLogger().Error("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		clientIP := getClientIP(c)
		services.GetLogger().Info("Vote updates WebSocket connection established - client_ip: %s", clientIP)

		// Create a channel for vote updates
		voteUpdateChan := make(chan WebSocketMessage, 100)

		// Handle the WebSocket client
		go handleWebSocketClient(conn, voteUpdateChan, services)

		// Set up vote event callback to send updates to this client
		// eventCallback := func(event *blockchain.SecureVotingSystemVoteCast) {
		// 	voteUpdate := WebSocketMessage{
		// 		Type: "vote_cast",
		// 		Data: map[string]interface{}{
		// 			"election_id":     event.ElectionId.String(),
		// 			"polling_unit_id": event.PollingUnitId.String(),
		// 			"vote_id":         event.VoteId.String(),
		// 			"timestamp":       event.Timestamp.Int64(),
		// 			"tx_hash":         event.Raw.TxHash.Hex(),
		// 			"block_number":    event.Raw.BlockNumber,
		// 		},
		// 		Timestamp: time.Now().Unix(),
		// 	}
		//
		// 	select {
		// 	case voteUpdateChan <- voteUpdate:
		// 	default:
		// 		// Channel full
		// 	}
		// }

		// Register the callback (in a real implementation, you'd have a proper event system)
		// For now, we'll just send periodic updates
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Send current election status
				currentElectionID, err := services.GetBlockchainClient().GetCurrentElectionID()
				if err == nil && currentElectionID != nil {
					electionData, err := services.GetBlockchainClient().GetElectionDetails(currentElectionID)
					if err == nil {
						updateMsg := WebSocketMessage{
							Type: "election_update",
							Data: map[string]interface{}{
								"election_id": currentElectionID.String(),
								"total_votes": electionData.TotalVotes.String(),
								"is_active":   electionData.IsActive,
							},
							Timestamp: time.Now().Unix(),
						}

						select {
						case voteUpdateChan <- updateMsg:
						default:
						}
					}
				}

			case <-c.Request.Context().Done():
				services.GetLogger().Info("Vote updates WebSocket client disconnected")
				return
			}
		}
	}
}

// handleWebSocketClient handles a WebSocket client connection
func handleWebSocketClient(conn *websocket.Conn, messageChan <-chan WebSocketMessage, services interfaces.Services) {
	// Set up ping/pong for connection health
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Handle incoming messages in a separate goroutine
	go func() {
		defer conn.Close()
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					services.GetLogger().Error("WebSocket error: %v", err)
				}
				break
			}

			if messageType == websocket.TextMessage {
				handleWebSocketMessage(message, services)
			}
		}
	}()

	// Send outgoing messages
	for {
		select {
		case message, ok := <-messageChan:
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(message); err != nil {
				services.GetLogger().Error("WebSocket write error: %v", err)
				return
			}

		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleWebSocketMessage processes incoming WebSocket messages
func handleWebSocketMessage(message []byte, services interfaces.Services) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		services.GetLogger().Error("Invalid WebSocket message: %v", err)
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "ping":
		// Client ping - respond with pong
		services.GetLogger().Debug("Received WebSocket ping")

	case "subscribe":
		// Client wants to subscribe to specific updates
		if topics, ok := msg["topics"].([]interface{}); ok {
			services.GetLogger().Info("Client subscribed to topics: %v", topics)
		}

	case "get_status":
		// Client requesting immediate status update
		services.GetLogger().Debug("Client requested status update")

	default:
		services.GetLogger().Warning("Unknown WebSocket message type: %s", msgType)
	}
}

// getSystemStatus returns current system status for WebSocket updates
func getSystemStatus(services interfaces.Services) map[string]interface{} {
	status := map[string]interface{}{
		"server_status":     "running",
		"blockchain_status": "disconnected",
		"pending_votes":     services.GetSyncManager().GetPendingVoteCount(),
		"sync_running":      services.GetSyncManager().IsRunning(),
		"current_election":  "none",
		"last_block":        uint64(0),
		"timestamp":         time.Now().Unix(),
	}

	// Update blockchain status
	if services.GetConnManager().IsConnected() {
		status["blockchain_status"] = "connected"

		if blockNumber, err := services.GetBlockchainClient().GetBlockNumber(); err == nil {
			status["last_block"] = blockNumber
		}

		if currentElectionID, err := services.GetBlockchainClient().GetCurrentElectionID(); err == nil && currentElectionID != nil {
			status["current_election"] = currentElectionID.String()
		}
	}

	return status
}

// AdminDashboard serves the admin dashboard page
func AdminDashboard(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin_dashboard.html", gin.H{
			"title": "Voting System Admin Dashboard",
		})
	}
}

// WebDashboard serves the main dashboard page
func WebDashboard(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"title": "Voting System Dashboard",
		})
	}
}

// WebVotingInterface serves the voting interface page
func WebVotingInterface(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get current election for the interface
		currentElectionID, err := services.GetBlockchainClient().GetCurrentElectionID()
		if err != nil {
			c.HTML(http.StatusOK, "error.html", gin.H{
				"title": "Error",
				"error": "Failed to load election data",
			})
			return
		}

		var election *types.ElectionInfo
		if currentElectionID != nil {
			electionData, err := services.GetBlockchainClient().GetElectionDetails(currentElectionID)
			if err == nil {
				election = &types.ElectionInfo{
					ID:         currentElectionID.String(),
					Name:       electionData.Name,
					StartTime:  electionData.StartTime.Int64(),
					EndTime:    electionData.EndTime.Int64(),
					IsActive:   electionData.IsActive,
					Candidates: electionData.Candidates,
					TotalVotes: electionData.TotalVotes.Int64(),
				}
			}
		}

		c.HTML(http.StatusOK, "voting_interface.html", gin.H{
			"title":    "Cast Your Vote",
			"election": election,
		})
	}
}

// WebResultsPage serves the election results page
func WebResultsPage(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "results.html", gin.H{
			"title": "Election Results",
		})
	}
}

// WebAdminPanel serves the admin panel page
func WebAdminPanel(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin_panel.html", gin.H{
			"title": "Admin Panel",
		})
	}
}

// WebLoginPage renders a simple login form
func WebLoginPage(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"title": "Login",
		})
	}
}

// WebLoginSubmit accepts username/password and sets a session cookie
func WebLoginSubmit(services interfaces.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		type creds struct {
			Username string `json:"username" form:"username"`
			Password string `json:"password" form:"password"`
		}
		var in creds
		if err := c.ShouldBind(&in); err != nil || in.Username == "" || in.Password == "" {
			c.HTML(http.StatusBadRequest, "login.html", gin.H{"title": "Login", "error": "Invalid credentials"})
			return
		}
		// TODO: real authentication; for now accept any non-empty
		// Set a simple session cookie (placeholder)
		c.SetCookie("session_id", "dev-session", 3600, "/", "", false, true)
		c.Redirect(http.StatusSeeOther, "/web/")
	}
}
