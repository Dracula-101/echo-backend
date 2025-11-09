# Echo Backend - Messaging Service Architecture Overview

**Generated:** November 4, 2025  
**Status:** Current state analysis + implementation roadmap  
**Purpose:** Comprehensive understanding of messaging service architecture for production-ready system design

---

## Executive Summary

The Echo Backend project has a **well-architected foundation** for a production-ready messaging system with:

- **Comprehensive database schema** designed for messaging, conversations, calls, and reactions
- **Kafka-based event streaming infrastructure** for async processing
- **Shared packages** for messaging, logging, caching, and monitoring
- **Stub message service** ready for implementation
- **API Gateway** with routing, rate limiting, and service discovery
- **Microservices architecture** following clean architecture patterns

**Current Implementation Status:** The message service is a structural stub that needs core implementation (repositories, services, WebSocket handlers).

---

## 1. PROJECT STRUCTURE & FILE ORGANIZATION

### Root Directory Layout
```
echo-backend/
├── services/                    # 9 microservices
│   ├── api-gateway/            # Main entry point (Port 8080)
│   ├── auth-service/           # Authentication (Port 8081)
│   ├── message-service/        # Messaging (Port 8083 HTTP, 50053 gRPC)
│   ├── notification-service/   # Push notifications
│   ├── user-service/           # User profiles & contacts
│   ├── media-service/          # File uploads & media
│   ├── presence-service/       # Online status tracking
│   ├── location-service/       # IP geolocation (Port 8090)
│   └── analytics-service/      # User behavior tracking
├── shared/                      # Shared libraries
│   ├── pkg/                    # Core packages
│   │   ├── messaging/          # Kafka producer/consumer
│   │   ├── cache/              # Redis caching
│   │   ├── database/           # DB connection pooling
│   │   ├── logger/             # Structured logging
│   │   ├── monitoring/         # Prometheus metrics
│   │   └── grpc/               # gRPC utilities
│   └── server/                 # HTTP server utilities
├── database/                    # Schema definitions
│   ├── schemas/                # 8 SQL schema files
│   ├── functions/              # PL/pgSQL functions
│   ├── triggers/               # Trigger definitions
│   ├── indexes/                # Performance indexes
│   └── views/                  # Analytics views
├── migrations/                  # Database migrations
├── infra/                       # Infrastructure as Code
│   ├── docker/                 # Docker Compose files
│   ├── kubernetes/             # K8s manifests
│   └── terraform/              # Terraform configs
├── docs/                        # Documentation
├── Makefile                     # Development commands
└── go.work                      # Go workspace configuration
```

### Message Service Directory Structure
```
services/message-service/
├── cmd/
│   └── server/
│       └── main.go             # Entry point (STUB - empty)
├── internal/
│   ├── config/                 # Configuration management
│   ├── domain/                 # Domain models
│   ├── infra/                  # Infrastructure setup
│   └── (TODO: handlers, service, repo, websocket/)
├── go.mod                       # Module definition
├── Dockerfile
├── Dockerfile.dev
└── migrations/                  # Service-specific migrations
```

---

## 2. DATABASE MODELS & SCHEMAS

### Core Database Architecture
- **System:** PostgreSQL 15+ with extensions (uuid-ossp, pgcrypto, pg_trgm)
- **Design Pattern:** Multi-schema architecture with 8 logical domains
- **Total Tables:** 60+ tables across all schemas
- **Security:** Row-Level Security (RLS) enabled on most tables

### Messages Schema (Primary Focus)
Located: `/database/schemas/message-schema.sql`

#### Core Tables (20 tables)

**1. `messages.conversations`**
- **Purpose:** Chat rooms, groups, channels, broadcasts
- **Key Fields:**
  - `conversation_type` (direct, group, channel, broadcast)
  - `title`, `description`, `avatar_url`
  - `creator_user_id`, `member_count`, `message_count`
  - `last_message_id`, `last_message_at`, `last_activity_at`
  - `is_encrypted`, `is_public`, `is_archived`
  - Permissions: `who_can_send_messages`, `who_can_add_members`, etc.
