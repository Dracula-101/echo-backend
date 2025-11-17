# API Reference

Complete API reference for all Echo Backend endpoints.

## Table of Contents

- [Authentication](#authentication)
- [User Management](#user-management)
- [Messaging](#messaging)
- [Presence](#presence)
- [Location](#location)
- [WebSocket API](#websocket-api)
- [Error Responses](#error-responses)
- [Rate Limiting](#rate-limiting)

## Base URL

```
Development: http://localhost:8080
Production:  https://api.echo.app
```

## Authentication

All authenticated endpoints require a JWT token in the `Authorization` header:

```http
Authorization: Bearer <access_token>
```

### Register User

Create a new user account with phone number verification.

```http
POST /api/v1/auth/register
```

**Request Body:**
```json
{
  "phone": "+1234567890",
  "password": "SecurePass123!",
  "name": "John Doe",
  "device_id": "device-uuid",
  "device_name": "iPhone 14 Pro",
  "platform": "ios"
}
```

**Field Specifications:**
- `phone` (required): E.164 format, unique
- `password` (required): Min 8 chars, max 128 chars, must include uppercase, lowercase, number
- `name` (required): Min 2 chars, max 100 chars
- `device_id` (optional): Unique device identifier
- `device_name` (optional): Human-readable device name
- `platform` (required): One of `ios`, `android`, `web`

**Success Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "phone": "+1234567890",
    "verified": false,
    "otp_sent": true,
    "otp_expires_at": "2024-01-15T10:35:45Z"
  },
  "metadata": {
    "request_id": "req_abc123",
    "timestamp": "2024-01-15T10:30:45Z",
    "duration": "45ms"
  }
}
```

**Error Responses:**

*400 Bad Request - Invalid Input*
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "type": "validation",
    "fields": {
      "phone": "Invalid phone number format",
      "password": "Password must be at least 8 characters"
    }
  }
}
```

*409 Conflict - Phone Already Exists*
```json
{
  "success": false,
  "error": {
    "code": "PHONE_EXISTS",
    "message": "Phone number already registered",
    "type": "conflict"
  }
}
```

---

### Verify OTP

Verify the OTP sent during registration.

```http
POST /api/v1/auth/verify-otp
```

**Request Body:**
```json
{
  "phone": "+1234567890",
  "otp": "123456"
}
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "verified": true,
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "rt_abc123...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

**Error Responses:**

*400 Bad Request - Invalid OTP*
```json
{
  "success": false,
  "error": {
    "code": "INVALID_OTP",
    "message": "OTP is invalid or expired",
    "type": "validation"
  }
}
```

*429 Too Many Requests - Rate Limited*
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many OTP verification attempts",
    "type": "rate_limit",
    "retry_after": 300
  }
}
```

---

### Login

Authenticate user with phone and password.

```http
POST /api/v1/auth/login
```

**Request Body:**
```json
{
  "phone": "+1234567890",
  "password": "SecurePass123!",
  "device_id": "device-uuid",
  "device_name": "iPhone 14 Pro",
  "platform": "ios"
}
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "rt_abc123...",
    "token_type": "Bearer",
    "expires_in": 900,
    "session_id": "sess_xyz789"
  }
}
```

**Error Responses:**

*401 Unauthorized - Invalid Credentials*
```json
{
  "success": false,
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid phone or password",
    "type": "unauthorized"
  }
}
```

*403 Forbidden - Account Locked*
```json
{
  "success": false,
  "error": {
    "code": "ACCOUNT_LOCKED",
    "message": "Account locked due to too many failed login attempts",
    "type": "forbidden",
    "locked_until": "2024-01-15T11:30:45Z"
  }
}
```

---

### Refresh Token

Refresh access token using refresh token.

```http
POST /api/v1/auth/refresh
```

**Request Body:**
```json
{
  "refresh_token": "rt_abc123..."
}
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

---

### Logout

Invalidate current session.

```http
POST /api/v1/auth/logout
Authorization: Bearer <access_token>
```

**Success Response (204 No Content)**

---

### Get Current User

Get authenticated user's information.

```http
GET /api/v1/auth/me
Authorization: Bearer <access_token>
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "phone": "+1234567890",
    "verified": true,
    "created_at": "2024-01-15T10:30:45Z"
  }
}
```

---

## User Management

### Get User Profile

Get user's profile information.

```http
GET /api/v1/users/:user_id/profile
Authorization: Bearer <access_token>
```

**Path Parameters:**
- `user_id` (required): User's UUID

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "phone": "+1234567890",
    "display_name": "John Doe",
    "bio": "Software Engineer",
    "avatar_url": "https://cdn.echo.app/avatars/abc123.jpg",
    "status": "online",
    "last_seen": "2024-01-15T10:30:45Z",
    "created_at": "2024-01-15T10:30:45Z"
  }
}
```

---

### Update Profile

Update user's profile.

```http
PUT /api/v1/users/profile
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "display_name": "John Doe",
  "bio": "Software Engineer",
  "avatar_url": "https://cdn.echo.app/avatars/abc123.jpg",
  "status": "away"
}
```

**Field Specifications:**
- `display_name` (optional): Min 2 chars, max 100 chars
- `bio` (optional): Max 500 chars
- `avatar_url` (optional): Valid URL
- `status` (optional): One of `online`, `away`, `busy`, `offline`

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "display_name": "John Doe",
    "bio": "Software Engineer",
    "avatar_url": "https://cdn.echo.app/avatars/abc123.jpg",
    "status": "away",
    "updated_at": "2024-01-15T10:30:45Z"
  }
}
```

---

### Get Contacts

Get user's contact list.

```http
GET /api/v1/users/contacts
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `limit` (optional): Max results per page (default: 20, max: 100)
- `offset` (optional): Pagination offset (default: 0)
- `search` (optional): Search by name or phone

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "contacts": [
      {
        "user_id": "123e4567-e89b-12d3-a456-426614174000",
        "display_name": "John Doe",
        "phone": "+1234567890",
        "avatar_url": "https://cdn.echo.app/avatars/abc123.jpg",
        "status": "online",
        "added_at": "2024-01-15T10:30:45Z"
      }
    ],
    "pagination": {
      "total": 150,
      "limit": 20,
      "offset": 0,
      "has_more": true
    }
  }
}
```

---

## Messaging

### Send Message

Send a message to a user.

```http
POST /api/v1/messages
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "to_user_id": "123e4567-e89b-12d3-a456-426614174000",
  "content": "Hello! How are you?",
  "type": "text",
  "reply_to_id": null,
  "metadata": {
    "client_id": "msg_client_123"
  }
}
```

**Field Specifications:**
- `to_user_id` (required): Recipient's UUID
- `content` (required): Message content (max 10,000 chars for text)
- `type` (required): One of `text`, `image`, `video`, `audio`, `document`, `location`, `contact`
- `reply_to_id` (optional): Message ID being replied to
- `metadata` (optional): Client-specific metadata

**Success Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "message_id": "msg_abc123",
    "conversation_id": "conv_xyz789",
    "from_user_id": "sender-uuid",
    "to_user_id": "recipient-uuid",
    "content": "Hello! How are you?",
    "type": "text",
    "status": "sent",
    "created_at": "2024-01-15T10:30:45Z",
    "delivered_at": null,
    "read_at": null
  }
}
```

