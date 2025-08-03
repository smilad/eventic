// Package sse provides a high-performance Server-Sent Events (SSE) implementation
// for Go applications. It supports concurrent connections, event broadcasting,
// and graceful shutdown capabilities.
package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Event represents a Server-Sent Event
type Event struct {
	Type string      `json:"type,omitempty"`
	Data interface{} `json:"data"`
	ID   string      `json:"id,omitempty"`
}

// Config holds the configuration for the SSE server
type Config struct {
	MaxConnections    int           `json:"max_connections"`
	RetryTimeout      int           `json:"retry_timeout"` // milliseconds
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	BufferSize        int           `json:"buffer_size"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		MaxConnections:    1000,
		RetryTimeout:      3000,
		HeartbeatInterval: 30 * time.Second,
		BufferSize:        1024,
	}
}

// Client represents a connected SSE client
type Client struct {
	ID      string
	EventCh chan Event
	Type    string
	conn    http.ResponseWriter
	mu      sync.Mutex
	closed  bool
	server  *Server
}

// Server represents the SSE server
type Server struct {
	config        Config
	clients       map[string]*Client
	clientsByType map[string]map[string]*Client
	mu            sync.RWMutex
	shutdown      chan struct{}
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewServer creates a new SSE server with default configuration
func NewServer() *Server {
	return NewServerWithConfig(DefaultConfig())
}

// NewServerWithConfig creates a new SSE server with custom configuration
func NewServerWithConfig(config Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config:        config,
		clients:       make(map[string]*Client),
		clientsByType: make(map[string]map[string]*Client),
		shutdown:      make(chan struct{}),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start heartbeat goroutine
	go server.heartbeat()

	return server
}

// HandleSSE handles incoming SSE connections
func (s *Server) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Check if connection supports flushing
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Check connection limit
	s.mu.RLock()
	if len(s.clients) >= s.config.MaxConnections {
		s.mu.RUnlock()
		http.Error(w, "Too many connections", http.StatusServiceUnavailable)
		return
	}
	s.mu.RUnlock()

	// Create client
	clientID := generateClientID()
	client := &Client{
		ID:      clientID,
		EventCh: make(chan Event, s.config.BufferSize),
		conn:    w,
		server:  s,
	}

	// Register client
	s.mu.Lock()
	s.clients[clientID] = client
	s.mu.Unlock()

	// Send initial connection event
	initialEvent := Event{
		Type: "connection",
		Data: map[string]interface{}{
			"client_id": clientID,
			"timestamp": time.Now().Unix(),
		},
	}

	if err := s.sendEventToClient(client, initialEvent); err != nil {
		s.removeClient(clientID)
		return
	}

	// Handle client events
	for {
		select {
		case event := <-client.EventCh:
			if err := s.sendEventToClient(client, event); err != nil {
				s.removeClient(clientID)
				return
			}
		case <-s.ctx.Done():
			s.removeClient(clientID)
			return
		case <-r.Context().Done():
			s.removeClient(clientID)
			return
		}
	}
}

// Broadcast sends an event to all connected clients
func (s *Server) Broadcast(event Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		select {
		case client.EventCh <- event:
		default:
			// Channel is full, remove client
			go s.removeClient(client.ID)
		}
	}
}

// BroadcastToType sends an event only to clients subscribed to a specific event type
func (s *Server) BroadcastToType(eventType string, event Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// For now, broadcast to all clients since we don't have type-based subscription
	// In a real implementation, you would track client subscriptions by type
	for _, client := range s.clients {
		select {
		case client.EventCh <- event:
		default:
			// Channel is full, remove client
			go s.removeClient(client.ID)
		}
	}
}

// GetConnectionCount returns the current number of active connections
func (s *Server) GetConnectionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// Shutdown gracefully shuts down the server and closes all connections
func (s *Server) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel context to stop heartbeat
	s.cancel()

	// Close all client connections
	for _, client := range s.clients {
		client.close()
	}

	// Clear maps
	s.clients = make(map[string]*Client)
	s.clientsByType = make(map[string]map[string]*Client)

	// Signal shutdown
	close(s.shutdown)
}

// sendEventToClient sends an event to a specific client
func (s *Server) sendEventToClient(client *Client, event Event) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.closed {
		return fmt.Errorf("client connection closed")
	}

	// Format event according to SSE specification
	var eventStr string

	if event.ID != "" {
		eventStr += fmt.Sprintf("id: %s\n", event.ID)
	}

	if event.Type != "" {
		eventStr += fmt.Sprintf("event: %s\n", event.Type)
	}

	// Convert data to string
	var dataStr string
	switch v := event.Data.(type) {
	case string:
		dataStr = v
	case []byte:
		dataStr = string(v)
	default:
		// Try to marshal as JSON
		if jsonData, err := json.Marshal(event.Data); err == nil {
			dataStr = string(jsonData)
		} else {
			dataStr = fmt.Sprintf("%v", event.Data)
		}
	}

	eventStr += fmt.Sprintf("data: %s\n\n", dataStr)

	// Write to connection
	if _, err := client.conn.Write([]byte(eventStr)); err != nil {
		return err
	}

	// Flush the response
	if flusher, ok := client.conn.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// removeClient removes a client from the server
func (s *Server) removeClient(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.clients[clientID]; exists {
		client.close()
		delete(s.clients, clientID)

		// Remove from type-specific maps
		for _, clients := range s.clientsByType {
			delete(clients, clientID)
		}
	}
}

// heartbeat sends periodic heartbeat events to keep connections alive
func (s *Server) heartbeat() {
	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.Broadcast(Event{
				Type: "heartbeat",
				Data: time.Now().Unix(),
			})
		case <-s.ctx.Done():
			return
		}
	}
}

// close closes the client connection
func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		close(c.EventCh)
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}