- **Features:** Invite links with expiration, group settings, archiving

**2. `messages.conversation_participants`**
- **Purpose:** N-to-N relationship between users and conversations
- **Key Fields:**
  - `conversation_id`, `user_id`, `role` (owner, admin, moderator, member)
  - `last_read_message_id`, `last_read_at`, `unread_count`
  - `is_muted`, `is_pinned`, `is_archived`
  - Fine-grained permissions: `can_send_messages`, `can_add_members`, `can_delete_messages`, etc.
  - Join metadata: `join_method`, `joined_at`, `left_at`, `removed_by`

**3. `messages.messages`**
- **Purpose:** Core message content table
- **Key Fields:**
  - `sender_user_id`, `conversation_id`, `parent_message_id` (for threads/replies)
  - `content`, `message_type` (text, image, video, audio, document, location, etc.)
  - `content_encrypted`, `content_hash` (for deduplication)
  - `status` (sending, sent, delivered, read, failed)
  - Metadata: `mentions` (JSONB), `hashtags`, `links`
  - Edit tracking: `is_edited`, `edited_at`, `edit_history`
  - Soft delete: `is_deleted`, `deleted_at`, `deleted_for`
  - Reactions: `reaction_count`, `reply_count`
  - Forward tracking: `is_forwarded`, `forwarded_from_message_id`, `forward_count`
  - Expiration: `expires_at`, `expire_after_seconds` (disappearing messages)
  - Device tracking: `sent_from_device_id`, `sent_from_ip`
  - Scheduled messages: `scheduled_at`, `is_scheduled`

**4. `messages.delivery_status`**
- **Purpose:** Per-recipient message delivery tracking
- **Key Fields:**
  - `message_id`, `user_id`, `status` (sent, delivered, read, failed)
  - `delivered_at`, `read_at`, `failed_reason`
  - `retry_count`, `device_id`
- **Usage:** Tracks delivery to each conversation participant separately

**5. `messages.reactions`**
- **Purpose:** Emoji/custom reactions on messages
- **Key Fields:**
  - `message_id`, `user_id`, `reaction_type`, `reaction_emoji`, `reaction_skin_tone`
- **Constraint:** UNIQUE(message_id, user_id, reaction_type) - one reaction per user per type

**6. `messages.typing_indicators`**
- **Purpose:** Real-time typing status
- **Key Fields:**
  - `conversation_id`, `user_id`, `started_at`, `expires_at`
- **Pattern:** Expires automatically (5-10 second TTL)

**7. `messages.message_media`**
- **Purpose:** Media attachments in messages
- **Key Fields:**
  - `message_id`, `media_id`, `media_type`
  - `display_order`, `caption`, `thumbnail_url`

**8. `messages.link_previews`**
- **Purpose:** URL preview metadata
- **Key Fields:**
  - `message_id`, `url`, `title`, `description`, `image_url`, `favicon_url`, `site_name`

**9. `messages.polls`**
- **Purpose:** Poll/voting functionality
- **Key Fields:**
  - `message_id`, `question`, `allow_multiple_answers`, `is_anonymous`, `is_quiz`
  - `correct_option_id`, `closes_at`, `is_closed`, `total_votes`

**10. `messages.poll_options` & `messages.poll_votes`**
- Supporting tables for poll voting mechanism

**11. `messages.calls`**
- **Purpose:** Voice/video call tracking
- **Key Fields:**
  - `conversation_id`, `call_type` (voice, video)
  - `initiator_user_id`, `status` (initiated, ringing, active, ended, missed)
  - `started_at`, `ended_at`, `duration_seconds`
  - Quality metrics: `video_quality`, `audio_quality`, `connection_quality`, `packet_loss_percentage`

