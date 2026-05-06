package redis

import "context"

// SeqCounter issues monotonically-increasing sequence numbers per conversation.
// Used so a reconnecting client can request "everything after seq N" without
// time-based windowing.
type SeqCounter struct {
	client *Client
}

func NewSeqCounter(client *Client) *SeqCounter {
	return &SeqCounter{client: client}
}

func (s *SeqCounter) Next(ctx context.Context, conversationID string) (int64, error) {
	return s.client.Raw().Incr(ctx, SeqKey(conversationID)).Result()
}

// Peek returns the current sequence value without incrementing. Returns 0 if
// the key doesn't exist yet (no messages have been sent in this conversation).
func (s *SeqCounter) Peek(ctx context.Context, conversationID string) (int64, error) {
	v, err := s.client.Raw().Get(ctx, SeqKey(conversationID)).Int64()
	if err != nil {
		if err.Error() == "redis: nil" {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}
