package encryption

import (
	"context"
	"crypto/rand"
	"testing"
)

func TestNewManager(t *testing.T) {
	key := Key{ID: "key1", Secret: make([]byte, 32)}
	rand.Read(key.Secret)

	mgr, err := NewManager(Config{Primary: key})
	if err != nil {
		t.Fatal(err)
	}
	if mgr == nil {
		t.Error("expected manager, got nil")
	}
	if mgr.cipher != CipherAESGCM {
		t.Errorf("expected default cipher aes-gcm, got %s", mgr.cipher)
	}
}

func TestNewManagerWithCipher(t *testing.T) {
	key := Key{ID: "key1", Secret: make([]byte, 32)}
	rand.Read(key.Secret)

	mgr, err := NewManager(Config{
		Primary: key,
		Cipher:  CipherChaCha20Poly1305,
	})
	if err != nil {
		t.Fatal(err)
	}
	if mgr.cipher != CipherChaCha20Poly1305 {
		t.Errorf("expected chacha20poly1305, got %s", mgr.cipher)
	}
}

func TestNewManagerInvalidKey(t *testing.T) {
	_, err := NewManager(Config{
		Primary: Key{ID: "", Secret: make([]byte, 32)},
	})
	if err == nil {
		t.Error("expected error for empty key ID")
	}

	_, err = NewManager(Config{
		Primary: Key{ID: "key1", Secret: nil},
	})
	if err == nil {
		t.Error("expected error for nil secret")
	}

	_, err = NewManager(Config{
		Primary: Key{ID: "key1", Secret: make([]byte, 10)},
	})
	if err == nil {
		t.Error("expected error for invalid secret length")
	}
}