**12. `messages.call_participants`**
- Supporting table for group calls

**13. `messages.message_reports`**
- **Purpose:** Content moderation/reporting
- **Key Fields:**
  - `message_id`, `reporter_user_id`, `report_type` (spam, harassment, violence, etc.)
  - `status`, `priority`, `action_taken`, `resolved_at`

**14. `messages.drafts`**
- **Purpose:** Auto-save message drafts
- **Key Fields:**
  - `user_id`, `conversation_id`, `content`, `reply_to_message_id`
  - `mentions`, `attachments` (JSONB)

**15. `messages.bookmarks`**
- **Purpose:** Saved messages for later
- **Key Fields:**
  - `user_id`, `message_id`, `collection_name`, `notes`, `tags`

**16. `messages.pinned_messages`**
- **Purpose:** Pinned messages in conversations
- **Key Fields:**
  - `conversation_id`, `message_id`, `pinned_by_user_id`, `pin_order`

**17. `messages.conversation_invites`**
- **Purpose:** Invitation management
- **Key Fields:**
  - `conversation_id`, `inviter_user_id`, `invitee_user_id`, `invitee_phone_number`, `invitee_email`
  - `invite_code`, `status` (pending, accepted, rejected, expired, revoked)
  - `max_uses`, `use_count`, `expires_at`

**18. `messages.search_index`**
- **Purpose:** Full-text search optimization
- **Key Fields:**
  - `message_id`, `conversation_id`, `content_tsvector` (PostgreSQL tsvector)

**19. `messages.conversation_settings`**
- **Purpose:** Conversation-level feature flags
- **Key Fields:**
  - `disappearing_messages_enabled`, `disappearing_messages_duration`
  - `message_history_enabled`, `screenshot_notification`
  - `read_receipts_enabled`, `typing_indicators_enabled`, `link_previews_enabled`

**20. Additional Tables:**
- `message_edits` - Edit history detail
- `message_receipts` - Detailed delivery tracking (alternative to delivery_status)

### Performance Indexes
```sql
CREATE INDEX idx_messages_conversation ON messages.messages(conversation_id, created_at DESC);
CREATE INDEX idx_messages_sender ON messages.messages(sender_user_id, created_at DESC);
CREATE INDEX idx_messages_parent ON messages.messages(parent_message_id);
CREATE INDEX idx_participants_user ON messages.conversation_participants(user_id);
CREATE INDEX idx_participants_conversation ON messages.conversation_participants(conversation_id);
CREATE INDEX idx_delivery_status_message ON messages.delivery_status(message_id);
CREATE INDEX idx_delivery_status_user ON messages.delivery_status(user_id);
CREATE INDEX idx_search_index_content ON messages.search_index USING GIN(content_tsvector);
```

### Database Functions
- `messages.is_conversation_participant(conv_id UUID)` - Check membership
- `messages.is_conversation_admin(conv_id UUID)` - Check admin role
- `messages.can_send_messages(conv_id UUID)` - Check permissions
- `messages.can_delete_message(msg_id UUID)` - Check deletion rights

### Database Triggers
- `update_conversation_stats()` - Auto-increment message count
- `update_unread_counts()` - Auto-increment unread count for participants
- `update_updated_at_column()` - Auto-update timestamps (all tables)

---

## 3. API ENDPOINTS & ROUTING

### API Gateway Configuration
**File:** `/services/api-gateway/internal/config/config.go`  
**Port:** 8080 (HTTP/REST)

### Service Registry Structure
```yaml
services:
  auth:
    protocol: grpc
    addresses:
      - localhost:50051
    timeout: 30s
    
  message:
    protocol: grpc
    addresses:
      - localhost:50053
    timeout: 30s
    
  user:
    protocol: grpc
    addresses:
      - localhost:50052
    timeout: 30s
```

### Planned Message Service Endpoints

