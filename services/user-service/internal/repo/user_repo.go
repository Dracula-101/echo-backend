package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	userErrors "user-service/internal/errors"

	"shared/pkg/database"
	"shared/pkg/database/postgres/models"
	"shared/pkg/logger"
)

// ============================================================================
// Repository Definition
// ============================================================================

type UserRepository struct {
	db  database.Database
	log logger.Logger
}

func NewUserRepository(db database.Database, log logger.Logger) *UserRepository {
	if db == nil {
		panic("Database is required for UserRepository")
	}
	if log == nil {
		panic("Logger is required for UserRepository")
	}

	log.Info("Initializing UserRepository",
		logger.String("service", userErrors.ServiceName),
	)

	return &UserRepository{
		db:  db,
		log: log,
	}
}

// ============================================================================
// Profile Operations
// ============================================================================

// GetProfileByUserID retrieves a user profile by user ID
func (r *UserRepository) GetProfileByUserID(ctx context.Context, userID string) (*models.Profile, error) {
	r.log.Debug("Fetching profile by user ID",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `SELECT * FROM users.profiles WHERE user_id = $1 AND deactivated_at IS NULL LIMIT 1`
	row := r.db.QueryRow(ctx, query, userID)

	var profile models.Profile
	err := row.ScanOne(&profile)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Debug("Profile not found",
				logger.String("service", userErrors.ServiceName),
				logger.String("user_id", userID),
			)
			return nil, nil
		}
		r.log.Error("Failed to get profile by user ID",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, err
	}

	r.log.Debug("Profile fetched successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("profile_id", profile.ID),
	)

	return &profile, nil
}

// GetProfileByUsername retrieves a user profile by username
func (r *UserRepository) GetProfileByUsername(ctx context.Context, username string) (*models.Profile, error) {
	r.log.Debug("Fetching profile by username",
		logger.String("service", userErrors.ServiceName),
		logger.String("username", username),
	)

	query := `SELECT * FROM users.profiles WHERE username = $1 AND deactivated_at IS NULL LIMIT 1`
	row := r.db.QueryRow(ctx, query, username)

	var profile models.Profile
	err := row.ScanOne(&profile)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Debug("Profile not found by username",
				logger.String("service", userErrors.ServiceName),
				logger.String("username", username),
			)
			return nil, nil
		}
		r.log.Error("Failed to get profile by username",
			logger.String("service", userErrors.ServiceName),
			logger.String("username", username),
			logger.Error(err),
		)
		return nil, err
	}

	r.log.Debug("Profile fetched successfully by username",
		logger.String("service", userErrors.ServiceName),
		logger.String("username", username),
		logger.String("profile_id", profile.ID),
	)

	return &profile, nil
}

// UpdateProfile updates a user profile
type UpdateProfileParams struct {
	UserID       string
	Username     *string
	DisplayName  *string
	FirstName    *string
	LastName     *string
	Bio          *string
	AvatarURL    *string
	LanguageCode *string
	Timezone     *string
	CountryCode  *string
}

func (r *UserRepository) UpdateProfile(ctx context.Context, params UpdateProfileParams) (*models.Profile, error) {
	r.log.Info("Updating profile",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", params.UserID),
	)

	// Build dynamic update query
	query := `UPDATE users.profiles SET updated_at = NOW()`
	args := []interface{}{params.UserID}
	argPos := 2

	if params.Username != nil {
		query += fmt.Sprintf(", username = $%d", argPos)
		args = append(args, *params.Username)
		argPos++
	}
	if params.DisplayName != nil {
		query += fmt.Sprintf(", display_name = $%d", argPos)
		args = append(args, *params.DisplayName)
		argPos++
	}
	if params.FirstName != nil {
		query += fmt.Sprintf(", first_name = $%d", argPos)
		args = append(args, *params.FirstName)
		argPos++
	}
	if params.LastName != nil {
		query += fmt.Sprintf(", last_name = $%d", argPos)
		args = append(args, *params.LastName)
		argPos++
	}
	if params.Bio != nil {
		query += fmt.Sprintf(", bio = $%d", argPos)
		args = append(args, *params.Bio)
		argPos++
	}
	if params.AvatarURL != nil {
		query += fmt.Sprintf(", avatar_url = $%d", argPos)
		args = append(args, *params.AvatarURL)
		argPos++
	}
	if params.LanguageCode != nil {
		query += fmt.Sprintf(", language_code = $%d", argPos)
		args = append(args, *params.LanguageCode)
		argPos++
	}
	if params.Timezone != nil {
		query += fmt.Sprintf(", timezone = $%d", argPos)
		args = append(args, *params.Timezone)
		argPos++
	}
	if params.CountryCode != nil {
		query += fmt.Sprintf(", country_code = $%d", argPos)
		args = append(args, *params.CountryCode)
		argPos++
	}

	query += ` WHERE user_id = $1 AND deactivated_at IS NULL RETURNING *`

	r.log.Debug("Executing update query",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", params.UserID),
	)

	row := r.db.QueryRow(ctx, query, args...)
	var profile models.Profile
	err := row.ScanOne(&profile)
	if err != nil {
		r.log.Error("Failed to update profile",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", params.UserID),
			logger.Error(err),
		)
		return nil, err
	}

	r.log.Info("Profile updated successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", params.UserID),
		logger.String("profile_id", profile.ID),
	)

	return &profile, nil
}

