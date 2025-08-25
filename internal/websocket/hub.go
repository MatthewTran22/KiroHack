package websocket

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Timestamp int64                  `json:"timestamp"`
	ID        string                 `json:"id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
}

// Client represents a WebSocket client
type Client struct {
	ID           string
	UserID       string
	Conn         *websocket.Conn
	Send         chan Message
	Hub          *Hub
	Consultations map[string]bool // Track which consultations this client has joined
	mu           sync.RWMutex
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan Message

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Consultation rooms - maps consultation ID to clients
	consultations map[string]map[*Client]bool

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		broadcast:     make(chan Message),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		consultations: make(map[string]map[*Client]bool),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			
			log.Printf("Client %s connected (User: %s)", client.ID, client.UserID)
			
			// Send connection confirmation
			select {
			case client.Send <- Message{
				Type:      "connection_confirmed",
				Data:      map[string]string{"status": "connected"},
				Timestamp: time.Now().Unix(),
			}:
			default:
				close(client.Send)
				h.mu.Lock()
				delete(h.clients, client)
				h.mu.Unlock()
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				
				// Remove client from all consultations
				for consultationID := range client.Consultations {
					if clients, exists := h.consultations[consultationID]; exists {
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.consultations, consultationID)
						}
					}
				}
			}
			h.mu.Unlock()
			
			log.Printf("Client %s disconnected (User: %s)", client.ID, client.UserID)

		case message := <-h.broadcast:
			h.handleMessage(message)
		}
	}
}

// handleMessage processes incoming messages
func (h *Hub) handleMessage(message Message) {
	switch message.Type {
	case "join_consultation":
		h.handleJoinConsultation(message)
	case "leave_consultation":
		h.handleLeaveConsultation(message)
	case "chat_message":
		h.handleChatMessage(message)
	case "typing_start", "typing_stop":
		h.handleTypingIndicator(message)
	case "ping":
		h.handlePing(message)
	default:
		log.Printf("Unknown message type: %s", message.Type)
	}
}

// handleJoinConsultation handles client joining a consultation
func (h *Hub) handleJoinConsultation(message Message) {
	data, ok := message.Data.(map[string]interface{})
	if !ok {
		return
	}
	
	consultationID, ok := data["consultationId"].(string)
	if !ok {
		return
	}

	// Find the client
	var client *Client
	h.mu.RLock()
	for c := range h.clients {
		if c.UserID == message.UserID {
			client = c
			break
		}
	}
	h.mu.RUnlock()

	if client == nil {
		return
	}

	h.mu.Lock()
	// Add client to consultation
	if h.consultations[consultationID] == nil {
		h.consultations[consultationID] = make(map[*Client]bool)
	}
	h.consultations[consultationID][client] = true
	
	// Track consultation in client
	client.mu.Lock()
	client.Consultations[consultationID] = true
	client.mu.Unlock()
	h.mu.Unlock()

	log.Printf("Client %s joined consultation %s", client.ID, consultationID)
}

// handleLeaveConsultation handles client leaving a consultation
func (h *Hub) handleLeaveConsultation(message Message) {
	data, ok := message.Data.(map[string]interface{})
	if !ok {
		return
	}
	
	consultationID, ok := data["consultationId"].(string)
	if !ok {
		return
	}

	// Find the client
	var client *Client
	h.mu.RLock()
	for c := range h.clients {
		if c.UserID == message.UserID {
			client = c
			break
		}
	}
	h.mu.RUnlock()

	if client == nil {
		return
	}

	h.mu.Lock()
	// Remove client from consultation
	if clients, exists := h.consultations[consultationID]; exists {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.consultations, consultationID)
		}
	}
	
	// Remove consultation from client
	client.mu.Lock()
	delete(client.Consultations, consultationID)
	client.mu.Unlock()
	h.mu.Unlock()

	log.Printf("Client %s left consultation %s", client.ID, consultationID)
}

// handleChatMessage handles chat messages
func (h *Hub) handleChatMessage(message Message) {
	data, ok := message.Data.(map[string]interface{})
	if !ok {
		return
	}
	
	consultationID, ok := data["consultationId"].(string)
	if !ok {
		return
	}

	// Broadcast to all clients in the consultation
	h.mu.RLock()
	clients, exists := h.consultations[consultationID]
	if !exists {
		h.mu.RUnlock()
		return
	}

	// Create response message
	responseMessage := Message{
		Type:      "chat_message",
		Data:      data,
		Timestamp: time.Now().Unix(),
		UserID:    message.UserID,
		SessionID: consultationID,
	}

	for client := range clients {
		select {
		case client.Send <- responseMessage:
		default:
			close(client.Send)
			delete(h.clients, client)
			delete(clients, client)
		}
	}
	h.mu.RUnlock()
}

// handleTypingIndicator handles typing indicators
func (h *Hub) handleTypingIndicator(message Message) {
	data, ok := message.Data.(map[string]interface{})
	if !ok {
		return
	}
	
	consultationID, ok := data["consultationId"].(string)
	if !ok {
		return
	}

	// Broadcast to all other clients in the consultation
	h.mu.RLock()
	clients, exists := h.consultations[consultationID]
	if !exists {
		h.mu.RUnlock()
		return
	}

	responseMessage := Message{
		Type:      message.Type,
		Data:      map[string]interface{}{"userId": message.UserID},
		Timestamp: time.Now().Unix(),
		SessionID: consultationID,
	}

	for client := range clients {
		// Don't send typing indicator back to the sender
		if client.UserID != message.UserID {
			select {
			case client.Send <- responseMessage:
			default:
				close(client.Send)
				delete(h.clients, client)
				delete(clients, client)
			}
		}
	}
	h.mu.RUnlock()
}

// handlePing handles ping messages
func (h *Hub) handlePing(message Message) {
	// Find the client and send pong
	h.mu.RLock()
	for client := range h.clients {
		if client.UserID == message.UserID {
			select {
			case client.Send <- Message{
				Type:      "pong",
				Data:      nil,
				Timestamp: time.Now().Unix(),
			}:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
			break
		}
	}
	h.mu.RUnlock()
}

// BroadcastToConsultation broadcasts a message to all clients in a consultation
func (h *Hub) BroadcastToConsultation(consultationID string, message Message) {
	h.mu.RLock()
	clients, exists := h.consultations[consultationID]
	if !exists {
		h.mu.RUnlock()
		return
	}

	message.Timestamp = time.Now().Unix()
	message.SessionID = consultationID

	for client := range clients {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(h.clients, client)
			delete(clients, client)
		}
	}
	h.mu.RUnlock()
}

// BroadcastToUser broadcasts a message to a specific user
func (h *Hub) BroadcastToUser(userID string, message Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	message.Timestamp = time.Now().Unix()

	for client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
}

// GetConnectedUsers returns the number of connected users
func (h *Hub) GetConnectedUsers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetConsultationParticipants returns the number of participants in a consultation
func (h *Hub) GetConsultationParticipants(consultationID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if clients, exists := h.consultations[consultationID]; exists {
		return len(clients)
	}
	return 0
}

// NewClient creates a new WebSocket client
func NewClient(id, userID string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:            id,
		UserID:        userID,
		Conn:          conn,
		Send:          make(chan Message, 256),
		Hub:           hub,
		Consultations: make(map[string]bool),
	}
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	// Set read deadline and pong handler
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var message Message
		err := c.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Set user ID from client
		message.UserID = c.UserID
		message.Timestamp = time.Now().Unix()

		// Send to hub
		c.Hub.broadcast <- message
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}