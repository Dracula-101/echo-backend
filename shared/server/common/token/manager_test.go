package token

import (
	"context"
	"testing"
	"time"
)

func TestManager_IssueParseValidate(t *testing.T) {
	secret := []byte("test-secret-key-which-is-long-enough")
	ks, err := NewStaticKeySet(secret)
	if err != nil {
		t.Fatalf("failed to create static key set: %v", err)
	}

	cfg := Config{
		KeySet:          ks,
		Issuer:          "test-issuer",
		Audience:        []string{"test_users"},
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: time.Hour * 24,
	}
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()
	pair, err := mgr.IssuePair(ctx, "user-123", IssueOptions{})
	if err != nil {
		t.Fatalf("issue pair failed: %v", err)
	}
	if pair.AccessToken.Token == "" || pair.RefreshToken.Token == "" {
		t.Fatalf("expected non-empty tokens")
	}

	// Parse access token
	claims, err := mgr.Parse(ctx, pair.AccessToken.Token)
	if err != nil {
		t.Fatalf("parse access token failed: %v", err)
	}
	if claims.Subject != "user-123" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}

	// Validate access token (expect access)
	vclaims, err := mgr.Validate(ctx, pair.AccessToken.Token, TokenTypeAccess)
	if err != nil {
		t.Fatalf("validate access token failed: %v", err)
	}
	if vclaims.TokenType != TokenTypeAccess {
		t.Fatalf("expected access token type, got %s", vclaims.TokenType)
	}

	// Validate refresh token
	rclaims, err := mgr.Validate(ctx, pair.RefreshToken.Token, TokenTypeRefresh)
	if err != nil {
		t.Fatalf("validate refresh token failed: %v", err)
	}
	if rclaims.TokenType != TokenTypeRefresh {
		t.Fatalf("expected refresh token type, got %s", rclaims.TokenType)
	}
}
