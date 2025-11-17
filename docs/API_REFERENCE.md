# API Reference

**IMPORTANT**: This documentation reflects ONLY the endpoints that are actually implemented in the codebase.

## Service Status

| Service | Status | Endpoints |
|---------|--------|-----------|
| Auth Service | ✅ IMPLEMENTED | 2 endpoints |
| Message Service | ✅ IMPLEMENTED | 9 endpoints + WebSocket |
| User Service | ⚠️ PARTIAL | 2 endpoints |
| Location Service | ✅ IMPLEMENTED | 2 endpoints |
| Presence Service | 🚧 STUBBED | Routes defined, no implementation |
| Media Service | ✅ IMPLEMENTED | Multiple endpoints |
| Notification Service | ❌ PLACEHOLDER | No implementation |
| Analytics Service | ❌ PLACEHOLDER | No implementation |

## Base URLs

```
API Gateway:     http://localhost:8080
Auth Service:    http://localhost:8081 (internal)
Message Service: http://localhost:8083 (internal)
User Service:    http://localhost:8082 (internal)
Location Service: http://localhost:8090
```

## Authentication Service

### POST /register

Register a new user with email and password.

**Endpoint**: `POST /api/v1/auth/register`

**Request Body**:
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "phone_number": "+1234567890",
  "phone_country_code": "+1",
  "accept_terms": true
}
```

**Request Fields**:
- `email` (required, string): Valid email address
- `password` (required, string): Min 8 characters
- `phone_number` (optional, string): E.164 format
- `phone_country_code` (optional, string): Country code
- `accept_terms` (required, boolean): Must be true

**Success Response (201 Created)**:
```json
{
  "success": true,
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "email_verified": false,
    "created_at": "2025-01-15T10:30:00Z"
  }
}
```

**Error Responses**:

*400 Bad Request*:
```json
{
  "success": false,
  "error": {
    "message": "Email is already registered",
    "code": "email_already_exists"
  }
}
```

**Implementation Details**:
- Email uniqueness check before registration
- Password hashed using bcrypt/Argon2
- IP address and user agent captured
- Email verification token generated (sent via email service if configured)

---

### POST /login

Authenticate user and create session.

**Endpoint**: `POST /api/v1/auth/login`

**Request Body**:
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "device_id": "device-12345",
  "device_name": "iPhone 14 Pro",
  "device_type": "mobile"
}
```

**Request Fields**:
- `email` (required, string): User's email
- `password` (required, string): User's password
- `device_id` (optional, string): Unique device identifier
- `device_name` (optional, string): Human-readable device name
- `device_type` (optional, string): mobile, tablet, desktop, web

**Success Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "session_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2025-01-15T11:30:00Z"
  }
}
```

**Error Responses**:

*401 Unauthorized*:
```json
{
  "success": false,
  "error": {
    "message": "Invalid email or password",
    "code": "invalid_credentials"
  }
}
```

**Implementation Details**:
- Failed login attempts tracked (account lockout after threshold)
- Session created with device information
- IP geolocation lookup performed (via location service)
- Login history recorded for security audit

---

## Message Service

### GET /ws

Establish WebSocket connection for real-time messaging.

**Endpoint**: `GET /ws`

**Connection Headers**:
```http
X-User-ID: 550e8400-e29b-41d4-a716-446655440000
X-Device-ID: device-12345
X-Platform: ios
X-App-Version: 1.0.0
```

**Connection Established**:
```json
{
  "type": "connection_ack",
  "payload": {
    "status": "connected",
    "timestamp": "2025-01-15T10:30:00Z",
    "client_id": "conn_abc123"
  }
}
```

**Supported Events**:
- `read_receipt` - Mark message as read
- `typing` - Send typing indicator
- `ping` - Connection keep-alive

See [WebSocket Protocol](./WEBSOCKET_PROTOCOL.md) for complete details.

---

### POST /

Send a new message to a conversation.

**Endpoint**: `POST /`

**Headers**:
```http
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request Body**:
```json
{
  "conversation_id": "660e8400-e29b-41d4-a716-446655440001",
  "content": "Hello, how are you?",
  "message_type": "text",
  "parent_message_id": null,
  "mentions": [],
  "metadata": {}
}
```

