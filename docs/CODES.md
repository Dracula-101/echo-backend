# Echo Backend — Error Codes Reference

Complete list of all error codes across all services. Use the **Code** string values to match against API responses on the client side.

---

## Table of Contents

1. [Shared / Global Codes](#1-shared--global-codes)
2. [Auth Service](#2-auth-service)
3. [User Service](#3-user-service)
4. [Message Service](#4-message-service)
5. [Media Service](#5-media-service)
6. [Location Service](#6-location-service)
7. [API Gateway](#7-api-gateway)
8. [WebSocket Service](#8-websocket-service)
9. [Event Types](#9-event-types)

---

## 1. Shared / Global Codes

Source: `shared/pkg/errors/codes.go`

These are the base error codes used as fallbacks across all services.

### 2xx: Success

| Code | HTTP Status |
|------|-------------|
| `OK` | 200 |
| `CREATED` | 201 |
| `ACCEPTED` | 202 |
| `NO_CONTENT` | 204 |

### 3xx: Redirection

| Code | HTTP Status |
|------|-------------|
| `MOVED_PERMANENTLY` | 301 |
| `FOUND` | 302 |
| `SEE_OTHER` | 303 |
| `NOT_MODIFIED` | 304 |
| `TEMPORARY_REDIRECT` | 307 |
| `PERMANENT_REDIRECT` | 308 |

### 4xx: Client Errors

| Code | HTTP Status |
|------|-------------|
| `BAD_REQUEST` | 400 |
| `UNAUTHORIZED` | 401 |
| `INVALID_ARGUMENT` | 400 |
| `ALREADY_EXISTS` | 409 |
| `PAYMENT_REQUIRED` | 402 |
| `FORBIDDEN` | 403 |
| `NOT_FOUND` | 404 |
| `METHOD_NOT_ALLOWED` | 405 |
| `NOT_ACCEPTABLE` | 406 |
| `ACCESS_DENIED` | — |
| `PROXY_AUTHENTICATION_REQUIRED` | 407 |
| `TIMEOUT` | — |
| `REQUEST_TIMEOUT` | 408 |
| `CONFLICT` | 409 |
| `GONE` | 410 |
| `LENGTH_REQUIRED` | 411 |
| `PRECONDITION_FAILED` | 412 |
| `REQUEST_ENTITY_TOO_LARGE` | 413 |
| `REQUEST_URI_TOO_LONG` | 414 |
| `UNSUPPORTED_MEDIA_TYPE` | 415 |
| `RANGE_NOT_SATISFIABLE` | 416 |
| `EXPECTATION_FAILED` | 417 |
| `TOO_MANY_REQUESTS` | 429 |
| `UNPROCESSABLE_ENTITY` | 422 |
| `RATE_LIMIT_EXCEEDED` | 429 |
| `VALIDATION_FAILED` | 422 |

### 4xx: Auth-Related

| Code | HTTP Status |
|------|-------------|
| `UNAUTHENTICATED` | 401 |
| `TOKEN_EXPIRED` | 401 |
| `TOKEN_INVALID` | 401 |
| `PERMISSION_DENIED` | 403 |
| `SESSION_EXPIRED` | — |
| `CSRF_TOKEN_INVALID` | — |

### 5xx: Server Errors

| Code | HTTP Status |
|------|-------------|
| `INTERNAL_ERROR` | 500 |
| `NOT_IMPLEMENTED` | 501 |
| `BAD_GATEWAY` | 502 |
| `SERVICE_UNAVAILABLE` | 503 |
| `GATEWAY_TIMEOUT` | 504 |
| `DATA_LOSS` | 500 |
| `DATABASE_ERROR` | 500 |
| `CACHE_ERROR` | 500 |
| `QUEUE_ERROR` | 500 |
| `DEADLINE_EXCEEDED` | 504 |
| `UNAVAILABLE` | 503 |
| `ABORTED` | 409 |
| `UNIMPLEMENTED` | 501 |
| `OUT_OF_RANGE` | 400 |
| `RESOURCE_EXHAUSTED` | 429 |
| `FAILED_PRECONDITION` | 400 |
| `INTERNAL_DEPENDENCY_FAILURE` | 503 |

### Cancellations

| Code | HTTP Status |
|------|-------------|
| `CANCELLED` | 499 |
| `DEADLOCK_DETECTED` | — |

---

## 2. Auth Service

Source: `services/auth-service/internal/error/codes.go`

### Authentication Errors

| Code |
|------|
| `AUTH_INVALID_CREDENTIALS` |
| `AUTH_USER_NOT_FOUND` |
| `AUTH_EMAIL_EXISTS` |
| `AUTH_PASSWORD_HASH_FAILED` |
| `AUTH_TOKEN_GEN_FAILED` |
| `AUTH_INVALID_TOKEN` |
| `AUTH_TOKEN_VALIDATE_FAILED` |
| `AUTH_SESSION_NOT_FOUND` |
| `AUTH_SESSION_EXPIRED` |
| `AUTH_ACCOUNT_LOCKED` |
| `AUTH_ACCOUNT_DISABLED` |
| `AUTH_PASSWORD_EXPIRED` |
| `AUTH_2FA_REQUIRED` |
| `AUTH_INVALID_2FA_CODE` |
| `AUTH_EMAIL_VERIFY_FAILED` |
| `AUTH_PHONE_VERIFY_FAILED` |

### Database Errors

| Code |
|------|
| `AUTH_DATABASE_ERROR` |

### Registration Errors

| Code |
|------|
| `AUTH_INVALID_EMAIL` |
| `AUTH_INVALID_PHONE` |
| `AUTH_PASSWORD_WEAK` |
| `AUTH_TERMS_NOT_ACCEPTED` |

### Session Errors

| Code |
|------|
| `AUTH_SESSION_CREATE_FAILED` |
| `AUTH_SESSION_UPDATE_FAILED` |
| `AUTH_INVALID_REFRESH_TOKEN` |
| `AUTH_REFRESH_TOKEN_EXPIRED` |

### Security Errors

| Code |
|------|
| `AUTH_TOO_MANY_FAILED_ATTEMPTS` |
| `AUTH_SUSPICIOUS_ACTIVITY` |
| `AUTH_IP_BLOCKED` |
| `AUTH_DEVICE_NOT_TRUSTED` |

---

## 3. User Service

Source: `services/user-service/internal/errors/codes.go`

### User Errors

| Code |
|------|
| `USER_NOT_FOUND` |
| `USER_ALREADY_EXISTS` |
| `INVALID_USER_ID` |
| `INVALID_USER_DATA` |
| `USERNAME_UNAVAILABLE` |

### Profile Errors

| Code |
|------|
| `PROFILE_NOT_FOUND` |
| `INVALID_PROFILE_DATA` |
| `PROFILE_UPDATE_FAILED` |

### Search Errors

| Code |
|------|
| `SEARCH_FAILED` |
| `INVALID_SEARCH_QUERY` |

### Database Errors

| Code |
|------|
| `DATABASE_ERROR` |
| `DATABASE_CONNECTION_ERROR` |

### Cache Errors

| Code |
|------|
| `CACHE_ERROR` |

### General Errors

| Code |
|------|
| `INTERNAL_ERROR` |
| `INVALID_REQUEST` |
| `UNAUTHORIZED` |
| `FORBIDDEN` |
| `VALIDATION_ERROR` |

---

## 4. Message Service

Source: `services/message-service/internal/error/codes.go`

### User Errors

| Code |
|------|
| `USER_BLOCKED` |
| `BLOCK_USER_NOT_FOUND` |
| `RATE_LIMIT_EXCEEDED` |

### Conversation Errors

| Code |
|------|
| `CONVERSATION_NOT_FOUND` |
| `CONVERSATION_VALIDATION_FAILED` |
| `CONVERSATION_INACTIVE` |
| `CONVERSATION_CREATE_FAILED` |
| `CONVERSATION_CREATION_FAILED` |
| `CONVERSATION_FETCH_FAILED` |

### Participant Errors

| Code |
|------|
| `PARTICIPANT_NOT_IN_CONVERSATION` |
| `PARTICIPANT_LEFT_CONVERSATION` |
| `PARTICIPANT_REMOVED_FROM_CONVERSATION` |
| `PARTICIPANT_NOT_ALLOWED_TO_SEND_MESSAGES` |

### Message Errors

| Code |
|------|
| `EMPTY_MESSAGE_CONTENT` |
| `MISSING_MESSAGE_METADATA` |
| `MESSAGE_EVENT_PUBLISH_FAILED` |
| `MESSAGE_RETRIEVAL_FAILED` |

---

## 5. Media Service

Source: `services/media-service/internal/errors/codes.go`

### Upload Errors

| Code |
|------|
| `MEDIA_UPLOAD_FAILED` |
| `MEDIA_FILE_TOO_LARGE` |
| `MEDIA_INVALID_FILE_TYPE` |
| `MEDIA_INVALID_FILE_NAME` |
| `MEDIA_UPLOAD_TIMEOUT` |
| `MEDIA_STORAGE_QUOTA_EXCEEDED` |
| `MEDIA_CONVERSATION_NOT_FOUND` |

### Download Errors

| Code |
|------|
| `MEDIA_DOWNLOAD_FAILED` |
| `MEDIA_FILE_NOT_FOUND` |
| `MEDIA_FILE_EXPIRED` |
| `MEDIA_ACCESS_DENIED` |
| `MEDIA_INVALID_ACCESS_TOKEN` |

### Processing Errors

| Code |
|------|
| `MEDIA_PROCESSING_FAILED` |
| `MEDIA_THUMBNAIL_GENERATION_FAILED` |
| `MEDIA_TRANSCODING_FAILED` |
| `MEDIA_COMPRESSION_FAILED` |

### Security Errors

| Code |
|------|
| `MEDIA_VIRUS_SCAN_FAILED` |
| `MEDIA_VIRUS_DETECTED` |
| `MEDIA_MODERATION_FAILED` |
| `MEDIA_CONTENT_REJECTED` |
| `MEDIA_NSFW_DETECTED` |

### Album Errors

| Code |
|------|
| `MEDIA_ALBUM_NOT_FOUND` |
| `MEDIA_ALBUM_CREATION_FAILED` |
| `MEDIA_ALBUM_LIMIT_EXCEEDED` |
| `MEDIA_FILE_ALREADY_IN_ALBUM` |

### Sticker Errors

| Code |
|------|
| `MEDIA_STICKER_PACK_NOT_FOUND` |
| `MEDIA_STICKER_NOT_FOUND` |
| `MEDIA_STICKER_LIMIT_EXCEEDED` |

### Share Errors

| Code |
|------|
| `MEDIA_SHARE_CREATION_FAILED` |
| `MEDIA_SHARE_NOT_FOUND` |
| `MEDIA_SHARE_EXPIRED` |
| `MEDIA_SHARE_LIMIT_EXCEEDED` |

### Validation Errors

| Code |
|------|
| `MEDIA_INVALID_METADATA` |
| `MEDIA_INVALID_DIMENSIONS` |
| `MEDIA_INVALID_DURATION` |
| `MEDIA_INVALID_FORMAT` |
| `MEDIA_CORRUPTED_FILE` |

### General Errors

| Code |
|------|
| `MEDIA_INTERNAL_ERROR` |
| `MEDIA_SERVICE_UNAVAILABLE` |

---

## 6. Location Service

Source: `services/location-service/errors/codes.go`

### Lookup Errors

| Code | HTTP Status |
|------|-------------|
| `LOC_LOOKUP_FAILED` | 500 |
| `LOC_INVALID_IP` | 400 |
| `LOC_IP_NOT_FOUND` | 404 |
| `LOC_PRIVATE_IP` | 400 |

### Database Errors

| Code | HTTP Status |
|------|-------------|
| `LOC_DATABASE_NOT_FOUND` | 503 |
| `LOC_DATABASE_LOAD_FAILED` | 500 |
| `LOC_DATABASE_CORRUPTED` | 500 |
| `LOC_DATABASE_OUTDATED` | 503 |

### Service Errors

| Code | HTTP Status |
|------|-------------|
| `LOC_SERVICE_UNAVAILABLE` | 503 |
| `LOC_RATE_LIMIT_EXCEEDED` | 429 |

### Data Quality Errors

| Code | HTTP Status |
|------|-------------|
| `LOC_INCOMPLETE_DATA` | 200 (returned with warning) |
| `LOC_LOW_ACCURACY` | 200 (returned with warning) |

---

## 7. API Gateway

Source: `services/api-gateway/internal/errors/codes.go`

### Service Discovery Errors

| Code | HTTP Status |
|------|-------------|
| `GW_SERVICE_NOT_FOUND` | 503 |
| `GW_SERVICE_UNAVAILABLE` | 503 |
| `GW_NO_HEALTHY_INSTANCES` | 503 |

### Routing Errors

| Code | HTTP Status |
|------|-------------|
| `GW_ROUTING_FAILED` | 502 |
| `GW_INVALID_ROUTE` | 400 |
| `GW_ROUTE_NOT_FOUND` | 404 |

### Proxy Errors

| Code | HTTP Status |
|------|-------------|
| `GW_PROXY_ERROR` | 502 |
| `GW_UPSTREAM_TIMEOUT` | 504 |
| `GW_UPSTREAM_ERROR` | 502 |
| `GW_CONNECTION_FAILED` | 503 |

### Rate Limiting Errors

| Code | HTTP Status |
|------|-------------|
| `GW_RATE_LIMIT_EXCEEDED` | 429 |
| `GW_QUOTA_EXCEEDED` | 429 |

### Authentication Errors

| Code | HTTP Status |
|------|-------------|
| `GW_MISSING_AUTH_HEADER` | 401 |
| `GW_INVALID_AUTH_TOKEN` | 401 |
| `GW_AUTH_FAILED` | 401 |

### Request Errors

| Code | HTTP Status |
|------|-------------|
| `GW_INVALID_REQUEST` | 400 |
| `GW_REQUEST_TOO_LARGE` | 413 |
| `GW_INVALID_CONTENT_TYPE` | 415 |

### Configuration Errors

| Code | HTTP Status |
|------|-------------|
| `GW_CONFIG_ERROR` | 500 |
| `GW_INVALID_CONFIG` | 500 |

---

## 8. WebSocket Service

Source: `services/ws-service/internal/domain/codes.go`

| Code |
|------|
| `INVALID_REQUEST` |
| `USER_NOT_FOUND` |
| `USER_NOT_ONLINE` |
| `INVALID_EVENT_TYPE` |
| `BROADCAST_FAILED` |
| `CONNECTION_FAILED` |
| `INVALID_USER_ID` |
| `INVALID_DEVICE_ID` |
| `DATABASE_ERROR` |
| `CACHE_ERROR` |

---

## 9. Event Types

Source: `services/message-service/internal/domain/codes.go`

These are not error codes but event type strings used in WebSocket/real-time communication.

### Message Events

| Event Type |
|------------|
| `message.sent` |
| `message.edited` |
| `message.deleted` |
| `message.delivered` |
| `message.read` |
| `message.failed` |

### Conversation Events

| Event Type |
|------------|
| `conversation.created` |
| `conversation.updated` |
| `conversation.deleted` |
| `conversation.participant_added` |
| `conversation.participant_left` |
| `conversation.typing_started` |
| `conversation.typing_stopped` |
