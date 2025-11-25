# WebSocket Service Architecture

## Overview

The WS-Service is a **pure WebSocket service** with NO REST API endpoints. It handles all real-time bidirectional communication using WebSocket connections only.

## Endpoints

- `/ws` - WebSocket connection endpoint (ONLY real endpoint)
- `/health`, `/live`, `/ready` - Health check endpoints only

## Architecture

### File Structure

```
ws-service/
├── cmd/server/main.go                      # Main entry point with full wiring
├── internal/
│   ├── config/                             # Configuration management
│   │   ├── config.go
│   │   ├── loader.go
│   │   └── validator.go
│   ├── websocket/                          # Core WebSocket infrastructure
│   │   ├── hub.go                          # Multi-device connection manager
│   │   ├── client.go                       # Individual client handling
│   │   ├── connection.go                   # HTTP upgrade & connection handler
│   │   └── types.go                        # WebSocket message types
│   ├── ws/                                 # WebSocket-specific features
│   │   ├── protocol/                       # Protocol definitions
│   │   │   ├── message_types.go            # All message type constants
│   │   │   └── messages.go                 # Message structures & payloads
│   │   ├── router/                         # Message routing
│   │   │   └── message_router.go           # Routes messages to handlers
│   │   ├── validator/                      # Message validation
│   │   │   └── message_validator.go        # Validates incoming messages
│   │   ├── handlers/                       # Message type handlers
│   │   │   ├── ping_handler.go             # Ping/pong keepalive
│   │   │   ├── authenticate_handler.go     # WebSocket authentication
│   │   │   ├── subscribe_handler.go        # Topic subscription
│   │   │   ├── unsubscribe_handler.go      # Topic unsubscription
│   │   │   ├── presence_handler.go         # Presence updates & queries
│   │   │   ├── typing_handler.go           # Typing indicators
│   │   │   ├── read_receipt_handler.go     # Read/delivered receipts
│   │   │   └── call_signaling_handler.go   # WebRTC signaling (offer/answer/ICE)
│   │   ├── subscription/                   # Subscription management
│   │   │   └── manager.go                  # Topic subscription manager
│   │   ├── broadcast/                      # Broadcasting strategies
│   │   │   └── broadcaster.go              # Broadcast to users/topics
│   │   └── presence/                       # Presence tracking
│   │       └── tracker.go                  # User presence tracking via WS
│   ├── service/                            # Business logic
│   │   └── ws_service.go                   # Minimal service (user validation)
│   ├── model/                              # Domain models
│   │   ├── events.go                       # Event types & categories
│   │   └── connection.go                   # Connection records
│   ├── health/                             # Health checks
│   │   ├── manager.go
│   │   ├── handler.go
│   │   └── checkers/
│   │       ├── cache.go
│   │       └── database.go
│   └── errors/                             # Error handling
│       └── errors.go
├── configs/
│   └── config.yaml                         # Configuration with WS settings
├── Dockerfile                              # Production build
└── Dockerfile.dev                          # Development with hot-reload
```

## WebSocket Protocol

### Client → Server Messages

| Type | Description |
|------|-------------|
| `ping` | Keepalive ping |
| `authenticate` | Authenticate connection |
| `subscribe` | Subscribe to topics |
| `unsubscribe` | Unsubscribe from topics |
| `presence.update` | Update user presence |
| `presence.query` | Query user presence |
| `typing.start` | Start typing indicator |
| `typing.stop` | Stop typing indicator |
| `mark.read` | Mark messages as read |
| `mark.delivered` | Mark messages as delivered |
| `call.offer` | WebRTC offer |
| `call.answer` | WebRTC answer |
| `call.ice` | WebRTC ICE candidate |
| `call.hangup` | End call |

### Server → Client Messages

| Type | Description |
|------|-------------|
| `connected` | Connection acknowledged |
| `auth.success` | Authentication successful |
| `auth.failed` | Authentication failed |
| `subscribed` | Subscription confirmed |
| `unsubscribed` | Unsubscription confirmed |
| `presence.update` | Presence status update |
| `presence.online` | User went online |
| `presence.offline` | User went offline |
| `typing.start` | User started typing |
| `typing.stop` | User stopped typing |
| `message.new` | New message |
| `message.delivered` | Message delivered |
| `message.read` | Message read |
| `call.incoming` | Incoming call |
| `call.offer` | Call offer |
| `call.answer` | Call answer |
| `call.ice` | ICE candidate |
| `call.ended` | Call ended |
| `error` | Error message |

## Subscription Topics

Clients can subscribe to specific topics to receive relevant events:

- `user` - User-specific events
- `conversation` - Conversation events
- `presence` - Presence updates
- `typing` - Typing indicators
- `calls` - Call events
- `notifications` - Notifications

## Key Components

### 1. Hub (`internal/websocket/hub.go`)
- Manages all active WebSocket connections
- Supports multi-device connections per user
- Handles connection registration/unregistration
- Broadcasts messages to users
- Tracks connection statistics

