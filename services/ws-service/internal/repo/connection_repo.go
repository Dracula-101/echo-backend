package repo

import (
	"context"
	"time"
	"ws-service/internal/model"

	"shared/pkg/database"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// ConnectionRepository handles persistence of connection records
type ConnectionRepository interface {
	// CreateConnection records a new WebSocket connection
	CreateConnection(ctx context.Context, conn *model.ConnectionRecord) error

	// UpdateConnectionStatus updates the status of a connection
	UpdateConnectionStatus(ctx context.Context, connectionID uuid.UUID, status string) error

	// GetActiveConnections retrieves all active connections for a user
	GetActiveConnections(ctx context.Context, userID uuid.UUID) ([]*model.ConnectionRecord, error)

	// DeleteConnection removes a connection record
	DeleteConnection(ctx context.Context, connectionID uuid.UUID) error

	// CleanupStaleConnections removes connections older than the specified duration
	CleanupStaleConnections(ctx context.Context, olderThan time.Duration) (int, error)
}

type connectionRepository struct {
	db  database.Database
	log logger.Logger
}

// NewConnectionRepository creates a new connection repository
func NewConnectionRepository(db database.Database, log logger.Logger) ConnectionRepository {
	return &connectionRepository{
		db:  db,
		log: log,
	}
}

// CreateConnection records a new WebSocket connection
func (r *connectionRepository) CreateConnection(ctx context.Context, conn *model.ConnectionRecord) error {
	query := `
		INSERT INTO websocket.connections (
			id, user_id, device_id, client_id, ip_address,
			user_agent, platform, app_version, connected_at, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(ctx, query,
		conn.ID,
		conn.UserID,
		conn.DeviceID,
		conn.ClientID,
		conn.IPAddress,
		conn.UserAgent,
		conn.Platform,
		conn.AppVersion,
		conn.ConnectedAt,
		conn.Status,
	)

	if err != nil {
		r.log.Error("Failed to create connection record",
			logger.String("connection_id", conn.ID.String()),
			logger.String("user_id", conn.UserID.String()),
			logger.Error(err),
		)
		return err
	}

	r.log.Debug("Connection record created",
		logger.String("connection_id", conn.ID.String()),
		logger.String("user_id", conn.UserID.String()),
	)

	return nil
}

// UpdateConnectionStatus updates the status of a connection
func (r *connectionRepository) UpdateConnectionStatus(ctx context.Context, connectionID uuid.UUID, status string) error {
	query := `
		UPDATE websocket.connections
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.Exec(ctx, query, status, connectionID)
	if err != nil {
		r.log.Error("Failed to update connection status",
			logger.String("connection_id", connectionID.String()),
			logger.Error(err),
		)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		r.log.Warn("No connection found to update",
			logger.String("connection_id", connectionID.String()),
		)
	}

	return nil
}

// GetActiveConnections retrieves all active connections for a user
func (r *connectionRepository) GetActiveConnections(ctx context.Context, userID uuid.UUID) ([]*model.ConnectionRecord, error) {
	query := `
		SELECT id, user_id, device_id, client_id, ip_address,
		       user_agent, platform, app_version, connected_at,
		       disconnected_at, status, created_at, updated_at
		FROM websocket.connections
		WHERE user_id = $1 AND status = 'active'
		ORDER BY connected_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to get active connections",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	connections := make([]*model.ConnectionRecord, 0)
	for rows.Next() {
		conn := &model.ConnectionRecord{}
		err := rows.Scan(
			&conn.ID,
			&conn.UserID,
			&conn.DeviceID,
			&conn.ClientID,
			&conn.IPAddress,
			&conn.UserAgent,
			&conn.Platform,
			&conn.AppVersion,
			&conn.ConnectedAt,
			&conn.DisconnectedAt,
			&conn.Status,
			&conn.CreatedAt,
			&conn.UpdatedAt,
		)
		if err != nil {
			r.log.Error("Failed to scan connection record",
				logger.Error(err),
			)
			continue
		}
		connections = append(connections, conn)
	}

	return connections, nil
}

// DeleteConnection removes a connection record
func (r *connectionRepository) DeleteConnection(ctx context.Context, connectionID uuid.UUID) error {
	query := `
		UPDATE websocket.connections
		SET status = 'disconnected', disconnected_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, connectionID)
	if err != nil {
		r.log.Error("Failed to delete connection",
			logger.String("connection_id", connectionID.String()),
			logger.Error(err),
		)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		r.log.Warn("No connection found to delete",
			logger.String("connection_id", connectionID.String()),
		)
	}

	return nil
}

// CleanupStaleConnections removes connections older than the specified duration
func (r *connectionRepository) CleanupStaleConnections(ctx context.Context, olderThan time.Duration) (int, error) {
	query := `
		UPDATE websocket.connections
		SET status = 'stale', updated_at = NOW()
		WHERE status = 'active'
		  AND connected_at < NOW() - $1::interval
	`

	result, err := r.db.Exec(ctx, query, olderThan.String())
	if err != nil {
		r.log.Error("Failed to cleanup stale connections",
			logger.Error(err),
		)
		return 0, err
	}

	affectedRows, _ := result.RowsAffected()
	rowsAffected := int(affectedRows)
	if rowsAffected > 0 {
		r.log.Info("Cleaned up stale connections",
			logger.Int("count", rowsAffected),
		)
	}

	return rowsAffected, nil
}