**REST API (Port 8083)**
```
POST   /api/v1/messages                    - Send message
GET    /api/v1/messages/{conversation_id} - Get messages in conversation
PUT    /api/v1/messages/{message_id}      - Edit message
DELETE /api/v1/messages/{message_id}      - Delete message
GET    /api/v1/conversations              - List conversations
POST   /api/v1/conversations              - Create conversation
GET    /api/v1/conversations/{id}         - Get conversation details
PUT    /api/v1/conversations/{id}         - Update conversation
GET    /api/v1/conversations/{id}/members - List participants
POST   /api/v1/conversations/{id}/members - Add participant
DELETE /api/v1/conversations/{id}/members/{user_id} - Remove participant
POST   /api/v1/messages/{id}/reactions    - Add reaction
POST   /api/v1/messages/{id}/read-receipt - Mark as read
POST   /api/v1/messages/{id}/typing       - Send typing indicator
```

**gRPC API (Port 50053)**
```protobuf
service MessageService {
  rpc SendMessage(SendMessageRequest) returns (SendMessageResponse);
  rpc GetMessages(GetMessagesRequest) returns (GetMessagesResponse);
  rpc GetConversations(GetConversationsRequest) returns (GetConversationsResponse);
  rpc CreateConversation(CreateConversationRequest) returns (ConversationResponse);
  rpc StreamMessages(StreamMessagesRequest) returns (stream MessageEvent);
}
```

**WebSocket API (Port 9000)**
```
ws://localhost:9000/ws

Message types:
  - new_message      (server → client)
  - message_read     (server → client)
  - typing_indicator (bidirectional)
  - read_receipt     (client → server)
  - user_online      (server → client)
  - user_offline     (server → client)
```

### Authentication & Authorization
- **Method:** JWT tokens in `Authorization: Bearer <token>` header
- **Validation:** Done at API Gateway level
- **Scope:** Per-user, per-conversation permissions

### Rate Limiting
**Configuration:** `/services/api-gateway/internal/config/config.go`
```yaml
ratelimit:
  enabled: true
  store: redis
  global:
    requests: 100
    window: 1m
  endpoints:
    /auth/login:
      requests: 10
      window: 1m
    /messages:
      requests: 100
      window: 1m
```

---

## 4. MESSAGE HANDLING & FLOW

### Current Message Flow (as documented)

```
1. Client sends message
   └─> POST /api/v1/messages {conversation_id, content, message_type}
   └─> API Gateway (port 8080)
       - Validates JWT token
       - Rate limiting
       - Input validation
   
2. Message Service (port 8083)
   └─> SaveMessage()
       - Validate sender is conversation participant
       - Check permissions (can_send_messages)
       - Insert into messages.messages
       - Update messages.conversations (last_message_at, message_count)
       - Get all conversation participants
   
3. Broadcast to participants
   └─> For each participant (except sender):
       ├─> Online (has active WebSocket)?
       │   └─> SendViaWebSocket()
       │       - Send via hub.SendToUser()
       │       - Update delivery_status = 'delivered'
       │       - Trigger: 'message_delivered' event
       │
       └─> Offline?
           └─> SendPushNotification()
               - Publish to Kafka topic: notifications
               - Update delivery_status = 'sent'
               - Notification Service picks up
               - Sends FCM/APNS push notification

4. Client receives read receipt
   └─> Client sends: {type: 'read_receipt', message_id, read_at}
   └─> Server updates: delivery_status.status = 'read', delivered_status.read_at = NOW()
   └─> Broadcast: MessageRead event to sender
```

### Message Status Lifecycle
```
sending (uploading)
  ↓
sent (on server)
  ↓
delivered (received by client)
  ↓
read (opened by user)

Failed states:
  ↓
failed (with retry_count, failed_reason)
```

