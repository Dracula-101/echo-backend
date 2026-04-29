package firebase

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type Client struct {
	auth *auth.Client
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(cfg.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("firebase: failed to init app: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase: failed to init auth client: %w", err)
	}

	return &Client{auth: authClient}, nil
}

func (c *Client) VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	token, err := c.auth.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("firebase: invalid token: %w", err)
	}
	return token, nil
}
