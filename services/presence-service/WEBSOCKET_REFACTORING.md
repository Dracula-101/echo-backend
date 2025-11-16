# WebSocket-Centralized Presence Service Refactoring

## Overview
The presence-service has been refactored to be **WebSocket-centralized**, meaning all real-time presence operations are now handled primarily through WebSocket connections, with HTTP endpoints serving as lightweight alternatives for non-real-time queries.

## Key Changes

### 1. New WebSocket Service Layer (`internal/websocket/service.go`)

Created a comprehensive `websocket.Service` that centralizes all WebSocket-related operations:

```go
type Service struct {
    hub   *Hub
    repo  repo.PresenceRepository
    cache cache.Cache
    log   logger.Logger
}
```

**Key Responsibilities:**
- ✅ Handle client lifecycle (connect/disconnect)
- ✅ Process WebSocket messages (presence updates, heartbeats, typing indicators)
- ✅ Automatic user online/offline management
- ✅ Real-time broadcasting to connected clients
- ✅ Integration with database and cache
- ✅ Privacy settings enforcement

### 2. WebSocket-Driven Operations

#### **Client Connection Flow**
```
1. Client connects via WebSocket → `/ws`
2. Handler.HandleConnection() authenticates and creates client
3. wsService.HandleClientConnect() automatically sets user online
4. User's contacts are notified of online status
5. Message handlers are registered for real-time events
```

#### **Client Disconnection Flow**
```
1. Client disconnects (network/user action)
2. Hub.unregisterClient() removes client from connections
3. wsService.HandleClientDisconnect() checks remaining devices
4. If no devices remain → set user offline
5. User's contacts are notified of offline status
```

### 3. WebSocket Message Handlers

The service now handles these WebSocket message types:

#### **Presence Update**
```json
{
  "type": "presence_update",
  "payload": {
    "online_status": "away",
    "custom_status": "In a meeting"
  }
}
```
- Updates database
- Broadcasts to user's contacts
- Sends acknowledgment to client

#### **Heartbeat**
```json
{
  "type": "heartbeat",
  "payload": {}
}
```
- Updates last_seen_at timestamp
- Keeps connection alive
- Sends acknowledgment

#### **Typing Indicator**
```json
{
  "type": "typing",
  "payload": {
    "conversation_id": "uuid",
    "is_typing": true
  }
}
```
- Broadcasts to conversation participants
- Stores in database with short TTL
- Real-time updates for other users

### 4. Enhanced Hub Integration

The Hub now works seamlessly with the WebSocket service:

**Features:**
- Multi-device support (users can have multiple connections)
- Real-time online status tracking
- Efficient message broadcasting to specific users
- Automatic stale connection cleanup
- Connection statistics and metrics

### 5. HTTP Endpoints (Lightweight Queries)

HTTP endpoints now serve as **query interfaces** rather than primary operations:

| Endpoint | Purpose | Notes |
|----------|---------|-------|
| `POST /` | Update presence | ⚠️ Use WebSocket for real-time |
| `GET /{user_id}` | Get user presence | Enhanced with real-time hub data |
| `POST /bulk` | Get bulk presence | Efficient for multiple users |
| `POST /heartbeat` | Send heartbeat | ⚠️ Use WebSocket for real-time |
| `POST /typing` | Set typing | ⚠️ Use WebSocket for real-time |

**Recommendation:** Clients should prefer WebSocket for real-time operations and use HTTP only for queries or when WebSocket is unavailable.

## Architecture Flow

