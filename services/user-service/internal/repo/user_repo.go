package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	dbModels "shared/pkg/database/postgres/models"
	"shared/pkg/utils"
	"user-service/internal/domain"
	userErrors "user-service/internal/errors"

	"shared/pkg/database"
	"shared/pkg/database/postgres"
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

// Generate Unique Username
func (r *UserRepository) GenerateUniqueUsername(ctx context.Context, baseUsername string) (*string, error) {
	r.log.Debug("Generating unique username",
		logger.String("service", userErrors.ServiceName),
		logger.String("base_username", baseUsername),
	)

	// Basic normalization: trim, lowercase, allow a-z0-9 and . _ -
	base := strings.ToLower(strings.TrimSpace(baseUsername))
	re := regexp.MustCompile(`[^a-z0-9._-]+`)
	base = re.ReplaceAllString(base, "")

	// Fallback if nothing remains after sanitization
	if base == "" {
		base = fmt.Sprintf("user%d", time.Now().Unix()%10000)
	}

	const maxLen = 30
	if len(base) > maxLen {
		base = base[:maxLen]
	}

	username := base
	query := `SELECT EXISTS(SELECT 1 FROM users.profiles WHERE username = $1 AND deactivated_at IS NULL)`
	rand.Seed(time.Now().UnixNano())
	maxAttempts := 1000
	for attempt := 0; attempt < maxAttempts; attempt++ {
		var exists bool
		err := r.db.QueryRow(ctx, query, username).Scan(&exists)
		if err != nil {
			r.log.Error("Failed to check username existence",
				logger.String("service", userErrors.ServiceName),
				logger.String("username", username),
				logger.Error(err),
			)
			return nil, err
		}

		if !exists {
			r.log.Debug("Unique username generated",
				logger.String("service", userErrors.ServiceName),
				logger.String("unique_username", username),
			)
			return &username, nil
		}

		if attempt < 50 {
			suffix := attempt + 1
			baseLimit := base
			maxBaseLen := maxLen - len(fmt.Sprintf("%d", suffix))
			if len(baseLimit) > maxBaseLen {
				baseLimit = baseLimit[:maxBaseLen]
			}
			username = fmt.Sprintf("%s%d", baseLimit, suffix)
		} else {
			suffixLen := 4
			suffix := randAlphaNum(suffixLen)
			maxBaseLen := maxLen - suffixLen
			baseLimit := base
			if len(baseLimit) > maxBaseLen {
				baseLimit = baseLimit[:maxBaseLen]
			}
			username = fmt.Sprintf("%s%s", baseLimit, suffix)
		}
	}

	r.log.Error("Unable to generate unique username after attempts",
		logger.String("service", userErrors.ServiceName),
		logger.String("base_username", baseUsername),
		logger.Int("attempts", maxAttempts),
	)

	return nil, fmt.Errorf("unable to generate unique username after %d attempts", maxAttempts)
}

