package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	sse "github.com/smilad/eventic"
)

// User represents a connected user
type User struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	JoinedAt time.Time `json:"joined_at"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Room      string    `json:"room"`
}

// Server represents our application server
type Server struct {
	sseServer *sse.Server
	users     map[string]*User
	rooms     map[string][]string // room -> user IDs
	mu        sync.RWMutex
}

// NewServer creates a new server instance
func NewServer() *Server {
	config := sse.Config{
		MaxConnections:    1000,
		RetryTimeout:      3000,
		HeartbeatInterval: 30 * time.Second,
		BufferSize:        1024,
	}

	return &Server{
		sseServer: sse.NewServerWithConfig(config),
		users:     make(map[string]*User),
		rooms:     make(map[string][]string),
	}
}

// AuthMiddleware checks for valid authentication
func (s *Server) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Simple token validation (in production, use proper JWT)
		if !strings.HasPrefix(token, "Bearer ") {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		userID := strings.TrimPrefix(token, "Bearer ")
		s.mu.RLock()
		_, exists := s.users[userID]
		s.mu.RUnlock()

		if !exists {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user ID to request context
		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// HandleSSE handles SSE connections with authentication
func (s *Server) HandleSSE(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	// Get user info
	s.mu.RLock()
	_, exists := s.users[userID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Add user to default room
	s.mu.Lock()
	if s.rooms["general"] == nil {
		s.rooms["general"] = []string{}
	}
	s.rooms["general"] = append(s.rooms["general"], userID)
	s.mu.Unlock()

	// Handle SSE connection
	s.sseServer.HandleSSE(w, r)

	// Remove user from room when connection closes
	s.mu.Lock()
	if roomUsers, exists := s.rooms["general"]; exists {
		for i, id := range roomUsers {
			if id == userID {
				s.rooms["general"] = append(roomUsers[:i], roomUsers[i+1:]...)
				break
			}
		}
	}
	s.mu.Unlock()
}

// HandleLogin handles user login
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var login struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if login.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Generate user ID
	userID := generateUserID()

	user := &User{
		ID:       userID,
		Name:     login.Name,
		JoinedAt: time.Now(),
	}

	s.mu.Lock()
	s.users[userID] = user
	s.mu.Unlock()

	// Broadcast user joined event
	s.sseServer.Broadcast(sse.Event{
		Type: "user_joined",
		Data: map[string]interface{}{
			"user": user,
			"room": "general",
		},
	})

	// Return user info and token
	response := map[string]interface{}{
		"user":  user,
		"token": userID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleMessage handles chat messages
func (s *Server) HandleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value("userID").(string)

	var msg struct {
		Message string `json:"message"`
		Room    string `json:"room"`
	}

	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if msg.Message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	if msg.Room == "" {
		msg.Room = "general"
	}

	// Get user info
	s.mu.RLock()
	user, exists := s.users[userID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Create chat message
	chatMsg := ChatMessage{
		ID:        generateMessageID(),
		UserID:    userID,
		UserName:  user.Name,
		Message:   msg.Message,
		Timestamp: time.Now(),
		Room:      msg.Room,
	}

	// Broadcast message to room
	s.sseServer.BroadcastToType("chat_"+msg.Room, sse.Event{
		Type: "chat_message",
		Data: chatMsg,
	})

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Message sent")
}

// HandleGetUsers returns list of online users
func (s *Server) HandleGetUsers(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

// StartStatsBroadcaster starts broadcasting server statistics
func (s *Server) StartStatsBroadcaster() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		userCount := len(s.users)
		roomCount := len(s.rooms)
		s.mu.RUnlock()

		connectionCount := s.sseServer.GetConnectionCount()

		s.sseServer.Broadcast(sse.Event{
			Type: "stats",
			Data: map[string]interface{}{
				"users":       userCount,
				"connections": connectionCount,
				"rooms":       roomCount,
				"timestamp":   time.Now().Unix(),
			},
		})
	}
}

// generateUserID generates a random user ID
func generateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// generateMessageID generates a random message ID
func generateMessageID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func main() {
	server := NewServer()

	// Create router
	r := mux.NewRouter()

	// Public routes
	r.HandleFunc("/login", server.HandleLogin).Methods("POST")
	r.HandleFunc("/users", server.HandleGetUsers).Methods("GET")

	// Protected routes
	r.HandleFunc("/events", server.AuthMiddleware(server.HandleSSE))
	r.HandleFunc("/message", server.AuthMiddleware(server.HandleMessage)).Methods("POST")

	// Serve static files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))

	// Start stats broadcaster
	go server.StartStatsBroadcaster()

	log.Println("Advanced SSE Server starting on :8080")
	log.Println("Open http://localhost:8080 in your browser")
	log.Fatal(http.ListenAndServe(":8080", r))
}