**Request Fields**:
- `conversation_id` (required, UUID): Conversation ID
- `content` (required, string): Message text (max 10,000 chars for text)
- `message_type` (required, string): text, image, video, audio, document, location, contact
- `parent_message_id` (optional, UUID): For threading/replies
- `mentions` (optional, array): Array of mentioned user IDs
- `metadata` (optional, object): Additional data for media messages

**Success Response (201 Created)**:
```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "conversation_id": "660e8400-e29b-41d4-a716-446655440001",
    "sender_user_id": "550e8400-e29b-41d4-a716-446655440000",
    "content": "Hello, how are you?",
    "message_type": "text",
    "status": "sent",
    "created_at": "2025-01-15T10:30:00Z"
  }
}
```

---

### GET /

Retrieve messages from a conversation with pagination.

**Endpoint**: `GET /`

**Query Parameters**:
- `conversation_id` (required, UUID): Conversation ID
- `limit` (optional, int): Messages per page (default: 50, max: 100)
- `before` (optional, UUID): Get messages before this message ID
- `after` (optional, UUID): Get messages after this message ID

**Example**:
```http
GET /?conversation_id=660e8400-e29b-41d4-a716-446655440001&limit=20&before=msg123
```

**Success Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "messages": [
      {
        "id": "770e8400-e29b-41d4-a716-446655440002",
        "conversation_id": "660e8400-e29b-41d4-a716-446655440001",
        "sender_user_id": "550e8400-e29b-41d4-a716-446655440000",
        "content": "Hello!",
        "message_type": "text",
        "created_at": "2025-01-15T10:30:00Z",
        "is_edited": false,
        "is_deleted": false
      }
    ],
    "pagination": {
      "has_more": true,
      "next_cursor": "770e8400-e29b-41d4-a716-446655440002"
    }
  }
}
```

---

### PUT /{id}

Edit an existing message.

**Endpoint**: `PUT /{id}`

**Path Parameters**:
- `id` (required, UUID): Message ID to edit

**Request Body**:
```json
{
  "content": "Updated message content"
}
```

**Success Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "content": "Updated message content",
    "is_edited": true,
    "edited_at": "2025-01-15T10:35:00Z"
  }
}
```

---

### DELETE /{id}

Delete a message.

**Endpoint**: `DELETE /{id}`

**Path Parameters**:
- `id` (required, UUID): Message ID to delete

**Query Parameters**:
- `for_everyone` (optional, boolean): Delete for all users (default: false)

**Success Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "is_deleted": true,
    "deleted_at": "2025-01-15T10:40:00Z"
  }
}
```

---

### POST /read

Mark message(s) as read.

**Endpoint**: `POST /read`

**Request Body**:
```json
{
  "message_ids": [
    "770e8400-e29b-41d4-a716-446655440002",
    "880e8400-e29b-41d4-a716-446655440003"
  ]
}
```

**Success Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "marked_count": 2,
    "read_at": "2025-01-15T10:45:00Z"
  }
}
```

---

### POST /typing

Send typing indicator.

**Endpoint**: `POST /typing`

**Request Body**:
```json
{
  "conversation_id": "660e8400-e29b-41d4-a716-446655440001",
  "is_typing": true
}
```