---

### Get Messages

Get messages in a conversation.

```http
GET /api/v1/messages
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `conversation_id` (required): Conversation UUID
- `limit` (optional): Max results (default: 50, max: 100)
- `before` (optional): Get messages before this message ID
- `after` (optional): Get messages after this message ID

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "messages": [
      {
        "message_id": "msg_abc123",
        "conversation_id": "conv_xyz789",
        "from_user_id": "sender-uuid",
        "to_user_id": "recipient-uuid",
        "content": "Hello!",
        "type": "text",
        "status": "read",
        "created_at": "2024-01-15T10:30:45Z",
        "delivered_at": "2024-01-15T10:30:46Z",
        "read_at": "2024-01-15T10:31:00Z",
        "edited": false,
        "deleted": false
      }
    ],
    "pagination": {
      "has_more": true,
      "next_cursor": "msg_xyz789"
    }
  }
}
```

---

### Edit Message

Edit a sent message.

```http
PUT /api/v1/messages/:message_id
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "content": "Updated message content"
}
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message_id": "msg_abc123",
    "content": "Updated message content",
    "edited": true,
    "edited_at": "2024-01-15T10:35:45Z"
  }
}
```

---

### Delete Message

Delete a message.

```http
DELETE /api/v1/messages/:message_id
Authorization: Bearer <access_token>
```

**Query Parameters:**
- `for_everyone` (optional): Delete for all users (default: false)

**Success Response (204 No Content)**

---

### Mark as Read

Mark messages as read.

```http
POST /api/v1/messages/read
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "message_ids": ["msg_abc123", "msg_xyz789"]
}
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "marked_count": 2
  }
}
```

---

## Presence

### Update Presence

Update user's online status.

```http
POST /api/v1/presence
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "status": "online",
  "activity": "typing"
}
```

