package encryption

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"sync"

	"golang.org/x/crypto/chacha20poly1305"
)

type Cipher string

type Key struct {
	ID     string
	Secret []byte
}

type Config struct {
	Primary  Key
	Fallback []Key
	Cipher   Cipher
}

type Manager struct {
	cipher   Cipher
	primary  Key
	fallback map[string]Key
	cache    map[string]cipher.AEAD
	mu       sync.RWMutex
}

type EncryptOptions struct {
	AssociatedData []byte
	KeyID          string
}

type DecryptOptions struct {
	AssociatedData []byte
}

type Ciphertext struct {
	Cipher Cipher
	KeyID  string
	Nonce  []byte
	Data   []byte
}

const (
	CipherAESGCM           Cipher = "aes-gcm"
	CipherChaCha20Poly1305 Cipher = "chacha20poly1305"
)

func NewManager(cfg Config) (*Manager, error) {
	cipherType := cfg.Cipher
	if cipherType == "" {
		cipherType = CipherAESGCM
	}
	if err := validateKey(cfg.Primary, cipherType); err != nil {
		return nil, err
	}
	fallback := make(map[string]Key, len(cfg.Fallback))
	for _, key := range cfg.Fallback {
		if err := validateKey(key, cipherType); err != nil {
			return nil, err
		}
		if key.ID == cfg.Primary.ID {
			return nil, fmt.Errorf("%w: fallback key cannot have same ID as primary", ErrInvalidConfig)
		}
		if _, exists := fallback[key.ID]; exists {
			return nil, fmt.Errorf("%w: duplicate fallback key ID: %s", ErrInvalidConfig, key.ID)
		}
		fallback[key.ID] = key
	}
	return &Manager{
		cipher:   cipherType,
		primary:  cfg.Primary,
		fallback: fallback,
		cache:    make(map[string]cipher.AEAD),
	}, nil
}

func (m *Manager) Encrypt(ctx context.Context, plaintext []byte, opts EncryptOptions) (Ciphertext, error) {
	if err := ctx.Err(); err != nil {
		return Ciphertext{}, err
	}
	key, err := m.selectKey(opts.KeyID)
	if err != nil {
		return Ciphertext{}, err
	}
	aead, err := m.aeadFor(key)
	if err != nil {
		return Ciphertext{}, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return Ciphertext{}, err
	}
	ciphertext := aead.Seal(nil, nonce, plaintext, opts.AssociatedData)
	return Ciphertext{
		Cipher: m.cipher,
		KeyID:  key.ID,
		Nonce:  nonce,
		Data:   ciphertext,
	}, nil
}

func (m *Manager) EncryptToString(ctx context.Context, plaintext []byte, opts EncryptOptions) (string, error) {
	ct, err := m.Encrypt(ctx, plaintext, opts)
	if err != nil {
		return "", err
	}
	return ct.Encode(), nil
}

func (m *Manager) Decrypt(ctx context.Context, ct Ciphertext, opts DecryptOptions) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if ct.Cipher != m.cipher {
		return nil, fmt.Errorf("%w: cipher mismatch (got %s, expected %s)", ErrInvalidCiphertext, ct.Cipher, m.cipher)
	}
	key, err := m.lookupKey(ct.KeyID)
	if err != nil {
		return nil, err
	}
	aead, err := m.aeadFor(key)
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, ct.Nonce, ct.Data, opts.AssociatedData)
	if err != nil {
		return nil, fmt.Errorf("encryption: decrypt failed: %w", err)
	}
	return plaintext, nil
}

func (m *Manager) DecryptString(ctx context.Context, encoded string, opts DecryptOptions) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ct, err := DecodeCiphertext(encoded)
	if err != nil {
		return nil, err
	}
	return m.Decrypt(ctx, ct, opts)
}

func (m *Manager) Rotate(newPrimary Key) error {
	if err := validateKey(newPrimary, m.cipher); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fallback == nil {
		m.fallback = make(map[string]Key)
	}
	m.fallback[m.primary.ID] = m.primary
	m.primary = newPrimary
	delete(m.fallback, newPrimary.ID)
	delete(m.cache, newPrimary.ID)
	return nil
}

func (m *Manager) AddFallback(key Key) error {
	if err := validateKey(key, m.cipher); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if key.ID == m.primary.ID {
		m.primary = key
		if m.fallback != nil {
			delete(m.fallback, key.ID)
		}
		delete(m.cache, key.ID)
		return nil
	}
	if m.fallback == nil {
		m.fallback = make(map[string]Key)
	}
	m.fallback[key.ID] = key
	delete(m.cache, key.ID)
	return nil
}