// SearchProfiles searches for profiles by query
func (r *UserRepository) SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*models.Profile, int, error) {
	r.log.Debug("Searching profiles",
		logger.String("service", userErrors.ServiceName),
		logger.String("query", query),
		logger.Int("limit", limit),
		logger.Int("offset", offset),
	)

	searchQuery := `
		SELECT * FROM users.profiles
		WHERE (
			username ILIKE $1 OR
			display_name ILIKE $1 OR
			first_name ILIKE $1 OR
			last_name ILIKE $1
		)
		AND deactivated_at IS NULL
		AND search_visibility = true
		ORDER BY
			CASE WHEN username ILIKE $1 THEN 1 ELSE 2 END,
			created_at DESC
		LIMIT $2 OFFSET $3
	`

	searchPattern := "%" + query + "%"
	rows, err := r.db.Query(ctx, searchQuery, searchPattern, limit, offset)
	if err != nil {
		r.log.Error("Failed to search profiles",
			logger.String("service", userErrors.ServiceName),
			logger.String("query", query),
			logger.Error(err),
		)
		return nil, 0, err
	}
	defer rows.Close()

	var profiles []*models.Profile
	for rows.Next() {
		var profile models.Profile
		if err := rows.Scan(&profile); err != nil {
			r.log.Error("Failed to scan profile",
				logger.String("service", userErrors.ServiceName),
				logger.Error(err),
			)
			continue
		}
		profiles = append(profiles, &profile)
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*) FROM users.profiles
		WHERE (
			username ILIKE $1 OR
			display_name ILIKE $1 OR
			first_name ILIKE $1 OR
			last_name ILIKE $1
		)
		AND deactivated_at IS NULL
		AND search_visibility = true
	`

	var totalCount int
	countRow := r.db.QueryRow(ctx, countQuery, searchPattern)
	if err := countRow.Scan(&totalCount); err != nil {
		r.log.Error("Failed to get search count",
			logger.String("service", userErrors.ServiceName),
			logger.Error(err),
		)
		totalCount = len(profiles) // Fallback to actual count
	}

	r.log.Debug("Profile search completed",
		logger.String("service", userErrors.ServiceName),
		logger.String("query", query),
		logger.Int("results", len(profiles)),
		logger.Int("total_count", totalCount),
	)

	return profiles, totalCount, nil
}

// UsernameExists checks if a username is already taken
func (r *UserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	r.log.Debug("Checking if username exists",
		logger.String("service", userErrors.ServiceName),
		logger.String("username", username),
	)

	query := `SELECT EXISTS(SELECT 1 FROM users.profiles WHERE username = $1 AND deactivated_at IS NULL)`
	var exists bool
	err := r.db.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		r.log.Error("Failed to check username existence",
			logger.String("service", userErrors.ServiceName),
			logger.String("username", username),
			logger.Error(err),
		)
		return false, err
	}

	r.log.Debug("Username existence check completed",
		logger.String("service", userErrors.ServiceName),
		logger.String("username", username),
		logger.Bool("exists", exists),
	)

	return exists, nil
}
