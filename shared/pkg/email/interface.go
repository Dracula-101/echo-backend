package email

import "shared/pkg/email/mailgun"

type EmailService interface {
	SendWelcomeEmail(toEmail string, name string) error
	SendPasswordResetEmail(toEmail string, name string, token string) error
	SendEmailVerificationEmail(toEmail string, token string) error
	SendAccountDeletionEmail(toEmail string, name string) error
}

type EmailConfig struct {
	Provider string
	Mailgun  mailgun.MailgunConfig
}
