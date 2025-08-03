# SSE Package API Documentation

## Overview

The SSE (Server-Sent Events) package provides a high-performance implementation for real-time event streaming in Go applications. It supports concurrent connections, event broadcasting, and graceful shutdown capabilities.

## Core Types

### Event

Represents a Server-Sent Event with optional type, data, and ID fields.

```go
type Event struct {
    Type string      `json:"type,omitempty"`
    Data interface{} `json:"data"`
    ID   string      `json:"id,omitempty"`
}
```

**Fields:**
- `Type`: Optional event type identifier
- `Data`: Event payload (string, []byte, or JSON-serializable)
- `ID`: Optional event ID for client-side event tracking

### Config

Configuration for the SSE server.

```go
type Config struct {
    MaxConnections    int           `json:"max_connections"`
    RetryTimeout      int           `json:"retry_timeout"`      // milliseconds
    HeartbeatInterval time.Duration `json:"heartbeat_interval"`
    BufferSize        int           `json:"buffer_size"`
}
```

**Fields:**
- `MaxConnections`: Maximum number of concurrent connections
- `RetryTimeout`: Client retry timeout in milliseconds
- `HeartbeatInterval`: Interval for heartbeat events
- `BufferSize`: Buffer size for event channels

### Server

Main SSE server instance that manages connections and event broadcasting.

```go
type Server struct {
    // Private fields
}
```

## Functions

### NewServer()

Creates a new SSE server with default configuration.

```go
func NewServer() *Server
```

**Returns:** A configured SSE server instance

**Example:**
```go
server := sse.NewServer()
```

### NewServerWithConfig(config Config)

Creates a new SSE server with custom configuration.

```go
func NewServerWithConfig(config Config) *Server
```

**Parameters:**
- `config`: Custom configuration for the server

**Returns:** A configured SSE server instance

**Example:**
```go
config := sse.Config{
    MaxConnections:    1000,
    RetryTimeout:      3000,
    HeartbeatInterval: 30 * time.Second,
    BufferSize:        1024,
}
server := sse.NewServerWithConfig(config)
```

### DefaultConfig()

Returns the default configuration.

```go
func DefaultConfig() Config
```

**Returns:** Default configuration values

## Server Methods

### HandleSSE(w http.ResponseWriter, r *http.Request)

Handles incoming SSE connections. Use this as your HTTP handler.

```go
func (s *Server) HandleSSE(w http.ResponseWriter, r *http.Request)
```

**Parameters:**
- `w`: HTTP response writer
- `r`: HTTP request

**Example:**
```go
http.HandleFunc("/events", server.HandleSSE)
```

### Broadcast(event Event)

Broadcasts an event to all connected clients.

```go
func (s *Server) Broadcast(event Event)
```

**Parameters:**
- `event`: Event to broadcast

**Example:**
```go
server.Broadcast(sse.Event{
    Type: "notification",
    Data: "Hello, World!",
    ID:   "msg-123",
})
```

### BroadcastToType(eventType string, event Event)

Broadcasts an event only to clients subscribed to a specific event type.

```go
func (s *Server) BroadcastToType(eventType string, event Event)
```

**Parameters:**
- `eventType`: Target event type
- `event`: Event to broadcast

**Example:**
```go
server.BroadcastToType("chat", sse.Event{
    Type: "message",
    Data: "New chat message",
})
```

### GetConnectionCount() int

Returns the current number of active connections.

```go
func (s *Server) GetConnectionCount() int
```

**Returns:** Number of active connections

**Example:**
```go
count := server.GetConnectionCount()
fmt.Printf("Active connections: %d\n", count)
```

### Shutdown()

Gracefully shuts down the server and closes all connections.

```go
func (s *Server) Shutdown()
```

**Example:**
```go
// Graceful shutdown
server.Shutdown()
```

## Usage Examples

### Basic Usage

```go
package main

import (
    "log"
    "net/http"
    "time"
    
    "github.com/miladsoleymani/sse"
)

func main() {
    server := sse.NewServer()
    
    http.HandleFunc("/events", server.HandleSSE)
    
    // Broadcast events periodically
    go func() {
        ticker := time.NewTicker(2 * time.Second)
        defer ticker.Stop()
        
        for range ticker.C {
            server.Broadcast(sse.Event{
                Type: "update",
                Data: "Server time: " + time.Now().Format(time.RFC3339),
            })
        }
    }()
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Advanced Usage with Custom Configuration

```go
package main

import (
    "log"
    "net/http"
    "time"
    
    "github.com/miladsoleymani/sse"
)

func main() {
    config := sse.Config{
        MaxConnections:    500,
        RetryTimeout:      5000,
        HeartbeatInterval: 60 * time.Second,
        BufferSize:        2048,
    }
    
    server := sse.NewServerWithConfig(config)
    
    // Add authentication middleware
    http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("Authorization") == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        server.HandleSSE(w, r)
    })
    
    // Monitor connection count
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        defer ticker.Stop()
        
        for range ticker.C {
            count := server.GetConnectionCount()
            log.Printf("Active connections: %d", count)
        }
    }()
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Client-Side JavaScript

```javascript
const eventSource = new EventSource('/events');

eventSource.onopen = function(event) {
    console.log('Connection established');
};

eventSource.onmessage = function(event) {
    console.log('Received:', event.data);
};

eventSource.addEventListener('notification', function(event) {
    console.log('Notification:', event.data);
});

eventSource.addEventListener('update', function(event) {
    console.log('Update:', event.data);
});

eventSource.onerror = function(event) {
    console.error('SSE error:', event);
    eventSource.close();
};
```

## Performance Considerations

- **Connection Limits**: Set appropriate `MaxConnections` based on your server capacity
- **Buffer Sizes**: Larger buffer sizes prevent blocking but use more memory
- **Heartbeat Intervals**: Shorter intervals keep connections alive but increase overhead
- **Event Frequency**: High-frequency events may require larger buffers

## Error Handling

The package handles common error scenarios:

- **Connection Limits**: Returns 503 when max connections reached
- **Streaming Support**: Returns 500 if response writer doesn't support flushing
- **Client Disconnection**: Automatically removes disconnected clients
- **Channel Overflow**: Removes clients when event channels are full

## Best Practices

1. **Graceful Shutdown**: Always call `Shutdown()` when stopping the server
2. **Connection Monitoring**: Use `GetConnectionCount()` to monitor server load
3. **Event Types**: Use meaningful event types for better client-side handling
4. **Error Recovery**: Implement client-side reconnection logic
5. **Resource Management**: Set appropriate connection and buffer limits

## Thread Safety

All public methods are thread-safe and can be called from multiple goroutines concurrently. 