### Typing Indicators Flow
```
1. Client types: {type: 'typing', conversation_id, is_typing: true}
   └─> Insert into messages.typing_indicators (expires_at = NOW() + 10s)

2. Server broadcasts:
   └─> To all other participants in conversation
   └─> {type: 'typing_indicator', user_id, is_typing: true}

3. Auto-cleanup:
   └─> Cron job deletes expired records
   └─> OR client sends is_typing: false
```

### Read Receipt Flow
```
1. Client reads message: {type: 'read_receipt', message_id, read_at}
   └─> Update delivery_status (message_id, user_id, status='read')

2. Server notifies sender:
   └─> {type: 'message_read', message_id, reader_id, read_at}

3. Sender's app updates UI:
   └─> Message changes from "delivered" to "read" status
```

---

## 5. DEPENDENCIES & INFRASTRUCTURE

### Message Queue System
**Technology:** Apache Kafka  
**Configuration:** `/shared/pkg/messaging/kafka/`

#### Implementation
- **Producer:** `kafka.NewProducer()` - Sync producer with WaitForAll acks
- **Consumer:** `kafka.NewConsumer()` - Group-based consumer
- **Serializer:** Custom JSON serialization for messages
- **Partitioner:** Hash-based partitioning (by conversation_id)

#### Kafka Topics
```
message.events              - Core message events
message.delivered           - Delivery confirmations
message.read                - Read receipts
notification.events         - Push notification queue
user.events                 - User activity events
media.events                - Media processing events
analytics.events            - Analytics tracking
```

#### Configuration
```go
Config{
  Brokers:           []string{"localhost:9092"}
  ClientID:          "message-service"
  GroupID:           "message-service-group"
  MaxRetries:        3
  RetryBackoff:      100ms
  SessionTimeout:    10s
  HeartbeatInterval: 3s
}
```

### WebSocket Infrastructure
**Library:** gorilla/websocket  
**Server:** Embedded in Message Service (port 9000)

#### Hub Architecture
```go
type Hub struct {
  clients map[string][]*Client      // user_id → [clients]
  register chan *Client             // Register new connection
  unregister chan *Client           // Unregister connection
  broadcast chan *BroadcastMessage  // Send to user
}

type Client struct {
  UserID string
  DeviceID string
  conn *websocket.Conn
  send chan []byte
  hub *Hub
}
```

#### Connection Management
- **Max message size:** 512 KB
- **Ping/Pong:** 60s pong timeout, 30s ping interval
- **Buffer size:** 256 messages per client
- **Write timeout:** 10 seconds
- **Multiple devices:** One client per device (identified by X-Device-ID header)

### Database Infrastructure
**Primary:** PostgreSQL 15  
**Location:** Port 5432  
**Connection Pooling:** 100 max connections, 10 min idle

#### Extensions Required
```sql
CREATE EXTENSION "uuid-ossp";        -- UUID generation
CREATE EXTENSION "pgcrypto";         -- Encryption functions
CREATE EXTENSION "pg_trgm";          -- Trigram text search
CREATE EXTENSION "pg_stat_statements"; -- Query statistics
```

### Caching Infrastructure
**Technology:** Redis 7  
**Location:** Port 6379  
**Configuration:** `/shared/pkg/cache/redis/`

#### Cache Strategy
- **User profiles:** 5 min TTL
- **Conversation metadata:** 1 min TTL
- **Active sessions:** Session duration TTL
- **Media URLs:** 1 hour TTL
- **Typing indicators:** 10 sec TTL

#### Usage in Messaging
- Cache conversation participants list (reduce DB queries)
- Cache user online status
- Cache recently fetched messages
- Rate limiting counters

### Monitoring & Observability

**Prometheus Metrics:**
- Message send rate
- Delivery latency (p50, p95, p99)
- WebSocket connection count
- Kafka producer lag
- Database query latency

**Tracing:**
- Jaeger distributed tracing
- Trace message lifecycle
- Identify bottlenecks

