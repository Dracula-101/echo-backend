package token

import "context"

type JWTTokenService struct {
	manager *Manager
}

func NewJWTTokenService(cfg Config) (*JWTTokenService, error) {
	mgr, err := NewManager(cfg)
	if err != nil {
		return nil, err
	}
	return &JWTTokenService{manager: mgr}, nil
}

func (s *JWTTokenService) IssueAccessToken(ctx context.Context, subject string, opts IssueOptions) (SignedToken, error) {
	if err := ctx.Err(); err != nil {
		return SignedToken{}, err
	}
	return s.manager.IssueAccessToken(ctx, subject, opts)
}

func (s *JWTTokenService) IssueRefreshToken(ctx context.Context, subject string, opts IssueOptions) (SignedToken, error) {
	if err := ctx.Err(); err != nil {
		return SignedToken{}, err
	}
	return s.manager.IssueRefreshToken(ctx, subject, opts)
}

func (s *JWTTokenService) IssuePair(ctx context.Context, subject string, opts IssueOptions) (TokenPair, error) {
	if err := ctx.Err(); err != nil {
		return TokenPair{}, err
	}
	return s.manager.IssuePair(ctx, subject, opts)
}

func (s *JWTTokenService) Parse(ctx context.Context, tokenString string) (*Claims, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.manager.Parse(ctx, tokenString)
}

func (s *JWTTokenService) Validate(ctx context.Context, tokenString string, expected TokenType) (*Claims, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.manager.Validate(ctx, tokenString, expected)
}
