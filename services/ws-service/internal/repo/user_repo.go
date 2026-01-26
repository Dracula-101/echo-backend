package repo

import (
	"context"

	"shared/pkg/database"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// UserRepository handles user-related database operations
type UserRepository interface {
	// UserExists checks if a user exists in the users.profiles table
	UserExists(ctx context.Context, userID uuid.UUID) (bool, error)

	// HasActiveSession checks if a user has any active sessions
	HasActiveSession(ctx context.Context, userID uuid.UUID) (bool, error)

	// ValidateUserAccess validates both user existence and active session
	ValidateUserAccess(ctx context.Context, userID uuid.UUID) (bool, error)

	// UpdateLastSeen updates the user's last_seen_at timestamp
	UpdateLastSeen(ctx context.Context, userID uuid.UUID) error

	// UpdateOnlineStatus updates the user's online_status field
	UpdateOnlineStatus(ctx context.Context, userID uuid.UUID, status string) error

	// GetUserContacts returns the list of accepted contacts for a user
	GetUserContacts(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type userRepository struct {
	db  database.Database
	log logger.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db database.Database, log logger.Logger) UserRepository {
	return &userRepository{
		db:  db,
		log: log,
	}
}

// UserExists checks if a user exists in the users.profiles table
func (r *userRepository) UserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM users.profiles
			WHERE user_id = $1
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		r.log.Error("Failed to check if user exists",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return false, err
	}

	r.log.Debug("User existence check completed",
		logger.String("user_id", userID.String()),
		logger.Bool("exists", exists),
	)

	return exists, nil
}

// HasActiveSession checks if a user has any active (non-revoked) sessions
func (r *userRepository) HasActiveSession(ctx context.Context, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM auth.sessions
			WHERE user_id = $1
			  AND revoked_at IS NULL
		)
	`

	var hasSession bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&hasSession)
	if err != nil {
		r.log.Error("Failed to check if user has active sessions",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return false, err
	}

	r.log.Debug("User active session check completed",
		logger.String("user_id", userID.String()),
		logger.Bool("has_active_session", hasSession),
	)

	return hasSession, nil
}

// ValidateUserAccess validates both user existence and active session
// Returns true only if user exists AND has at least one active session
func (r *userRepository) ValidateUserAccess(ctx context.Context, userID uuid.UUID) (bool, error) {
	// First check if user exists
	exists, err := r.UserExists(ctx, userID)
	if err != nil {
		return false, err
	}

	if !exists {
		r.log.Debug("User validation failed: user does not exist",
			logger.String("user_id", userID.String()),
		)
		return false, nil
	}

	// Check if user has active session
	hasSession, err := r.HasActiveSession(ctx, userID)
	if err != nil {
		return false, err
	}

	if !hasSession {
		r.log.Debug("User validation failed: no active session",
			logger.String("user_id", userID.String()),
		)
		return false, nil
	}

	r.log.Debug("User validation successful",
		logger.String("user_id", userID.String()),
	)

	return true, nil
}

// UpdateLastSeen updates the user's last_seen_at timestamp
func (r *userRepository) UpdateLastSeen(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users.profiles
		SET last_seen_at = NOW(),
		    updated_at = NOW()
		WHERE user_id = $1
	`

	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to update last seen",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		r.log.Warn("No rows updated for last seen",
			logger.String("user_id", userID.String()),
		)
	}

	return nil
}

// UpdateOnlineStatus updates the user's online_status field
func (r *userRepository) UpdateOnlineStatus(ctx context.Context, userID uuid.UUID, status string) error {
	query := `
		UPDATE users.profiles
		SET online_status = $1,
		    updated_at = NOW()
		WHERE user_id = $2
	`

	result, err := r.db.Exec(ctx, query, status, userID)
	if err != nil {
		r.log.Error("Failed to update online status",
			logger.String("user_id", userID.String()),
			logger.String("status", status),
			logger.Error(err),
		)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		r.log.Warn("No rows updated for online status",
			logger.String("user_id", userID.String()),
		)
	}

	r.log.Debug("Online status updated",
		logger.String("user_id", userID.String()),
		logger.String("status", status),
	)

	return nil
}

// GetUserContacts returns the list of accepted contacts for a user
func (r *userRepository) GetUserContacts(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT contact_user_id
		FROM users.contacts
		WHERE user_id = $1
		  AND status = 'accepted'
		  AND relationship_type IN ('friend', 'contact')
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to get user contacts",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	var contacts []uuid.UUID
	for rows.Next() {
		var contactID uuid.UUID
		if err := rows.Scan(&contactID); err != nil {
			r.log.Error("Failed to scan contact ID",
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
			return nil, err
		}
		contacts = append(contacts, contactID)
	}

	if err := rows.Err(); err != nil {
		r.log.Error("Error iterating over contacts",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, err
	}

	r.log.Debug("Retrieved user contacts",
		logger.String("user_id", userID.String()),
		logger.Int("count", len(contacts)),
	)

	return contacts, nil
}
