package sse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	server := NewServer()
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	if server.config.MaxConnections != 1000 {
		t.Errorf("Expected MaxConnections to be 1000, got %d", server.config.MaxConnections)
	}
}

func TestNewServerWithConfig(t *testing.T) {
	config := Config{
		MaxConnections:    500,
		RetryTimeout:      5000,
		HeartbeatInterval: 60 * time.Second,
		BufferSize:        2048,
	}

	server := NewServerWithConfig(config)
	if server == nil {
		t.Fatal("NewServerWithConfig() returned nil")
	}

	if server.config.MaxConnections != 500 {
		t.Errorf("Expected MaxConnections to be 500, got %d", server.config.MaxConnections)
	}
}

func TestHandleSSE(t *testing.T) {
	server := NewServer()

	// Create test request
	req := httptest.NewRequest("GET", "/events", http.NoBody)
	w := httptest.NewRecorder()

	// Start SSE handler in goroutine
	go func() {
		server.HandleSSE(w, req)
	}()

	// Wait a bit for connection to establish
	time.Sleep(100 * time.Millisecond)

	// Check connection count
	count := server.GetConnectionCount()
	if count != 1 {
		t.Errorf("Expected 1 connection, got %d", count)
	}

	// Shutdown server
	server.Shutdown()
}

func TestBroadcast(t *testing.T) {
	server := NewServer()

	// Create test request
	req := httptest.NewRequest("GET", "/events", http.NoBody)
	w := httptest.NewRecorder()

	// Start SSE handler in goroutine
	go func() {
		server.HandleSSE(w, req)
	}()

	// Wait for connection to establish
	time.Sleep(100 * time.Millisecond)

	// Broadcast an event
	event := Event{
		Type: "test",
		Data: "Hello, World!",
		ID:   "test-123",
	}

	server.Broadcast(event)

	// Wait for event to be processed
	time.Sleep(100 * time.Millisecond)

	// Shutdown server before reading response body to avoid race
	server.Shutdown()

	// Check response body contains the event
	body := w.Body.String()
	if !strings.Contains(body, "event: test") {
		t.Error("Response body does not contain event type")
	}

	if !strings.Contains(body, "data: Hello, World!") {
		t.Error("Response body does not contain event data")
	}

	if !strings.Contains(body, "id: test-123") {
		t.Error("Response body does not contain event ID")
	}
}

func TestBroadcastToType(t *testing.T) {
	server := NewServer()

	// Create test request
	req := httptest.NewRequest("GET", "/events", http.NoBody)
	w := httptest.NewRecorder()

	// Start SSE handler in goroutine
	go func() {
		server.HandleSSE(w, req)
	}()

	// Wait for connection to establish
	time.Sleep(100 * time.Millisecond)

	// Broadcast to specific type
	event := Event{
		Type: "notification",
		Data: "Test notification",
	}

	server.BroadcastToType("notification", event)

	// Wait for event to be processed
	time.Sleep(100 * time.Millisecond)

	// Shutdown server before reading response body to avoid race
	server.Shutdown()

	// Check response body contains the event
	body := w.Body.String()
	if !strings.Contains(body, "event: notification") {
		t.Error("Response body does not contain notification event type")
	}
}

func TestGetConnectionCount(t *testing.T) {
	server := NewServer()

	// Initially should have 0 connections
	count := server.GetConnectionCount()
	if count != 0 {
		t.Errorf("Expected 0 connections initially, got %d", count)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/events", http.NoBody)
	w := httptest.NewRecorder()

	// Start SSE handler in goroutine
	go func() {
		server.HandleSSE(w, req)
	}()

	// Wait for connection to establish
	time.Sleep(100 * time.Millisecond)

	// Should have 1 connection
	count = server.GetConnectionCount()
	if count != 1 {
		t.Errorf("Expected 1 connection, got %d", count)
	}

	server.Shutdown()
}

func TestShutdown(t *testing.T) {
	server := NewServer()

	// Create test request
	req := httptest.NewRequest("GET", "/events", http.NoBody)
	w := httptest.NewRecorder()

	// Start SSE handler in goroutine
	go func() {
		server.HandleSSE(w, req)
	}()

	// Wait for connection to establish
	time.Sleep(100 * time.Millisecond)

	// Verify connection exists
	count := server.GetConnectionCount()
	if count != 1 {
		t.Errorf("Expected 1 connection before shutdown, got %d", count)
	}

	// Shutdown server
	server.Shutdown()

	// Verify all connections are closed
	count = server.GetConnectionCount()
	if count != 0 {
		t.Errorf("Expected 0 connections after shutdown, got %d", count)
	}
}

func TestConcurrentConnections(t *testing.T) {
	server := NewServer()
	var wg sync.WaitGroup

	// Create multiple concurrent connections
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/events", http.NoBody)
			w := httptest.NewRecorder()

			server.HandleSSE(w, req)
		}()
	}

	// Wait for connections to establish
	time.Sleep(200 * time.Millisecond)

	// Check connection count
	count := server.GetConnectionCount()
	if count != 5 {
		t.Errorf("Expected 5 connections, got %d", count)
	}

	server.Shutdown()
}