**Logging:**
- Structured logging with zap
- Log levels: debug, info, warn, error, fatal
- Request/response logging with correlation IDs

---

## 6. SERVICE INTERCONNECTIONS

### Message Service → Auth Service
- Validate JWT tokens
- Verify user identity
- Get user permissions

**Calls:**
```
gRPC: localhost:50051 (Auth Service)
  - ValidateToken(token) → User ID
  - CheckPermission(user_id, resource, action) → boolean
```

### Message Service → User Service
- Get user profiles
- Check contact relationships
- Get conversation member list

**Calls:**
```
gRPC: localhost:50052 (User Service)
  - GetUserProfile(user_id) → Profile
  - GetContacts(user_id) → []Contact
  - AreContacts(user_a, user_b) → boolean
```

### Message Service → Notification Service
- Send push notifications (offline users)
- Send message notifications

**Calls:**
```
Kafka: message-service publishes to "notification.events"
  - {user_id, message_id, conversation_id, sender_id, content, timestamp}

gRPC: localhost:50054 (Notification Service)
  - SendPushNotification(user_id, title, body, data)
```

### Message Service → Media Service
- Process image uploads (thumbnails)
- Store media in R2/S3

**Calls:**
```
gRPC: localhost:50057 (Media Service)
  - UploadFile(file_bytes, metadata) → {media_id, cdn_url}
  - GetFile(media_id) → file_bytes
  - DeleteFile(media_id)
```

### Message Service → Presence Service
- Check user online status
- Subscribe to presence changes

**Calls:**
```
gRPC: localhost:50055 (Presence Service)
  - IsUserOnline(user_id, conversation_id) → boolean
  - SubscribeToPresence(conversation_id) → stream PresenceEvent
```

### Message Service → Analytics Service
- Track message events
- Track engagement metrics

**Calls:**
```
Kafka: message-service publishes to "analytics.events"
  - {event_type, user_id, conversation_id, timestamp, properties}

gRPC: localhost:50056 (Analytics Service)
  - LogEvent(event) → success
```

### API Gateway → Message Service
- Routes `/api/v1/messages/*` to Message Service
- Handles HTTP → gRPC conversion

---

## 7. CONFIGURATION & ENVIRONMENT

### Environment Variables (`.env` file)

**Messaging-specific:**
```bash
# Service Ports
MESSAGE_SERVICE_HTTP_PORT=8083
MESSAGE_SERVICE_GRPC_PORT=50053
MESSAGE_SERVICE_WS_PORT=9000

# Kafka Configuration
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=message-service
KAFKA_TOPIC_MESSAGE_EVENTS=message.events

# WebSocket Configuration
WS_READ_BUFFER_SIZE=1024
WS_WRITE_BUFFER_SIZE=1024
WS_MAX_MESSAGE_SIZE=65536
WS_PING_INTERVAL=30s
WS_PONG_TIMEOUT=10s

# Feature Flags
FEATURE_READ_RECEIPTS=true
FEATURE_TYPING_INDICATORS=true
FEATURE_MESSAGE_REACTIONS=true
FEATURE_END_TO_END_ENCRYPTION=true
FEATURE_DISAPPEARING_MESSAGES=false
FEATURE_VOICE_MESSAGES=true
FEATURE_VIDEO_CALLS=false

# Database
POSTGRES_URL=postgres://echo:echo_password@localhost:5432/echo_db

# Redis
REDIS_URL=redis://localhost:6379/0

# Cache TTL
CACHE_CONVERSATION_TTL=1m
CACHE_TYPING_INDICATOR_TTL=10s
```

### Configuration Files

**API Gateway routing config:**
```yaml
services:
  message:
    protocol: grpc
    addresses:
      - localhost:50053
    timeout: 30s
    retry_attempts: 3
```

---

## 8. CURRENT IMPLEMENTATION STATUS

