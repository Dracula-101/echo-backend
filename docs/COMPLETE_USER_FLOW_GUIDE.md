# Complete User Flow Guide - Echo Backend

**From Login to Message Broadcasting**

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Phase 1: User Registration & Authentication](#phase-1-user-registration--authentication)
3. [Phase 2: User Profile Creation](#phase-2-user-profile-creation)
4. [Phase 3: Starting a Conversation](#phase-3-starting-a-conversation)
5. [Phase 4: Sending Messages](#phase-4-sending-messages)
6. [Phase 5: Message Broadcasting & Delivery](#phase-5-message-broadcasting--delivery)
7. [Database Schema Reference](#database-schema-reference)
8. [API Endpoints Reference](#api-endpoints-reference)
9. [Service Communication Flow](#service-communication-flow)
10. [Error Handling & Edge Cases](#error-handling--edge-cases)

---

## Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                    Client Applications                           │
│              (Mobile: iOS/Android, Web, Desktop)                 │
└──────────────────────────┬──────────────────────────────────────┘
                           │ HTTPS/REST
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                  API Gateway (Port 8080)                         │
│  • JWT Authentication & Authorization                            │
│  • Rate Limiting (Fixed/Sliding/Token Bucket)                   │
│  • Request Routing & Load Balancing                             │
│  • CORS, Security Headers, Compression                          │
│  • Request/Response Logging & Tracing                           │
└────┬─────────────┬────────────────┬────────────────┬────────────┘
     │             │                │                │
     ▼             ▼                ▼                ▼
┌──────────┐  ┌──────────┐  ┌──────────────┐  ┌──────────────┐
│  Auth    │  │  User    │  │   Message    │  │  Presence    │
│ Service  │  │ Service  │  │   Service    │  │  Service     │
│ :8081    │  │ :50052   │  │   :50051     │  │  :50053      │
│ HTTP/gRPC│  │  gRPC    │  │   gRPC/WS    │  │   gRPC/WS    │
└────┬─────┘  └────┬─────┘  └──────┬───────┘  └──────┬───────┘
     │             │                │                  │
     │             │                │                  │
     ▼             ▼                ▼                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                   PostgreSQL Database (Port 5432)                │
│  Schemas: auth, users, messages, notifications, media           │
└─────────────────────────────────────────────────────────────────┘
                           ▲
                           │
     ┌─────────────────────┴─────────────────────┐
     │                                             │
┌────▼──────┐  ┌──────────────┐  ┌──────────────────┐
│   Redis   │  │    Kafka     │  │   Location       │
│   Cache   │  │  (Planned)   │  │   Service        │
│   :6379   │  │              │  │   :8090          │
└───────────┘  └──────────────┘  └──────────────────┘
```

### Service Responsibilities

| Service | Port | Technology | Responsibility |
|---------|------|------------|----------------|
| **API Gateway** | 8080 | Go + HTTP | Entry point, routing, auth validation |
| **Auth Service** | 8081 | Go + HTTP/gRPC | User authentication, JWT, sessions |
| **User Service** | 50052 | Go + gRPC | User profiles, contacts, settings |
| **Message Service** | 50051 | Go + gRPC/WebSocket | Message CRUD, conversations, real-time |
| **Presence Service** | 50053 | Go + gRPC/WebSocket | Online status, typing indicators |
| **Location Service** | 8090 | Go + HTTP | IP geolocation, location tracking |
| **Notification Service** | TBD | Go + gRPC | Push notifications (FCM/APNS) |
| **Media Service** | TBD | Go + HTTP | File uploads, media storage |

---

## Phase 1: User Registration & Authentication

### 1.1 User Registration Flow

```
Client                API Gateway         Auth Service        PostgreSQL
  │                        │                    │                  │
  │  POST /api/v1/auth/    │                    │                  │
  │       register         │                    │                  │
  ├───────────────────────>│                    │                  │
  │                        │ Validate Request   │                  │
  │                        │ Rate Limit Check   │                  │
  │                        │                    │                  │
  │                        │ Forward Request    │                  │
  │                        ├───────────────────>│                  │
  │                        │                    │ Check Email      │
  │                        │                    │ Availability     │
  │                        │                    ├─────────────────>│
  │                        │                    │<─────────────────┤
  │                        │                    │                  │
  │                        │                    │ Hash Password    │
  │                        │                    │ (bcrypt)         │
  │                        │                    │                  │
  │                        │                    │ Generate Email   │
  │                        │                    │ Verification JWT │
  │                        │                    │                  │
  │                        │                    │ INSERT INTO      │
  │                        │                    │ auth.users       │
  │                        │                    ├─────────────────>│
  │                        │                    │<─────────────────┤
  │                        │                    │ user_id          │
  │                        │                    │                  │
  │                        │  Response          │                  │
  │                        │  {user_id, email}  │                  │
  │                        │<───────────────────┤                  │
  │  201 Created           │                    │                  │
  │  {user_id, email,      │                    │                  │
  │   verification_sent}   │                    │                  │
  │<───────────────────────┤                    │                  │
  │                        │                    │                  │
```

#### Request Body

```json
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "phone_number": "+1234567890",
  "phone_country_code": "+1"
}
```

#### Database Changes

**Table: `auth.users`**
```sql
INSERT INTO auth.users (
    id,                     -- UUID (auto-generated)
    email,                  -- user@example.com
    password_hash,          -- bcrypt hashed password
    password_salt,          -- base64 encoded salt
    password_algorithm,     -- 'bcrypt'
    phone_number,           -- '+1234567890'
    phone_country_code,     -- '+1'
    email_verified,         -- FALSE
    phone_verified,         -- FALSE
    account_status,         -- 'pending' or 'active'
    failed_login_attempts,  -- 0
    two_factor_enabled,     -- FALSE
    created_at,             -- NOW()
    created_by_ip,          -- Client IP
    created_by_user_agent   -- User Agent string
) VALUES (...);
```

#### Response

```json
HTTP/1.1 201 Created
Content-Type: application/json

{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "email_verification_sent": true,
  "verification_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

---

### 1.2 User Login Flow

```
Client          API Gateway       Auth Service      Location Service    PostgreSQL
  │                  │                  │                    │                │
  │  POST /api/v1/   │                  │                    │                │
  │  auth/login      │                  │                    │                │
  ├─────────────────>│                  │                    │                │
  │                  │ Extract:         │                    │                │
  │                  │ • Client IP      │                    │                │
  │                  │ • User-Agent     │                    │                │
  │                  │ • Device Info    │                    │                │
  │                  │                  │                    │                │
  │                  │ Lookup IP        │                    │                │
  │                  │ Geolocation      │                    │                │
  │                  ├──────────────────────────────────────>│                │
  │                  │<──────────────────────────────────────┤                │
  │                  │ {country, city,                       │                │
  │                  │  lat, lng, ISP}                       │                │
  │                  │                  │                    │                │
  │                  │ Forward Login    │                    │                │
  │                  ├─────────────────>│                    │                │
  │                  │                  │ Fetch User by Email│                │
  │                  │                  ├───────────────────────────────────>│
  │                  │                  │<───────────────────────────────────┤
  │                  │                  │                    │                │
  │                  │                  │ Verify Password    │                │
  │                  │                  │ (bcrypt compare)   │                │
  │                  │                  │                    │                │
  │                  │                  │ Generate JWT Tokens│                │
  │                  │                  │ • Access Token     │                │
  │                  │                  │ • Refresh Token    │                │
  │                  │                  │                    │                │
  │                  │                  │ Create/Update      │                │
  │                  │                  │ Session            │                │
  │                  │                  ├───────────────────────────────────>│
  │                  │                  │ INSERT INTO        │                │
  │                  │                  │ auth.sessions      │                │
  │                  │                  │<───────────────────────────────────┤
  │                  │                  │                    │                │
  │                  │                  │ Log Login History  │                │
  │                  │                  ├───────────────────────────────────>│
  │                  │                  │ INSERT INTO        │                │
  │                  │                  │ auth.login_history │                │
  │                  │                  │<───────────────────────────────────┤
  │                  │                  │                    │                │
  │                  │                  │ Log Security Event │                │
  │                  │                  ├───────────────────────────────────>│
  │                  │                  │ INSERT INTO        │                │
  │                  │                  │ auth.security_evt  │                │
  │                  │                  │<───────────────────────────────────┤
  │                  │                  │                    │                │
  │                  │  Login Response  │                    │                │
  │                  │<─────────────────┤                    │                │
  │  200 OK          │                  │                    │                │
  │  {user, session} │                  │                    │                │
  │<─────────────────┤                  │                    │                │
  │                  │                  │                    │                │
```

#### Request Body

```json
POST /api/v1/auth/login
Content-Type: application/json
X-Device-ID: device_abc123
X-Device-Type: mobile
User-Agent: EchoApp/1.0 (iOS 17.0)

{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "fcm_token": "fcm_token_here",    // Optional: Firebase token
  "apns_token": "apns_token_here"   // Optional: Apple Push token
}
```

#### Database Changes

**1. Fetch User from `auth.users`**
```sql
SELECT id, email, password_hash, account_status, 
       email_verified, two_factor_enabled, 
       account_locked_until, failed_login_attempts
FROM auth.users
WHERE email = 'user@example.com' 
  AND deleted_at IS NULL;
```

**2. Create Session in `auth.sessions`**
```sql
INSERT INTO auth.sessions (
    id,                     -- UUID
    user_id,                -- User's UUID
    session_token,          -- JWT access token
    refresh_token,          -- JWT refresh token
    device_id,              -- 'device_abc123'
    device_name,            -- 'iPhone 15 Pro'
    device_type,            -- 'mobile'
    device_os,              -- 'iOS'
    device_os_version,      -- '17.0'
    browser_name,           -- 'Safari' (if web)
    user_agent,             -- Full user agent string
    ip_address,             -- Client IP
    ip_country,             -- 'United States'
    ip_city,                -- 'New York'
    latitude,               -- 40.7128
    longitude,              -- -74.0060
    is_mobile,              -- TRUE
    is_trusted_device,      -- FALSE (first login)
    fcm_token,              -- Firebase token
    apns_token,             -- Apple token
    push_enabled,           -- TRUE
    session_type,           -- 'user'
    expires_at,             -- NOW() + access_token_ttl
    last_activity_at,       -- NOW()
    created_at              -- NOW()
) VALUES (...);
```

**3. Record Login History in `auth.login_history`**
```sql
INSERT INTO auth.login_history (
    user_id,
    session_id,
    login_method,           -- 'password'
    status,                 -- 'success'
    ip_address,
    user_agent,
    device_id,
    device_fingerprint,     -- Generated hash
    location_country,
    location_city,
    latitude,
    longitude,
    is_new_device,          -- TRUE/FALSE
    is_new_location,        -- TRUE/FALSE
    created_at
) VALUES (...);
```

**4. Log Security Event in `auth.security_events`**
```sql
INSERT INTO auth.security_events (
    user_id,
    session_id,
    event_type,             -- 'login_attempt'
    event_category,         -- 'authentication'
    severity,               -- 'medium'
    status,                 -- 'initiated'
    description,            -- 'User login attempt initiated'
    ip_address,
    user_agent,
    device_id,
    location_country,
    location_city,
    is_suspicious,          -- FALSE
    created_at
) VALUES (...);
```

#### Response

```json
HTTP/1.1 200 OK
Content-Type: application/json
X-Request-ID: req_abc123
X-Correlation-ID: corr_xyz789

{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "phone_number": "+1234567890",
    "phone_country_code": "+1",
    "email_verified": true,
    "phone_verified": true,
    "account_status": "active",
    "tfa_enabled": false,
    "created_at": 1699123200,
    "updated_at": 1699123200
  },
  "session": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": 1699209600,
    "token_type": "Bearer"
  }
}
```

---

## Phase 2: User Profile Creation

### 2.1 Create User Profile

After successful login, the user profile is automatically created in the `users.profiles` table.

```
Client          API Gateway       User Service      PostgreSQL
  │                  │                  │                │
  │  POST /api/v1/   │                  │                │
  │  users/profile   │                  │                │
  ├─────────────────>│                  │                │
  │ Authorization:   │ Validate JWT     │                │
  │ Bearer <token>   │                  │                │
  │                  │                  │                │
  │                  │ Forward Request  │                │
  │                  ├─────────────────>│                │
  │                  │                  │ INSERT INTO    │
  │                  │                  │ users.profiles │
  │                  │                  ├───────────────>│
  │                  │                  │<───────────────┤
  │                  │                  │                │
  │                  │                  │ INSERT INTO    │
  │                  │                  │ users.settings │
  │                  │                  ├───────────────>│
  │                  │                  │<───────────────┤
  │                  │  Response        │                │
  │                  │<─────────────────┤                │
  │  201 Created     │                  │                │
  │<─────────────────┤                  │                │
  │                  │                  │                │
```

#### Request Body

```json
POST /api/v1/users/profile
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
Content-Type: application/json

{
  "username": "john_doe",
  "display_name": "John Doe",
  "first_name": "John",
  "last_name": "Doe",
  "bio": "Software Engineer | Tech Enthusiast",
  "date_of_birth": "1990-01-15",
  "gender": "male",
  "language_code": "en",
  "timezone": "America/New_York",
  "country_code": "US",
  "profile_visibility": "public"
}
```

#### Database Changes

**1. Create Profile in `users.profiles`**
```sql
INSERT INTO users.profiles (
    id,                     -- UUID
    user_id,                -- From JWT token
    username,               -- 'john_doe' (unique)
    display_name,           -- 'John Doe'
    first_name,             -- 'John'
    last_name,              -- 'Doe'
    bio,                    -- 'Software Engineer...'
    date_of_birth,          -- '1990-01-15'
    gender,                 -- 'male'
    language_code,          -- 'en'
    timezone,               -- 'America/New_York'
    country_code,           -- 'US'
    online_status,          -- 'offline'
    profile_visibility,     -- 'public'
    search_visibility,      -- TRUE
    is_verified,            -- FALSE
    created_at,             -- NOW()
    updated_at              -- NOW()
) VALUES (...);
```

**2. Create Default Settings in `users.settings`**
```sql
INSERT INTO users.settings (
    user_id,                        -- From JWT token
    
    -- Privacy defaults
    profile_visibility,             -- 'public'
    last_seen_visibility,           -- 'everyone'
    online_status_visibility,       -- 'everyone'
    read_receipts_enabled,          -- TRUE
    typing_indicators_enabled,      -- TRUE
    
    -- Notification defaults
    push_notifications_enabled,     -- TRUE
    message_notifications,          -- TRUE
    group_message_notifications,    -- TRUE
    mention_notifications,          -- TRUE
    call_notifications,             -- TRUE
    
    -- Chat defaults
    enter_key_to_send,              -- FALSE
    auto_download_photos,           -- TRUE
    auto_download_videos,           -- FALSE
    chat_backup_enabled,            -- TRUE
    
    -- Display defaults
    theme,                          -- 'system'
    font_size,                      -- 'medium'
    
    created_at,
    updated_at
) VALUES (...);
```

---

## Phase 3: Starting a Conversation

### 3.1 Creating a Direct Conversation

```
Client          API Gateway       Message Service      PostgreSQL
  │                  │                  │                    │
  │  POST /api/v1/   │                  │                    │
  │  conversations   │                  │                    │
  ├─────────────────>│                  │                    │
  │ Authorization:   │ Validate JWT     │                    │
  │ Bearer <token>   │                  │                    │
  │                  │                  │                    │
  │                  │ Forward Request  │                    │
  │                  ├─────────────────>│                    │
  │                  │                  │ Check if Conv      │
  │                  │                  │ Already Exists     │
  │                  │                  ├───────────────────>│
  │                  │                  │<───────────────────┤
  │                  │                  │                    │
  │                  │                  │ INSERT INTO        │
  │                  │                  │ messages.          │
  │                  │                  │ conversations      │
  │                  │                  ├───────────────────>│
  │                  │                  │<───────────────────┤
  │                  │                  │                    │
  │                  │                  │ Add Participants   │
  │                  │                  ├───────────────────>│
  │                  │                  │ INSERT INTO        │
  │                  │                  │ messages.          │
  │                  │                  │ conversation_      │
  │                  │                  │ participants (x2)  │
  │                  │                  │<───────────────────┤
  │                  │                  │                    │
  │                  │                  │ INSERT INTO        │
  │                  │                  │ messages.          │
  │                  │                  │ conversation_      │
  │                  │                  │ settings           │
  │                  │                  ├───────────────────>│
  │                  │                  │<───────────────────┤
  │                  │  Response        │                    │
  │                  │<─────────────────┤                    │
  │  201 Created     │                  │                    │
  │<─────────────────┤                  │                    │
  │                  │                  │                    │
```

#### Request Body

```json
POST /api/v1/conversations
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
Content-Type: application/json

{
  "conversation_type": "direct",
  "participant_user_ids": [
    "550e8400-e29b-41d4-a716-446655440001"
  ],
  "is_encrypted": true
}
```

#### Database Changes

**1. Check for Existing Conversation**
```sql
-- Check if direct conversation already exists between these users
SELECT c.id 
FROM messages.conversations c
INNER JOIN messages.conversation_participants cp1 
    ON c.id = cp1.conversation_id
INNER JOIN messages.conversation_participants cp2 
    ON c.id = cp2.conversation_id
WHERE c.conversation_type = 'direct'
  AND c.deleted_at IS NULL
  AND cp1.user_id = '550e8400-e29b-41d4-a716-446655440000'  -- Current user
  AND cp2.user_id = '550e8400-e29b-41d4-a716-446655440001'  -- Other user
  AND cp1.left_at IS NULL
  AND cp2.left_at IS NULL;
```

**2. Create Conversation in `messages.conversations`**
```sql
INSERT INTO messages.conversations (
    id,                     -- UUID
    conversation_type,      -- 'direct'
    creator_user_id,        -- Current user's UUID
    is_group,               -- FALSE
    is_channel,             -- FALSE
    is_encrypted,           -- TRUE
    encryption_key_id,      -- Generated key ID
    is_public,              -- FALSE
    who_can_send_messages,  -- 'all'
    is_active,              -- TRUE
    member_count,           -- 2
    message_count,          -- 0
    last_activity_at,       -- NOW()
    created_at,             -- NOW()
    updated_at              -- NOW()
) VALUES (...)
RETURNING id;
```

**3. Add Participants to `messages.conversation_participants`**
```sql
-- Add current user
INSERT INTO messages.conversation_participants (
    id,                     -- UUID
    conversation_id,        -- From step 2
    user_id,                -- Current user UUID
    role,                   -- 'member'
    can_send_messages,      -- TRUE
    can_send_media,         -- TRUE
    is_muted,               -- FALSE
    is_pinned,              -- FALSE
    is_archived,            -- FALSE
    unread_count,           -- 0
    mention_count,          -- 0
    join_method,            -- 'added'
    joined_at,              -- NOW()
    created_at              -- NOW()
) VALUES (...);

-- Add other participant
INSERT INTO messages.conversation_participants (
    id,                     -- UUID
    conversation_id,        -- From step 2
    user_id,                -- Other user UUID
    role,                   -- 'member'
    can_send_messages,      -- TRUE
    can_send_media,         -- TRUE
    is_muted,               -- FALSE
    is_pinned,              -- FALSE
    is_archived,            -- FALSE
    unread_count,           -- 0
    mention_count,          -- 0
    join_method,            -- 'added'
    invited_by_user_id,     -- Current user UUID
    joined_at,              -- NOW()
    created_at              -- NOW()
) VALUES (...);
```

**4. Create Conversation Settings in `messages.conversation_settings`**
```sql
INSERT INTO messages.conversation_settings (
    conversation_id,                    -- From step 2
    disappearing_messages_enabled,      -- FALSE
    message_history_enabled,            -- TRUE
    screenshot_notification,            -- FALSE
    read_receipts_enabled,              -- TRUE
    typing_indicators_enabled,          -- TRUE
    link_previews_enabled,              -- TRUE
    auto_download_media,                -- TRUE
    created_at,
    updated_at
) VALUES (...);
```

#### Response

```json
HTTP/1.1 201 Created
Content-Type: application/json

{
  "conversation_id": "660e8400-e29b-41d4-a716-446655440000",
  "conversation_type": "direct",
  "is_encrypted": true,
  "participants": [
    {
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "role": "member",
      "joined_at": 1699123200
    },
    {
      "user_id": "550e8400-e29b-41d4-a716-446655440001",
      "role": "member",
      "joined_at": 1699123200
    }
  ],
  "member_count": 2,
  "message_count": 0,
  "created_at": 1699123200
}
```

---

## Phase 4: Sending Messages

### 4.1 Send a Text Message

```
Client          API Gateway       Message Service      PostgreSQL      WebSocket Hub
  │                  │                  │                    │                 │
  │  POST /api/v1/   │                  │                    │                 │
  │  messages        │                  │                    │                 │
  ├─────────────────>│                  │                    │                 │
  │                  │ Validate JWT     │                    │                 │
  │                  │ Extract user_id  │                    │                 │
  │                  │                  │                    │                 │
  │                  │ Forward Request  │                    │                 │
  │                  ├─────────────────>│                    │                 │
  │                  │                  │ Validate User is   │                 │
  │                  │                  │ Participant        │                 │
  │                  │                  ├───────────────────>│                 │
  │                  │                  │<───────────────────┤                 │
  │                  │                  │                    │                 │
  │                  │                  │ INSERT Message     │                 │
  │                  │                  ├───────────────────>│                 │
  │                  │                  │<───────────────────┤                 │
  │                  │                  │ message_id         │                 │
  │                  │                  │                    │                 │
  │                  │                  │ Get All Participants│                │
  │                  │                  ├───────────────────>│                 │
  │                  │                  │<───────────────────┤                 │
  │                  │                  │ [user_ids]         │                 │
  │                  │                  │                    │                 │
  │                  │                  │ Update Conversation│                 │
  │                  │                  ├───────────────────>│                 │
  │                  │                  │<───────────────────┤                 │
  │                  │                  │                    │                 │
  │                  │  Response        │                    │                 │
  │                  │<─────────────────┤                    │                 │
  │  201 Created     │                  │                    │                 │
  │<─────────────────┤                  │                    │                 │
  │                  │                  │                    │                 │
  │                  │                  │ [Async] Broadcast  │                 │
  │                  │                  │ to Recipients      │                 │
  │                  │                  ├────────────────────────────────────>│
  │                  │                  │                    │                 │
```

#### Request Body

```json
POST /api/v1/messages
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
Content-Type: application/json

{
  "conversation_id": "660e8400-e29b-41d4-a716-446655440000",
  "content": "Hey! How are you doing?",
  "message_type": "text",
  "reply_to_id": null,
  "mentions": []
}
```

#### Database Changes

**1. Validate Participant Permission**
```sql
SELECT 1 
FROM messages.conversation_participants
WHERE conversation_id = '660e8400-e29b-41d4-a716-446655440000'
  AND user_id = '550e8400-e29b-41d4-a716-446655440000'
  AND can_send_messages = TRUE
  AND left_at IS NULL;
```

**2. Insert Message into `messages.messages`**
```sql
INSERT INTO messages.messages (
    id,                     -- UUID (generated)
    conversation_id,        -- '660e8400-e29b-41d4-a716-446655440000'
    sender_user_id,         -- Current user UUID
    content,                -- 'Hey! How are you doing?'
    message_type,           -- 'text'
    status,                 -- 'sent'
    content_encrypted,      -- TRUE (if E2E enabled)
    format_type,            -- 'plain'
    mentions,               -- '[]'::JSONB
    is_edited,              -- FALSE
    is_deleted,             -- FALSE
    is_pinned,              -- FALSE
    delivery_count,         -- 0
    read_count,             -- 0
    reaction_count,         -- 0
    reply_count,            -- 0
    sent_from_device_id,    -- Device ID from request
    sent_from_ip,           -- Client IP
    created_at,             -- NOW()
    updated_at              -- NOW()
) VALUES (...)
RETURNING id, created_at;
```

**3. Get All Conversation Participants**
```sql
SELECT user_id 
FROM messages.conversation_participants
WHERE conversation_id = '660e8400-e29b-41d4-a716-446655440000'
  AND left_at IS NULL;
```

**4. Update Conversation Metadata**
```sql
UPDATE messages.conversations
SET
    last_message_id = '770e8400-e29b-41d4-a716-446655440000',
    last_message_at = NOW(),
    last_activity_at = NOW(),
    message_count = message_count + 1,
    updated_at = NOW()
WHERE id = '660e8400-e29b-41d4-a716-446655440000';
```

**5. Initialize Delivery Status for Each Participant**
```sql
-- For each participant (except sender)
INSERT INTO messages.delivery_status (
    id,                     -- UUID
    message_id,             -- Message UUID from step 2
    user_id,                -- Participant UUID
    status,                 -- 'sent'
    created_at,             -- NOW()
    updated_at              -- NOW()
) VALUES (...);
```

**6. Update Unread Count for Recipients**
```sql
-- For each participant (except sender)
UPDATE messages.conversation_participants
SET unread_count = unread_count + 1,
    updated_at = NOW()
WHERE conversation_id = '660e8400-e29b-41d4-a716-446655440000'
  AND user_id != '550e8400-e29b-41d4-a716-446655440000';  -- Not sender
```

#### Response

```json
HTTP/1.1 201 Created
Content-Type: application/json
X-Message-ID: 770e8400-e29b-41d4-a716-446655440000

{
  "id": "770e8400-e29b-41d4-a716-446655440000",
  "conversation_id": "660e8400-e29b-41d4-a716-446655440000",
  "sender_user_id": "550e8400-e29b-41d4-a716-446655440000",
  "content": "Hey! How are you doing?",
  "message_type": "text",
  "status": "sent",
  "mentions": [],
  "created_at": 1699123456,
  "delivery_status": {
    "sent": 1,
    "delivered": 0,
    "read": 0
  }
}
```

---

## Phase 5: Message Broadcasting & Delivery

### 5.1 Real-Time Broadcasting Flow

```
Message Service          WebSocket Hub       Presence Service     PostgreSQL
       │                       │                      │                 │
       │ Get Participants      │                      │                 │
       │ for Broadcast         │                      │                 │
       │                       │                      │                 │
       │ For Each Recipient:   │                      │                 │
       │ Check Online Status   │                      │                 │
       ├──────────────────────────────────────────────>│                 │
       │                       │                      │                 │
       │                       │   Is User Online?    │                 │
       │<──────────────────────────────────────────────┤                 │
       │                       │   {user_id, online}  │                 │
       │                       │                      │                 │
       │ If Online:            │                      │                 │
       │ Broadcast Message     │                      │                 │
       ├──────────────────────>│                      │                 │
       │                       │ Send to User         │                 │
       │                       │ (All Devices)        │                 │
       │                       │                      │                 │
       │                       ├──> Device 1          │                 │
       │                       ├──> Device 2          │                 │
       │                       │                      │                 │
       │ Update Delivery       │                      │                 │
       │ Status: 'delivered'   │                      │                 │
       ├───────────────────────────────────────────────────────────────>│
       │                       │                      │                 │
       │ If Offline:           │                      │                 │
       │ Queue Push            │                      │                 │
       │ Notification          │                      │                 │
       │                       │                      │                 │
```

### 5.2 WebSocket Message Format

When a message is broadcast to online users via WebSocket:

```json
{
  "type": "new_message",
  "event_id": "evt_abc123",
  "timestamp": 1699123456,
  "data": {
    "message": {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "conversation_id": "660e8400-e29b-41d4-a716-446655440000",
      "sender_user_id": "550e8400-e29b-41d4-a716-446655440000",
      "sender": {
        "username": "john_doe",
        "display_name": "John Doe",
        "avatar_url": "https://cdn.example.com/avatars/john.jpg"
      },
      "content": "Hey! How are you doing?",
      "message_type": "text",
      "status": "delivered",
      "mentions": [],
      "created_at": 1699123456
    }
  }
}
```

### 5.3 Recipient Receives Message

```
WebSocket        Client App       Local Storage      UI
    │                 │                  │            │
    │ new_message     │                  │            │
    │ event           │                  │            │
    ├────────────────>│                  │            │
    │                 │ Decrypt (E2E)    │            │
    │                 │                  │            │
    │                 │ Save to Local DB │            │
    │                 ├─────────────────>│            │
    │                 │                  │            │
    │                 │ Update UI        │            │
    │                 ├───────────────────────────────>│
    │                 │                  │  Display   │
    │                 │                  │  Message   │
    │                 │                  │            │
    │                 │ Send Read Receipt│            │
    │<────────────────┤                  │            │
    │                 │                  │            │
```

### 5.4 Read Receipt Flow

```
Client          WebSocket        Message Service      PostgreSQL
  │                  │                  │                  │
  │ Mark as Read     │                  │                  │
  │ {message_id,     │                  │                  │
  │  conversation_id}│                  │                  │
  ├─────────────────>│                  │                  │
  │                  │ read_receipt     │                  │
  │                  │ event            │                  │
  │                  ├─────────────────>│                  │
  │                  │                  │ UPDATE           │
  │                  │                  │ delivery_status  │
  │                  │                  ├─────────────────>│
  │                  │                  │ SET status=read  │
  │                  │                  │ read_at=NOW()    │
  │                  │                  │<─────────────────┤
  │                  │                  │                  │
  │                  │                  │ UPDATE           │
  │                  │                  │ conversation_    │
  │                  │                  │ participants     │
  │                  │                  ├─────────────────>│
  │                  │                  │ unread_count=0   │
  │                  │                  │<─────────────────┤
  │                  │                  │                  │
  │                  │                  │ Notify Sender    │
  │                  │                  │ via WebSocket    │
  │                  │                  │                  │
```

#### Database Changes for Read Receipt

**1. Update Delivery Status**
```sql
UPDATE messages.delivery_status
SET
    status = 'read',
    read_at = NOW(),
    updated_at = NOW()
WHERE message_id = '770e8400-e29b-41d4-a716-446655440000'
  AND user_id = '550e8400-e29b-41d4-a716-446655440001';
```

**2. Update Message Read Count**
```sql
UPDATE messages.messages
SET
    read_count = (
        SELECT COUNT(*) 
        FROM messages.delivery_status 
        WHERE message_id = '770e8400-e29b-41d4-a716-446655440000' 
          AND status = 'read'
    ),
    updated_at = NOW()
WHERE id = '770e8400-e29b-41d4-a716-446655440000';
```

**3. Update Participant Last Read**
```sql
UPDATE messages.conversation_participants
SET
    last_read_message_id = '770e8400-e29b-41d4-a716-446655440000',
    last_read_at = NOW(),
    unread_count = 0,
    updated_at = NOW()
WHERE conversation_id = '660e8400-e29b-41d4-a716-446655440000'
  AND user_id = '550e8400-e29b-41d4-a716-446655440001';
```

---

## Database Schema Reference

### Core Tables Overview

#### 1. Authentication Schema (`auth`)

| Table | Purpose | Key Fields |
|-------|---------|------------|
| `auth.users` | User credentials | email, password_hash, account_status |
| `auth.sessions` | Active user sessions | user_id, session_token, expires_at |
| `auth.login_history` | Login attempts log | user_id, status, ip_address, device_id |
| `auth.security_events` | Security audit trail | event_type, severity, is_suspicious |
| `auth.otp_verifications` | OTP codes | identifier, otp_code, purpose |

#### 2. User Schema (`users`)

| Table | Purpose | Key Fields |
|-------|---------|------------|
| `users.profiles` | User profile data | username, display_name, bio, avatar_url |
| `users.contacts` | Friend relationships | user_id, contact_user_id, status |
| `users.settings` | User preferences | privacy settings, notification settings |
| `users.blocked_users` | Blocked users list | user_id, blocked_user_id |
| `users.devices` | User devices | device_id, fcm_token, apns_token |

#### 3. Message Schema (`messages`)

| Table | Purpose | Key Fields |
|-------|---------|------------|
| `messages.conversations` | Chat conversations | conversation_type, member_count |
| `messages.conversation_participants` | Conversation members | user_id, role, unread_count |
| `messages.messages` | Chat messages | content, message_type, status |
| `messages.delivery_status` | Per-user delivery | status, delivered_at, read_at |
| `messages.reactions` | Message reactions | message_id, user_id, reaction_emoji |
| `messages.typing_indicators` | Typing status | conversation_id, user_id, expires_at |

#### 4. Notification Schema (`notifications`)

| Table | Purpose | Key Fields |
|-------|---------|------------|
| `notifications.notifications` | App notifications | notification_type, title, body |
| `notifications.push_delivery_log` | Push delivery tracking | push_token, status, provider |

---

## API Endpoints Reference

### Authentication Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | User login |
| POST | `/api/v1/auth/logout` | User logout |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| POST | `/api/v1/auth/verify-email` | Verify email address |
| POST | `/api/v1/auth/forgot-password` | Request password reset |
| POST | `/api/v1/auth/reset-password` | Reset password |

### User Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users/profile` | Get current user profile |
| PUT | `/api/v1/users/profile` | Update user profile |
| GET | `/api/v1/users/{user_id}` | Get user by ID |
| GET | `/api/v1/users/search` | Search users |
| GET | `/api/v1/users/contacts` | Get user contacts |
| POST | `/api/v1/users/contacts` | Add contact |
| DELETE | `/api/v1/users/contacts/{contact_id}` | Remove contact |

### Conversation Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/conversations` | List user conversations |
| POST | `/api/v1/conversations` | Create conversation |
| GET | `/api/v1/conversations/{id}` | Get conversation details |
| PUT | `/api/v1/conversations/{id}` | Update conversation |
| DELETE | `/api/v1/conversations/{id}` | Delete conversation |
| POST | `/api/v1/conversations/{id}/participants` | Add participants |
| DELETE | `/api/v1/conversations/{id}/participants/{user_id}` | Remove participant |

### Message Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/conversations/{id}/messages` | Get messages |
| POST | `/api/v1/messages` | Send message |
| GET | `/api/v1/messages/{id}` | Get message details |
| PUT | `/api/v1/messages/{id}` | Edit message |
| DELETE | `/api/v1/messages/{id}` | Delete message |
| POST | `/api/v1/messages/{id}/reactions` | Add reaction |
| POST | `/api/v1/messages/{id}/read` | Mark as read |

### WebSocket Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /ws` | Establish WebSocket connection |

---

## Service Communication Flow

### Complete End-to-End Flow Summary

```
1. USER REGISTRATION
   Client → API Gateway → Auth Service → PostgreSQL (auth.users)
   
2. USER LOGIN
   Client → API Gateway → Auth Service → Location Service → PostgreSQL
   Creates: Session, Login History, Security Event
   Returns: JWT Access Token + Refresh Token
   
3. CREATE PROFILE
   Client → API Gateway → User Service → PostgreSQL (users.profiles, users.settings)
   
4. START CONVERSATION
   Client → API Gateway → Message Service → PostgreSQL
   Creates: Conversation, Participants, Settings
   
5. SEND MESSAGE
   Client → API Gateway → Message Service → PostgreSQL
   Creates: Message, Delivery Status
   Updates: Conversation metadata, Unread counts
   
6. BROADCAST MESSAGE
   Message Service → Presence Service (check online status)
   If Online: Message Service → WebSocket Hub → Connected Clients
   If Offline: Message Service → Notification Service → Push (FCM/APNS)
   
7. RECEIVE MESSAGE
   WebSocket → Client → Local Storage → UI
   Client sends Read Receipt → Message Service → PostgreSQL
   
8. READ RECEIPT
   Updates: Delivery Status, Message Read Count, Unread Count
   Notifies sender via WebSocket
```

---

## Error Handling & Edge Cases

### Common Error Scenarios

#### 1. Authentication Errors

```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid email or password",
    "status": 401
  }
}
```

#### 2. Conversation Not Found

```json
{
  "error": {
    "code": "CONVERSATION_NOT_FOUND",
    "message": "Conversation not found or you don't have access",
    "status": 404
  }
}
```

#### 3. Rate Limiting

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please try again later.",
    "status": 429,
    "retry_after": 60
  }
}
```

#### 4. Message Send Failure

```json
{
  "error": {
    "code": "MESSAGE_SEND_FAILED",
    "message": "Failed to send message",
    "status": 500,
    "details": "User not authorized to send messages in this conversation"
  }
}
```

### Edge Cases Handled

1. **Duplicate Conversations**: Check for existing direct conversations before creating
2. **Offline Users**: Queue push notifications when WebSocket unavailable
3. **Multiple Devices**: Broadcast to all user devices via WebSocket Hub
4. **Deleted Users**: Cascade deletes or set to NULL with ON DELETE clauses
5. **Token Expiration**: Automatic refresh token rotation
6. **Network Failures**: Client retry logic with exponential backoff
7. **Message Ordering**: Use created_at timestamp and database constraints
8. **Read Receipts Privacy**: Respect user settings for read receipts
9. **Blocked Users**: Filter blocked users from conversation participants
10. **Account Suspension**: Check account_status before allowing actions

---

## Environment Setup Checklist

### Required Services

- [x] PostgreSQL 15+ (Port 5432)
- [x] Redis 7+ (Port 6379)
- [x] API Gateway (Port 8080)
- [x] Auth Service (Port 8081)
- [x] Location Service (Port 8090)
- [ ] Message Service (Port 50051) - Stub
- [ ] User Service (Port 50052) - Stub
- [ ] Presence Service (Port 50053) - Stub
- [ ] Notification Service - Stub

### Database Schemas

- [x] `auth` schema
- [x] `users` schema
- [x] `messages` schema
- [x] `notifications` schema
- [x] `media` schema (partial)
- [x] `analytics` schema (partial)

### Next Steps for Implementation

1. **Implement Message Service** with WebSocket support
2. **Implement User Service** for profile management
3. **Implement Presence Service** for online status
4. **Add Kafka** for asynchronous message processing
5. **Implement Notification Service** with FCM/APNS
6. **Add E2E Encryption** for messages
7. **Implement Media Service** for file uploads
8. **Add Monitoring** with Prometheus/Grafana

---

**Document Version**: 1.0  
**Last Updated**: November 3, 2025  
**Author**: Echo Backend Team