// small helper for random alphanumeric suffixes
func randAlphaNum(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// GetProfileByUserID retrieves a user profile by user ID
func (r *UserRepository) GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	r.log.Debug("Fetching profile by user ID",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
	)

	// user_id has UNIQUE constraint, so LIMIT 1 is unnecessary
	query := `SELECT * FROM users.profiles WHERE user_id = $1 AND deactivated_at IS NULL`
	row := r.db.QueryRow(ctx, query, userID)

	var profile dbModels.Profile
	err := row.ScanModel(&profile)
	if err != nil {
		if postgres.IsNotFoundError(err) {
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

	return &domain.Profile{
		ID:           profile.ID,
		UserID:       profile.UserID,
		Username:     profile.Username,
		DisplayName:  profile.DisplayName,
		FirstName:    profile.FirstName,
		LastName:     profile.LastName,
		Bio:          profile.Bio,
		AvatarURL:    profile.AvatarURL,
		LanguageCode: profile.LanguageCode,
		Timezone:     profile.Timezone,
		CountryCode:  profile.CountryCode,
		IsVerified:   profile.IsVerified,
		CreatedAt:    profile.CreatedAt,
		UpdatedAt:    profile.UpdatedAt,
	}, nil
}

// GetProfileByUsername retrieves a user profile by username
func (r *UserRepository) GetProfileByUsername(ctx context.Context, username string) (*domain.Profile, error) {
	r.log.Debug("Fetching profile by username",
		logger.String("service", userErrors.ServiceName),
		logger.String("username", username),
	)

	// username has UNIQUE constraint, so LIMIT 1 is unnecessary
	query := `SELECT * FROM users.profiles WHERE username = $1 AND deactivated_at IS NULL`
	row := r.db.QueryRow(ctx, query, username)

	var profile dbModels.Profile
	err := row.ScanModel(&profile)
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

	return &domain.Profile{
		ID:           profile.ID,
		UserID:       profile.UserID,
		Username:     profile.Username,
		DisplayName:  profile.DisplayName,
		FirstName:    profile.FirstName,
		LastName:     profile.LastName,
		Bio:          profile.Bio,
		AvatarURL:    profile.AvatarURL,
		LanguageCode: profile.LanguageCode,
		Timezone:     profile.Timezone,
		CountryCode:  profile.CountryCode,
		IsVerified:   profile.IsVerified,
		CreatedAt:    profile.CreatedAt,
		UpdatedAt:    profile.UpdatedAt,
	}, nil
}

// CreateProfile creates a new user profile
func (r *UserRepository) CreateProfile(ctx context.Context, profile domain.Profile) (*domain.Profile, error) {
	r.log.Info("Creating new profile",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", profile.UserID),
	)

	var profileModel = dbModels.Profile{
		UserID:            profile.UserID,
		Username:          profile.Username,
		DisplayName:       profile.DisplayName,
		FirstName:         profile.FirstName,
		LastName:          profile.LastName,
		Bio:               profile.Bio,
		AvatarURL:         profile.AvatarURL,
		LanguageCode:      profile.LanguageCode,
		OnlineStatus:      dbModels.OnlineStatusOffline,
		ProfileVisibility: dbModels.ProfileVisibilityPublic,
		SearchVisibility:  profile.SearchVisibility,
		Timezone:          profile.Timezone,
		CountryCode:       profile.CountryCode,
		IsVerified:        profile.IsVerified,
		CreatedAt:         utils.Ptr(time.Now()),
		UpdatedAt:         utils.Ptr(time.Now()),
	}
	id, err := r.db.Insert(ctx, &profileModel)
	if err != nil {
		r.log.Error("Failed to create profile",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", profile.UserID),
			logger.Error(err),
		)
		return nil, err
	}

	createdProfile, dbErr := r.GetProfileByUserID(ctx, profile.UserID)
	if dbErr != nil {
		r.log.Error("Failed to retrieve created profile",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", profile.UserID),
			logger.String("profile_id", *id),
			logger.Error(dbErr),
		)
		return nil, dbErr
	}
	if createdProfile == nil {
		r.log.Error("Created profile not found",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", profile.UserID),
		)
		return nil, fmt.Errorf("created profile not found for user_id: %s", profile.UserID)
	}

	r.log.Info("Profile created successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", profile.UserID),
		logger.String("profile_id", createdProfile.ID),
	)
	return createdProfile, nil
}

// UpdateProfile updates a user profile
type UpdateProfileParams struct {
	UserID             string
	Username           *string
	DisplayName        *string
	FirstName          *string
	LastName           *string
	Bio                *string
	AvatarURL          *string
	AvatarThumbnailURL *string
	CoverImageURL      *string
	LanguageCode       *string
	Timezone           *string
	CountryCode        *string
}

func (r *UserRepository) UpdateProfile(ctx context.Context, params UpdateProfileParams) (*domain.Profile, error) {
	r.log.Info("Updating profile",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", params.UserID),
	)

	// Build dynamic SET clause
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argPos := 1

	if params.Username != nil {
		setClauses = append(setClauses, fmt.Sprintf("username = $%d", argPos))
		args = append(args, *params.Username)
		argPos++
	}
	if params.DisplayName != nil {
		setClauses = append(setClauses, fmt.Sprintf("display_name = $%d", argPos))
		args = append(args, *params.DisplayName)
		argPos++
	}
	if params.FirstName != nil {
		setClauses = append(setClauses, fmt.Sprintf("first_name = $%d", argPos))
		args = append(args, *params.FirstName)
		argPos++
	}
	if params.LastName != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_name = $%d", argPos))
		args = append(args, *params.LastName)
		argPos++
	}
	if params.Bio != nil {
		setClauses = append(setClauses, fmt.Sprintf("bio = $%d", argPos))
		args = append(args, *params.Bio)
		argPos++
	}
	if params.AvatarURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar_url = $%d", argPos))
		args = append(args, *params.AvatarURL)
		argPos++
	}
	if params.AvatarThumbnailURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("avatar_thumbnail_url = $%d", argPos))
		args = append(args, *params.AvatarThumbnailURL)
		argPos++
	}
	if params.CoverImageURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("cover_image_url = $%d", argPos))
		args = append(args, *params.CoverImageURL)
		argPos++
	}
	if params.LanguageCode != nil {
		setClauses = append(setClauses, fmt.Sprintf("language_code = $%d", argPos))
		args = append(args, *params.LanguageCode)
		argPos++
	}
	if params.Timezone != nil {
		setClauses = append(setClauses, fmt.Sprintf("timezone = $%d", argPos))
		args = append(args, *params.Timezone)
		argPos++
	}
	if params.CountryCode != nil {
		setClauses = append(setClauses, fmt.Sprintf("country_code = $%d", argPos))
		args = append(args, *params.CountryCode)
		argPos++
	}

	args = append(args, params.UserID)

	query := fmt.Sprintf(`
		UPDATE users.profiles 
		SET %s
		WHERE user_id = $%d AND deactivated_at IS NULL 
		RETURNING *`,
		strings.Join(setClauses, ", "),
		argPos,
	)

	r.log.Debug("Executing update query",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", params.UserID),
	)

	var profile dbModels.Profile
	if err := r.db.FindOneAndUpdate(ctx, &profile, query, args...); err != nil {
		if postgres.IsNotFoundError(err) {
			r.log.Debug("No profile updated - not found",
				logger.String("service", userErrors.ServiceName),
				logger.String("user_id", params.UserID),
			)
			return nil, nil
		}
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

	return &domain.Profile{
		ID:           profile.ID,
		UserID:       profile.UserID,
		Username:     profile.Username,
		DisplayName:  profile.DisplayName,
		FirstName:    profile.FirstName,
		LastName:     profile.LastName,
		Bio:          profile.Bio,
		AvatarURL:    profile.AvatarURL,
		LanguageCode: profile.LanguageCode,
		Timezone:     profile.Timezone,
		CountryCode:  profile.CountryCode,
		IsVerified:   profile.IsVerified,
		CreatedAt:    profile.CreatedAt,
		UpdatedAt:    profile.UpdatedAt,
	}, nil
}