func TestEncrypt(t *testing.T) {
	key := Key{ID: "key1", Secret: make([]byte, 32)}
	rand.Read(key.Secret)
	mgr, _ := NewManager(Config{Primary: key})
	ctx := context.Background()

	plaintext := []byte("hello world")
	ct, err := mgr.Encrypt(ctx, plaintext, EncryptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if ct.Cipher != CipherAESGCM {
		t.Errorf("expected aes-gcm, got %s", ct.Cipher)
	}
	if ct.KeyID != "key1" {
		t.Errorf("expected key1, got %s", ct.KeyID)
	}
	if len(ct.Nonce) == 0 {
		t.Error("nonce is empty")
	}
	if len(ct.Data) == 0 {
		t.Error("data is empty")
	}
}

func TestEncryptWithAssociatedData(t *testing.T) {
	key := Key{ID: "key1", Secret: make([]byte, 32)}
	rand.Read(key.Secret)
	mgr, _ := NewManager(Config{Primary: key})
	ctx := context.Background()

	plaintext := []byte("hello world")
	ad := []byte("metadata")
	ct, err := mgr.Encrypt(ctx, plaintext, EncryptOptions{AssociatedData: ad})
	if err != nil {
		t.Fatal(err)
	}
	if len(ct.Data) == 0 {
		t.Error("data is empty")
	}
}

func TestEncryptContextCancelled(t *testing.T) {
	key := Key{ID: "key1", Secret: make([]byte, 32)}
	rand.Read(key.Secret)
	mgr, _ := NewManager(Config{Primary: key})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mgr.Encrypt(ctx, []byte("test"), EncryptOptions{})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestEncryptToString(t *testing.T) {
	key := Key{ID: "key1", Secret: make([]byte, 32)}
	rand.Read(key.Secret)
	mgr, _ := NewManager(Config{Primary: key})
	ctx := context.Background()

	encoded, err := mgr.EncryptToString(ctx, []byte("hello"), EncryptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if encoded == "" {
		t.Error("encoded string is empty")
	}
	if len(encoded) < 10 {
		t.Error("encoded string too short")
	}
}

func TestDecrypt(t *testing.T) {
	key := Key{ID: "key1", Secret: make([]byte, 32)}
	rand.Read(key.Secret)
	mgr, _ := NewManager(Config{Primary: key})
	ctx := context.Background()

	plaintext := []byte("hello world")
	ct, _ := mgr.Encrypt(ctx, plaintext, EncryptOptions{})

	decrypted, err := mgr.Decrypt(ctx, ct, DecryptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("expected %s, got %s", plaintext, decrypted)
	}
}

func TestDecryptWithAssociatedData(t *testing.T) {
	key := Key{ID: "key1", Secret: make([]byte, 32)}
	rand.Read(key.Secret)
	mgr, _ := NewManager(Config{Primary: key})
	ctx := context.Background()

	plaintext := []byte("hello world")
	ad := []byte("metadata")
	ct, _ := mgr.Encrypt(ctx, plaintext, EncryptOptions{AssociatedData: ad})

	decrypted, err := mgr.Decrypt(ctx, ct, DecryptOptions{AssociatedData: ad})
	if err != nil {
		t.Fatal(err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("expected %s, got %s", plaintext, decrypted)
	}

	_, err = mgr.Decrypt(ctx, ct, DecryptOptions{AssociatedData: []byte("wrong")})
	if err == nil {
		t.Error("expected error for wrong associated data")
	}
}

func TestDecryptContextCancelled(t *testing.T) {
	key := Key{ID: "key1", Secret: make([]byte, 32)}
	rand.Read(key.Secret)
	mgr, _ := NewManager(Config{Primary: key})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mgr.Decrypt(ctx, Ciphertext{}, DecryptOptions{})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestDecryptString(t *testing.T) {
	key := Key{ID: "key1", Secret: make([]byte, 32)}
	rand.Read(key.Secret)
	mgr, _ := NewManager(Config{Primary: key})
	ctx := context.Background()

	plaintext := []byte("hello world")
	encoded, _ := mgr.EncryptToString(ctx, plaintext, EncryptOptions{})

	decrypted, err := mgr.DecryptString(ctx, encoded, DecryptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("expected %s, got %s", plaintext, decrypted)
	}
}

func TestRotate(t *testing.T) {
	key1 := Key{ID: "key1", Secret: make([]byte, 32)}
	key2 := Key{ID: "key2", Secret: make([]byte, 32)}
	rand.Read(key1.Secret)
	rand.Read(key2.Secret)

	mgr, _ := NewManager(Config{Primary: key1})
	ctx := context.Background()

	ct, _ := mgr.Encrypt(ctx, []byte("test"), EncryptOptions{})
	if ct.KeyID != "key1" {
		t.Error("expected key1")
	}

	err := mgr.Rotate(key2)
	if err != nil {
		t.Fatal(err)
	}

	ct2, _ := mgr.Encrypt(ctx, []byte("test2"), EncryptOptions{})
	if ct2.KeyID != "key2" {
		t.Error("expected key2 after rotation")
	}

	decrypted, err := mgr.Decrypt(ctx, ct, DecryptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(decrypted) != "test" {
		t.Error("failed to decrypt with old key after rotation")
	}
}

func TestAddFallback(t *testing.T) {
	key1 := Key{ID: "key1", Secret: make([]byte, 32)}
	key2 := Key{ID: "key2", Secret: make([]byte, 32)}
	rand.Read(key1.Secret)
	rand.Read(key2.Secret)

	mgr, _ := NewManager(Config{Primary: key1})
	ctx := context.Background()

	err := mgr.AddFallback(key2)
	if err != nil {
		t.Fatal(err)
	}

	ct, _ := mgr.Encrypt(ctx, []byte("test"), EncryptOptions{KeyID: "key2"})
	if ct.KeyID != "key2" {
		t.Error("expected key2")
	}

	decrypted, err := mgr.Decrypt(ctx, ct, DecryptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(decrypted) != "test" {
		t.Error("failed to decrypt with fallback key")
	}
}

func TestAddFallbackUpdatesPrimary(t *testing.T) {
	key1 := Key{ID: "key1", Secret: make([]byte, 32)}
	key1Updated := Key{ID: "key1", Secret: make([]byte, 32)}
	rand.Read(key1.Secret)
	rand.Read(key1Updated.Secret)

	mgr, _ := NewManager(Config{Primary: key1})

	err := mgr.AddFallback(key1Updated)
	if err != nil {
		t.Fatal(err)
	}

	if string(mgr.primary.Secret) != string(key1Updated.Secret) {
		t.Error("primary key was not updated")
	}
}

func TestSelectKey(t *testing.T) {
	key1 := Key{ID: "key1", Secret: make([]byte, 32)}
	key2 := Key{ID: "key2", Secret: make([]byte, 32)}
	rand.Read(key1.Secret)
	rand.Read(key2.Secret)

	mgr, _ := NewManager(Config{
		Primary:  key1,
		Fallback: []Key{key2},
	})

	t.Run("default to primary", func(t *testing.T) {
		key, err := mgr.selectKey("")
		if err != nil {
			t.Fatal(err)
		}
		if key.ID != "key1" {
			t.Errorf("expected key1, got %s", key.ID)
		}
	})

	t.Run("select fallback", func(t *testing.T) {
		key, err := mgr.selectKey("key2")
		if err != nil {
			t.Fatal(err)
		}
		if key.ID != "key2" {
			t.Errorf("expected key2, got %s", key.ID)
		}
	})

	t.Run("key not found", func(t *testing.T) {
		_, err := mgr.selectKey("key3")
		if err == nil {
			t.Error("expected error for non-existent key")
		}
	})
}

func TestLookupKey(t *testing.T) {
	key1 := Key{ID: "key1", Secret: make([]byte, 32)}
	key2 := Key{ID: "key2", Secret: make([]byte, 32)}
	rand.Read(key1.Secret)
	rand.Read(key2.Secret)

	mgr, _ := NewManager(Config{
		Primary:  key1,
		Fallback: []Key{key2},
	})

	t.Run("lookup primary", func(t *testing.T) {
		key, err := mgr.lookupKey("key1")
		if err != nil {
			t.Fatal(err)
		}
		if key.ID != "key1" {
			t.Error("expected key1")
		}
	})

	t.Run("lookup empty returns primary", func(t *testing.T) {
		key, err := mgr.lookupKey("")
		if err != nil {
			t.Fatal(err)
		}
		if key.ID != "key1" {
			t.Error("expected key1")
		}
	})

	t.Run("lookup fallback", func(t *testing.T) {
		key, err := mgr.lookupKey("key2")
		if err != nil {
			t.Fatal(err)
		}
		if key.ID != "key2" {
			t.Error("expected key2")
		}
	})

	t.Run("key not found", func(t *testing.T) {
		_, err := mgr.lookupKey("key3")
		if err == nil {
			t.Error("expected error for non-existent key")
		}
	})
}

func TestValidateKey(t *testing.T) {
	t.Run("valid aes key", func(t *testing.T) {
		key := Key{ID: "key1", Secret: make([]byte, 32)}
		err := validateKey(key, CipherAESGCM)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("valid chacha key", func(t *testing.T) {
		key := Key{ID: "key1", Secret: make([]byte, 32)}
		err := validateKey(key, CipherChaCha20Poly1305)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("empty id", func(t *testing.T) {
		key := Key{ID: "", Secret: make([]byte, 32)}
		err := validateKey(key, CipherAESGCM)
		if err == nil {
			t.Error("expected error for empty id")
		}
	})

	t.Run("empty secret", func(t *testing.T) {
		key := Key{ID: "key1", Secret: nil}
		err := validateKey(key, CipherAESGCM)
		if err == nil {
			t.Error("expected error for empty secret")
		}
	})

	t.Run("invalid aes length", func(t *testing.T) {
		key := Key{ID: "key1", Secret: make([]byte, 10)}
		err := validateKey(key, CipherAESGCM)
		if err == nil {
			t.Error("expected error for invalid aes length")
		}
	})

	t.Run("invalid chacha length", func(t *testing.T) {
		key := Key{ID: "key1", Secret: make([]byte, 16)}
		err := validateKey(key, CipherChaCha20Poly1305)
		if err == nil {
			t.Error("expected error for invalid chacha length")
		}
	})
}

func TestCiphertextEncode(t *testing.T) {
	ct := Ciphertext{
		Cipher: CipherAESGCM,
		KeyID:  "key1",
		Nonce:  []byte("nonce123"),
		Data:   []byte("data456"),
	}

	encoded := ct.Encode()
	if encoded == "" {
		t.Error("encoded is empty")
	}
	if len(encoded) < 10 {
		t.Error("encoded too short")
	}
}

func TestDecodeCiphertext(t *testing.T) {
	original := Ciphertext{
		Cipher: CipherAESGCM,
		KeyID:  "key1",
		Nonce:  []byte("nonce123"),
		Data:   []byte("data456"),
	}

	encoded := original.Encode()
	decoded, err := DecodeCiphertext(encoded)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Cipher != original.Cipher {
		t.Error("cipher mismatch")
	}
	if decoded.KeyID != original.KeyID {
		t.Error("keyID mismatch")
	}
	if string(decoded.Nonce) != string(original.Nonce) {
		t.Error("nonce mismatch")
	}
	if string(decoded.Data) != string(original.Data) {
		t.Error("data mismatch")
	}
}

func TestDecodeCiphertextInvalid(t *testing.T) {
	_, err := DecodeCiphertext("invalid")
	if err == nil {
		t.Error("expected error for invalid format")
	}

	_, err = DecodeCiphertext("a$b$c")
	if err == nil {
		t.Error("expected error for too few parts")
	}
}

func TestIntegrationAllCiphers(t *testing.T) {
	ciphers := []Cipher{CipherAESGCM, CipherChaCha20Poly1305}

	for _, cipher := range ciphers {
		t.Run(string(cipher), func(t *testing.T) {
			key := Key{ID: "key1", Secret: make([]byte, 32)}
			rand.Read(key.Secret)

			mgr, err := NewManager(Config{
				Primary: key,
				Cipher:  cipher,
			})
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			plaintext := []byte("secret message")

			ct, err := mgr.Encrypt(ctx, plaintext, EncryptOptions{})
			if err != nil {
				t.Fatal(err)
			}

			decrypted, err := mgr.Decrypt(ctx, ct, DecryptOptions{})
			if err != nil {
				t.Fatal(err)
			}

			if string(decrypted) != string(plaintext) {
				t.Errorf("expected %s, got %s", plaintext, decrypted)
			}
		})
	}
}
