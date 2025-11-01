package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Lock struct {
	client *redis.Client
	key    string
	token  string
	ttl    time.Duration
}

func NewLock(client *redis.Client, key string, ttl time.Duration) *Lock {
	return &Lock{
		client: client,
		key:    key,
		token:  generateToken(),
		ttl:    ttl,
	}
}

func (l *Lock) Acquire(ctx context.Context) (bool, error) {
	success, err := l.client.SetNX(ctx, l.key, l.token, l.ttl).Result()
	if err != nil {
		return false, err
	}
	return success, nil
}

func (l *Lock) Release(ctx context.Context) error {
	script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.token).Result()
	if err != nil {
		return err
	}

	if result.(int64) == 0 {
		return errors.New("lock not held")
	}

	return nil
}

func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("pexpire", KEYS[1], ARGV[2])
        else
            return 0
        end
    `

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.token, int64(ttl/time.Millisecond)).Result()
	if err != nil {
		return err
	}

	if result.(int64) == 0 {
		return errors.New("lock not held")
	}

	return nil
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
