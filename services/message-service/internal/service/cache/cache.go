package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"shared/pkg/cache"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	msgError "echo-backend/services/message-service/internal/error"
)

type MessageCache interface {
	GetConversationList(ctx context.Context, userID string) ([]byte, pkgErrors.AppError)
	SetConversationList(ctx context.Context, userID string, data []byte) pkgErrors.AppError
	InvalidateConversationList(ctx context.Context, userID string) pkgErrors.AppError

	GetConversation(ctx context.Context, conversationID string) ([]byte, pkgErrors.AppError)
	SetConversation(ctx context.Context, conversationID string, data []byte) pkgErrors.AppError
	InvalidateConversation(ctx context.Context, conversationID string) pkgErrors.AppError

	GetUnreadCount(ctx context.Context, userID, conversationID string) (int64, pkgErrors.AppError)
	IncrementUnreadCount(ctx context.Context, userID, conversationID string) (int64, pkgErrors.AppError)
	ResetUnreadCount(ctx context.Context, userID, conversationID string) pkgErrors.AppError

	GetTotalUnread(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	IncrementTotalUnread(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	DecrementTotalUnread(ctx context.Context, userID string, delta int64) (int64, pkgErrors.AppError)
	InvalidateTotalUnread(ctx context.Context, userID string) pkgErrors.AppError

	GetMessages(ctx context.Context, conversationID string, page int) ([]byte, pkgErrors.AppError)
	SetMessages(ctx context.Context, conversationID string, page int, data []byte) pkgErrors.AppError
	InvalidateMessages(ctx context.Context, conversationID string, page int) pkgErrors.AppError

	SetTyping(ctx context.Context, conversationID, userID string, data []byte) pkgErrors.AppError
	DeleteTyping(ctx context.Context, conversationID, userID string) pkgErrors.AppError
	GetTypingList(ctx context.Context, conversationID string) ([]string, pkgErrors.AppError)
	AddToTypingList(ctx context.Context, conversationID, userID string) pkgErrors.AppError
	RemoveFromTypingList(ctx context.Context, conversationID, userID string) pkgErrors.AppError

	GetDelivery(ctx context.Context, messageID string) ([]byte, pkgErrors.AppError)
	SetDelivery(ctx context.Context, messageID string, data []byte) pkgErrors.AppError
	InvalidateDelivery(ctx context.Context, messageID string) pkgErrors.AppError

	GetParticipants(ctx context.Context, conversationID string) ([]string, pkgErrors.AppError)
	SetParticipants(ctx context.Context, conversationID string, userIDs []string) pkgErrors.AppError
	InvalidateParticipants(ctx context.Context, conversationID string) pkgErrors.AppError

	GetPinnedMessages(ctx context.Context, conversationID string) ([]byte, pkgErrors.AppError)
	SetPinnedMessages(ctx context.Context, conversationID string, data []byte) pkgErrors.AppError
	InvalidatePinnedMessages(ctx context.Context, conversationID string) pkgErrors.AppError

	GetDMPair(ctx context.Context, uid1, uid2 string) (string, pkgErrors.AppError)
	SetDMPair(ctx context.Context, uid1, uid2, conversationID string) pkgErrors.AppError

	AcquireDMLock(ctx context.Context, uid1, uid2 string) (bool, pkgErrors.AppError)
	ReleaseDMLock(ctx context.Context, uid1, uid2 string) pkgErrors.AppError
	AcquireMessageLock(ctx context.Context, idempotencyKey string) (bool, pkgErrors.AppError)
	GetMessageLock(ctx context.Context, idempotencyKey string) (string, pkgErrors.AppError)
	SetMessageLock(ctx context.Context, idempotencyKey, messageID string) pkgErrors.AppError
	ReleaseMessageLock(ctx context.Context, idempotencyKey string) pkgErrors.AppError
}

type messageCache struct {
	cache cache.Cache
	log   logger.Logger
}

func NewMessageCache(c cache.Cache, log logger.Logger) MessageCache {
	return &messageCache{cache: c, log: log}
}

func dmPairKey(uid1, uid2 string) string {
	if uid1 <= uid2 {
		return fmt.Sprintf("%s%s:%s", DMPairPrefix, uid1, uid2)
	}
	return fmt.Sprintf("%s%s:%s", DMPairPrefix, uid2, uid1)
}

func dmLockKey(uid1, uid2 string) string {
	if uid1 <= uid2 {
		return fmt.Sprintf("%s%s:%s", LockDMPrefix, uid1, uid2)
	}
	return fmt.Sprintf("%s%s:%s", LockDMPrefix, uid2, uid1)
}

func (m *messageCache) GetConversationList(ctx context.Context, userID string) ([]byte, pkgErrors.AppError) {
	m.log.Info("Getting conversation list", logger.String("user_id", userID))
	data, err := m.cache.Get(ctx, ConvListPrefix+userID)
	if err != nil {
		m.log.Error("Failed to get conversation list", logger.String("user_id", userID), logger.Error(err))
		return nil, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to get conversation list").
			WithService(msgError.ServiceName).
			WithDetail("user_id", userID)
	}
	return data, nil
}

func (m *messageCache) SetConversationList(ctx context.Context, userID string, data []byte) pkgErrors.AppError {
	m.log.Info("Setting conversation list", logger.String("user_id", userID))
	if err := m.cache.Set(ctx, ConvListPrefix+userID, data, ConvListTTL); err != nil {
		m.log.Error("Failed to set conversation list", logger.String("user_id", userID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to set conversation list").
			WithService(msgError.ServiceName).
			WithDetail("user_id", userID)
	}
	return nil
}

func (m *messageCache) InvalidateConversationList(ctx context.Context, userID string) pkgErrors.AppError {
	m.log.Info("Invalidating conversation list", logger.String("user_id", userID))
	if err := m.cache.Delete(ctx, ConvListPrefix+userID); err != nil {
		m.log.Error("Failed to invalidate conversation list", logger.String("user_id", userID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to delete conversation list").
			WithService(msgError.ServiceName).
			WithDetail("user_id", userID)
	}
	return nil
}

func (m *messageCache) GetConversation(ctx context.Context, conversationID string) ([]byte, pkgErrors.AppError) {
	m.log.Info("Getting conversation", logger.String("conversation_id", conversationID))
	data, err := m.cache.Get(ctx, ConvPrefix+conversationID)
	if err != nil {
		m.log.Error("Failed to get conversation", logger.String("conversation_id", conversationID), logger.Error(err))
		return nil, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to get conversation").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	return data, nil
}

func (m *messageCache) SetConversation(ctx context.Context, conversationID string, data []byte) pkgErrors.AppError {
	m.log.Info("Setting conversation", logger.String("conversation_id", conversationID))
	if err := m.cache.Set(ctx, ConvPrefix+conversationID, data, ConvTTL); err != nil {
		m.log.Error("Failed to set conversation", logger.String("conversation_id", conversationID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to set conversation").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	return nil
}

func (m *messageCache) InvalidateConversation(ctx context.Context, conversationID string) pkgErrors.AppError {
	m.log.Info("Invalidating conversation", logger.String("conversation_id", conversationID))
	if err := m.cache.Delete(ctx, ConvPrefix+conversationID); err != nil {
		m.log.Error("Failed to invalidate conversation", logger.String("conversation_id", conversationID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to delete conversation").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	return nil
}

func (m *messageCache) unreadKey(userID, conversationID string) string {
	return fmt.Sprintf("%s%s:%s", UnreadPrefix, userID, conversationID)
}

func (m *messageCache) GetUnreadCount(ctx context.Context, userID, conversationID string) (int64, pkgErrors.AppError) {
	m.log.Info("Getting unread count", logger.String("user_id", userID), logger.String("conversation_id", conversationID))
	key := m.unreadKey(userID, conversationID)
	count, err := m.cache.GetInt(ctx, key)
	if err != nil {
		m.log.Error("Failed to get unread count", logger.String("key", key), logger.Error(err))
		return 0, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to get unread count").
			WithService(msgError.ServiceName).
			WithDetail("user_id", userID).
			WithDetail("conversation_id", conversationID)
	}
	return count, nil
}

func (m *messageCache) IncrementUnreadCount(ctx context.Context, userID, conversationID string) (int64, pkgErrors.AppError) {
	m.log.Info("Incrementing unread count", logger.String("user_id", userID), logger.String("conversation_id", conversationID))
	key := m.unreadKey(userID, conversationID)
	count, err := m.cache.Increment(ctx, key, 1)
	if err != nil {
		m.log.Error("Failed to increment unread count", logger.String("key", key), logger.Error(err))
		return 0, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to increment unread count").
			WithService(msgError.ServiceName).
			WithDetail("user_id", userID).
			WithDetail("conversation_id", conversationID)
	}
	if expErr := m.cache.Expire(ctx, key, UnreadTTL); expErr != nil {
		m.log.Warn("Failed to refresh unread TTL", logger.String("key", key), logger.Error(expErr))
	}
	return count, nil
}

func (m *messageCache) ResetUnreadCount(ctx context.Context, userID, conversationID string) pkgErrors.AppError {
	m.log.Info("Resetting unread count", logger.String("user_id", userID), logger.String("conversation_id", conversationID))
	if err := m.cache.Delete(ctx, m.unreadKey(userID, conversationID)); err != nil {
		m.log.Error("Failed to reset unread count", logger.String("user_id", userID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to reset unread count").
			WithService(msgError.ServiceName).
			WithDetail("user_id", userID).
			WithDetail("conversation_id", conversationID)
	}
	return nil
}

func (m *messageCache) GetTotalUnread(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	m.log.Info("Getting total unread", logger.String("user_id", userID))
	count, err := m.cache.GetInt(ctx, TotalUnreadPrefix+userID)
	if err != nil {
		m.log.Error("Failed to get total unread", logger.String("user_id", userID), logger.Error(err))
		return 0, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to get total unread count").
			WithService(msgError.ServiceName).
			WithDetail("user_id", userID)
	}
	return count, nil
}

func (m *messageCache) IncrementTotalUnread(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	m.log.Info("Incrementing total unread", logger.String("user_id", userID))
	key := TotalUnreadPrefix + userID
	count, err := m.cache.Increment(ctx, key, 1)
	if err != nil {
		m.log.Error("Failed to increment total unread", logger.String("user_id", userID), logger.Error(err))
		return 0, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to increment total unread count").
			WithService(msgError.ServiceName).
			WithDetail("user_id", userID)
	}
	if expErr := m.cache.Expire(ctx, key, TotalUnreadTTL); expErr != nil {
		m.log.Warn("Failed to refresh total unread TTL", logger.String("key", key), logger.Error(expErr))
	}
	return count, nil
}

func (m *messageCache) DecrementTotalUnread(ctx context.Context, userID string, delta int64) (int64, pkgErrors.AppError) {
	m.log.Info("Decrementing total unread", logger.String("user_id", userID))
	key := TotalUnreadPrefix + userID
	count, err := m.cache.Decrement(ctx, key, delta)
	if err != nil {
		m.log.Error("Failed to decrement total unread", logger.String("user_id", userID), logger.Error(err))
		return 0, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to decrement total unread count").
			WithService(msgError.ServiceName).
			WithDetail("user_id", userID).
			WithDetail("delta", delta)
	}
	if count < 0 {
		_ = m.cache.SetInt(ctx, key, 0, TotalUnreadTTL)
		return 0, nil
	}
	return count, nil
}

func (m *messageCache) InvalidateTotalUnread(ctx context.Context, userID string) pkgErrors.AppError {
	m.log.Info("Invalidating total unread", logger.String("user_id", userID))
	if err := m.cache.Delete(ctx, TotalUnreadPrefix+userID); err != nil {
		m.log.Error("Failed to invalidate total unread", logger.String("user_id", userID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to delete total unread count").
			WithService(msgError.ServiceName).
			WithDetail("user_id", userID)
	}
	return nil
}

func (m *messageCache) msgsKey(conversationID string, page int) string {
	return fmt.Sprintf("%s%s:%d", MsgsPrefix, conversationID, page)
}

func (m *messageCache) GetMessages(ctx context.Context, conversationID string, page int) ([]byte, pkgErrors.AppError) {
	m.log.Info("Getting messages", logger.String("conversation_id", conversationID), logger.Int("page", page))
	data, err := m.cache.Get(ctx, m.msgsKey(conversationID, page))
	if err != nil {
		m.log.Error("Failed to get messages", logger.String("conversation_id", conversationID), logger.Error(err))
		return nil, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to get messages").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID).
			WithDetail("page", page)
	}
	return data, nil
}

func (m *messageCache) SetMessages(ctx context.Context, conversationID string, page int, data []byte) pkgErrors.AppError {
	m.log.Info("Setting messages", logger.String("conversation_id", conversationID), logger.Int("page", page))
	if err := m.cache.Set(ctx, m.msgsKey(conversationID, page), data, MsgsTTL); err != nil {
		m.log.Error("Failed to set messages", logger.String("conversation_id", conversationID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to set messages").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID).
			WithDetail("page", page)
	}
	return nil
}

func (m *messageCache) InvalidateMessages(ctx context.Context, conversationID string, page int) pkgErrors.AppError {
	m.log.Info("Invalidating messages", logger.String("conversation_id", conversationID), logger.Int("page", page))
	if err := m.cache.Delete(ctx, m.msgsKey(conversationID, page)); err != nil {
		m.log.Error("Failed to invalidate messages", logger.String("conversation_id", conversationID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to delete messages").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID).
			WithDetail("page", page)
	}
	return nil
}

func (m *messageCache) SetTyping(ctx context.Context, conversationID, userID string, data []byte) pkgErrors.AppError {
	m.log.Info("Setting typing indicator", logger.String("conversation_id", conversationID), logger.String("user_id", userID))
	key := fmt.Sprintf("%s%s:%s", TypingPrefix, conversationID, userID)
	if err := m.cache.Set(ctx, key, data, TypingTTL); err != nil {
		m.log.Error("Failed to set typing indicator", logger.String("key", key), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to set typing indicator").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID).
			WithDetail("user_id", userID)
	}
	return nil
}

func (m *messageCache) DeleteTyping(ctx context.Context, conversationID, userID string) pkgErrors.AppError {
	m.log.Info("Deleting typing indicator", logger.String("conversation_id", conversationID), logger.String("user_id", userID))
	key := fmt.Sprintf("%s%s:%s", TypingPrefix, conversationID, userID)
	if err := m.cache.Delete(ctx, key); err != nil {
		m.log.Error("Failed to delete typing indicator", logger.String("key", key), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to delete typing indicator").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID).
			WithDetail("user_id", userID)
	}
	return nil
}

func (m *messageCache) GetTypingList(ctx context.Context, conversationID string) ([]string, pkgErrors.AppError) {
	m.log.Info("Getting typing list", logger.String("conversation_id", conversationID))
	key := TypingListPrefix + conversationID
	data, err := m.cache.Get(ctx, key)
	if err != nil {
		m.log.Error("Failed to get typing list", logger.String("key", key), logger.Error(err))
		return nil, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to get typing list").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	if data == nil {
		return []string{}, nil
	}
	var userIDs []string
	if jsonErr := json.Unmarshal(data, &userIDs); jsonErr != nil {
		return nil, pkgErrors.FromError(jsonErr, msgError.CodeCacheError, "failed to unmarshal typing list").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	return userIDs, nil
}

func (m *messageCache) AddToTypingList(ctx context.Context, conversationID, userID string) pkgErrors.AppError {
	m.log.Info("Adding to typing list", logger.String("conversation_id", conversationID), logger.String("user_id", userID))
	userIDs, mErr := m.GetTypingList(ctx, conversationID)
	if mErr != nil {
		return mErr
	}
	for _, id := range userIDs {
		if id == userID {
			_ = m.cache.Expire(ctx, TypingListPrefix+conversationID, TypingListTTL)
			return nil
		}
	}
	userIDs = append(userIDs, userID)
	return m.persistTypingList(ctx, conversationID, userIDs)
}

func (m *messageCache) RemoveFromTypingList(ctx context.Context, conversationID, userID string) pkgErrors.AppError {
	m.log.Info("Removing from typing list", logger.String("conversation_id", conversationID), logger.String("user_id", userID))
	userIDs, mErr := m.GetTypingList(ctx, conversationID)
	if mErr != nil {
		return mErr
	}
	filtered := userIDs[:0]
	for _, id := range userIDs {
		if id != userID {
			filtered = append(filtered, id)
		}
	}
	return m.persistTypingList(ctx, conversationID, filtered)
}

func (m *messageCache) persistTypingList(ctx context.Context, conversationID string, userIDs []string) pkgErrors.AppError {
	data, jsonErr := json.Marshal(userIDs)
	if jsonErr != nil {
		return pkgErrors.FromError(jsonErr, msgError.CodeCacheError, "failed to marshal typing list").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	if err := m.cache.Set(ctx, TypingListPrefix+conversationID, data, TypingListTTL); err != nil {
		m.log.Error("Failed to persist typing list", logger.String("conversation_id", conversationID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to set typing list").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	return nil
}

func (m *messageCache) GetDelivery(ctx context.Context, messageID string) ([]byte, pkgErrors.AppError) {
	m.log.Info("Getting delivery status", logger.String("message_id", messageID))
	data, err := m.cache.Get(ctx, DeliveryPrefix+messageID)
	if err != nil {
		m.log.Error("Failed to get delivery status", logger.String("message_id", messageID), logger.Error(err))
		return nil, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to get delivery status").
			WithService(msgError.ServiceName).
			WithDetail("message_id", messageID)
	}
	return data, nil
}

func (m *messageCache) SetDelivery(ctx context.Context, messageID string, data []byte) pkgErrors.AppError {
	m.log.Info("Setting delivery status", logger.String("message_id", messageID))
	if err := m.cache.Set(ctx, DeliveryPrefix+messageID, data, DeliveryTTL); err != nil {
		m.log.Error("Failed to set delivery status", logger.String("message_id", messageID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to set delivery status").
			WithService(msgError.ServiceName).
			WithDetail("message_id", messageID)
	}
	return nil
}

func (m *messageCache) InvalidateDelivery(ctx context.Context, messageID string) pkgErrors.AppError {
	m.log.Info("Invalidating delivery cache", logger.String("message_id", messageID))
	if err := m.cache.Delete(ctx, DeliveryPrefix+messageID); err != nil {
		m.log.Error("Failed to invalidate delivery status", logger.String("message_id", messageID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to delete delivery status").
			WithService(msgError.ServiceName).
			WithDetail("message_id", messageID)
	}
	return nil
}

func (m *messageCache) GetParticipants(ctx context.Context, conversationID string) ([]string, pkgErrors.AppError) {
	m.log.Info("Getting participants", logger.String("conversation_id", conversationID))
	data, err := m.cache.Get(ctx, ParticipantPrefix+conversationID)
	if err != nil {
		m.log.Error("Failed to get participants", logger.String("conversation_id", conversationID), logger.Error(err))
		return nil, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to get participants").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	if data == nil {
		return nil, nil
	}
	var userIDs []string
	if jsonErr := json.Unmarshal(data, &userIDs); jsonErr != nil {
		return nil, pkgErrors.FromError(jsonErr, msgError.CodeCacheError, "failed to unmarshal participants").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	return userIDs, nil
}

func (m *messageCache) SetParticipants(ctx context.Context, conversationID string, userIDs []string) pkgErrors.AppError {
	m.log.Info("Setting participants", logger.String("conversation_id", conversationID))
	data, jsonErr := json.Marshal(userIDs)
	if jsonErr != nil {
		return pkgErrors.FromError(jsonErr, msgError.CodeCacheError, "failed to marshal participants").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	if err := m.cache.Set(ctx, ParticipantPrefix+conversationID, data, ParticipantTTL); err != nil {
		m.log.Error("Failed to set participants", logger.String("conversation_id", conversationID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to set participants").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	return nil
}

func (m *messageCache) InvalidateParticipants(ctx context.Context, conversationID string) pkgErrors.AppError {
	m.log.Info("Invalidating participants", logger.String("conversation_id", conversationID))
	if err := m.cache.Delete(ctx, ParticipantPrefix+conversationID); err != nil {
		m.log.Error("Failed to invalidate participants", logger.String("conversation_id", conversationID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to delete participants").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	return nil
}

func (m *messageCache) GetPinnedMessages(ctx context.Context, conversationID string) ([]byte, pkgErrors.AppError) {
	m.log.Info("Getting pinned messages", logger.String("conversation_id", conversationID))
	data, err := m.cache.Get(ctx, PinnedPrefix+conversationID)
	if err != nil {
		m.log.Error("Failed to get pinned messages", logger.String("conversation_id", conversationID), logger.Error(err))
		return nil, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to get pinned messages").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	return data, nil
}

func (m *messageCache) SetPinnedMessages(ctx context.Context, conversationID string, data []byte) pkgErrors.AppError {
	m.log.Info("Setting pinned messages", logger.String("conversation_id", conversationID))
	if err := m.cache.Set(ctx, PinnedPrefix+conversationID, data, PinnedTTL); err != nil {
		m.log.Error("Failed to set pinned messages", logger.String("conversation_id", conversationID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to set pinned messages").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	return nil
}

func (m *messageCache) InvalidatePinnedMessages(ctx context.Context, conversationID string) pkgErrors.AppError {
	m.log.Info("Invalidating pinned messages", logger.String("conversation_id", conversationID))
	if err := m.cache.Delete(ctx, PinnedPrefix+conversationID); err != nil {
		m.log.Error("Failed to invalidate pinned messages", logger.String("conversation_id", conversationID), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to delete pinned messages").
			WithService(msgError.ServiceName).
			WithDetail("conversation_id", conversationID)
	}
	return nil
}

func (m *messageCache) GetDMPair(ctx context.Context, uid1, uid2 string) (string, pkgErrors.AppError) {
	m.log.Info("Getting DM pair", logger.String("uid1", uid1), logger.String("uid2", uid2))
	key := dmPairKey(uid1, uid2)
	conversationID, err := m.cache.GetString(ctx, key)
	if err != nil {
		m.log.Error("Failed to get DM pair", logger.String("key", key), logger.Error(err))
		return "", pkgErrors.FromError(err, msgError.CodeCacheError, "failed to get DM pair").
			WithService(msgError.ServiceName).
			WithDetail("uid1", uid1).
			WithDetail("uid2", uid2)
	}
	return conversationID, nil
}

func (m *messageCache) SetDMPair(ctx context.Context, uid1, uid2, conversationID string) pkgErrors.AppError {
	m.log.Info("Setting DM pair", logger.String("uid1", uid1), logger.String("uid2", uid2))
	key := dmPairKey(uid1, uid2)
	if err := m.cache.SetString(ctx, key, conversationID, DMPairTTL); err != nil {
		m.log.Error("Failed to set DM pair", logger.String("key", key), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to set DM pair").
			WithService(msgError.ServiceName).
			WithDetail("uid1", uid1).
			WithDetail("uid2", uid2).
			WithDetail("conversation_id", conversationID)
	}
	return nil
}

func (m *messageCache) AcquireDMLock(ctx context.Context, uid1, uid2 string) (bool, pkgErrors.AppError) {
	m.log.Info("Acquiring DM lock", logger.String("uid1", uid1), logger.String("uid2", uid2))
	key := dmLockKey(uid1, uid2)
	locked, err := m.cache.AcquireLock(ctx, key, LockDMTTL)
	if err != nil {
		m.log.Error("Failed to acquire DM lock", logger.String("key", key), logger.Error(err))
		return false, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to acquire DM lock").
			WithService(msgError.ServiceName).
			WithDetail("uid1", uid1).
			WithDetail("uid2", uid2)
	}
	return locked, nil
}

func (m *messageCache) ReleaseDMLock(ctx context.Context, uid1, uid2 string) pkgErrors.AppError {
	m.log.Info("Releasing DM lock", logger.String("uid1", uid1), logger.String("uid2", uid2))
	key := dmLockKey(uid1, uid2)
	if err := m.cache.ReleaseLock(ctx, key); err != nil {
		m.log.Error("Failed to release DM lock", logger.String("key", key), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to release DM lock").
			WithService(msgError.ServiceName).
			WithDetail("uid1", uid1).
			WithDetail("uid2", uid2)
	}
	return nil
}

func (m *messageCache) AcquireMessageLock(ctx context.Context, idempotencyKey string) (bool, pkgErrors.AppError) {
	m.log.Info("Acquiring message lock", logger.String("idempotency_key", idempotencyKey))
	key := LockMsgPrefix + idempotencyKey
	locked, err := m.cache.AcquireLock(ctx, key, LockMsgTTL)
	if err != nil {
		m.log.Error("Failed to acquire message lock", logger.String("key", key), logger.Error(err))
		return false, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to acquire message lock").
			WithService(msgError.ServiceName).
			WithDetail("idempotency_key", idempotencyKey)
	}
	return locked, nil
}

func (m *messageCache) GetMessageLock(ctx context.Context, idempotencyKey string) (string, pkgErrors.AppError) {
	m.log.Info("Getting message lock", logger.String("idempotency_key", idempotencyKey))
	key := LockMsgPrefix + idempotencyKey
	messageID, err := m.cache.GetString(ctx, key)
	if err != nil {
		m.log.Error("Failed to get message lock", logger.String("key", key), logger.Error(err))
		return "", pkgErrors.FromError(err, msgError.CodeCacheError, "failed to get message lock").
			WithService(msgError.ServiceName).
			WithDetail("idempotency_key", idempotencyKey)
	}
	return messageID, nil
}

func (m *messageCache) SetMessageLock(ctx context.Context, idempotencyKey, messageID string) pkgErrors.AppError {
	m.log.Info("Setting message lock", logger.String("idempotency_key", idempotencyKey), logger.String("message_id", messageID))
	key := LockMsgPrefix + idempotencyKey
	if err := m.cache.SetString(ctx, key, messageID, LockMsgTTL); err != nil {
		m.log.Error("Failed to set message lock", logger.String("key", key), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to set message lock").
			WithService(msgError.ServiceName).
			WithDetail("idempotency_key", idempotencyKey).
			WithDetail("message_id", messageID)
	}
	return nil
}

func (m *messageCache) ReleaseMessageLock(ctx context.Context, idempotencyKey string) pkgErrors.AppError {
	m.log.Info("Releasing message lock", logger.String("idempotency_key", idempotencyKey))
	key := LockMsgPrefix + idempotencyKey
	if err := m.cache.ReleaseLock(ctx, key); err != nil {
		m.log.Error("Failed to release message lock", logger.String("key", key), logger.Error(err))
		return pkgErrors.FromError(err, msgError.CodeCacheError, "failed to release message lock").
			WithService(msgError.ServiceName).
			WithDetail("idempotency_key", idempotencyKey)
	}
	return nil
}