func (m *Manager) RemoveFallback(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if keyID == m.primary.ID {
		return fmt.Errorf("%w: cannot remove primary key", ErrInvalidConfig)
	}
	if m.fallback == nil {
		return ErrKeyNotFound
	}
	if _, exists := m.fallback[keyID]; !exists {
		return ErrKeyNotFound
	}
	delete(m.fallback, keyID)
	delete(m.cache, keyID)
	return nil
}

func (m *Manager) selectKey(requested string) (Key, error) {
	if requested != "" {
		key, err := m.lookupKey(requested)
		if err != nil {
			return Key{}, err
		}
		return key, nil
	}
	m.mu.RLock()
	primary := m.primary
	m.mu.RUnlock()
	return primary, nil
}

func (m *Manager) lookupKey(id string) (Key, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if id == "" || id == m.primary.ID {
		return m.primary, nil
	}
	key, ok := m.fallback[id]
	if !ok {
		return Key{}, fmt.Errorf("%w: %s", ErrKeyNotFound, id)
	}
	return key, nil
}

func (m *Manager) aeadFor(key Key) (cipher.AEAD, error) {
	m.mu.RLock()
	aead, ok := m.cache[key.ID]
	m.mu.RUnlock()
	if ok {
		return aead, nil
	}
	created, err := newAEAD(m.cipher, key.Secret)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	if m.cache == nil {
		m.cache = make(map[string]cipher.AEAD)
	}
	m.cache[key.ID] = created
	m.mu.Unlock()
	return created, nil
}

func newAEAD(cipherType Cipher, secret []byte) (cipher.AEAD, error) {
	switch cipherType {
	case CipherAESGCM:
		block, err := aes.NewCipher(secret)
		if err != nil {
			return nil, fmt.Errorf("encryption: aes setup failed: %w", err)
		}
		return cipher.NewGCM(block)
	case CipherChaCha20Poly1305:
		aead, err := chacha20poly1305.New(secret)
		if err != nil {
			return nil, fmt.Errorf("encryption: chacha setup failed: %w", err)
		}
		return aead, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidCipher, cipherType)
	}
}

func validateKey(key Key, cipherType Cipher) error {
	if key.ID == "" {
		return fmt.Errorf("%w: key id required", ErrInvalidConfig)
	}
	if len(key.Secret) == 0 {
		return fmt.Errorf("%w: secret required", ErrInvalidConfig)
	}
	switch cipherType {
	case CipherAESGCM:
		if l := len(key.Secret); l != 16 && l != 24 && l != 32 {
			return fmt.Errorf("%w: aes secret must be 16, 24, or 32 bytes", ErrInvalidConfig)
		}
	case CipherChaCha20Poly1305:
		if len(key.Secret) != chacha20poly1305.KeySize {
			return fmt.Errorf("%w: chacha secret must be %d bytes", ErrInvalidConfig, chacha20poly1305.KeySize)
		}
	default:
		return fmt.Errorf("%w: %s", ErrInvalidCipher, cipherType)
	}
	return nil
}

func (c Ciphertext) Encode() string {
	parts := []string{
		string(c.Cipher),
		c.KeyID,
		base64.RawStdEncoding.EncodeToString(c.Nonce),
		base64.RawStdEncoding.EncodeToString(c.Data),
	}
	return strings.Join(parts, "$")
}

func DecodeCiphertext(encoded string) (Ciphertext, error) {
	if encoded == "" {
		return Ciphertext{}, ErrInvalidCiphertext
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 {
		return Ciphertext{}, fmt.Errorf("%w: expected 4 parts, got %d", ErrInvalidCiphertext, len(parts))
	}
	if parts[0] == "" || parts[1] == "" {
		return Ciphertext{}, fmt.Errorf("%w: cipher or keyID cannot be empty", ErrInvalidCiphertext)
	}
	nonce, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return Ciphertext{}, fmt.Errorf("%w: invalid nonce encoding: %w", ErrInvalidCiphertext, err)
	}
	data, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return Ciphertext{}, fmt.Errorf("%w: invalid data encoding: %w", ErrInvalidCiphertext, err)
	}
	return Ciphertext{
		Cipher: Cipher(parts[0]),
		KeyID:  parts[1],
		Nonce:  nonce,
		Data:   data,
	}, nil
}
