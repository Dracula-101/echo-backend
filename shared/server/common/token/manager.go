package token

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Manager struct {
	keySet     KeySet
	issuer     string
	audience   []string
	accessTTL  time.Duration
	refreshTTL time.Duration
	leeway     time.Duration
	clock      func() time.Time
	parser     *jwt.Parser
}

type Config struct {
	KeySet          KeySet
	Issuer          string
	Audience        []string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Clock           func() time.Time
	Leeway          time.Duration
	ParserOptions   []jwt.ParserOption
}

type IssueOptions struct {
	Metadata  map[string]any
	Audience  []string
	ExpiresIn time.Duration
	NotBefore time.Time
}

type SignedToken struct {
	Token  string
	Claims *Claims
}

type TokenPair struct {
	AccessToken  SignedToken
	RefreshToken SignedToken
}

func NewManager(cfg Config) (*Manager, error) {
	if cfg.KeySet == nil {
		return nil, fmt.Errorf("%w: key set required", ErrInvalidConfig)
	}
	if cfg.AccessTokenTTL <= 0 {
		return nil, fmt.Errorf("%w: access ttl required", ErrInvalidConfig)
	}
	if cfg.RefreshTokenTTL <= 0 {
		return nil, fmt.Errorf("%w: refresh ttl required", ErrInvalidConfig)
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	parser := jwt.NewParser(cfg.ParserOptions...)
	manager := &Manager{
		keySet:     cfg.KeySet,
		issuer:     cfg.Issuer,
		audience:   cfg.Audience,
		accessTTL:  cfg.AccessTokenTTL,
		refreshTTL: cfg.RefreshTokenTTL,
		leeway:     cfg.Leeway,
		clock:      clock,
		parser:     parser,
	}
	if manager.leeway < 0 {
		manager.leeway = 0
	}
	return manager, nil
}

func (m *Manager) IssueAccessToken(ctx context.Context, subject string, opts IssueOptions) (SignedToken, error) {
	claims := m.buildClaims(subject, TokenTypeAccess, opts, m.accessTTL)
	return m.sign(ctx, claims)
}

func (m *Manager) IssueRefreshToken(ctx context.Context, subject string, opts IssueOptions) (SignedToken, error) {
	claims := m.buildClaims(subject, TokenTypeRefresh, opts, m.refreshTTL)
	return m.sign(ctx, claims)
}

func (m *Manager) IssuePair(ctx context.Context, subject string, opts IssueOptions) (TokenPair, error) {
	access, err := m.IssueAccessToken(ctx, subject, opts)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := m.IssueRefreshToken(ctx, subject, opts)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (m *Manager) Parse(ctx context.Context, tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := m.parser.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		kid, _ := t.Header["kid"].(string)
		key, err := m.keySet.Lookup(ctx, kid)
		if err != nil {
			return nil, err
		}
		claims.IssuedKey = key.ID
		method := jwt.GetSigningMethod(key.Algorithm)
		if method == nil {
			return nil, fmt.Errorf("token: unsupported signing method %s", key.Algorithm)
		}
		if t.Method != method {
			return nil, errors.New("token: signing method mismatch")
		}
		return key.Secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) || errors.Is(err, jwt.ErrTokenInvalidClaims) {
			return nil, ErrInvalidToken
		}
		if errors.Is(err, ErrKeyNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("token: parse failed: %w", err)
	}
	claims.Capture(claims)
	return claims, nil
}

func (m *Manager) Validate(ctx context.Context, tokenString string, expected TokenType) (*Claims, error) {
	claims, err := m.Parse(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	if err := claims.Validate(m.clock(), m.leeway, expected, m.audience); err != nil {
		return nil, err
	}
	return claims, nil
}

func (m *Manager) buildClaims(subject string, tokenType TokenType, opts IssueOptions, ttl time.Duration) *Claims {
	now := m.clock()
	expires := now.Add(ttl)
	audience := opts.Audience
	if len(audience) == 0 {
		audience = m.audience
	}
	metadata := make(map[string]any)
	for k, v := range opts.Metadata {
		metadata[k] = v
	}
	return &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   subject,
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings(audience),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(nonZero(opts.NotBefore, now)),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
		TokenType: tokenType,
		Metadata:  metadata,
	}
}

func (m *Manager) sign(ctx context.Context, claims *Claims) (SignedToken, error) {
	key, err := m.keySet.Current(ctx)
	if err != nil {
		return SignedToken{}, err
	}
	method := jwt.GetSigningMethod(key.Algorithm)
	if method == nil {
		return SignedToken{}, fmt.Errorf("token: unsupported signing method %s", key.Algorithm)
	}
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = key.ID
	signed, err := token.SignedString(key.Secret)
	if err != nil {
		return SignedToken{}, fmt.Errorf("token: sign failed: %w", err)
	}
	claims.IssuedKey = key.ID
	return SignedToken{Token: signed, Claims: claims}, nil
}

func nonZero(t time.Time, fallback time.Time) time.Time {
	if t.IsZero() {
		return fallback
	}
	return t
}
