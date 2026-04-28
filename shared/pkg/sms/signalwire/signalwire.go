package signalwire

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{}}
}

func (c *Client) Send(ctx context.Context, to, body string) error {
	endpoint := fmt.Sprintf(
		"https://%s/api/laml/2010-04-01/Accounts/%s/Messages.json",
		c.cfg.SpaceURL, c.cfg.ProjectID,
	)

	form := url.Values{}
	form.Set("To", to)
	form.Set("From", c.cfg.FromNumber)
	form.Set("Body", body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.cfg.ProjectID + ":" + c.cfg.APIToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("signalwire: request failed with status %d", resp.StatusCode)
	}

	return nil
}
