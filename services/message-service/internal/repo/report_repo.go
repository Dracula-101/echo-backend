package repo

import (
	"context"
	"time"

	"shared/pkg/database"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// Report represents a message report record.
type Report struct {
	ID             uuid.UUID
	MessageID      uuid.UUID
	ReporterUserID uuid.UUID
	ReportType     string
	Description    string
	Status         string
	CreatedAt      time.Time
}

// ReportRepository defines the interface for message report interactions.
type ReportRepository interface {
	CreateReport(ctx context.Context, messageID, reporterUserID uuid.UUID, reportType, description string) (*Report, error)
	HasUserReported(ctx context.Context, messageID, userID uuid.UUID) (bool, error)
}

type reportRepository struct {
	db     database.Database
	logger logger.Logger
}

// NewReportRepository creates a new instance of ReportRepository.
func NewReportRepository(db database.Database, log logger.Logger) ReportRepository {
	return &reportRepository{
		db:     db,
		logger: log,
	}
}

// CreateReport inserts a new message report and returns the created record.
func (r *reportRepository) CreateReport(ctx context.Context, messageID, reporterUserID uuid.UUID, reportType, description string) (*Report, error) {
	query := `
		INSERT INTO messages.message_reports (
			message_id, reporter_user_id, report_type, description
		) VALUES (
			$1, $2, $3, $4
		)
		RETURNING id, message_id, reporter_user_id, report_type, description, status, created_at
	`

	row := r.db.QueryRow(ctx, query, messageID, reporterUserID, reportType, description)

	var report Report
	err := row.Scan(
		&report.ID,
		&report.MessageID,
		&report.ReporterUserID,
		&report.ReportType,
		&report.Description,
		&report.Status,
		&report.CreatedAt,
	)
	if err != nil {
		r.logger.Error("Failed to create message report",
			logger.String("message_id", messageID.String()),
			logger.String("reporter_user_id", reporterUserID.String()),
			logger.Error(err),
		)
		return nil, err
	}

	return &report, nil
}

// HasUserReported checks whether a user has already reported a specific message.
func (r *reportRepository) HasUserReported(ctx context.Context, messageID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM messages.message_reports
			WHERE message_id = $1 AND reporter_user_id = $2
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, messageID, userID).Scan(&exists)
	if err != nil {
		r.logger.Error("Failed to check if user has reported message",
			logger.String("message_id", messageID.String()),
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return false, err
	}

	return exists, nil
}