**Success Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "conversation_id": "660e8400-e29b-41d4-a716-446655440001",
    "is_typing": true,
    "timestamp": "2025-01-15T10:30:00Z"
  }
}
```

---

### POST /conversations

Create a new conversation.

**Endpoint**: `POST /conversations`

**Request Body**:
```json
{
  "conversation_type": "direct",
  "participant_ids": ["user-id-1", "user-id-2"],
  "title": "Optional group name",
  "description": "Optional description"
}
```

**Request Fields**:
- `conversation_type` (required, string): direct, group, channel
- `participant_ids` (required, array): Array of user IDs
- `title` (optional, string): For groups/channels
- `description` (optional, string): For groups/channels

**Success Response (201 Created)**:
```json
{
  "success": true,
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "conversation_type": "direct",
    "participant_count": 2,
    "created_at": "2025-01-15T10:30:00Z"
  }
}
```

---

### GET /conversations

Get user's conversations with pagination.

**Endpoint**: `GET /conversations`

**Query Parameters**:
- `limit` (optional, int): Conversations per page (default: 20, max: 100)
- `offset` (optional, int): Pagination offset (default: 0)
- `type` (optional, string): Filter by type (direct, group, channel)

**Success Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "conversations": [
      {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "conversation_type": "direct",
        "title": null,
        "last_message_at": "2025-01-15T10:30:00Z",
        "unread_count": 3,
        "participant_count": 2
      }
    ],
    "pagination": {
      "total": 45,
      "limit": 20,
      "offset": 0,
      "has_more": true
    }
  }
}
```

---

## User Service

### POST /

Create or update user profile.

**Endpoint**: `POST /`

**Request Body**:
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "john_doe",
  "display_name": "John Doe",
  "bio": "Software engineer",
  "avatar_url": "https://example.com/avatar.jpg"
}
```

**Success Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "id": "profile-id",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "display_name": "John Doe",
    "created_at": "2025-01-15T10:30:00Z"
  }
}
```

**Implementation Notes**:
- Automatically generates username if not provided
- Updates existing profile if one exists for the user

---

### GET /{user_id}

Get user profile by ID.

**Endpoint**: `GET /{user_id}`

**Path Parameters**:
- `user_id` (required, UUID): User's ID

**Success Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "id": "profile-id",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "display_name": "John Doe",
    "bio": "Software engineer",
    "avatar_url": "https://example.com/avatar.jpg",
    "created_at": "2025-01-15T10:30:00Z"
  }
}
```

---

## Location Service

### GET /health

Health check endpoint.

**Endpoint**: `GET /health`

**Success Response (200 OK)**:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

---

### GET /lookup

IP geolocation lookup.

**Endpoint**: `GET /lookup`

**Query Parameters**:
- `ip` (required, string): IP address to lookup

**Example**:
```http
GET /lookup?ip=8.8.8.8
```

**Success Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "ip": "8.8.8.8",
    "country": "United States",
    "country_code": "US",
    "region": "California",
    "city": "Mountain View",
    "timezone": "America/Los_Angeles",
    "isp": "Google LLC",
    "coordinates": {
      "latitude": 37.386,
      "longitude": -122.084
    }
  }
}
```

---

## Error Response Format

All errors follow this standard format:

```json
{
  "success": false,
  "error": {
    "message": "Human-readable error message",
    "code": "ERROR_CODE",
    "details": {}
  },
  "metadata": {
    "request_id": "req_abc123",
    "timestamp": "2025-01-15T10:30:00Z"
  }
}
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request (validation error) |
| 401 | Unauthorized (authentication required) |
| 403 | Forbidden (insufficient permissions) |
| 404 | Not Found |
| 409 | Conflict (duplicate resource) |
| 429 | Too Many Requests (rate limited) |
| 500 | Internal Server Error |

---

## Rate Limiting

Rate limits are enforced at multiple levels:

- **Global**: 1000 requests/minute per IP
- **Auth endpoints**: 5 requests/minute per IP
- **WebSocket**: 100 messages/minute per connection

Rate limit headers:
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 995
X-RateLimit-Reset: 1610000000
```

---

**Last Updated**: January 2025
**Version**: Based on actual implementation
