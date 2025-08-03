# SSE (Server-Sent Events) Package

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Code Coverage](https://img.shields.io/badge/coverage-87%25-brightgreen.svg)](https://github.com/smilad/eventic)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)](https://github.com/smilad/eventic)
[![Race Detection](https://img.shields.io/badge/race--detection-passing-brightgreen.svg)](https://github.com/smilad/eventic)
[![Linting](https://img.shields.io/badge/linting-passing-brightgreen.svg)](https://github.com/smilad/eventic)
[![Static Analysis](https://img.shields.io/badge/static--analysis-passing-brightgreen.svg)](https://github.com/smilad/eventic)

A professional Go package for implementing Server-Sent Events (SSE) with high performance, reliability, and ease of use.

## 📊 Project Status

| Metric | Status |
|--------|--------|
| **Test Coverage** | 87% |
| **Go Version** | 1.21+ |
| **License** | MIT |
| **Build Status** | ✅ Passing |
| **Code Quality** | ✅ Linting Passed |

## Features

- **High Performance**: Efficient event broadcasting with minimal memory footprint
- **Connection Management**: Automatic connection cleanup and health monitoring
- **Event Types**: Support for different event types and custom data formats
- **Middleware Support**: Easy integration with existing HTTP frameworks
- **Graceful Shutdown**: Proper cleanup on server shutdown
- **Thread Safe**: Concurrent-safe operations for multiple goroutines
- **Customizable**: Configurable retry policies and connection limits

## 🚀 Installation

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

```bash
go get github.com/smilad/eventic
```

## ⚡ Quick Start

### Basic Usage

```go
package main

import (
    "log"
    "net/http"
    "time"
    
    "github.com/smilad/eventic"
)

func main() {
    // Create SSE server
    sseServer := sse.NewServer()
    
    // Handle SSE connections
    http.HandleFunc("/events", sseServer.HandleSSE)
    
    // Start broadcasting events
    go func() {
        ticker := time.NewTicker(2 * time.Second)
        defer ticker.Stop()
        
        for range ticker.C {
            sseServer.Broadcast(sse.Event{
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
    
    "github.com/smilad/eventic"
)

func main() {
    // Create SSE server with custom configuration
    config := sse.Config{
        MaxConnections: 1000,
        RetryTimeout:   3000, // milliseconds
        HeartbeatInterval: 30 * time.Second,
        BufferSize:     1024,
    }
    
    sseServer := sse.NewServerWithConfig(config)
    
    // Add middleware for authentication
    http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
        // Your authentication logic here
        if r.Header.Get("Authorization") == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        sseServer.HandleSSE(w, r)
    })
    
    // Broadcast different types of events
    go func() {
        for {
            time.Sleep(5 * time.Second)
            
            // Broadcast to all clients
            sseServer.Broadcast(sse.Event{
                Type: "heartbeat",
                Data: "ping",
            })
            
            // Broadcast to specific event type
            sseServer.BroadcastToType("notification", sse.Event{
                Type: "notification",
                Data: "New message received",
            })
        }
    }()
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## API Reference

### Server

#### `NewServer() *Server`
Creates a new SSE server with default configuration.

#### `NewServerWithConfig(config Config) *Server`
Creates a new SSE server with custom configuration.

#### `Server.HandleSSE(w http.ResponseWriter, r *http.Request)`
Handles incoming SSE connections. Use this as your HTTP handler.

#### `Server.Broadcast(event Event)`
Broadcasts an event to all connected clients.

#### `Server.BroadcastToType(eventType string, event Event)`
Broadcasts an event only to clients subscribed to a specific event type.

#### `Server.GetConnectionCount() int`
Returns the current number of active connections.

#### `Server.Shutdown()`
Gracefully shuts down the server and closes all connections.

### Event

```go
type Event struct {
    Type string      // Event type (optional)
    Data interface{} // Event data
    ID   string      // Event ID (optional)
}
```

### Config

```go
type Config struct {
    MaxConnections    int           // Maximum number of concurrent connections
    RetryTimeout      int           // Retry timeout in milliseconds
    HeartbeatInterval time.Duration // Interval for heartbeat events
    BufferSize        int           // Buffer size for event channels
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

eventSource.onerror = function(event) {
    console.error('SSE error:', event);
    eventSource.close();
};
```

## Performance Considerations

- The package uses buffered channels to prevent blocking
- Automatic connection cleanup prevents memory leaks
- Configurable buffer sizes for optimal performance
- Efficient event broadcasting with minimal overhead

## 🧪 Testing

[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)](https://github.com/smilad/eventic)
[![Race Detection](https://img.shields.io/badge/race--detection-passing-brightgreen.svg)](https://github.com/smilad/eventic)

```bash
go test ./...
```

## 🤝 Contributing

[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)
[![Issues](https://img.shields.io/badge/issues-welcome-brightgreen.svg)](https://github.com/smilad/eventic/issues)

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## 📄 License

[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

MIT License - see LICENSE file for details. 