func (r *UserRepository) AddUserDevice(ctx context.Context, input *domain.UserDevice, isCurrentDevice bool) error {
	r.log.Info("Adding user device",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", input.UserID),
		logger.String("device_id", input.DeviceID),
	)

	deviceModel := dbModels.Device{
		UserID:             input.UserID,
		DeviceID:           input.DeviceID,
		DeviceName:         utils.PtrString(input.DeviceName),
		DeviceType:         utils.PtrString(input.DeviceType),
		DeviceModel:        utils.PtrString(input.DeviceModel),
		DeviceManufacturer: utils.PtrString(input.DeviceManufacturer),
		OSName:             utils.PtrString(input.OSName),
		OSVersion:          utils.PtrString(input.OSVersion),
		AppVersion:         input.AppVersion,
		IsCurrentDevice:    isCurrentDevice,
		IsActive:           input.FCMToken != nil || input.APNSToken != nil,
		LastActiveAt:       time.Now(),
		RegisteredAt:       time.Now(),
		FCMToken:           input.FCMToken,
		APNSToken:          input.APNSToken,
		PushEnabled:        input.PushEnabled,
		Metadata:           json.RawMessage("{}"),
	}

	_, err := r.db.Insert(ctx, &deviceModel)
	if err != nil {
		r.log.Error("Failed to add user device",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", input.UserID),
			logger.String("device_id", input.DeviceID),
			logger.Error(err),
		)
		return err
	}

	r.log.Info("User device added successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", input.UserID),
		logger.String("device_id", input.DeviceID),
	)

	return nil
}