### What Exists
✅ **Database Schema** - Fully designed and comprehensive  
✅ **Kafka Infrastructure** - Producer/consumer implementation  
✅ **API Gateway** - Routing, load balancing, middleware  
✅ **Shared Libraries** - Logging, caching, database, monitoring  
✅ **Docker Compose** - Full dev environment setup  
✅ **Documentation** - MESSAGE_FLOW_GUIDE.md with implementation examples  

### What's Missing (Message Service)
🚧 **Service Implementation**
- [ ] Repository layer (Postgres queries)
- [ ] Service layer (business logic)
- [ ] HTTP handlers (REST endpoints)
- [ ] gRPC server (gRPC endpoints)
- [ ] WebSocket manager (connection handling)
- [ ] Middleware (authentication, logging)

🚧 **Features**
- [ ] Message sending & persistence
- [ ] Conversation management
- [ ] WebSocket real-time delivery
- [ ] Typing indicators
- [ ] Read receipts
- [ ] Reactions
- [ ] Message editing/deletion
- [ ] Message search

🚧 **Infrastructure**
- [ ] Kafka consumer (listen for events)
- [ ] Database migrations
- [ ] Health checks
- [ ] Metrics collection
- [ ] Error handling

---

## 9. ARCHITECTURE PATTERNS & DESIGN

### Design Patterns Used

**1. Hexagonal Architecture (Ports & Adapters)**
- Domain models (core business logic)
- Ports (interfaces)
- Adapters (implementations - DB, HTTP, gRPC)

**2. Microservices Pattern**
- Independent services
- gRPC for service-to-service communication
- Kafka for async events

**3. Repository Pattern**
- Data access layer abstraction
- Query builders
- Connection pooling

**4. Service Pattern**
- Business logic orchestration
- Transaction management
- Event publishing

**5. Middleware Pattern**
- Authentication/authorization
- Logging
- Error handling
- Rate limiting

### Code Organization (Per Auth Service Model)

```
message-service/
├── internal/
│   ├── config/          # Configuration loading & validation
│   ├── domain/          # Domain models (Message, Conversation, etc.)
│   ├── service/         # Business logic (MessageService, ConversationService)
│   ├── repo/            # Data access (MessageRepo, ConversationRepo)
│   ├── handler/         # HTTP/gRPC handlers
│   ├── websocket/       # WebSocket hub & clients
│   ├── model/           # API request/response models
│   ├── errors/          # Error definitions
│   ├── middleware/      # HTTP middleware
│   └── infra/           # Infrastructure setup (DB, Kafka, etc.)
├── proto/               # gRPC proto files
├── cmd/server/main.go   # Entry point
└── tests/               # Unit & integration tests
```

---

## 10. NEXT STEPS FOR PRODUCTION-READY IMPLEMENTATION

### Phase 1: Core Infrastructure (Week 1)
1. [ ] Set up message service main.go with dependency injection
2. [ ] Create domain models (Message, Conversation, Participant)
3. [ ] Create repository interfaces
4. [ ] Implement Postgres repositories with connection pooling
5. [ ] Create database migrations

### Phase 2: Core Services (Week 2-3)
1. [ ] Implement MessageService (send, get, list, search)
2. [ ] Implement ConversationService (create, update, delete)
3. [ ] Implement ParticipantService (add, remove, update roles)
4. [ ] Add caching layer (Redis integration)
5. [ ] Implement error handling & logging

### Phase 3: Real-time Delivery (Week 3-4)
1. [ ] Implement WebSocket hub & connection manager
2. [ ] Create WebSocket handlers
3. [ ] Implement typing indicators
4. [ ] Implement read receipts
5. [ ] Add presence integration

### Phase 4: API & HTTP (Week 4)
1. [ ] Create REST API handlers
2. [ ] Implement request validation
3. [ ] Add authentication middleware
4. [ ] Create gRPC service definitions
5. [ ] Implement gRPC handlers

