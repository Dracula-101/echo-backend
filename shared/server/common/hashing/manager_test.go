package hashing

import (
	"context"
	"testing"
)

func TestNewService(t *testing.T) {
	svc, err := NewService(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if svc == nil {
		t.Error("expected service, got nil")
	}
	if svc.manager == nil {
		t.Error("expected manager, got nil")
	}
}

func TestHashPassword(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	result, err := svc.HashPassword(ctx, "password123")
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Error("expected result, got nil")
	}
	if result.Encoded == "" {
		t.Error("encoded is empty")
	}
	if result.Algorithm != AlgorithmArgon2id {
		t.Errorf("expected argon2id, got %s", result.Algorithm)
	}
}

func TestHashPasswordContextCancelled(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.HashPassword(ctx, "password123")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestHashPasswordWithAlgorithm(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	t.Run("argon2id", func(t *testing.T) {
		result, err := svc.HashPasswordWithAlgorithm(ctx, AlgorithmArgon2id, "password123")
		if err != nil {
			t.Fatal(err)
		}
		if result.Algorithm != AlgorithmArgon2id {
			t.Errorf("expected argon2id, got %s", result.Algorithm)
		}
	})

	t.Run("bcrypt", func(t *testing.T) {
		result, err := svc.HashPasswordWithAlgorithm(ctx, AlgorithmBcrypt, "password123")
		if err != nil {
			t.Fatal(err)
		}
		if result.Algorithm != AlgorithmBcrypt {
			t.Errorf("expected bcrypt, got %s", result.Algorithm)
		}
	})

	t.Run("scrypt", func(t *testing.T) {
		result, err := svc.HashPasswordWithAlgorithm(ctx, AlgorithmScrypt, "password123")
		if err != nil {
			t.Fatal(err)
		}
		if result.Algorithm != AlgorithmScrypt {
			t.Errorf("expected scrypt, got %s", result.Algorithm)
		}
	})
}

func TestHashPasswordWithAlgorithmContextCancelled(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.HashPasswordWithAlgorithm(ctx, AlgorithmBcrypt, "password123")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestVerifyPassword(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	result, _ := svc.HashPassword(ctx, "password123")

	t.Run("correct password", func(t *testing.T) {
		ok, algo, err := svc.VerifyPassword(ctx, "password123", result.Encoded)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected verification to succeed")
		}
		if algo != AlgorithmArgon2id {
			t.Errorf("expected argon2id, got %s", algo)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		ok, _, err := svc.VerifyPassword(ctx, "wrongpassword", result.Encoded)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected verification to fail")
		}
	})
}

func TestVerifyPasswordContextCancelled(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := svc.VerifyPassword(ctx, "password123", "encoded")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestNeedsRehash(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	result, _ := svc.HashPassword(ctx, "password123")

	needsRehash, err := svc.NeedsRehash(result.Encoded)
	if err != nil {
		t.Fatal(err)
	}
	if needsRehash {
		t.Error("should not need rehash with same config")
	}
}

func TestNeedsRehashDifferentConfig(t *testing.T) {
	svc1, _ := NewService(Config{
		Argon2: Argon2Config{
			Time:       3,
			Memory:     64 * 1024,
			Threads:    2,
			KeyLength:  32,
			SaltLength: 16,
		},
	})
	ctx := context.Background()
	result, _ := svc1.HashPassword(ctx, "password123")

	svc2, _ := NewService(Config{
		Argon2: Argon2Config{
			Time:       5,
			Memory:     64 * 1024,
			Threads:    2,
			KeyLength:  32,
			SaltLength: 16,
		},
	})

	needsRehash, err := svc2.NeedsRehash(result.Encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !needsRehash {
		t.Error("should need rehash with different config")
	}
}

func TestVerifyAndRehash(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx := context.Background()

	result, _ := svc.HashPassword(ctx, "password123")

	t.Run("no rehash needed", func(t *testing.T) {
		ok, rehashResult, err := svc.VerifyAndRehash(ctx, "password123", result.Encoded)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected verification to succeed")
		}
		if rehashResult != nil {
			t.Error("expected no rehash result")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		ok, rehashResult, err := svc.VerifyAndRehash(ctx, "wrongpassword", result.Encoded)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected verification to fail")
		}
		if rehashResult != nil {
			t.Error("expected no rehash result")
		}
	})
}

func TestVerifyAndRehashWithRehash(t *testing.T) {
	svc1, _ := NewService(Config{
		Argon2: Argon2Config{
			Time:       3,
			Memory:     64 * 1024,
			Threads:    2,
			KeyLength:  32,
			SaltLength: 16,
		},
	})
	ctx := context.Background()
	result, _ := svc1.HashPassword(ctx, "password123")

	svc2, _ := NewService(Config{
		Argon2: Argon2Config{
			Time:       5,
			Memory:     64 * 1024,
			Threads:    2,
			KeyLength:  32,
			SaltLength: 16,
		},
	})

	ok, rehashResult, err := svc2.VerifyAndRehash(ctx, "password123", result.Encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected verification to succeed")
	}
	if rehashResult == nil {
		t.Fatal("expected rehash result")
	}
	if rehashResult.Encoded == "" {
		t.Error("rehash encoded is empty")
	}
	if rehashResult.Params["previous"] != string(AlgorithmArgon2id) {
		t.Errorf("expected previous=argon2id, got %s", rehashResult.Params["previous"])
	}
}

func TestVerifyAndRehashContextCancelled(t *testing.T) {
	svc, _ := NewService(Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := svc.VerifyAndRehash(ctx, "password123", "encoded")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestIntegrationAllAlgorithms(t *testing.T) {
	algorithms := []Algorithm{AlgorithmArgon2id, AlgorithmBcrypt, AlgorithmScrypt}
	svc, _ := NewService(Config{})
	ctx := context.Background()

	for _, algo := range algorithms {
		t.Run(string(algo), func(t *testing.T) {
			result, err := svc.HashPasswordWithAlgorithm(ctx, algo, "password123")
			if err != nil {
				t.Fatal(err)
			}

			ok, verifiedAlgo, err := svc.VerifyPassword(ctx, "password123", result.Encoded)
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Error("verification failed")
			}
			if verifiedAlgo != algo {
				t.Errorf("expected %s, got %s", algo, verifiedAlgo)
			}

			needsRehash, err := svc.NeedsRehash(result.Encoded)
			if err != nil {
				t.Fatal(err)
			}
			if needsRehash {
				t.Error("should not need rehash")
			}
		})
	}
}