```
┌─────────────┐
│   Client    │
│  (Mobile/   │
│    Web)     │
└──────┬──────┘
       │
       │ WebSocket Connection
       ↓
┌──────────────────────────────────────┐
│         WebSocket Handler            │
│  • Upgrades HTTP → WebSocket         │
│  • Authenticates user                │
│  • Creates client instance           │
└──────┬───────────────────────────────┘
       │
       │ Delegates to
       ↓
┌──────────────────────────────────────┐
│        WebSocket Service             │
│  • HandleClientConnect()             │
│  • HandlePresenceUpdate()            │
│  • HandleHeartbeat()                 │
│  • HandleTypingIndicator()           │
│  • HandleClientDisconnect()          │
└──┬───────────────────┬───────────────┘
   │                   │
   │                   │ Broadcasts via
   ↓                   ↓
┌──────────────┐  ┌──────────────┐
│ Repository   │  │     Hub      │
│  (Database)  │  │  • Clients   │
└──────────────┘  │  • Broadcast │
                  └──────┬───────┘
                         │
                         │ Real-time events to
                         ↓
                   ┌─────────────┐
                   │  Connected  │
                   │   Clients   │
                   └─────────────┘
```

## Benefits

### 1. **Real-Time by Default**
- All presence operations happen in real-time
- Immediate updates to connected clients
- No polling required

### 2. **Automatic State Management**
- User online/offline managed automatically
- No manual heartbeat endpoints needed
- Graceful handling of disconnections

### 3. **Multi-Device Support**
- Users can connect from multiple devices
- User stays "online" as long as one device is connected
- Individual device tracking

### 4. **Efficient Broadcasting**
- Hub maintains active connections map
- Targeted message delivery
- No database queries for online status

### 5. **Separation of Concerns**
- WebSocket logic isolated in `websocket` package
- Service layer handles business logic
- Repository handles persistence
- Clear boundaries and interfaces

## Migration Guide for Clients

### Before (HTTP-based)
```javascript
// Poll for presence updates
setInterval(async () => {
  const response = await fetch(`/presence/${userId}`);
  const presence = await response.json();
  updateUI(presence);
}, 5000); // Poll every 5 seconds

// Update own presence
await fetch('/presence', {
  method: 'POST',
  body: JSON.stringify({ online_status: 'away' })
});
```

### After (WebSocket-based)
```javascript
// Connect once
const ws = new WebSocket('ws://localhost:8085/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  switch(data.type) {
    case 'presence_update':
      updateUI(data.payload);
      break;
    case 'typing_indicator':
      showTypingIndicator(data.payload);
      break;
  }
};

// Update own presence
ws.send(JSON.stringify({
  type: 'presence_update',
  payload: { online_status: 'away' }
}));

// Send heartbeat every 30 seconds
setInterval(() => {
  ws.send(JSON.stringify({ type: 'heartbeat' }));
}, 30000);
```

## Performance Improvements

1. **Reduced Database Load**: Online status queries from hub instead of database
2. **Lower Latency**: WebSocket push instead of HTTP polling
3. **Better Scalability**: Efficient connection management and broadcasting
4. **Resource Optimization**: One connection handles all real-time operations

## Future Enhancements

- [ ] Add Redis Pub/Sub for multi-instance broadcasting
- [ ] Implement connection rate limiting
- [ ] Add WebSocket message batching
- [ ] Support for custom presence states (e.g., "in a call", "presenting")
- [ ] Analytics for connection patterns
- [ ] WebSocket reconnection strategies
- [ ] Message queue for offline message delivery

## Testing

```bash
# Build service
cd services/presence-service
go build ./...

# Run service
./main

# Connect via WebSocket
wscat -c ws://localhost:8085/ws -H "X-User-ID: user-uuid" -H "X-Device-ID: device-123"

# Send presence update
{"type": "presence_update", "payload": {"online_status": "away", "custom_status": "Busy"}}

# Send heartbeat
{"type": "heartbeat"}

# Send typing indicator
{"type": "typing", "payload": {"conversation_id": "conv-uuid", "is_typing": true}}
```

## Configuration

No additional configuration required. The service uses existing database and cache configurations.

## Backwards Compatibility

✅ All existing HTTP endpoints still work
✅ Clients can migrate gradually from HTTP to WebSocket
✅ No breaking changes to API contracts

---

**Summary**: The presence-service is now a true real-time service built around WebSocket connections, providing instant presence updates, typing indicators, and online status tracking with minimal latency and optimal resource usage.
