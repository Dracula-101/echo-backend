# Message Service - Key Flows

## 1. Send Message Flow

```
Client POST / -> Handler validates auth + DTO -> Service checks:
  - User is participant in conversation
  - Conversation is active
  - User is not blocked
  - Rate limit not exceeded
  - Message type validation (text, image, video, etc.)
-> Gets recipient user IDs from conversation participants
-> Builds ChatMessageEvent with metadata (device info, IP, content hash)
-> Publishes to Kafka `chat-messages` topic
-> Returns preliminary Message to client (status=sent)

Kafka Consumer (async):
-> ProcessChatMessage receives event
-> Inserts message to DB in transaction
-> Creates delivery_status rows for each recipient
-> DB triggers fire:
   - Update conversation last_message_*, message_count
   - Increment unread_count for each participant
   - Index message content in search_index table
```

## 2. Message Sync / Catch-up Flow

```
Client reconnects after being offline
-> GET /sync?conversation_id={id}&last_message_id={id}&limit=100
-> Service queries messages after the cursor in ascending order
-> Returns { messages, has_more }
-> Client repeats until has_more=false
-> Client calls POST /{last_message_id}/read to mark caught up
-> Client also refreshes conversation list via GET /conversations/me
   for updated unread counts and last message previews
```

## 3. Read Receipt Flow

```
Client views message -> POST /{message_id}/read
-> Service validates user is participant in the conversation
-> Calls deliveryRepo.MarkConversationAsRead using DB transaction:
   1. UPDATE delivery_status SET status='read', read_at=NOW()
      for all messages up to and including the target message
   2. UPDATE conversation_participants SET unread_count=0
   3. Recalculate read_count on affected messages
-> Publishes DeliveryEventRead to Kafka
-> ws-service broadcasts read receipt to message senders
```

## 4. Typing Indicator Flow

```
Client starts typing:
-> POST /conversations/{id}/typing?is_typing=true
-> Service upserts typing_indicators row (expires in 10s)
-> Publishes ConversationEventTypeTypingStarted
-> ws-service broadcasts to conversation members

Client keeps typing:
-> Re-sends POST every 5s to keep indicator alive

Client stops typing:
-> POST /conversations/{id}/typing?is_typing=false
-> Service deletes typing_indicators row
-> Publishes ConversationEventTypeTypingStopped

Expiry:
-> Rows with expires_at < NOW() are automatically excluded from queries
-> Expired rows cleaned up on next INSERT (DB trigger)
```

## 5. Delivery Guarantee Flow

```
1. Client sends message -> Kafka publish with ack
   - If publish fails, client gets 500 and should retry
2. DB insert uses ON CONFLICT (id) DO NOTHING for idempotent inserts
3. Kafka consumer commits offset only after successful ProcessChatMessage
4. If consumer crashes mid-batch, transaction rolls back, Kafka redelivers
5. Offline users sync via /sync endpoint on reconnect
6. content_hash field enables application-level dedup detection
```

## 6. Message Edit Flow

```
Client PUT /{id} with new content
-> Handler parses EditMessageRequest
-> Service validates:
   - Message exists and is not deleted
   - User is the message sender
   - Edit window not expired (24h from creation)
   - New content is not empty
-> Repo: UPDATE content, updated_at
-> DB trigger automatically sets is_edited=TRUE, edited_at=NOW()
-> Publishes MessageEventTypeEdited to Kafka
-> ws-service broadcasts edit to conversation members
```

## 7. Message Delete Flow

```
Client DELETE /{id}?delete_for=everyone
-> Handler validates auth and path param
-> Service validates:
   - Message exists
   - User is sender OR has admin can_delete_messages permission
   - Message not already deleted (idempotent)
-> Repo calls DB function messages.delete_message()
   - Soft-delete: sets is_deleted=TRUE, replaces content
   - Search index trigger removes entry
-> Publishes MessageEventTypeDeleted to Kafka
-> ws-service broadcasts deletion to conversation members
```

## 8. Forward Message Flow

```
Client POST /{id}/forward with target_conversation_id
-> Service fetches original message (must exist, not deleted)
-> Validates user is participant in target conversation
-> Builds new ChatMessageEvent:
   - IsForwarded=true
   - ForwardedFromMessageID set to original
   - Copies content and metadata
-> Publishes to Kafka chat-messages topic
-> DB trigger on original: increment forward_count
-> Normal message processing flow applies for the new message
```

## 9. Pin/Unpin Message Flow

```
Pin: POST /{id}/pin
-> Validates user has can_pin_messages or manage permission
-> Repo executes in transaction:
   1. INSERT INTO pinned_messages
   2. UPDATE messages SET is_pinned=TRUE
-> Publishes MessageEventTypePinned

Unpin: DELETE /{id}/pin
-> Same permission check
-> Repo executes in transaction:
   1. DELETE FROM pinned_messages
   2. UPDATE messages SET is_pinned=FALSE
-> Publishes MessageEventTypeUnpinned

Get pinned: GET /conversations/{id}/pinned
-> Returns all pinned messages ordered by pin_order
```

