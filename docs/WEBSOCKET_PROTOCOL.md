# WebSocket Protocol

Real-time WebSocket communication protocol for Echo Backend - based on ACTUAL implementation.

## Overview

The Message Service provides WebSocket support for real-time bidirectional communication. The implementation uses the Gorilla WebSocket library with a Hub-based architecture for managing multiple client connections per user.

**WebSocket Endpoint**: `ws://localhost:8083/ws`

**Connection Limits**:
- Max message size: 1 MB
- Client send buffer: 256 messages
- Hub broadcast buffer: 1024 messages
- Ping interval: 45 seconds
- Connection timeout: 90 seconds

## Connection

### Establishing Connection

**Endpoint**: `GET /ws`

**Required Headers**:
```http
X-User-ID: 550e8400-e29b-41d4-a716-446655440000
```

**Optional Headers**:
```http
X-Device-ID: device-12345
X-Platform: ios
X-App-Version: 1.0.0
```

**JavaScript Example**:
```javascript
// Note: Browser WebSocket doesn't support custom headers
// Use query parameters or authenticate via token endpoint first

const userId = '550e8400-e29b-41d4-a716-446655440000';
const socket = new WebSocket(`ws://localhost:8083/ws`);

// Server extracts user_id from context (set by auth middleware)
```

**Go Example**:
```go
import (
    "github.com/gorilla/websocket"
    "net/http"
)

headers := http.Header{}
headers.Add("X-User-ID", userID)
headers.Add("X-Device-ID", deviceID)
headers.Add("X-Platform", "ios")

conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8083/ws", headers)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()
```

### Connection Established

Server sends acknowledgment when connection is established:

```json
{
  "type": "connection_ack",
  "payload": {
    "status": "connected",
    "timestamp": "2025-01-15T10:30:00Z",
    "client_id": "conn_abc123xyz"
  }
}
```

---

## Message Format

All WebSocket messages follow this format:

```json
{
  "type": "event_name",
  "payload": {
    // Event-specific data
  }
}
```

**Fields**:
- `type` (required, string): Event type identifier
- `payload` (required, object): Event data

---

## Client → Server Events

Events sent from client to server.

### 1. read_receipt

Mark a message as read.

**Type**: `read_receipt`

**Payload**:
```json
{
  "type": "read_receipt",
  "payload": {
    "message_id": "770e8400-e29b-41d4-a716-446655440002"
  }
}
```

**Handler**: `handleReadReceipt()`

**Server Action**:
- Updates message delivery_status to 'read'
- Broadcasts read receipt to message sender

---

### 2. typing

Send typing indicator.

**Type**: `typing`

**Payload**:
```json
{
  "type": "typing",
  "payload": {
    "conversation_id": "660e8400-e29b-41d4-a716-446655440001",
    "is_typing": true
  }
}
```

**Handler**: `handleTypingIndicator()`

**Server Action**:
- Updates typing_indicators table with TTL
- Broadcasts typing status to conversation participants

**Client Behavior**:
- Send `is_typing: true` when user starts typing
- Send `is_typing: false` when user stops or sends message
- Auto-expire after 3 seconds if no update

---

### 3. ping

Connection keep-alive.

**Type**: `ping`

**Payload**:
```json
{
  "type": "ping",
  "payload": {}
}
```

**Handler**: `handlePing()`

**Server Response**:
```json
{
  "type": "pong",
  "payload": {
    "timestamp": "2025-01-15T10:30:00Z"
  }
}
```

**Recommended Frequency**: Every 30 seconds

---

## Server → Client Events

Events sent from server to client.

### 1. connection_ack

Connection acknowledgment (automatic on connect).

**Type**: `connection_ack`

**Payload**:
```json
{
  "type": "connection_ack",
  "payload": {
    "status": "connected",
    "timestamp": "2025-01-15T10:30:00Z",
    "client_id": "conn_abc123xyz"
  }
}
```

---

### 2. pong

Heartbeat response.

**Type**: `pong`

**Payload**:
```json
{
  "type": "pong",
  "payload": {
    "timestamp": "2025-01-15T10:30:00Z"
  }
}
```

---

### 3. message (Broadcast)

New message received (broadcasted by Hub).

**Type**: `message`

**Payload**:
```json
{
  "type": "message",
  "payload": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "conversation_id": "660e8400-e29b-41d4-a716-446655440001",
    "sender_user_id": "550e8400-e29b-41d4-a716-446655440000",
    "content": "Hello!",
    "message_type": "text",
    "created_at": "2025-01-15T10:30:00Z"
  }
}
```

---

### 4. error

Error occurred.

**Type**: `error`

**Payload**:
```json
{
  "type": "error",
  "payload": {
    "code": "INVALID_MESSAGE_FORMAT",
    "message": "Failed to parse message"
  }
}
```

**Error Codes**:
- `INVALID_MESSAGE_FORMAT` - Malformed JSON
- `UNKNOWN_EVENT_TYPE` - Unrecognized type
- `HANDLER_ERROR` - Event handler failed

---

## Hub Architecture

### Overview

The WebSocket Hub manages all active connections with multi-device support.

```
Hub
├── clients: map[UserID]map[*Client]bool
├── connections: map[ClientID]*Client
├── register: chan *Client (buffer: 256)
├── unregister: chan *Client (buffer: 256)
└── broadcast: chan *BroadcastMessage (buffer: 1024)
```

### Multi-Device Support

Each user can have multiple connected devices:

```
User A (UUID)
  ├── Client 1 (iPhone)
  ├── Client 2 (Web Browser)
  └── Client 3 (iPad)

User B (UUID)
  └── Client 4 (Android)
```

**Hub Methods**:

```go
// Send to all user's devices
SendToUser(userID, message)