**Field Specifications:**
- `status` (required): One of `online`, `away`, `busy`, `offline`
- `activity` (optional): One of `typing`, `recording`, `uploading`

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "status": "online",
    "activity": "typing",
    "last_seen": "2024-01-15T10:30:45Z"
  }
}
```

---

### Get User Presence

Get presence information for a user.

```http
GET /api/v1/presence/:user_id
Authorization: Bearer <access_token>
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "status": "online",
    "activity": null,
    "last_seen": "2024-01-15T10:30:45Z",
    "devices": [
      {
        "device_id": "device-1",
        "platform": "ios",
        "last_active": "2024-01-15T10:30:45Z"
      }
    ]
  }
}
```

---

## Location

### Lookup Phone Number

Get location information for a phone number.

```http
GET /api/v1/location/lookup
```

**Query Parameters:**
- `phone` (required): Phone number in E.164 format

**Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "phone": "+1234567890",
    "country": "US",
    "country_code": "+1",
    "country_name": "United States",
    "region": "California",
    "timezone": "America/Los_Angeles",
    "carrier": "AT&T"
  }
}
```

---

## WebSocket API

### Connection

Connect to WebSocket for real-time updates.

```
ws://localhost:8083/ws
```

**Connection Headers:**
```
X-User-ID: <user_uuid>
X-Device-ID: <device_id>
X-Platform: ios|android|web
X-Access-Token: <access_token>
```

**Connection Established:**
```json
{
  "type": "connection.established",
  "data": {
    "connection_id": "conn_abc123",
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "connected_at": "2024-01-15T10:30:45Z"
  }
}
```

### Events

**Message Received:**
```json
{
  "type": "message.received",
  "data": {
    "message_id": "msg_abc123",
    "conversation_id": "conv_xyz789",
    "from_user_id": "sender-uuid",
    "content": "Hello!",
    "type": "text",
    "created_at": "2024-01-15T10:30:45Z"
  }
}
```

**Message Delivered:**
```json
{
  "type": "message.delivered",
  "data": {
    "message_id": "msg_abc123",
    "delivered_at": "2024-01-15T10:30:46Z"
  }
}
```

**Message Read:**
```json
{
  "type": "message.read",
  "data": {
    "message_id": "msg_abc123",
    "read_by": "recipient-uuid",
    "read_at": "2024-01-15T10:31:00Z"
  }
}
```

**Typing Indicator:**
```json
{
  "type": "presence.typing",
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "conversation_id": "conv_xyz789",
    "is_typing": true
  }
}
```

**Presence Update:**
```json
{
  "type": "presence.updated",
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "status": "online",
    "last_seen": "2024-01-15T10:30:45Z"
  }
}
```

**Heartbeat:**

Send every 30 seconds to keep connection alive:
```json
{
  "type": "ping"
}
```

Response:
```json
{
  "type": "pong",
  "timestamp": "2024-01-15T10:30:45Z"
}
```

---

## Error Responses

All error responses follow this format:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "type": "error_type",
    "fields": {},
    "context": {}
  },
  "metadata": {
    "request_id": "req_abc123",
    "correlation_id": "corr_xyz789",
    "timestamp": "2024-01-15T10:30:45Z"
  }
}
```

### Error Types

| Type | Description | HTTP Status |
|------|-------------|-------------|
| `validation` | Input validation failed | 400 |
| `not_found` | Resource not found | 404 |
| `unauthorized` | Authentication required | 401 |
| `forbidden` | Permission denied | 403 |
| `conflict` | Resource conflict | 409 |
| `rate_limit` | Rate limit exceeded | 429 |
| `internal` | Internal server error | 500 |
| `service_unavailable` | Service unavailable | 503 |

### Common Error Codes

| Code | Message | Type |
|------|---------|------|
| `VALIDATION_ERROR` | Validation failed | validation |
| `INVALID_CREDENTIALS` | Invalid credentials | unauthorized |
| `TOKEN_EXPIRED` | Token expired | unauthorized |
| `TOKEN_INVALID` | Invalid token | unauthorized |
| `USER_NOT_FOUND` | User not found | not_found |
| `MESSAGE_NOT_FOUND` | Message not found | not_found |
| `PHONE_EXISTS` | Phone already exists | conflict |
| `RATE_LIMIT_EXCEEDED` | Rate limit exceeded | rate_limit |
| `INTERNAL_ERROR` | Internal server error | internal |

---

## Rate Limiting

All endpoints are rate limited to prevent abuse.

**Rate Limit Headers:**
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1610704845
```

**Default Limits:**

| Endpoint Pattern | Limit | Window |
|-----------------|-------|--------|
| `/api/v1/auth/login` | 5 requests | 1 minute |
| `/api/v1/auth/register` | 3 requests | 5 minutes |
| `/api/v1/auth/verify-otp` | 5 requests | 1 minute |
| `/api/v1/messages` | 100 requests | 1 minute |
| `/api/v1/*` (default) | 1000 requests | 1 minute |

**Rate Limit Exceeded Response:**
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded",
    "type": "rate_limit",
    "retry_after": 45
  }
}
```

---

**Last Updated**: January 2025
**Version**: 1.0.0
