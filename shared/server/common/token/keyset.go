package token

import (
	"context"
	"crypto/hmac"
	"encoding/base64"
	"errors"
	"fmt"
)

type Key struct {
	ID        string
	Secret    []byte
	Algorithm string
}

type KeySet interface {
	Current(ctx context.Context) (Key, error)
	Lookup(ctx context.Context, keyID string) (Key, error)
}

type StaticKeySet struct {
	key Key
}

func NewStaticKeySet(secret []byte) (*StaticKeySet, error) {
	if len(secret) == 0 {
		return nil, errors.New("token: static key must not be empty")
	}
	encoded := base64.StdEncoding.EncodeToString(secret)
	return &StaticKeySet{
		key: Key{
			ID:        fmt.Sprintf("static-%s", encoded[:12]),
			Secret:    secret,
			Algorithm: "HS256",
		},
	}, nil
}

func (s *StaticKeySet) Current(ctx context.Context) (Key, error) {
	return s.key, nil
}

func (s *StaticKeySet) Lookup(ctx context.Context, keyID string) (Key, error) {
	if keyID == "" || keyID == s.key.ID {
		return s.key, nil
	}
	return Key{}, ErrKeyNotFound
}

type RotatingKeySet struct {
	primary Key
	backups map[string]Key
}

func NewRotatingKeySet(primary Key, backups ...Key) (*RotatingKeySet, error) {
	if err := validateKey(primary); err != nil {
		return nil, err
	}
	set := &RotatingKeySet{primary: primary, backups: make(map[string]Key)}
	for _, k := range backups {
		if err := validateKey(k); err != nil {
			return nil, err
		}
		set.backups[k.ID] = k
	}
	return set, nil
}

func (r *RotatingKeySet) Current(ctx context.Context) (Key, error) {
	return r.primary, nil
}

func (r *RotatingKeySet) Lookup(ctx context.Context, keyID string) (Key, error) {
	if keyID == "" || keyID == r.primary.ID {
		return r.primary, nil
	}
	if key, ok := r.backups[keyID]; ok {
		return key, nil
	}
	return Key{}, ErrKeyNotFound
}

func (r *RotatingKeySet) Rotate(newPrimary Key) error {
	if err := validateKey(newPrimary); err != nil {
		return err
	}
	if r.backups == nil {
		r.backups = make(map[string]Key)
	}
	r.backups[r.primary.ID] = r.primary
	r.primary = newPrimary
	return nil
}

func validateKey(key Key) error {
	if key.ID == "" {
		return errors.New("token: key id required")
	}
	if len(key.Secret) == 0 {
		return errors.New("token: key secret required")
	}
	if !hmac.Equal([]byte(key.Algorithm), []byte(key.Algorithm)) {
		return errors.New("token: invalid algorithm")
	}
	return nil
}