func (r *UserRepository) GetUserDevices(ctx context.Context, userID string) ([]*domain.UserDevice, error) {
	r.log.Debug("Fetching user devices",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `SELECT * FROM users.devices WHERE user_id = $1 AND is_active = true`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to fetch user devices",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	var devices []*domain.UserDevice
	for rows.Next() {
		var device dbModels.Device
		if err := rows.ScanModel(&device); err != nil {
			r.log.Error("Failed to scan user device",
				logger.String("service", userErrors.ServiceName),
				logger.String("user_id", userID),
				logger.Error(err),
			)
			continue
		}
		devices = append(devices, domain.NewUserDevice(device))
	}

	r.log.Debug("User devices fetched successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int("device_count", len(devices)),
	)

	return devices, nil
}

func (r *UserRepository) UpdateUserDevice(ctx context.Context, input *domain.UpdateUserDevice, userID string, deviceID string) error {
	r.log.Info("Updating user device",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
	)

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if input.PushEnabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("push_enabled = $%d", argPos))
		args = append(args, *input.PushEnabled)
		argPos++
	}
	if input.FCMToken != nil {
		setClauses = append(setClauses, fmt.Sprintf("fcm_token = $%d", argPos))
		args = append(args, *input.FCMToken)
		argPos++
	}
	if input.APNSToken != nil {
		setClauses = append(setClauses, fmt.Sprintf("apns_token = $%d", argPos))
		args = append(args, *input.APNSToken)
		argPos++
	}
	if input.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, *input.IsActive)
		argPos++
	}
	setClauses = append(setClauses, fmt.Sprintf("last_active_at = $%d", argPos))
	args = append(args, time.Now())
	argPos++

	args = append(args, userID)
	args = append(args, deviceID)

	query := fmt.Sprintf(`
		UPDATE users.devices 
		SET %s
		WHERE user_id = $%d AND device_id = $%d`,
		strings.Join(setClauses, ", "),
		argPos-2,
		argPos-1,
	)

	r.log.Debug("Executing update device query",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
	)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		r.log.Error("Failed to update user device",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userID),
			logger.String("device_id", deviceID),
			logger.Error(err),
		)
		return err
	}

	r.log.Info("User device updated successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
	)

	return nil
}

// SearchProfiles searches for profiles by query
func (r *UserRepository) SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int, error) {
	r.log.Debug("Searching profiles",
		logger.String("service", userErrors.ServiceName),
		logger.String("query", query),
		logger.Int("limit", limit),
		logger.Int("offset", offset),
	)

	searchQuery := `
		SELECT *
			FROM users.profiles
			WHERE
				deactivated_at IS NULL
				AND search_visibility = true
				AND (
					username ILIKE '%' || $1 || '%'
					OR display_name ILIKE '%' || $1 || '%'
				)
		ORDER BY
			CASE WHEN username ILIKE $1 THEN 1 ELSE 0 END DESC,
			CASE WHEN username ILIKE $1 || '%' THEN 1 ELSE 0 END DESC,
			LEAST(
				NULLIF(POSITION(LOWER($1) IN LOWER(username)), 0),
				NULLIF(POSITION(LOWER($1) IN LOWER(display_name)), 0)
			) ASC NULLS LAST,
			LENGTH(username) ASC,
			created_at DESC
		LIMIT $2 OFFSET $3;
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

	var profiles []*domain.Profile
	for rows.Next() {
		var profile dbModels.Profile
		if err := rows.ScanModel(&profile); err != nil {
			r.log.Error("Failed to scan profile",
				logger.String("service", userErrors.ServiceName),
				logger.Error(err),
			)
			continue
		}
		profiles = append(profiles, &domain.Profile{
			ID:           profile.ID,
			UserID:       profile.UserID,
			Username:     profile.Username,
			DisplayName:  profile.DisplayName,
			FirstName:    profile.FirstName,
			LastName:     profile.LastName,
			Bio:          profile.Bio,
			AvatarURL:    profile.AvatarURL,
			LanguageCode: profile.LanguageCode,
			Timezone:     profile.Timezone,
			CountryCode:  profile.CountryCode,
			IsVerified:   profile.IsVerified,
			CreatedAt:    profile.CreatedAt,
			UpdatedAt:    profile.UpdatedAt,
		})
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