// Send to multiple users (exclude some)
SendToUsers(userIDs, message, excludeUsers)

// Check if user is online
IsUserOnline(userID) bool

// Get user's device count
GetUserDeviceCount(userID) int

// Get statistics
GetStats() HubStats
```

### Connection Lifecycle

```
1. Client connects → 2. Register with Hub → 3. Receive connection_ack
4. Send/receive messages → 5. Heartbeat (ping/pong)
6. Client disconnects → 7. Unregister from Hub
```

### Automatic Cleanup

Hub runs periodic cleanup:
- **Interval**: Every 30 seconds
- **Timeout**: 90 seconds of inactivity
- **Action**: Stale connections automatically removed

---

## Client Implementation

### JavaScript/TypeScript

```javascript
class EchoWebSocket {
  constructor(wsUrl, userId) {
    this.wsUrl = wsUrl;
    this.userId = userId;
    this.socket = null;
    this.messageHandlers = new Map();
    this.reconnectDelay = 1000;
    this.maxReconnectDelay = 30000;
  }

  connect() {
    this.socket = new WebSocket(this.wsUrl);

    this.socket.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectDelay = 1000;
      this.startHeartbeat();
    };

    this.socket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        this.handleMessage(message);
      } catch (err) {
        console.error('Failed to parse message:', err);
      }
    };

    this.socket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.socket.onclose = () => {
      console.log('WebSocket disconnected');
      this.stopHeartbeat();
      this.reconnect();
    };
  }

  send(type, payload) {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify({ type, payload }));
    } else {
      console.error('WebSocket not connected');
    }
  }

  handleMessage(message) {
    const handler = this.messageHandlers.get(message.type);
    if (handler) {
      handler(message.payload);
    } else {
      console.warn('No handler for message type:', message.type);
    }
  }

  on(type, handler) {
    this.messageHandlers.set(type, handler);
  }

  // Send read receipt
  sendReadReceipt(messageId) {
    this.send('read_receipt', { message_id: messageId });
  }

  // Send typing indicator
  setTyping(conversationId, isTyping) {
    this.send('typing', { conversation_id: conversationId, is_typing: isTyping });
  }

  // Heartbeat
  startHeartbeat() {
    this.heartbeatInterval = setInterval(() => {
      this.send('ping', {});
    }, 30000); // 30 seconds
  }

  stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
    }
  }

  // Reconnection with exponential backoff
  reconnect() {
    setTimeout(() => {
      console.log(`Reconnecting in ${this.reconnectDelay}ms...`);
      this.connect();
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
    }, this.reconnectDelay);
  }

  disconnect() {
    this.stopHeartbeat();
    if (this.socket) {
      this.socket.close();
    }
  }
}

// Usage
const ws = new EchoWebSocket('ws://localhost:8083/ws', userId);

ws.on('connection_ack', (payload) => {
  console.log('Connected:', payload);
});

ws.on('message', (payload) => {
  console.log('New message:', payload);
  // Display message in UI
});

ws.on('pong', (payload) => {
  console.log('Pong received:', payload.timestamp);
});

ws.connect();

// Send typing indicator
ws.setTyping(conversationId, true);

// Send read receipt
ws.sendReadReceipt(messageId);
```

---

## Connection Management

### Heartbeat

**Client Side**:
- Send `ping` every 30 seconds
- Expect `pong` response
- If no `pong` after 3 attempts, reconnect

**Server Side**:
- Automatically sends `pong` in response to `ping`
- Sends ping to clients every 45 seconds
- Closes connection if no activity for 90 seconds

### Reconnection Strategy

**Exponential Backoff**:
```
1st attempt: 1 second
2nd attempt: 2 seconds
3rd attempt: 4 seconds
4th attempt: 8 seconds
5th attempt: 16 seconds
6th+ attempts: 30 seconds (max)
```

**Reconnection Flow**:
1. Detect disconnection
2. Wait backoff duration
3. Attempt reconnection
4. On success: reset backoff
5. On failure: increase backoff, retry

---

## Buffer Management

**Message Size Limits**:
- **Read buffer**: 1024 bytes (configurable)
- **Write buffer**: 1024 bytes (configurable)
- **Max message size**: 1 MB

**Queue Limits**:
- **Client send buffer**: 256 messages
- **Hub register channel**: 256 clients
- **Hub unregister channel**: 256 clients
- **Hub broadcast channel**: 1024 messages

**Buffer Full Behavior**:
- If client send buffer full → connection closed
- Prevents memory exhaustion from slow clients

---

## Error Handling

### Connection Errors

**Upgrade Failed**:
```
HTTP 400 Bad Request - Invalid WebSocket upgrade request
```

**Authentication Failed**:
```
HTTP 401 Unauthorized - Missing or invalid X-User-ID header
```

### Message Errors

**Invalid JSON**:
```json
{
  "type": "error",
  "payload": {
    "code": "INVALID_MESSAGE_FORMAT",
    "message": "Failed to parse JSON"
  }
}
```

**Unknown Event Type**:
```json
{
  "type": "error",
  "payload": {
    "code": "UNKNOWN_EVENT_TYPE",
    "message": "Unknown event: custom_event"
  }
}
```

---

## Performance & Scalability

**Current Implementation**:
- Single Hub instance (in-memory)
- All connections in one process
- Suitable for: ~10,000 concurrent connections

**Scaling Considerations**:
- For >10K connections: Add Redis Pub/Sub for multi-instance support
- For >100K connections: Consider horizontal scaling with sticky sessions

**Metrics Tracked**:
- Total active connections
- Total messages sent
- Total broadcasts
- Connections per user
- Average message latency

---

**Last Updated**: January 2025
**Based on actual WebSocket implementation in `/services/message-service/internal/websocket/`**