## 10. Thread / Reply Chain Flow

```
GET /{parent_message_id}/thread?limit=50&before={cursor}
-> Queries messages WHERE parent_message_id = {id}
-> Returns replies in ascending order (oldest first)
-> Supports cursor-based pagination via before param
-> Returns { messages, has_more }

Sending a reply:
-> POST / with parent_message_id set in the request body
-> Normal send flow, but parent_message_id links to parent
-> DB trigger increments reply_count on parent message
```

## 11. Search Messages Flow

```
POST /search with { query, conversation_id?, limit, offset }
-> Service validates non-empty query
-> Repo uses PostgreSQL full-text search:
   SELECT m.*, ts_rank(si.content_tsvector, query) as rank
   FROM messages.search_index si
   JOIN messages.messages m ON m.id = si.message_id
   JOIN messages.conversation_participants cp
     ON cp.conversation_id = si.conversation_id
     AND cp.user_id = {user_id}
   WHERE si.content_tsvector @@ plainto_tsquery('english', {query})
-> Participant join ensures users only see messages from their conversations
-> Returns { messages, total_count, limit, offset }
```

## 12. Conversation Invite Flow

```
Create: POST /conversations/{id}/invites
-> Generates random 8-char hex invite code
-> Stores with max_uses, expires_at
-> Returns invite with code

Accept: POST /invites/{code}/accept
-> Validates invite: not expired, not revoked, use_count < max_uses
-> Increments use_count
-> Adds user to conversation as participant

Revoke: DELETE /conversations/{id}/invites/{invite_id}
-> Sets status='revoked', revoked_at=NOW()
```

## 13. Draft Flow

```
Save: PUT /conversations/{id}/draft
-> Upserts draft (one per user per conversation)
-> Stores content and optional reply_to_message_id

Get: GET /conversations/{id}/draft
-> Returns draft if exists, null otherwise

Delete: DELETE /conversations/{id}/draft
-> Removes draft (e.g., after message is sent)

Client should auto-save drafts periodically and delete on send.
```

## API Endpoints Summary

### Messages
| Method | Path | Description |
|--------|------|-------------|
| POST | `/` | Send message |
| GET | `/` | Get messages (paginated) |
| GET | `/sync` | Sync messages after cursor |
| POST | `/search` | Search messages |
| GET | `/bookmarks` | Get bookmarked messages |
| GET | `/{id}` | Get message by ID |
| PUT | `/{id}` | Edit message |
| DELETE | `/{id}` | Delete message |
| POST | `/{id}/read` | Mark as read |
| GET | `/{id}/delivery` | Get delivery status |
| POST | `/{id}/forward` | Forward message |
| POST | `/{id}/pin` | Pin message |
| DELETE | `/{id}/pin` | Unpin message |
| GET | `/{id}/thread` | Get thread replies |
| POST | `/{id}/bookmark` | Bookmark message |
| DELETE | `/{id}/bookmark` | Remove bookmark |
| POST | `/{id}/report` | Report message |
| POST | `/{id}/reactions` | Add reaction |
| DELETE | `/{id}/reactions/{rid}` | Remove reaction |
| GET | `/{id}/reactions` | Get reactions |

### Polls
| Method | Path | Description |
|--------|------|-------------|
| POST | `/polls` | Create poll |
| POST | `/polls/{id}/vote` | Vote on poll |
| GET | `/polls/{id}/results` | Get poll results |

### Conversations
| Method | Path | Description |
|--------|------|-------------|
| POST | `/conversations` | Create conversation |
| GET | `/conversations/me` | Get user's conversations |
| GET | `/conversations/{id}` | Get conversation by ID |
| PUT | `/conversations/{id}` | Update conversation |
| DELETE | `/conversations/{id}` | Delete conversation |
| POST | `/conversations/{id}/participants` | Add participants |
| DELETE | `/conversations/{id}/participants/{uid}` | Remove participant |
| PUT | `/conversations/{id}/participants/{uid}` | Update participant role |
| GET | `/conversations/{id}/pinned` | Get pinned messages |
| POST | `/conversations/{id}/mute` | Mute conversation |
| DELETE | `/conversations/{id}/mute` | Unmute conversation |
| GET | `/conversations/{id}/unread` | Get unread count |
| POST | `/conversations/{id}/typing` | Set typing indicator |
| GET | `/conversations/{id}/typing` | Get typing users |
| PUT | `/conversations/{id}/draft` | Save draft |
| GET | `/conversations/{id}/draft` | Get draft |
| DELETE | `/conversations/{id}/draft` | Delete draft |
| GET | `/conversations/{id}/settings` | Get settings |
| PUT | `/conversations/{id}/settings` | Update settings |
| POST | `/conversations/{id}/invites` | Create invite |
| GET | `/conversations/{id}/invites` | Get invites |
| DELETE | `/conversations/{id}/invites/{iid}` | Revoke invite |

### Invites
| Method | Path | Description |
|--------|------|-------------|
| POST | `/invites/{code}/accept` | Accept invite |
