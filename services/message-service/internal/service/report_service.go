package service

import (
	"context"

	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/repo"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// ReportServiceInterface defines the interface for message report operations.
type ReportServiceInterface interface {
	ReportMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, reportType, description string) *msgError.MsgError
}

type reportService struct {
	reportRepo repo.ReportRepository
	logger     logger.Logger
}

// NewReportService creates a new instance of ReportServiceInterface.
func NewReportService(reportRepo repo.ReportRepository, log logger.Logger) ReportServiceInterface {
	return &reportService{
		reportRepo: reportRepo,
		logger:     log,
	}
}

// ReportMessage checks if the user has already reported the message and creates a new report.
func (s *reportService) ReportMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, reportType, description string) *msgError.MsgError {
	// Check if the user has already reported this message
	alreadyReported, err := s.reportRepo.HasUserReported(ctx, messageID, userID)
	if err != nil {
		s.logger.Error("Failed to check existing report",
			logger.String("message_id", messageID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return &msgError.MsgError{
			Code:    msgError.CodeConversationValidationFailed,
			Message: "Failed to check existing report",
			Error:   pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "Failed to check existing report"),
		}
	}

	if alreadyReported {
		return &msgError.MsgError{
			Code:    msgError.CodeConversationValidationFailed,
			Message: "You have already reported this message",
		}
	}

	// Create the report
	_, createErr := s.reportRepo.CreateReport(ctx, messageID, userID, reportType, description)
	if createErr != nil {
		s.logger.Error("Failed to create message report",
			logger.String("message_id", messageID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(createErr),
		)
		return &msgError.MsgError{
			Code:    msgError.CodeConversationUpdateFailed,
			Message: "Failed to create message report",
			Error:   pkgErrors.FromError(createErr, pkgErrors.CodeDatabaseError, "Failed to create message report"),
		}
	}

	return nil
}