func TestMaxConnections(t *testing.T) {
	config := Config{
		MaxConnections:    2,
		BufferSize:        1024,
		HeartbeatInterval: 30 * time.Second,
	}
	server := NewServerWithConfig(config)

	// Create first connection
	req1 := httptest.NewRequest("GET", "/events", http.NoBody)
	w1 := httptest.NewRecorder()

	go func() {
		server.HandleSSE(w1, req1)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create second connection
	req2 := httptest.NewRequest("GET", "/events", http.NoBody)
	w2 := httptest.NewRecorder()

	go func() {
		server.HandleSSE(w2, req2)
	}()

	time.Sleep(100 * time.Millisecond)

	// Try to create third connection (should be rejected)
	req3 := httptest.NewRequest("GET", "/events", http.NoBody)
	w3 := httptest.NewRecorder()

	server.HandleSSE(w3, req3)

	// Check that third connection was rejected
	if w3.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w3.Code)
	}

	server.Shutdown()
}

func TestEventDataTypes(t *testing.T) {
	server := NewServer()

	req := httptest.NewRequest("GET", "/events", http.NoBody)
	w := httptest.NewRecorder()

	go func() {
		server.HandleSSE(w, req)
	}()

	time.Sleep(100 * time.Millisecond)

	// Test string data
	server.Broadcast(Event{
		Type: "string",
		Data: "string data",
	})

	// Test JSON data
	server.Broadcast(Event{
		Type: "json",
		Data: map[string]interface{}{
			"key":    "value",
			"number": 42,
		},
	})

	// Test byte data
	server.Broadcast(Event{
		Type: "bytes",
		Data: []byte("byte data"),
	})

	time.Sleep(100 * time.Millisecond)

	// Shutdown server before reading response body to avoid race
	server.Shutdown()

	body := w.Body.String()

	// Check string data
	if !strings.Contains(body, "data: string data") {
		t.Error("String data not found in response")
	}

	// Check JSON data
	if !strings.Contains(body, `"key":"value"`) {
		t.Error("JSON data not found in response")
	}

	// Check byte data
	if !strings.Contains(body, "data: byte data") {
		t.Error("Byte data not found in response")
	}
}

func TestContextCancellation(t *testing.T) {
	server := NewServer()

	// Create request with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/events", http.NoBody).WithContext(ctx)
	w := httptest.NewRecorder()

	go func() {
		server.HandleSSE(w, req)
	}()

	time.Sleep(100 * time.Millisecond)

	// Verify connection exists
	count := server.GetConnectionCount()
	if count != 1 {
		t.Errorf("Expected 1 connection, got %d", count)
	}

	// Cancel context
	cancel()

	time.Sleep(100 * time.Millisecond)

	// Verify connection is removed
	count = server.GetConnectionCount()
	if count != 0 {
		t.Errorf("Expected 0 connections after context cancellation, got %d", count)
	}

	server.Shutdown()
}

func BenchmarkBroadcast(b *testing.B) {
	server := NewServer()

	// Create test connection
	req := httptest.NewRequest("GET", "/events", http.NoBody)
	w := httptest.NewRecorder()

	go func() {
		server.HandleSSE(w, req)
	}()

	time.Sleep(100 * time.Millisecond)

	event := Event{
		Type: "benchmark",
		Data: "test data",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		server.Broadcast(event)
	}

	server.Shutdown()
}

func BenchmarkMultipleConnections(b *testing.B) {
	server := NewServer()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/events", http.NoBody)
		w := httptest.NewRecorder()

		go func() {
			server.HandleSSE(w, req)
		}()
	}

	time.Sleep(100 * time.Millisecond)
	server.Shutdown()
}
