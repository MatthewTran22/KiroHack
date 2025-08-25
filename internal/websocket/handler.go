package websocket

import (
	"log"
	"net/http"
	"strings"

	"ai-government-consultant/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from localhost for development
		origin := r.Header.Get("Origin")
		log.Printf("WebSocket connection attempt from origin: %s", origin)
		
		// Allow connections without origin (for testing tools like curl)
		if origin == "" {
			log.Printf("WebSocket connection allowed: no origin header (testing)")
			return true
		}
		
		// Allow localhost connections on common ports
		allowedOrigins := []string{
			"http://localhost:3000",
			"https://localhost:3000",
			"http://localhost:8080",
			"https://localhost:8080",
			"http://127.0.0.1:3000",
			"https://127.0.0.1:3000",
			"http://127.0.0.1:8080",
			"https://127.0.0.1:8080",
		}
		
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				log.Printf("WebSocket connection allowed from origin: %s", origin)
				return true
			}
		}
		
		// Also allow any localhost origin for development
		if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
			log.Printf("WebSocket connection allowed from localhost origin: %s", origin)
			return true
		}
		
		log.Printf("WebSocket connection rejected from origin: %s", origin)
		return false
	},
}

// Handler handles WebSocket connections
type Handler struct {
	hub         *Hub
	authService *auth.AuthService
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, authService *auth.AuthService) *Handler {
	return &Handler{
		hub:         hub,
		authService: authService,
	}
}

// HandleWebSocket handles WebSocket upgrade requests
func (h *Handler) HandleWebSocket(c *gin.Context) {
	log.Printf("WebSocket connection attempt from %s", c.ClientIP())
	
	// Get token from query parameter or header
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}
	}

	if token == "" {
		log.Printf("WebSocket connection rejected: No authentication token provided")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No authentication token provided"})
		return
	}

	// Validate token
	ctx := c.Request.Context()
	tokenValidation, err := h.authService.ValidateToken(ctx, token)
	if err != nil {
		log.Printf("WebSocket authentication failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
		return
	}

	log.Printf("WebSocket authentication successful for user: %s", tokenValidation.Claims.UserID)

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade connection"})
		return
	}

	// Create client
	clientID := uuid.New().String()
	client := NewClient(clientID, tokenValidation.Claims.UserID, conn, h.hub)

	// Register client with hub
	h.hub.register <- client

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()

	log.Printf("WebSocket connection established for user %s (client %s)", tokenValidation.Claims.UserID, clientID)
}

// GetHub returns the WebSocket hub
func (h *Handler) GetHub() *Hub {
	return h.hub
}