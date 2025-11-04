# Message Flow Implementation Guide

Complete guide for implementing message sending from phone clients and broadcasting to recipients in Echo Backend.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Message Flow](#message-flow)
3. [Backend Implementation](#backend-implementation)
4. [Real-time Delivery Options](#real-time-delivery-options)
5. [Client Implementation](#client-implementation)
6. [Delivery Tracking](#delivery-tracking)
7. [Testing](#testing)

---

## Architecture Overview

### Message Journey

```
┌─────────────┐
│   Phone     │
│   Client    │
└──────┬──────┘
       │ 1. POST /api/v1/messages
       │    {conversation_id, content, type}
       ▼
┌─────────────────────┐
│   API Gateway       │
│   (Port 8080)       │
│   • Auth validation │
│   • Rate limiting   │
└──────┬──────────────┘a
       │ 2. Forward to Message Service
       ▼
┌─────────────────────┐
│  Message Service    │
│  • Save to DB       │
│  • Get participants │
│  • Publish to Queue │
└──────┬──────────────┘
       │ 3. Broadcast
       ├──────────────┐
       ▼              ▼
┌──────────────┐  ┌──────────────┐
│  WebSocket   │  │   Push       │
│  Server      │  │   Service    │
│  (online)    │  │  (offline)   │
└──────┬───────┘  └──────┬───────┘
       │                 │
       ▼                 ▼
┌─────────────┐   ┌─────────────┐
│ Recipient 1 │   │ Recipient 2 │
│ (Online)    │   │ (Offline)   │
└─────────────┘   └─────────────┘
```

---

## Message Flow

### Step-by-Step Flow

1. **Client sends message** → API Gateway
2. **API Gateway** validates auth & forwards → Message Service
3. **Message Service**:
   - Validates user is participant
   - Saves message to database
   - Gets all conversation participants
   - Publishes to message queue/broadcast
4. **Delivery**:
   - **Online users** → Receive via WebSocket
   - **Offline users** → Receive push notification
5. **Acknowledgment**:
   - Clients send read receipts
   - Update delivery status

---

## Backend Implementation

### 1. Message Service Structure

Create the message service (currently a stub):

```bash
# Service structure
services/message-service/
├── cmd/server/
│   └── main.go
├── internal/
│   ├── handler/
│   │   ├── grpc_handler.go      # gRPC endpoints
│   │   └── websocket_handler.go  # WebSocket connections
│   ├── service/
│   │   ├── message_service.go
│   │   └── broadcast_service.go
│   ├── repo/
│   │   ├── message_repo.go
│   │   └── conversation_repo.go
│   ├── websocket/
│   │   ├── manager.go            # Connection manager
│   │   ├── client.go             # Client connection
│   │   └── hub.go                # Broadcast hub
│   └── config/
│       └── config.go
├── proto/
│   └── message.proto             # gRPC definitions
├── Dockerfile
└── go.mod
```

### 2. Message Service - Core Implementation

**`internal/service/message_service.go`**

```go
package service

import (
    "context"
    "database/sql"
    "encoding/json"
    "time"

    "github.com/google/uuid"
    "echo-backend/shared/pkg/logger"
    "echo-backend/shared/pkg/messaging" // Kafka client
)

type MessageService struct {
    db           *sql.DB
    kafka        *messaging.KafkaProducer
    logger       *logger.Logger
    hub          *websocket.Hub
}

type SendMessageRequest struct {
    ConversationID string `json:"conversation_id" validate:"required,uuid"`
    SenderUserID   string `json:"sender_user_id" validate:"required,uuid"`
    Content        string `json:"content" validate:"required,max=10000"`
    MessageType    string `json:"message_type" validate:"required,oneof=text image video audio"`
    Mentions       []Mention `json:"mentions,omitempty"`
    ReplyToID      *string `json:"reply_to_id,omitempty" validate:"omitempty,uuid"`
}

type Message struct {
    ID             string    `json:"id"`
    ConversationID string    `json:"conversation_id"`
    SenderUserID   string    `json:"sender_user_id"`
    Content        string    `json:"content"`
    MessageType    string    `json:"message_type"`
    Status         string    `json:"status"`
    CreatedAt      time.Time `json:"created_at"`
    Mentions       []Mention `json:"mentions,omitempty"`
}

type Mention struct {
    UserID string `json:"user_id"`
    Offset int    `json:"offset"`
    Length int    `json:"length"`
}

// SendMessage - Main function to send a message
func (s *MessageService) SendMessage(ctx context.Context, req *SendMessageRequest) (*Message, error) {
    // 1. Validate sender is a participant
    if err := s.validateParticipant(ctx, req.ConversationID, req.SenderUserID); err != nil {
        return nil, err
    }

    // 2. Create message in database
    message, err := s.createMessage(ctx, req)
    if err != nil {
        s.logger.Error("Failed to create message", "error", err)
        return nil, err
    }

    // 3. Get all participants (for broadcasting)
    participants, err := s.getConversationParticipants(ctx, req.ConversationID)
    if err != nil {
        s.logger.Error("Failed to get participants", "error", err)
        return nil, err
    }

    // 4. Update conversation metadata
    if err := s.updateConversationLastMessage(ctx, req.ConversationID, message.ID); err != nil {
        s.logger.Warn("Failed to update conversation", "error", err)
    }

    // 5. Broadcast to participants
    go s.broadcastMessage(message, participants)

    return message, nil
}

// createMessage - Save message to database
func (s *MessageService) createMessage(ctx context.Context, req *SendMessageRequest) (*Message, error) {
    message := &Message{
        ID:             uuid.New().String(),
        ConversationID: req.ConversationID,
        SenderUserID:   req.SenderUserID,
        Content:        req.Content,
        MessageType:    req.MessageType,
        Status:         "sent",
        CreatedAt:      time.Now(),
        Mentions:       req.Mentions,
    }

    mentionsJSON, _ := json.Marshal(req.Mentions)

    query := `
        INSERT INTO messages.messages (
            id, conversation_id, sender_user_id, content,
            message_type, status, mentions, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id, created_at
    `

    err := s.db.QueryRowContext(ctx, query,
        message.ID, message.ConversationID, message.SenderUserID,
        message.Content, message.MessageType, message.Status,
        mentionsJSON, message.CreatedAt,
    ).Scan(&message.ID, &message.CreatedAt)

    if err != nil {
        return nil, err
    }

    return message, nil
}

// validateParticipant - Check if sender is in conversation
func (s *MessageService) validateParticipant(ctx context.Context, conversationID, userID string) error {
    query := `
        SELECT 1 FROM messages.conversation_participants
        WHERE conversation_id = $1 AND user_id = $2 AND can_send_messages = true
    `

    var exists int
    err := s.db.QueryRowContext(ctx, query, conversationID, userID).Scan(&exists)
    if err == sql.ErrNoRows {
        return fmt.Errorf("user not authorized to send messages in this conversation")
    }
    return err
}

// getConversationParticipants - Get all participants
func (s *MessageService) getConversationParticipants(ctx context.Context, conversationID string) ([]string, error) {
    query := `
        SELECT user_id FROM messages.conversation_participants
        WHERE conversation_id = $1 AND left_at IS NULL
    `

    rows, err := s.db.QueryContext(ctx, query, conversationID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var participants []string
    for rows.Next() {
        var userID string
        if err := rows.Scan(&userID); err != nil {
            return nil, err
        }
        participants = append(participants, userID)
    }

    return participants, nil
}

// updateConversationLastMessage - Update conversation metadata
func (s *MessageService) updateConversationLastMessage(ctx context.Context, conversationID, messageID string) error {
    query := `
        UPDATE messages.conversations
        SET
            last_message_id = $1,
            last_message_at = NOW(),
            last_activity_at = NOW(),
            message_count = message_count + 1
        WHERE id = $2
    `

    _, err := s.db.ExecContext(ctx, query, messageID, conversationID)
    return err
}

// broadcastMessage - Send to all participants
func (s *MessageService) broadcastMessage(message *Message, participants []string) {
    // Prepare broadcast event
    event := BroadcastEvent{
        Type:    "new_message",
        Message: message,
    }

    eventJSON, _ := json.Marshal(event)

    for _, participantID := range participants {
        // Skip sender (they already have the message)
        if participantID == message.SenderUserID {
            continue
        }

        // Option 1: Send via WebSocket (if online)
        if s.hub.IsUserOnline(participantID) {
            s.hub.SendToUser(participantID, eventJSON)

            // Track delivery
            s.trackDelivery(message.ID, participantID, "delivered")
        } else {
            // Option 2: Send to Kafka for push notification
            s.sendPushNotification(message, participantID)

            // Track as pending
            s.trackDelivery(message.ID, participantID, "sent")
        }
    }
}

// trackDelivery - Track message delivery status
func (s *MessageService) trackDelivery(messageID, userID, status string) {
    query := `
        INSERT INTO messages.delivery_status (message_id, user_id, status)
        VALUES ($1, $2, $3)
        ON CONFLICT (message_id, user_id) DO UPDATE SET
            status = EXCLUDED.status,
            delivered_at = CASE WHEN EXCLUDED.status = 'delivered' THEN NOW() ELSE delivery_status.delivered_at END
    `

    _, err := s.db.Exec(query, messageID, userID, status)
    if err != nil {
        s.logger.Error("Failed to track delivery", "error", err)
    }
}

// sendPushNotification - Send to notification service
func (s *MessageService) sendPushNotification(message *Message, recipientID string) {
    notification := map[string]interface{}{
        "user_id":         recipientID,
        "type":            "message",
        "message_id":      message.ID,
        "conversation_id": message.ConversationID,
        "sender_id":       message.SenderUserID,
        "content":         message.Content,
        "timestamp":       message.CreatedAt,
    }

    notifJSON, _ := json.Marshal(notification)

    // Publish to Kafka topic for notification service
    s.kafka.Publish("notifications", string(notifJSON))
}
```

### 3. WebSocket Manager

**`internal/websocket/hub.go`**

```go
package websocket

import (
    "sync"
    "encoding/json"
)

// Hub maintains active WebSocket connections
type Hub struct {
    // user_id -> []*Client (supports multiple devices)
    clients map[string][]*Client

    // Register new client
    register chan *Client

    // Unregister client
    unregister chan *Client

    // Broadcast to specific user
    broadcast chan *BroadcastMessage

    mu sync.RWMutex
}

type BroadcastMessage struct {
    UserID  string
    Payload []byte
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[string][]*Client),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        broadcast:  make(chan *BroadcastMessage),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client.UserID] = append(h.clients[client.UserID], client)
            h.mu.Unlock()

        case client := <-h.unregister:
            h.mu.Lock()
            if clients, ok := h.clients[client.UserID]; ok {
                for i, c := range clients {
                    if c == client {
                        h.clients[client.UserID] = append(clients[:i], clients[i+1:]...)
                        close(client.send)
                        break
                    }
                }
                if len(h.clients[client.UserID]) == 0 {
                    delete(h.clients, client.UserID)
                }
            }
            h.mu.Unlock()

        case message := <-h.broadcast:
            h.mu.RLock()
            clients := h.clients[message.UserID]
            h.mu.RUnlock()

            // Send to all devices of this user
            for _, client := range clients {
                select {
                case client.send <- message.Payload:
                default:
                    // Client buffer full, disconnect
                    h.unregister <- client
                }
            }
        }
    }
}

// SendToUser - Send message to specific user (all devices)
func (h *Hub) SendToUser(userID string, payload []byte) {
    h.broadcast <- &BroadcastMessage{
        UserID:  userID,
        Payload: payload,
    }
}

// IsUserOnline - Check if user has active connection
func (h *Hub) IsUserOnline(userID string) bool {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return len(h.clients[userID]) > 0
}
```

**`internal/websocket/client.go`**

```go
package websocket

import (
    "time"
    "github.com/gorilla/websocket"
)

type Client struct {
    UserID     string
    DeviceID   string
    conn       *websocket.Conn
    send       chan []byte
    hub        *Hub
}

const (
    writeWait      = 10 * time.Second
    pongWait       = 60 * time.Second
    pingPeriod     = (pongWait * 9) / 10
    maxMessageSize = 512 * 1024 // 512 KB
)

// Read from WebSocket
func (c *Client) ReadPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            break
        }

        // Handle incoming messages (read receipts, typing indicators, etc.)
        c.handleIncomingMessage(message)
    }
}

// Write to WebSocket
func (c *Client) WritePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

func (c *Client) handleIncomingMessage(message []byte) {
    // Parse and handle client messages
    // e.g., read receipts, typing indicators
}
```

**`internal/handler/websocket_handler.go`**

```go
package handler

import (
    "net/http"
    "github.com/gorilla/websocket"
    "echo-backend/services/message-service/internal/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // TODO: Implement proper origin check
        return true
    },
}

type WebSocketHandler struct {
    hub *websocket.Hub
}

func NewWebSocketHandler(hub *websocket.Hub) *WebSocketHandler {
    return &WebSocketHandler{hub: hub}
}

// HandleWebSocket - Upgrade HTTP to WebSocket
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Get user ID from JWT token (set by auth middleware)
    userID := r.Context().Value("user_id").(string)
    deviceID := r.Header.Get("X-Device-ID")

    // Upgrade connection
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        http.Error(w, "Could not upgrade connection", http.StatusBadRequest)
        return
    }

    // Create client
    client := &websocket.Client{
        UserID:   userID,
        DeviceID: deviceID,
        conn:     conn,
        send:     make(chan []byte, 256),
        hub:      h.hub,
    }

    // Register client
    h.hub.register <- client

    // Start pumps
    go client.WritePump()
    go client.ReadPump()
}
```

### 4. gRPC Handler (Alternative to REST)

**`proto/message.proto`**

```protobuf
syntax = "proto3";

package message;

option go_package = "echo-backend/services/message-service/proto";

service MessageService {
    rpc SendMessage(SendMessageRequest) returns (SendMessageResponse);
    rpc GetMessages(GetMessagesRequest) returns (GetMessagesResponse);
    rpc StreamMessages(StreamMessagesRequest) returns (stream MessageEvent);
}

message SendMessageRequest {
    string conversation_id = 1;
    string sender_user_id = 2;
    string content = 3;
    string message_type = 4;
    repeated Mention mentions = 5;
    optional string reply_to_id = 6;
}

message SendMessageResponse {
    Message message = 1;
}

message Message {
    string id = 1;
    string conversation_id = 2;
    string sender_user_id = 3;
    string content = 4;
    string message_type = 5;
    string status = 6;
    int64 created_at = 7;
    repeated Mention mentions = 8;
}

message Mention {
    string user_id = 1;
    int32 offset = 2;
    int32 length = 3;
}

message MessageEvent {
    string type = 1;
    Message message = 2;
}

message StreamMessagesRequest {
    string user_id = 1;
}

message GetMessagesRequest {
    string conversation_id = 1;
    int32 limit = 2;
    optional string before_id = 3;
}

message GetMessagesResponse {
    repeated Message messages = 1;
    bool has_more = 2;
}
```

### 5. Main Server Setup

**`cmd/server/main.go`**

```go
package main

import (
    "context"
    "database/sql"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "echo-backend/services/message-service/internal/handler"
    "echo-backend/services/message-service/internal/service"
    "echo-backend/services/message-service/internal/websocket"
    "echo-backend/shared/pkg/database"
    "echo-backend/shared/pkg/logger"
    "echo-backend/shared/pkg/messaging"
    "github.com/gorilla/mux"
)

func main() {
    // Initialize logger
    log := logger.NewLogger("message-service", "info")

    // Connect to database
    db, err := database.Connect(database.Config{
        Host:     os.Getenv("DB_HOST"),
        Port:     os.Getenv("DB_PORT"),
        User:     os.Getenv("DB_USER"),
        Password: os.Getenv("DB_PASSWORD"),
        Database: os.Getenv("DB_NAME"),
    })
    if err != nil {
        log.Fatal("Failed to connect to database", "error", err)
    }
    defer db.Close()

    // Initialize Kafka
    kafka, err := messaging.NewKafkaProducer([]string{os.Getenv("KAFKA_BROKERS")})
    if err != nil {
        log.Fatal("Failed to initialize Kafka", "error", err)
    }
    defer kafka.Close()

    // Initialize WebSocket hub
    hub := websocket.NewHub()
    go hub.Run()

    // Initialize services
    msgService := service.NewMessageService(db, kafka, log, hub)

    // Initialize handlers
    wsHandler := handler.NewWebSocketHandler(hub)
    httpHandler := handler.NewHTTPHandler(msgService)

    // Setup HTTP router
    router := mux.NewRouter()

    // WebSocket endpoint
    router.HandleFunc("/ws", wsHandler.HandleWebSocket)

    // REST endpoints
    router.HandleFunc("/messages", httpHandler.SendMessage).Methods("POST")
    router.HandleFunc("/messages/{conversation_id}", httpHandler.GetMessages).Methods("GET")
    router.HandleFunc("/health", httpHandler.Health).Methods("GET")

    // Start server
    srv := &http.Server{
        Addr:         ":8083",
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
    }

    // Graceful shutdown
    go func() {
        log.Info("Message service starting", "port", 8083)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("Server failed", "error", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Info("Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Error("Server forced to shutdown", "error", err)
    }

    log.Info("Server stopped")
}
```

---

## Real-time Delivery Options

### Option 1: WebSocket (Recommended for Real-time)

**Pros:**
- True real-time bidirectional communication
- Low latency
- Persistent connection

**Cons:**
- Requires connection management
- More complex scaling

### Option 2: Server-Sent Events (SSE)

**Pros:**
- Simple unidirectional streaming
- Built on HTTP
- Auto-reconnect

**Cons:**
- One-way only (server → client)
- Not supported in all browsers

### Option 3: Long Polling

**Pros:**
- Works everywhere
- Simple to implement

**Cons:**
- Higher latency
- More server resources

### Option 4: gRPC Streaming

**Pros:**
- Efficient binary protocol
- Bidirectional streaming
- Built-in load balancing

**Cons:**
- Not supported in web browsers natively
- Requires HTTP/2

**Recommendation:** Use **WebSocket** for mobile/web apps with fallback to **push notifications** for offline users.

---

## Client Implementation

### Mobile Client (iOS - Swift)

**WebSocket Connection:**

```swift
import Foundation
import Starscream

class MessageService: WebSocketDelegate {
    private var socket: WebSocket?
    private let baseURL = "ws://localhost:8083/ws"

    func connect(token: String) {
        var request = URLRequest(url: URL(string: baseURL)!)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        request.setValue(UIDevice.current.identifierForVendor?.uuidString,
                        forHTTPHeaderField: "X-Device-ID")

        socket = WebSocket(request: request)
        socket?.delegate = self
        socket?.connect()
    }

    func sendMessage(conversationId: String, content: String) {
        let message: [String: Any] = [
            "conversation_id": conversationId,
            "content": content,
            "message_type": "text"
        ]

        guard let url = URL(string: "http://localhost:8080/api/v1/messages") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try? JSONSerialization.data(withJSONObject: message)

        URLSession.shared.dataTask(with: request) { data, response, error in
            // Handle response
        }.resume()
    }

    // WebSocket delegates
    func didReceive(event: WebSocketEvent, client: WebSocket) {
        switch event {
        case .connected(_):
            print("WebSocket connected")

        case .text(let text):
            handleIncomingMessage(text)

        case .disconnected(let reason, let code):
            print("WebSocket disconnected: \(reason) with code: \(code)")

        case .error(let error):
            print("WebSocket error: \(error?.localizedDescription ?? "")")

        default:
            break
        }
    }

    func handleIncomingMessage(_ json: String) {
        guard let data = json.data(using: .utf8),
              let event = try? JSONDecoder().decode(MessageEvent.self, from: data) else {
            return
        }

        switch event.type {
        case "new_message":
            // Update UI with new message
            NotificationCenter.default.post(
                name: .newMessageReceived,
                object: event.message
            )

            // Send read receipt
            sendReadReceipt(messageId: event.message.id)

        case "message_read":
            // Update message status
            break

        case "typing":
            // Show typing indicator
            break

        default:
            break
        }
    }

    func sendReadReceipt(messageId: String) {
        let receipt: [String: Any] = [
            "type": "read_receipt",
            "message_id": messageId,
            "read_at": ISO8601DateFormatter().string(from: Date())
        ]

        if let data = try? JSONSerialization.data(withJSONObject: receipt),
           let text = String(data: data, encoding: .utf8) {
            socket?.write(string: text)
        }
    }
}

struct MessageEvent: Codable {
    let type: String
    let message: Message
}

struct Message: Codable {
    let id: String
    let conversationId: String
    let senderUserId: String
    let content: String
    let messageType: String
    let createdAt: Date
}
```

### Web Client (React/TypeScript)

**WebSocket Hook:**

```typescript
import { useEffect, useRef, useState } from 'react';

interface Message {
  id: string;
  conversationId: string;
  senderUserId: string;
  content: string;
  messageType: string;
  createdAt: string;
}

interface MessageEvent {
  type: string;
  message: Message;
}

export const useWebSocket = (token: string) => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [connected, setConnected] = useState(false);
  const ws = useRef<WebSocket | null>(null);

  useEffect(() => {
    // Connect to WebSocket
    ws.current = new WebSocket('ws://localhost:8083/ws');

    ws.current.onopen = () => {
      console.log('WebSocket connected');
      setConnected(true);

      // Send authentication
      ws.current?.send(JSON.stringify({
        type: 'auth',
        token: token
      }));
    };

    ws.current.onmessage = (event) => {
      const data: MessageEvent = JSON.parse(event.data);

      switch (data.type) {
        case 'new_message':
          setMessages(prev => [...prev, data.message]);

          // Send read receipt
          sendReadReceipt(data.message.id);
          break;

        case 'message_read':
          // Update message status
          break;
      }
    };

    ws.current.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.current.onclose = () => {
      console.log('WebSocket disconnected');
      setConnected(false);

      // Reconnect after 3 seconds
      setTimeout(() => {
        // Reconnect logic
      }, 3000);
    };

    return () => {
      ws.current?.close();
    };
  }, [token]);

  const sendMessage = async (conversationId: string, content: string) => {
    const response = await fetch('http://localhost:8080/api/v1/messages', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        conversation_id: conversationId,
        content: content,
        message_type: 'text'
      })
    });

    return await response.json();
  };

  const sendReadReceipt = (messageId: string) => {
    ws.current?.send(JSON.stringify({
      type: 'read_receipt',
      message_id: messageId,
      read_at: new Date().toISOString()
    }));
  };

  return {
    messages,
    connected,
    sendMessage,
    sendReadReceipt
  };
};
```

**React Component:**

```typescript
import React, { useState } from 'react';
import { useWebSocket } from './useWebSocket';

const ChatView: React.FC = () => {
  const [messageText, setMessageText] = useState('');
  const { messages, connected, sendMessage } = useWebSocket(authToken);
  const conversationId = 'conversation-id-here';

  const handleSend = async () => {
    if (!messageText.trim()) return;

    await sendMessage(conversationId, messageText);
    setMessageText('');
  };

  return (
    <div className="chat-view">
      <div className="connection-status">
        {connected ? '🟢 Connected' : '🔴 Disconnected'}
      </div>

      <div className="messages">
        {messages.map(msg => (
          <div key={msg.id} className="message">
            <div className="sender">{msg.senderUserId}</div>
            <div className="content">{msg.content}</div>
            <div className="time">{new Date(msg.createdAt).toLocaleTimeString()}</div>
          </div>
        ))}
      </div>

      <div className="input-area">
        <input
          type="text"
          value={messageText}
          onChange={(e) => setMessageText(e.target.value)}
          onKeyPress={(e) => e.key === 'Enter' && handleSend()}
          placeholder="Type a message..."
        />
        <button onClick={handleSend}>Send</button>
      </div>
    </div>
  );
};
```

### Android Client (Kotlin)

```kotlin
import okhttp3.*
import org.json.JSONObject
import java.util.concurrent.TimeUnit

class MessageService(private val token: String) {
    private var webSocket: WebSocket? = null
    private val client = OkHttpClient.Builder()
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    fun connect() {
        val request = Request.Builder()
            .url("ws://localhost:8083/ws")
            .addHeader("Authorization", "Bearer $token")
            .build()

        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                println("WebSocket connected")
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                handleMessage(text)
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                println("WebSocket error: ${t.message}")
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                println("WebSocket closed: $reason")
            }
        })
    }

    fun sendMessage(conversationId: String, content: String) {
        val json = JSONObject().apply {
            put("conversation_id", conversationId)
            put("content", content)
            put("message_type", "text")
        }

        val requestBody = RequestBody.create(
            MediaType.parse("application/json"),
            json.toString()
        )

        val request = Request.Builder()
            .url("http://localhost:8080/api/v1/messages")
            .addHeader("Authorization", "Bearer $token")
            .post(requestBody)
            .build()

        client.newCall(request).enqueue(object : Callback {
            override fun onResponse(call: Call, response: Response) {
                // Handle response
            }

            override fun onFailure(call: Call, e: IOException) {
                // Handle error
            }
        })
    }

    private fun handleMessage(text: String) {
        val json = JSONObject(text)
        val type = json.getString("type")

        when (type) {
            "new_message" -> {
                val message = json.getJSONObject("message")
                // Update UI
                // Send read receipt
                sendReadReceipt(message.getString("id"))
            }
        }
    }

    private fun sendReadReceipt(messageId: String) {
        val receipt = JSONObject().apply {
            put("type", "read_receipt")
            put("message_id", messageId)
            put("read_at", System.currentTimeMillis())
        }

        webSocket?.send(receipt.toString())
    }

    fun disconnect() {
        webSocket?.close(1000, "User disconnected")
    }
}
```

---

## Delivery Tracking

### Read Receipts Implementation

**Backend - Handle Read Receipt:**

```go
func (s *MessageService) MarkAsRead(ctx context.Context, messageID, userID string) error {
    query := `
        UPDATE messages.delivery_status
        SET
            status = 'read',
            read_at = NOW()
        WHERE message_id = $1 AND user_id = $2
    `

    _, err := s.db.ExecContext(ctx, query, messageID, userID)
    if err != nil {
        return err
    }

    // Notify sender about read receipt
    go s.notifyReadReceipt(messageID, userID)

    return nil
}

func (s *MessageService) notifyReadReceipt(messageID, readerID string) {
    // Get message sender
    var senderID string
    query := `SELECT sender_user_id FROM messages.messages WHERE id = $1`
    s.db.QueryRow(query, messageID).Scan(&senderID)

    // Send read receipt to sender
    event := map[string]interface{}{
        "type":       "message_read",
        "message_id": messageID,
        "reader_id":  readerID,
        "read_at":    time.Now(),
    }

    eventJSON, _ := json.Marshal(event)
    s.hub.SendToUser(senderID, eventJSON)
}
```

### Typing Indicators

**Backend - Handle Typing:**

```go
func (s *MessageService) SetTyping(ctx context.Context, conversationID, userID string, isTyping bool) error {
    if isTyping {
        query := `
            INSERT INTO messages.typing_indicators (conversation_id, user_id, expires_at)
            VALUES ($1, $2, NOW() + INTERVAL '10 seconds')
            ON CONFLICT (conversation_id, user_id) DO UPDATE SET
                started_at = NOW(),
                expires_at = NOW() + INTERVAL '10 seconds'
        `
        _, err := s.db.ExecContext(ctx, query, conversationID, userID)
        if err != nil {
            return err
        }
    } else {
        query := `DELETE FROM messages.typing_indicators WHERE conversation_id = $1 AND user_id = $2`
        _, err := s.db.ExecContext(ctx, query, conversationID, userID)
        if err != nil {
            return err
        }
    }

    // Broadcast typing status to other participants
    go s.broadcastTypingStatus(conversationID, userID, isTyping)

    return nil
}
```

---

## Testing

### 1. Test Message Sending (cURL)

```bash
# Send a message
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "11111111-1111-1111-1111-111111111111",
    "content": "Hello from the API!",
    "message_type": "text"
  }'

# Response:
# {
#   "id": "new-message-id",
#   "conversation_id": "11111111-1111-1111-1111-111111111111",
#   "sender_user_id": "a1111111-1111-1111-1111-111111111111",
#   "content": "Hello from the API!",
#   "status": "sent",
#   "created_at": "2024-11-03T12:00:00Z"
# }
```

### 2. Test WebSocket Connection (JavaScript)

```javascript
// Browser console test
const ws = new WebSocket('ws://localhost:8083/ws');

ws.onopen = () => {
  console.log('Connected');

  // Send auth
  ws.send(JSON.stringify({
    type: 'auth',
    token: 'YOUR_TOKEN'
  }));
};

ws.onmessage = (event) => {
  console.log('Received:', JSON.parse(event.data));
};

// Send message via REST API
fetch('http://localhost:8080/api/v1/messages', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer YOUR_TOKEN',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    conversation_id: '11111111-1111-1111-1111-111111111111',
    content: 'Test message',
    message_type: 'text'
  })
})
.then(r => r.json())
.then(console.log);
```

### 3. Test with Postman

1. **Send Message:**
   - Method: POST
   - URL: `http://localhost:8080/api/v1/messages`
   - Headers:
     - `Authorization: Bearer YOUR_TOKEN`
     - `Content-Type: application/json`
   - Body:
     ```json
     {
       "conversation_id": "11111111-1111-1111-1111-111111111111",
       "content": "Hello!",
       "message_type": "text"
     }
     ```

2. **WebSocket Test:**
   - Use Postman WebSocket feature
   - URL: `ws://localhost:8083/ws`
   - Headers: `Authorization: Bearer YOUR_TOKEN`

---

## Complete Flow Summary

```
1. Phone Client
   └─> POST /api/v1/messages {conversation_id, content}

2. API Gateway (8080)
   └─> Validates JWT token
   └─> Forwards to Message Service (8083)

3. Message Service
   └─> Validates user is participant
   └─> Saves to database (messages.messages)
   └─> Gets conversation participants
   └─> Broadcasts via WebSocket Hub

4. WebSocket Hub
   ├─> Online users: Send via WebSocket
   │   └─> Client receives instantly
   │   └─> Client sends read receipt
   └─> Offline users: Publish to Kafka
       └─> Notification Service picks up
       └─> Sends push notification (FCM/APNS)

5. Delivery Tracking
   └─> Update messages.delivery_status
   └─> Notify sender about read receipts
```

---

## Next Steps

1. **Implement Message Service** using the code above
2. **Setup Kafka** for async message queue
3. **Add Push Notifications** for offline users
4. **Implement File Upload** for media messages
5. **Add Encryption** for secure messaging
6. **Setup Load Balancer** for multiple WebSocket servers

---

This guide provides everything you need to implement the complete message flow from phone to API and broadcast to recipients! 🚀
