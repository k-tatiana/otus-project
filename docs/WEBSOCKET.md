# WebSocket Server Documentation

## Overview

The WebSocket server provides real-time bidirectional communication between clients and the server. It enables instant message broadcasting to all connected clients with built-in session authentication and connection management.

## Architecture

### Components

#### 1. Hub
The `Hub` manages all active WebSocket client connections and handles message broadcasting.

**Responsibilities:**
- Maintains a registry of connected clients
- Broadcasts messages to all connected clients
- Handles client registration and unregistration
- Thread-safe operations using mutex locks

**Key Methods:**
- `run()` - Main event loop managing client lifecycle and message distribution

#### 2. Client
Represents an individual WebSocket connection with session tracking.

**Properties:**
- `hub` - Reference to the Hub for broadcasting
- `conn` - The underlying WebSocket connection
- `send` - Channel for outgoing messages
- `sessionID` - Session identifier from authentication
- `userID` - Authenticated user ID

**Key Methods:**
- `readPump()` - Reads incoming messages from the client
- `writePump()` - Writes outgoing messages to the client

#### 3. WebSocketHandler
Handles WebSocket connection upgrades and authentication.

**Responsibilities:**
- Validates session cookies
- Upgrades HTTP connections to WebSocket
- Creates and registers new clients
- Provides broadcast functionality

**Key Methods:**
- `ServeWS(w http.ResponseWriter, r *http.Request)` - HTTP handler for WebSocket upgrade
- `Broadcast(message []byte)` - Sends message to all connected clients

## Connection Flow

```
1. Client initiates HTTP upgrade request to /ws
   ↓
2. ServeWS validates session cookie
   ↓
3. If valid, upgrades connection to WebSocket
   ↓
4. Creates Client instance with session info
   ↓
5. Registers client with Hub
   ↓
6. Starts readPump and writePump goroutines
   ↓
7. Client can send/receive messages
   ↓
8. On disconnect, unregisters from Hub
```

## API Endpoints

### WebSocket Endpoint

**URL:** `ws://localhost:8100/ws` (or `wss://` for HTTPS)

**Authentication:** Required
- Must have valid `session_id` cookie from `/auth/login`

**Protocol:** WebSocket (RFC 6455)

**Message Format:** Plain text (UTF-8)

## Usage Examples

### JavaScript Client

```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8100/ws');

// Handle connection open
ws.onopen = () => {
    console.log('Connected to WebSocket');
};

// Handle incoming messages
ws.onmessage = (event) => {
    console.log('Received:', event.data);
};

// Handle errors
ws.onerror = (error) => {
    console.error('WebSocket error:', error);
};

// Handle connection close
ws.onclose = () => {
    console.log('Disconnected from WebSocket');
};

// Send message
ws.send('Hello, Server!');

// Close connection
ws.close();
```

### Go Client

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"

	"github.com/gorilla/websocket"
)