### 2. Message Router (`internal/ws/router/message_router.go`)
- Routes incoming messages to appropriate handlers
- Validates message format
- Handles errors and sends error responses
- Extensible handler registration system

### 3. Subscription Manager (`internal/ws/subscription/manager.go`)
- Manages topic subscriptions per client
- Supports resource-specific subscriptions (e.g., specific conversation)
- Tracks client subscriptions
- Returns subscribers for broadcasting

### 4. Broadcaster (`internal/ws/broadcast/broadcaster.go`)
- Broadcasts to specific users (all devices)
- Broadcasts to multiple users
- Broadcasts to topic subscribers
- Broadcasts to all connected clients
- Excludes sender from broadcasts

### 5. Presence Tracker (`internal/ws/presence/tracker.go`)
- Tracks user online/offline status
- Tracks custom presence status (away, busy)
- Auto-updates based on device connections
- Broadcasts presence changes
- Cleans up stale presence data

### 6. Message Handlers (`internal/ws/handlers/`)
Each handler implements the `MessageHandler` interface:
```go
type MessageHandler interface {
    Handle(ctx context.Context, client *Client, msg *ClientMessage) error
    MessageType() ClientMessageType
}
```

## Message Flow

```
1. Client connects to /ws
   ↓
2. Connection upgraded to WebSocket
   ↓
3. Client registered in Hub
   ↓
4. Client sends JSON message
   ↓
5. Message Router receives raw message
   ↓
6. Message Validator validates format
   ↓
7. Router finds appropriate Handler
   ↓
8. Handler processes message
   ↓
9. Handler may broadcast to other clients
   ↓
10. Response sent back to client
```

## Connection Lifecycle

```
1. HTTP upgrade request → /ws
2. User validation (database check)
3. WebSocket upgrade
4. Client object created
5. Hub registration
6. Welcome message sent
7. Message processing begins
8. Heartbeat/ping-pong
9. Disconnect event
10. Hub unregistration
11. Subscription cleanup
12. Presence update (offline)
```

## Features

### Multi-Device Support
- Each user can have multiple connected devices
- Messages broadcast to all user devices
- Device-specific tracking

### Topic Subscriptions
- Subscribe to specific topics/resources
- Filter-based subscriptions
- Efficient subscriber lookup
- Auto-cleanup on disconnect

### Presence Tracking
- Real-time online/offline status
- Custom status messages
- Device count tracking
- Auto-detection via connections

### WebRTC Signaling
- Complete WebRTC signaling support
- Offer/Answer exchange
- ICE candidate trickling
- Call management

### Typing Indicators
- Real-time typing status
- Conversation-scoped
- Start/stop events

### Read Receipts
- Message delivered notifications
- Message read notifications
- Bulk receipt support

## Configuration

Key WebSocket-specific settings in `config.yaml`:

```yaml
websocket:
  write_wait: 10s
  pong_wait: 60s
  ping_period: 54s
  read_buffer_size: 1024
  write_buffer_size: 1024
  max_message_size: 10485760  # 10MB
  client_buffer_size: 256
  cleanup_interval: 30s
  stale_connection_timeout: 90s
  register_buffer: 256
  unregister_buffer: 256
  broadcast_buffer: 1024
```

## Usage Example

### Client Connection
```javascript
const ws = new WebSocket('ws://localhost:8086/ws?user_id=<uuid>');

ws.onopen = () => {
  // Send ping
  ws.send(JSON.stringify({
    id: 'msg_1',
    type: 'ping',
    payload: {},
    timestamp: new Date().toISOString()
  }));

  // Subscribe to topics
  ws.send(JSON.stringify({
    id: 'msg_2',
    type: 'subscribe',
    payload: {
      topics: ['conversation', 'presence'],
      filters: {
        conversation_id: '<conv-uuid>'
      }
    }
  }));
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message.type, message.payload);
};
```

## Scalability Considerations

- **Horizontal Scaling**: Can run multiple instances with Redis pub/sub for cross-instance messaging
- **Connection Limits**: Configure based on available resources
- **Message Buffering**: Configurable client buffers prevent slow clients from blocking
- **Cleanup**: Automatic cleanup of stale connections and presence data

## Security

- User validation before connection
- Origin checking (TODO: implement production checks)
- Message validation
- Rate limiting (can be added per message type)
- Token-based authentication support

## Monitoring

- Connection statistics via `Hub.GetStats()`
- Subscription statistics via `Manager.GetStats()`
- Presence statistics via `Tracker.GetStats()`
- Health checks for database and cache
- Structured logging with correlation IDs

## Future Enhancements

- [ ] Message persistence for offline users
- [ ] Redis pub/sub for multi-instance deployment
- [ ] Rate limiting per user/message type
- [ ] Message acknowledgment tracking
- [ ] Reconnection with message replay
- [ ] Binary message support (protobuf)
- [ ] Compression support (permessage-deflate)
- [ ] Authentication token refresh
- [ ] Admin API for connection management
