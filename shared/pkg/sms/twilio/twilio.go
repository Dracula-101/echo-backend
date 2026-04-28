package twilio

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"shared/pkg/logger"
	"strings"
)

type Client struct {
	cfg  Config
	http *http.Client
	log  logger.Logger
}

func New(cfg Config, log logger.Logger) *Client {
	return &Client{cfg: cfg, http: &http.Client{}, log: log}
}

func (c *Client) Send(ctx context.Context, to, body string) error {
	endpoint := fmt.Sprintf(
		"https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json",
		c.cfg.AccountSID,
	)
	c.log.Info("Sending SMS via Twilio",
		logger.String("to", to),
		logger.String("from", c.cfg.FromNumber),
		logger.String("endpoint", endpoint),
		logger.String("service", "twilio"),
	)

	form := url.Values{}
	form.Set("To", to)
	form.Set("From", c.cfg.FromNumber)
	form.Set("Body", body)
	form.Set("MessagingServiceSid", c.cfg.MessagingServiceSID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.cfg.AccountSID + ":" + c.cfg.AuthToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		c.log.Error("Failed to send SMS via Twilio",
			logger.String("to", to),
			logger.String("from", c.cfg.FromNumber),
			logger.String("endpoint", endpoint),
			logger.String("service", "twilio"),
			logger.Error(err),
		)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.log.Error("Twilio API returned error",
			logger.String("to", to),
			logger.String("from", c.cfg.FromNumber),
			logger.String("endpoint", endpoint),
			logger.String("service", "twilio"),
			logger.Int("status_code", resp.StatusCode),
			logger.String("response_status", resp.Status),
			logger.String("response_status_text", http.StatusText(resp.StatusCode)),
			logger.String("response_body", fmt.Sprintf("Status: %s, StatusCode: %d", resp.Status, resp.StatusCode)),
			logger.String("error_message", fmt.Sprintf("Twilio API error: %s", http.StatusText(resp.StatusCode))),
			logger.String("suggestion", "Check Twilio account SID, auth token, and messaging service SID. Also verify that the 'From' number is valid and associated with your Twilio account."),
		)
		return fmt.Errorf("twilio: request failed with status %d", resp.StatusCode)
	}

	c.log.Info("SMS sent successfully via Twilio",
		logger.String("to", to),
		logger.String("from", c.cfg.FromNumber),
		logger.String("endpoint", endpoint),
		logger.String("service", "twilio"),
		logger.String("response_status", resp.Status),
		logger.Int("response_status_code", resp.StatusCode),
		logger.String("response_status_text", http.StatusText(resp.StatusCode)),
		logger.String("response_body", fmt.Sprintf("Status: %s, StatusCode: %d", resp.Status, resp.StatusCode)),
	)

	return nil
}