### Phase 5: Kafka Integration (Week 5)
1. [ ] Implement Kafka consumer
2. [ ] Handle message events
3. [ ] Connect with notification service
4. [ ] Add retry logic & dead letter queues
5. [ ] Implement message deduplication

### Phase 6: Testing & Monitoring (Week 6)
1. [ ] Write unit tests (>80% coverage)
2. [ ] Write integration tests
3. [ ] Add Prometheus metrics
4. [ ] Setup Jaeger tracing
5. [ ] Create load tests

### Phase 7: Deployment & Optimization (Week 7-8)
1. [ ] Docker containerization
2. [ ] Kubernetes manifests
3. [ ] Performance optimization
4. [ ] Security hardening
5. [ ] Production deployment

---

## 11. KEY REFERENCES & FILE LOCATIONS

### Documentation
- **Message Flow Guide:** `/docs/MESSAGE_FLOW_GUIDE.md`
- **Complete User Flow:** `/docs/COMPLETE_USER_FLOW_GUIDE.md`
- **Database Documentation:** `/database/DOC.md`
- **Quickstart:** `/QUICKSTART.md`

### Database Schemas
- **Message Schema:** `/database/schemas/message-schema.sql`
- **All Schemas:** `/database/schemas/*.sql`
- **Migrations:** `/migrations/postgres/`

### Shared Libraries
- **Messaging (Kafka):** `/shared/pkg/messaging/`
- **Cache (Redis):** `/shared/pkg/cache/redis/`
- **Database:** `/shared/pkg/database/`
- **Logger:** `/shared/pkg/logger/`
- **Monitoring:** `/shared/pkg/monitoring/`

### Services
- **Auth Service:** `/services/auth-service/` (reference implementation)
- **API Gateway:** `/services/api-gateway/`
- **Message Service:** `/services/message-service/` (to implement)

### Infrastructure
- **Docker Compose:** `/infra/docker/docker-compose.dev.yml`
- **Makefile:** `/Makefile`
- **Environment:** `.env.example`

---

## 12. PRODUCTION CONSIDERATIONS

### Scalability
- **Message Queue:** Kafka with multiple partitions (by conversation_id)
- **Database:** Connection pooling (100 max), prepared statements
- **Caching:** Redis cluster for high availability
- **WebSocket:** Load balancer with sticky sessions OR Redis pub/sub for cross-instance messaging

### Reliability
- **Message Delivery:** At-least-once with idempotency (content_hash)
- **Error Handling:** Retry logic with exponential backoff
- **Health Checks:** Readiness & liveness probes
- **Graceful Shutdown:** Drain connections, complete in-flight messages

### Security
- **Authentication:** JWT tokens with expiration
- **Authorization:** Row-Level Security (RLS) in database
- **Encryption:** TLS for transport, AES-256 for E2E messages
- **Rate Limiting:** Redis-backed rate limiter
- **Input Validation:** Schema validation on all inputs

### Monitoring
- **Metrics:** Message latency, throughput, error rates
- **Logs:** Structured logging with correlation IDs
- **Tracing:** Distributed tracing for debugging
- **Alerts:** Alert on SLO violations (delivery latency >5s)

---

## Summary

The Echo Backend project provides a **solid foundation** for building a production-ready messaging system. The database schema is comprehensive, infrastructure is well-architected, and shared libraries are mature. The main work ahead is implementing the message service's handler, service, and repository layers, along with WebSocket infrastructure for real-time delivery.

The system is designed to handle:
- **100M+ messages** with proper indexing and partitioning
- **1M+ daily active users** with Kafka scaling
- **Sub-second message delivery** with WebSocket + Redis
- **Multiple message types** (text, media, calls, reactions, polls)
- **Advanced features** (encryption, typing indicators, read receipts, message search)

**Recommended approach:** Follow the clean architecture pattern established by the auth-service, implement in phases with 80%+ test coverage, and load test each critical path.

