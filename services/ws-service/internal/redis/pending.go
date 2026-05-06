package redis

import (
	"context"

	"shared/pkg/logger"
)

// PendingStore buffers messages destined for a user who has no live connection
// across the cluster, so a quick reconnect (cellular handoff, app foreground)
// can replay them without hitting Postgres. Capped at MaxPendingMessages with
// a TTLPending TTL — anything older falls back to the Postgres catch-up path.
type PendingStore struct {
	client *Client
	log    logger.Logger
}

func NewPendingStore(client *Client, log logger.Logger) *PendingStore {
	return &PendingStore{client: client, log: log}
}

// Push appends a serialized frame to the user's pending list and refreshes
// TTL. Trims to keep at most MaxPendingMessages most-recent entries.
func (p *PendingStore) Push(ctx context.Context, userID string, payload []byte) error {
	key := PendingKey(userID)
	pipe := p.client.Pipeline()
	pipe.RPush(ctx, key, payload)
	pipe.LTrim(ctx, key, -int64(MaxPendingMessages), -1)
	pipe.Expire(ctx, key, TTLPending)
	if _, err := pipe.Exec(ctx); err != nil {
		p.log.Warn("failed to push pending message",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return err
	}
	return nil
}

// Drain returns all buffered frames for the user and deletes the list
// atomically. Returns an empty slice if there's nothing pending.
func (p *PendingStore) Drain(ctx context.Context, userID string) ([][]byte, error) {
	key := PendingKey(userID)
	pipe := p.client.Pipeline()
	rangeCmd := pipe.LRange(ctx, key, 0, -1)
	pipe.Del(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	raws, err := rangeCmd.Result()
	if err != nil {
		return nil, err
	}
	out := make([][]byte, len(raws))
	for i, s := range raws {
		out[i] = []byte(s)
	}
	return out, nil
}