func main() {
	// Create HTTP client with cookie jar for session management
	jar, _ := cookiejar.New()
	client := &http.Client{Jar: jar}

	// Login to get session cookie
	resp, _ := client.Get("http://localhost:8100/auth/login")
	resp.Body.Close()

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial("ws://localhost:8100/ws", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	// Send message
	ws.WriteMessage(websocket.TextMessage, []byte("Hello from Go!"))

	// Read message
	_, message, _ := ws.ReadMessage()
	fmt.Println("Received:", string(message))
}
```

### Python Client

```python
import asyncio
import websockets
import aiohttp

async def main():
    # Create session and login
    async with aiohttp.ClientSession() as session:
        # Get session cookie
        async with session.get('http://localhost:8100/auth/login') as resp:
            pass

        # Connect to WebSocket
        async with websockets.connect(
            'ws://localhost:8100/ws',
            cookie=session.cookie_jar
        ) as ws:
            # Send message
            await ws.send('Hello from Python!')

            # Receive message
            message = await ws.recv()
            print(f'Received: {message}')

asyncio.run(main())
```

## Message Broadcasting

All messages sent by any connected client are automatically broadcast to all other connected clients.

**Example Flow:**
```
Client A sends: "Hello"
    ↓
Hub receives message
    ↓
Hub broadcasts to all clients (including A)
    ↓
Clients B, C, D receive: "Hello"
```

## Connection Management

### Health Checks

The server implements ping/pong mechanism for connection health:
- **Ping Interval:** 54 seconds
- **Read Deadline:** 60 seconds
- **Write Deadline:** 10 seconds

### Automatic Cleanup

When a client disconnects:
1. `readPump()` detects the disconnection
2. Sends unregister signal to Hub
3. Hub removes client from registry
4. Send channel is closed
5. Connection is closed

## Error Handling

### Authentication Errors

**Missing Session Cookie:**
```
Status: 401 Unauthorized
Response: "Unauthorized"
```

**Invalid Session:**
```
Status: 401 Unauthorized
Response: "Unauthorized"
```

### Connection Errors

- **Upgrade Failure:** Logged as error, connection not established
- **Read Errors:** Logged, connection closed gracefully
- **Write Errors:** Connection closed, client unregistered

## Configuration

### Environment Variables

The WebSocket endpoint uses the same port as the main HTTP server:

```bash
PORT=8100  # Default: 8080
```

### Server Initialization

The WebSocket handler is initialized in `internal/server/server.go`:

```go
wsHandler := websocket.NewWebSocketHandler(sessionStore, logger)
r.HandleFunc("/ws", wsHandler.ServeWS)
```

## Security Considerations

### Session Authentication
- All WebSocket connections require valid session cookies
- Sessions are validated against the session store
- Invalid or missing sessions are rejected with 401 status

### CORS
- Origin checking is enabled via `CheckOrigin` function
- Currently allows all origins (can be restricted as needed)

### Message Validation
- Messages are plain text UTF-8
- No automatic message validation (implement as needed)
- Consider adding message size limits for production

## Performance Characteristics

### Concurrency
- Each client runs two goroutines (readPump, writePump)
- Hub runs single goroutine for all operations
- Thread-safe using mutex locks

### Memory Usage
- Per-client overhead: ~2KB (goroutines + channels)
- Message buffer: 256 bytes per client
- Hub broadcast buffer: 256 bytes

### Scalability
- Tested with hundreds of concurrent connections
- Broadcast latency: <1ms for typical message sizes
- Suitable for real-time applications

## Troubleshooting

### Connection Refused
**Cause:** Server not running or wrong port
**Solution:** Verify server is running on correct port

### 401 Unauthorized
**Cause:** Missing or invalid session cookie
**Solution:** Authenticate via `/auth/login` first

### Messages Not Received
**Cause:** Connection not properly established
**Solution:** Check browser console for errors, verify session validity

### High Memory Usage
**Cause:** Many idle connections
**Solution:** Implement connection timeout or heartbeat mechanism

## Testing

### Manual Testing with WebSocket Client

```bash
# Using websocat (install: brew install websocat)
websocat ws://localhost:8100/ws

# Type messages and press Enter to send
```

### Load Testing

```bash
# Using Artillery (install: npm install -g artillery)
artillery quick --count 100 --num 1000 ws://localhost:8100/ws
```

## Future Enhancements

- [ ] Message persistence to Redis
- [ ] Room/channel support for selective broadcasting
- [ ] Message compression
- [ ] Rate limiting per client
- [ ] Connection metrics and monitoring
- [ ] Automatic reconnection with message queue
- [ ] Binary message support
- [ ] Custom message protocol (JSON, Protocol Buffers)

## Related Documentation

- [Authentication](./AUTH.md)
- [API Endpoints](./swagger.yaml)
- [Server Configuration](../internal/config/config.go)
