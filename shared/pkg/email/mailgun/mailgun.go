package mailgun

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	mailgun "github.com/mailgun/mailgun-go/v5"
)

type MailgunConfig struct {
	Domain  string
	APIKey  string
	BaseURL string // optional, for EU region
}

type MailgunService struct {
	mg     *mailgun.Client
	domain string
}

// ---------- INIT ----------

func New(config MailgunConfig) *MailgunService {
	mg := mailgun.NewMailgun(config.APIKey)

	// Optional: EU region
	if config.BaseURL != "" {
		mg.SetAPIBase(config.BaseURL)
	}

	return &MailgunService{
		mg:     mg,
		domain: config.Domain,
	}
}

// ---------- TEMPLATE ----------

func readTemplate(templateName string) (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}

	base := strings.Split(root, "/services/")[0]
	path := filepath.Join(base, "shared/pkg/email/templates", templateName)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read template: %w", err)
	}

	return string(data), nil
}

func replace(template string, values map[string]string) string {
	for k, v := range values {
		template = strings.ReplaceAll(template, "{{"+k+"}}", v)
	}
	return template
}

// ---------- CORE SEND ----------

func (s *MailgunService) send(to, subject, html string, variables map[string]string, tags ...string) error {

	msg := mailgun.NewMessage(
		s.domain,
		fmt.Sprintf("Echo App <mail@%s>", s.domain),
		subject,
		"",
		to,
	)

	msg.SetHTML(html)
	for k, v := range variables {
		msg.AddVariable(k, v)
	}

	for _, t := range tags {
		msg.AddTag(t)
	}

	msg.SetTracking(true)
	msg.SetTrackingClicks(true)
	msg.SetTrackingOpens(true)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.mg.Send(ctx, msg)
	return err
}

// ---------- EMAILS ----------

func (s *MailgunService) SendWelcomeEmail(to, name string) error {
	tpl, err := readTemplate("welcome-email.html")
	if err != nil {
		return err
	}

	return s.send(to, "Welcome to Echo!", tpl, map[string]string{"name": name}, "welcome")
}

func (s *MailgunService) SendPasswordResetEmail(to, name, token string) error {
	tpl, err := readTemplate("password-reset.html")
	if err != nil {
		return err
	}

	resetLink := fmt.Sprintf("https://%s/reset-password?token=%s", s.domain, token)

	body := replace(tpl, map[string]string{
		"name":      name,
		"reset_url": resetLink,
	})

	return s.send(to, "Reset Your Password", body, map[string]string{"name": name, "reset_url": resetLink}, "password-reset")
}

func (s *MailgunService) SendEmailVerificationEmail(to, name, link string) error {
	tpl, err := readTemplate("email-verification.html")
	if err != nil {
		return err
	}

	body := replace(tpl, map[string]string{
		"name":              name,
		"verification_link": link,
	})

	return s.send(to, "Verify Your Email", body, map[string]string{"name": name, "verification_link": link}, "verification")
}

func (s *MailgunService) SendAccountDeletionEmail(to, name string) error {
	tpl, err := readTemplate("account-deletion.html")
	if err != nil {
		return err
	}

	body := replace(tpl, map[string]string{
		"name": name,
	})

	return s.send(to, "Account Deleted", body, map[string]string{"name": name}, "account-deletion")
}
