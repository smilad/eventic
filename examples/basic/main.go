package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/miladsoleymani/sse"
)

type Message struct {
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	User      string    `json:"user"`
}

func main() {
	// Create SSE server
	server := sse.NewServer()

	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("static")))

	// Handle SSE connections
	http.HandleFunc("/events", server.HandleSSE)

	// Handle message posting
	http.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var msg Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		msg.Timestamp = time.Now()

		// Broadcast message to all clients
		server.Broadcast(sse.Event{
			Type: "message",
			Data: msg,
		})

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Message sent")
	})

	// Start broadcasting system events
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			server.Broadcast(sse.Event{
				Type: "system",
				Data: fmt.Sprintf("Server heartbeat at %s", time.Now().Format(time.RFC3339)),
			})
		}
	}()

	// Start broadcasting user count updates
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			count := server.GetConnectionCount()
			server.Broadcast(sse.Event{
				Type: "stats",
				Data: map[string]interface{}{
					"connections": count,
					"timestamp":   time.Now().Unix(),
				},
			})
		}
	}()

	log.Println("Server starting on :8080")
	log.Println("Open http://localhost:8080 in your browser")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
