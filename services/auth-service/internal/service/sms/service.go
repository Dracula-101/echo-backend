package sms

import (
	authErrors "auth-service/internal/error"
	"context"
	"fmt"
	"shared/pkg/logger"
	"shared/pkg/sms"
	"shared/pkg/sms/twilio"
)

type SmsService interface {
	SendOTP(ctx context.Context, phone string, otp string) *authErrors.AuthError
}

type smsService struct {
	provider sms.Sender
	log      logger.Logger
}

func NewSmsService(accountSid string, messagingServiceSid string, authToken string, fromNumber string, log logger.Logger) SmsService {
	log.Info("Initializing SMS service with Twilio",
		logger.String("service", authErrors.ServiceName),
		logger.String("provider", "twilio"),
	)
	return &smsService{
		provider: twilio.New(twilio.Config{
			AccountSID:          accountSid,
			AuthToken:           authToken,
			FromNumber:          fromNumber,
			MessagingServiceSID: messagingServiceSid,
		}, log),
		log: log,
	}
}

func (s *smsService) SendOTP(ctx context.Context, phone string, otp string) *authErrors.AuthError {
	s.log.Info("Sending OTP via SMS",
		logger.String("phone", phone),
	)
	template := fmt.Sprintf("Your verification code is: %s. This code will expire in 10 minutes.", otp)
	err := s.provider.Send(ctx, phone, template)
	if err != nil {
		s.log.Error("Failed to send OTP via SMS",
			logger.String("phone", phone),
			logger.Error(err),
		)
		return &authErrors.AuthError{
			Code:    authErrors.CodeSmsSendFailed,
			Message: fmt.Sprintf("Failed to send OTP to %s: %v", phone, err),
		}
	}

	s.log.Info("OTP sent successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("phone", phone),
	)
	return nil
}
