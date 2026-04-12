package security_event

import (
	"context"
	"encoding/json"

	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/request"
)

type SecurityEventInterface interface {
	LogLoginEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError
	LogLogoutEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError
	LogPasswordChangeEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError
	LogPasswordResetRequestEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError
	LogEmailVerificationEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError
	LogSessionRevocationEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError
	LogRegistrationEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError
}

func (r *SecurityEventRepository) logEvent(ctx context.Context, input enrichedEventInput) pkgErrors.AppError {
	r.log.Info("logging security event",
		logger.String("user_id", input.UserID),
		logger.String("event_type", string(input.eventType)),
		logger.String("event_category", input.eventCategory),
		logger.String("severity", string(input.severity)),
		logger.String("status", input.Status),
	)

	locationInfo, ok := request.GetIPAddressInfoFromContext(ctx)
	if !ok {
		r.log.Warn("IP location info not found in context, proceeding without location data",
			logger.String("user_id", input.UserID),
			logger.String("event_type", string(input.eventType)),
		)
		locationInfo = request.IpAddressInfo{
			Country: "Unknown",
			City:    "Unknown",
			IP:      "127.0.0.1",
		}
	}

	emptyMetadata := json.RawMessage(`{}`)

	_, err := r.db.Insert(ctx, &models.SecurityEvent{
		UserID:          &input.UserID,
		SessionID:       input.SessionID,
		EventType:       input.eventType,
		EventCategory:   &input.eventCategory,
		Severity:        input.severity,
		Status:          &input.Status,
		Description:     &input.Description,
		LocationCountry: &locationInfo.Country,
		LocationCity:    &locationInfo.City,
		IPAddress:       &locationInfo.IP,
		UserAgent:       &input.UserAgent,
		DeviceID:        &input.DeviceID,
		IsSuspicious:    false,
		RiskScore:       utils.Ptr(0),
		Metadata:        &emptyMetadata,
	})
	if err != nil {
		r.log.Warn("failed to log security event (non-critical)",
			logger.String("user_id", input.UserID),
			logger.String("event_type", string(input.eventType)),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to log security event")
	}

	return nil
}

func (r *SecurityEventRepository) LogLoginEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError {
	return r.logEvent(ctx, enrichedEventInput{
		SecurityEventInput: event,
		eventType:          models.SecurityEventLogin,
		eventCategory:      "authentication",
		severity:           models.SecuritySeverityLow,
	})
}

func (r *SecurityEventRepository) LogLogoutEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError {
	return r.logEvent(ctx, enrichedEventInput{
		SecurityEventInput: event,
		eventType:          models.SecurityEventLogout,
		eventCategory:      "authentication",
		severity:           models.SecuritySeverityLow,
	})
}

func (r *SecurityEventRepository) LogPasswordChangeEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError {
	return r.logEvent(ctx, enrichedEventInput{
		SecurityEventInput: event,
		eventType:          models.SecurityEventPasswordChange,
		eventCategory:      "account",
		severity:           models.SecuritySeverityMedium,
	})
}

func (r *SecurityEventRepository) LogPasswordResetRequestEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError {
	return r.logEvent(ctx, enrichedEventInput{
		SecurityEventInput: event,
		eventType:          models.SecurityEventPasswordResetRequested,
		eventCategory:      "account",
		severity:           models.SecuritySeverityMedium,
	})
}

func (r *SecurityEventRepository) LogEmailVerificationEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError {
	return r.logEvent(ctx, enrichedEventInput{
		SecurityEventInput: event,
		eventType:          models.SecurityEventEmailVerified,
		eventCategory:      "account",
		severity:           models.SecuritySeverityLow,
	})
}

func (r *SecurityEventRepository) LogSessionRevocationEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError {
	return r.logEvent(ctx, enrichedEventInput{
		SecurityEventInput: event,
		eventType:          models.SecurityEventSessionRevoked,
		eventCategory:      "session",
		severity:           models.SecuritySeverityMedium,
	})
}

func (r *SecurityEventRepository) LogRegistrationEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError {
	return r.logEvent(ctx, enrichedEventInput{
		SecurityEventInput: event,
		eventType:          models.SecurityEventRegistration,
		eventCategory:      "account",
		severity:           models.SecuritySeverityLow,
	})
}
