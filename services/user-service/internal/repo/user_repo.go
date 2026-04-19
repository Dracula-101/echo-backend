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

	"user-service/internal/domain"
	userErrors "user-service/internal/errors"

	"shared/pkg/database"
	"shared/pkg/database/postgres"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
)

// ============================================================================
// Repository Definition
// ============================================================================

type userRepository struct {
	db  database.Database
	log logger.Logger
}

func NewUserRepository(db database.Database, log logger.Logger) UserRepository {
	if db == nil {
		panic("Database is required for UserRepository")
	}
	if log == nil {
		panic("Logger is required for UserRepository")
	}

	log.Info("Initializing UserRepository",
		logger.String("service", userErrors.ServiceName),
	)

	return &userRepository{
		db:  db,
		log: log,
	}
}

// ============================================================================
// Profile Operations
// ============================================================================

// Generate Unique Username
func (r *userRepository) GenerateUniqueUsername(ctx context.Context, baseUsername string) (*string, pkgErrors.AppError) {
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
			return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to check username existence")
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

	return nil, pkgErrors.FromError(fmt.Errorf("unable to generate unique username after %d attempts", maxAttempts), userErrors.ErrCodeUsernameGenerationFailed, "Failed to generate unique username")
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
func (r *userRepository) GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, pkgErrors.AppError) {
	r.log.Debug("Fetching profile by user ID",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
	)

	// user_id has UNIQUE constraint, so LIMIT 1 is unnecessary
	query := `SELECT * FROM users.profiles WHERE user_id = $1`
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
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to get profile by user ID")
	}

	// Also load auth.users partial: SELECT email_verified, phone_verified, two_factor_enabled, account_status FROM auth.users WHERE id = $1.
	query = `SELECT email_verified, phone_verified, two_factor_enabled, account_status FROM auth.users WHERE id = $1`
	authRow := r.db.QueryRow(ctx, query, userID)
	var emailVerified, phoneVerified, twoFactorEnabled bool
	var accountStatus string
	err = authRow.Scan(&emailVerified, &phoneVerified, &twoFactorEnabled, &accountStatus)
	if err != nil {
		r.log.Error("Failed to get auth details for profile",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
			logger.String("query", query),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to get auth details for profile")
	}

	r.log.Debug("Profile fetched successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("profile_id", profile.ID),
	)

	return &domain.Profile{
		UserID:             profile.UserID,
		Username:           profile.Username,
		DisplayName:        profile.DisplayName,
		FirstName:          profile.FirstName,
		LastName:           profile.LastName,
		Bio:                profile.Bio,
		AvatarURL:          profile.AvatarURL,
		AvatarThumbnailURL: profile.AvatarThumbnailURL,
		LanguageCode:       profile.LanguageCode,
		Timezone:           profile.Timezone,
		CountryCode:        profile.CountryCode,
		PhoneVisible:       profile.PhoneVisible,
		EmailVisible:       profile.EmailVisible,
		OnlineStatus:       utils.Ptr(domain.OnlineStatus(profile.OnlineStatus)),
		LastSeenAt:         profile.LastSeenAt,
		ProfileVisibility:  domain.ProfileVisibility(profile.ProfileVisibility),
		SearchVisibility:   profile.SearchVisibility,
		IsVerified:         profile.IsVerified,
		PhoneVerified:      phoneVerified,
		EmailVerified:      emailVerified,
		TwoFactorEnabled:   twoFactorEnabled,
		AccountStatus:      domain.AccountStatus(accountStatus),
	}, nil
}

// GetProfileByUsername retrieves a user profile by username
func (r *userRepository) GetProfileByUsername(ctx context.Context, username string) (*domain.Profile, pkgErrors.AppError) {
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
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to get profile by username")
	}

	// Also load auth.users partial: SELECT email_verified, phone_verified, two_factor_enabled, account_status FROM auth.users WHERE id = $1.
	query = `SELECT email_verified, phone_verified, two_factor_enabled, account_status FROM auth.users WHERE id = $1`
	authRow := r.db.QueryRow(ctx, query, profile.UserID)
	var emailVerified, phoneVerified, twoFactorEnabled bool
	var accountStatus string
	err = authRow.Scan(&emailVerified, &phoneVerified, &twoFactorEnabled, &accountStatus)
	if err != nil {
		r.log.Error("Failed to get auth details for profile",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", profile.UserID),
			logger.Error(err),
			logger.String("query", query),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to get auth details for profile")
	}

	r.log.Debug("Profile fetched successfully by username",
		logger.String("service", userErrors.ServiceName),
		logger.String("username", username),
		logger.String("profile_id", profile.ID),
	)

	return &domain.Profile{
		UserID:            profile.UserID,
		Username:          profile.Username,
		DisplayName:       profile.DisplayName,
		FirstName:         profile.FirstName,
		LastName:          profile.LastName,
		Bio:               profile.Bio,
		AvatarURL:         profile.AvatarURL,
		LanguageCode:      profile.LanguageCode,
		Timezone:          profile.Timezone,
		CountryCode:       profile.CountryCode,
		PhoneVisible:      profile.PhoneVisible,
		EmailVisible:      profile.EmailVisible,
		OnlineStatus:      utils.Ptr(domain.OnlineStatus(profile.OnlineStatus)),
		LastSeenAt:        profile.LastSeenAt,
		ProfileVisibility: domain.ProfileVisibility(profile.ProfileVisibility),
		SearchVisibility:  profile.SearchVisibility,
		IsVerified:        profile.IsVerified,
		PhoneVerified:     phoneVerified,
		EmailVerified:     emailVerified,
		TwoFactorEnabled:  twoFactorEnabled,
		AccountStatus:     domain.AccountStatus(accountStatus),
	}, nil
}

// CreateProfile creates a new user profile
type CreateProfileInput struct {
	DisplayName       *string
	FirstName         *string
	LastName          *string
	Bio               *string
	LanguageCode      *string
	Timezone          *string
	CountryCode       *string
	City              *string
	PhoneVisible      *bool
	EmailVisible      *bool
	ProfileVisibility *string
	SearchVisibility  *bool
}

func (r *userRepository) CreateProfile(ctx context.Context, userId string, profile CreateProfileInput) (*domain.Profile, pkgErrors.AppError) {
	r.log.Info("Creating new profile",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userId),
	)

	userName, err := r.GenerateUniqueUsername(ctx, *profile.DisplayName)
	if err != nil {
		r.log.Error("Failed to generate unique username",
			logger.String("service", userErrors.ServiceName),
			logger.String("display_name", *profile.DisplayName),
			logger.Error(err),
		)
		return nil, err
	}

	var profileModel = dbModels.Profile{
		UserID:            userId,
		Username:          *userName,
		DisplayName:       profile.DisplayName,
		FirstName:         profile.FirstName,
		LastName:          profile.LastName,
		Bio:               profile.Bio,
		LanguageCode:      *utils.SafePtrString(profile.LanguageCode),
		OnlineStatus:      dbModels.OnlineStatusOffline,
		ProfileVisibility: dbModels.ProfileVisibility(*utils.SafePtrString(profile.ProfileVisibility)),
		SearchVisibility:  *utils.SafePtrBool(profile.SearchVisibility),
		Timezone:          profile.Timezone,
		City:              profile.City,
		CountryCode:       profile.CountryCode,
		PhoneVisible:      *utils.SafePtrBool(profile.PhoneVisible),
		EmailVisible:      *utils.SafePtrBool(profile.EmailVisible),
		CreatedAt:         utils.Ptr(time.Now()),
		UpdatedAt:         utils.Ptr(time.Now()),
	}
	id, dbErr := r.db.Insert(ctx, &profileModel)
	if dbErr != nil {
		r.log.Error("Failed to create profile",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userId),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "Failed to create profile")
	}

	createdProfile, err := r.GetProfileByUserID(ctx, userId)
	if err != nil {
		r.log.Error("Failed to retrieve created profile",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userId),
			logger.String("profile_id", *id),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "Failed to retrieve created profile")
	}
	if createdProfile == nil {
		r.log.Error("Created profile not found",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userId),
		)
		return nil, pkgErrors.FromError(fmt.Errorf("created profile not found"), userErrors.ErrCodeProfileNotFound, "Created profile not found")
	}

	r.log.Info("Profile created successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userId),
		logger.String("profile_id", createdProfile.UserID),
	)
	return createdProfile, nil
}

// UpdateProfile updates a user profile
type UpdateProfileParams struct {
	DisplayName        *string
	FirstName          *string
	LastName           *string
	Bio                *string
	BioLinks           *[]string
	DateOfBirth        *time.Time
	AvatarURL          *string
	AvatarThumbnailURL *string
	Gender             *string
	Pronouns           *string
	LanguageCode       *string
	Timezone           *string
	CountryCode        *string
	City               *string
	WebsiteURL         *string
	SocialLinks        *[]string
	Interests          *[]string
	PhoneVisible       *bool
	EmailVisible       *bool
	ProfileVisibility  *string
	SearchVisibility   *bool
}

func (r *userRepository) UpdateProfile(ctx context.Context, userId string, params UpdateProfileParams) (*domain.Profile, pkgErrors.AppError) {
	r.log.Info("Updating profile",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userId),
	)

	// Build dynamic SET clause
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argPos := 1

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
	if params.BioLinks != nil {
		bioLinksJSON, err := json.Marshal(*params.BioLinks)
		if err != nil {
			r.log.Error("Failed to marshal bio links",
				logger.String("service", userErrors.ServiceName),
				logger.String("user_id", userId),
				logger.Error(err),
			)
			return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to marshal bio links")
		}
		setClauses = append(setClauses, fmt.Sprintf("bio_links = $%d", argPos))
		args = append(args, string(bioLinksJSON))
		argPos++
	}
	if params.DateOfBirth != nil {
		setClauses = append(setClauses, fmt.Sprintf("date_of_birth = $%d", argPos))
		args = append(args, *params.DateOfBirth)
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
	if params.PhoneVisible != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone_visible = $%d", argPos))
		args = append(args, *params.PhoneVisible)
		argPos++
	}
	if params.EmailVisible != nil {
		setClauses = append(setClauses, fmt.Sprintf("email_visible = $%d", argPos))
		args = append(args, *params.EmailVisible)
		argPos++
	}
	if params.ProfileVisibility != nil {
		setClauses = append(setClauses, fmt.Sprintf("profile_visibility = $%d", argPos))
		args = append(args, *params.ProfileVisibility)
		argPos++
	}
	if params.SearchVisibility != nil {
		setClauses = append(setClauses, fmt.Sprintf("search_visibility = $%d", argPos))
		args = append(args, *params.SearchVisibility)
		argPos++
	}

	if len(setClauses) == 1 {
		r.log.Debug("No fields to update for profile",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userId),
		)
		return r.GetProfileByUserID(ctx, userId)
	}

	args = append(args, userId)
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
		logger.String("user_id", userId),
	)

	var profile dbModels.Profile
	if err := r.db.FindOneAndUpdate(ctx, &profile, query, args...); err != nil {
		if postgres.IsNotFoundError(err) {
			r.log.Debug("No profile updated - not found",
				logger.String("service", userErrors.ServiceName),
				logger.String("user_id", userId),
			)
			return nil, nil
		}
		r.log.Error("Failed to update profile",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userId),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to update profile")
	}

	var emailVerified, phoneVerified, twoFactorEnabled bool
	var accountStatus string
	authQuery := `SELECT email_verified, phone_verified, two_factor_enabled, account_status FROM auth.users WHERE id = $1 AND deactivated_at IS NULL`
	err := r.db.QueryRow(ctx, authQuery, profile.UserID).Scan(&emailVerified, &phoneVerified, &twoFactorEnabled, &accountStatus)
	if err != nil {
		r.log.Error("Failed to get auth details after profile update",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", profile.UserID),
			logger.Error(err),
			logger.String("query", authQuery),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to get auth details after profile update")
	}

	r.log.Info("Profile updated successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userId),
		logger.String("profile_id", profile.ID),
	)

	return &domain.Profile{
		UserID:             profile.UserID,
		Username:           profile.Username,
		DisplayName:        profile.DisplayName,
		FirstName:          profile.FirstName,
		LastName:           profile.LastName,
		Bio:                profile.Bio,
		AvatarURL:          profile.AvatarURL,
		AvatarThumbnailURL: profile.AvatarThumbnailURL,
		LanguageCode:       profile.LanguageCode,
		Timezone:           profile.Timezone,
		CountryCode:        profile.CountryCode,
		IsVerified:         profile.IsVerified,
		PhoneVisible:       profile.PhoneVisible,
		EmailVisible:       profile.EmailVisible,
		OnlineStatus:       utils.Ptr(domain.OnlineStatus(profile.OnlineStatus)),
		LastSeenAt:         profile.LastSeenAt,
		ProfileVisibility:  domain.ProfileVisibility(profile.ProfileVisibility),
		SearchVisibility:   profile.SearchVisibility,
		PhoneVerified:      emailVerified,
		EmailVerified:      emailVerified,
		TwoFactorEnabled:   twoFactorEnabled,
		AccountStatus:      domain.AccountStatus(accountStatus),
	}, nil
}

func (r *userRepository) IsBlocked(ctx context.Context, requesterID, targetID string) (bool, pkgErrors.AppError) {
	r.log.Debug("Checking if requester has blocked target",
		logger.String("service", userErrors.ServiceName),
		logger.String("requester_id", requesterID),
		logger.String("target_id", targetID),
	)

	query := `SELECT EXISTS (
		SELECT 1 FROM users.blocked_users
		WHERE user_id = $1
		AND blocked_user_id = $2
		AND unblocked_at IS NULL
	)`
	var isBlocked bool
	err := r.db.QueryRow(ctx, query, requesterID, targetID).Scan(&isBlocked)
	if err != nil {
		r.log.Error("Failed to check block status",
			logger.String("service", userErrors.ServiceName),
			logger.String("requester_id", requesterID),
			logger.String("target_id", targetID),
			logger.Error(err),
		)
		return false, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to check block status")
	}

	r.log.Debug("Block status checked successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("requester_id", requesterID),
		logger.String("target_id", targetID),
		logger.Bool("is_blocked", isBlocked),
	)

	return isBlocked, nil
}

func (r *userRepository) GetBlockedUsers(ctx context.Context, userID string) ([]string, pkgErrors.AppError) {
	r.log.Debug("Fetching blocked users",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `SELECT blocked_user_id FROM users.blocked_users WHERE user_id = $1 AND unblocked_at IS NULL`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to fetch blocked users",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to fetch blocked users")
	}
	defer rows.Close()

	var blockedUserIDs []string
	for rows.Next() {
		var blockedUserID string
		if err := rows.Scan(&blockedUserID); err != nil {
			r.log.Error("Failed to scan blocked user ID",
				logger.String("service", userErrors.ServiceName),
				logger.String("user_id", userID),
				logger.Error(err),
			)
			continue
		}
		blockedUserIDs = append(blockedUserIDs, blockedUserID)
	}

	r.log.Debug("Blocked users fetched successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int("blocked_count", len(blockedUserIDs)),
	)

	return blockedUserIDs, nil
}

func (r *userRepository) GetContacts(ctx context.Context, userID string) (*[]domain.Profile, pkgErrors.AppError) {
	r.log.Debug("Fetching contacts",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `SELECT contact_user_id FROM users.contacts WHERE user_id = $1 AND status = $2`
	rows, err := r.db.Query(ctx, query, userID, dbModels.ContactStatusActive)
	if err != nil {
		r.log.Error("Failed to fetch contact IDs",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to fetch contact IDs")
	}
	defer rows.Close()

	var contacts []domain.Profile
	for rows.Next() {
		var contactUserID string
		if err := rows.Scan(&contactUserID); err != nil {
			r.log.Error("Failed to scan contact user ID",
				logger.String("service", userErrors.ServiceName),
				logger.String("user_id", userID),
				logger.Error(err),
			)
			continue
		}

		profile, err := r.GetProfileByUserID(ctx, contactUserID)
		if err != nil {
			r.log.Error("Failed to fetch profile for contact",
				logger.String("service", userErrors.ServiceName),
				logger.String("user_id", userID),
				logger.String("contact_user_id", contactUserID),
				logger.Error(err),
			)
			continue
		}
		if profile != nil {
			contacts = append(contacts, *profile)
		}
	}

	r.log.Debug("Contacts fetched successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int("contact_count", len(contacts)),
	)

	return &contacts, nil
}

func (r *userRepository) GetSettings(ctx context.Context, userID string) (*domain.Settings, pkgErrors.AppError) {
	r.log.Debug("Fetching user settings",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `SELECT * FROM users.settings WHERE user_id = $1`
	row := r.db.QueryRow(ctx, query, userID)

	var settings dbModels.UserSettings
	err := row.ScanModel(&settings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Debug("Settings not found",
				logger.String("service", userErrors.ServiceName),
				logger.String("user_id", userID),
			)
			return nil, nil
		}
		r.log.Error("Failed to get user settings",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to get user settings")
	}

	r.log.Debug("User settings fetched successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
	)

	return &domain.Settings{
		ProfileVisibility:         domain.ProfileVisibility(settings.ProfileVisibility),
		LastSeenVisibility:        domain.ProfileVisibility(settings.LastSeenVisibility),
		OnlineStatusVisibility:    domain.ProfileVisibility(settings.OnlineStatusVisibility),
		ProfilePhotoVisibility:    domain.ProfileVisibility(settings.ProfilePhotoVisibility),
		AboutVisibility:           domain.ProfileVisibility(settings.AboutVisibility),
		ReadReceiptsEnabled:       settings.ReadReceiptsEnabled,
		TypingIndicatorsEnabled:   settings.TypingIndicatorsEnabled,
		PushNotificationsEnabled:  settings.PushNotificationsEnabled,
		EmailNotificationsEnabled: settings.EmailNotificationsEnabled,
		SmsNotificationsEnabled:   settings.SMSNotificationsEnabled,
		MessageNotifications:      settings.MessageNotifications,
		GroupMessageNotifications: settings.GroupMessageNotifications,
		MentionNotifications:      settings.MentionNotifications,
		ReactionNotifications:     settings.ReactionNotifications,
		CallNotifications:         settings.CallNotifications,
		NotificationSound:         domain.NotificationSound(settings.NotificationSound),
		VibrationEnabled:          settings.VibrationEnabled,
		NotificationPreview:       domain.NotificationPreview(settings.NotificationPreview),
		QuietHoursEnabled:         settings.QuietHoursEnabled,
		QuietHoursStart:           settings.QuietHoursStart,
		QuietHoursEnd:             settings.QuietHoursEnd,
		EnterKeyToSend:            settings.EnterKeyToSend,
		AutoDownloadPhotos:        settings.AutoDownloadPhotos,
		AutoDownloadVideos:        settings.AutoDownloadVideos,
		AutoDownloadDocuments:     settings.AutoDownloadDocuments,
		AutoDownloadOnWifiOnly:    settings.AutoDownloadOnWifiOnly,
		CompressImages:            settings.CompressImages,
		SaveToGallery:             settings.SaveToGallery,
		ChatBackupEnabled:         settings.ChatBackupEnabled,
		ChatBackupFrequency:       domain.BackupFrequency(settings.ChatBackupFrequency),
		ScreenLockEnabled:         settings.ScreenLockEnabled,
		ScreenLockTimeout:         settings.ScreenLockTimeout,
		FingerprintUnlock:         settings.FingerprintUnlock,
		FaceUnlock:                settings.FaceUnlock,
		ShowSecurityNotifications: settings.ShowSecurityNotifications,
		Theme:                     domain.Theme(settings.Theme),
		FontSize:                  domain.FontSize(settings.FontSize),
		ChatWallpaper:             settings.ChatWallpaper,
		UseSystemEmoji:            settings.UseSystemEmoji,
		LanguageCode:              settings.LanguageCode,
		Timezone:                  settings.Timezone,
		DateFormat:                domain.DateFormat(settings.DateFormat),
		TimeFormat:                domain.TimeFormat(settings.TimeFormat),
		LowDataMode:               settings.LowDataMode,
	}, nil
}

func (r *userRepository) GetPrivacyOverrides(ctx context.Context, userID string, targetUserID string) (*domain.PrivacyOverride, pkgErrors.AppError) {
	r.log.Debug("Fetching user privacy overrides",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `SELECT * FROM users.privacy_overrides WHERE user_id = $1 AND target_user_id = $2`
	row := r.db.QueryRow(ctx, query, userID, targetUserID)

	var overrides dbModels.PrivacyOverride
	err := row.ScanModel(&overrides)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Debug("Privacy override not found",
				logger.String("service", userErrors.ServiceName),
				logger.String("user_id", userID),
				logger.String("target_user_id", targetUserID),
			)
			return nil, nil
		}
		r.log.Error("Failed to scan user privacy override",
			logger.String("service", userErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to scan user privacy override")
	}

	return &domain.PrivacyOverride{
		ID:                      overrides.ID,
		UserID:                  overrides.UserID,
		TargetUserID:            overrides.TargetUserID,
		LastSeenVisible:         overrides.LastSeenVisible,
		OnlineStatusVisible:     overrides.OnlineStatusVisible,
		ProfilePhotoVisible:     overrides.ProfilePhotoVisible,
		AboutVisible:            overrides.AboutVisible,
		ReadReceiptsEnabled:     overrides.ReadReceiptsEnabled,
		TypingIndicatorsEnabled: overrides.TypingIndicatorsEnabled,
		CreatedAt:               overrides.CreatedAt,
		UpdatedAt:               overrides.UpdatedAt,
	}, nil
}

func (r *userRepository) AddUserDevice(ctx context.Context, input *domain.UserDevice, isCurrentDevice bool) pkgErrors.AppError {
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
		return pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to add user device")
	}

	r.log.Info("User device added successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", input.UserID),
		logger.String("device_id", input.DeviceID),
	)

	return nil
}

func (r *userRepository) GetUserDevices(ctx context.Context, userID string) ([]*domain.UserDevice, pkgErrors.AppError) {
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
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to fetch user devices")
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

func (r *userRepository) UpdateUserDevice(ctx context.Context, input *domain.UpdateUserDevice, userID string, deviceID string) pkgErrors.AppError {
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
		return pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to update user device")
	}

	r.log.Info("User device updated successfully",
		logger.String("service", userErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
	)

	return nil
}

// SearchProfiles searches for profiles by query
func (r *userRepository) SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int, pkgErrors.AppError) {
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
		return nil, 0, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to search profiles")
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
			UserID:             profile.UserID,
			Username:           profile.Username,
			DisplayName:        profile.DisplayName,
			FirstName:          profile.FirstName,
			LastName:           profile.LastName,
			Bio:                profile.Bio,
			AvatarURL:          profile.AvatarURL,
			AvatarThumbnailURL: profile.AvatarThumbnailURL,
			LanguageCode:       profile.LanguageCode,
			Timezone:           profile.Timezone,
			PhoneVisible:       profile.PhoneVisible,
			EmailVisible:       profile.EmailVisible,
			OnlineStatus:       utils.Ptr(domain.OnlineStatus(profile.OnlineStatus)),
			LastSeenAt:         profile.LastSeenAt,
			ProfileVisibility:  domain.ProfileVisibility(profile.ProfileVisibility),
			SearchVisibility:   profile.SearchVisibility,
			IsVerified:         profile.IsVerified,
			CountryCode:        profile.CountryCode,
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
func (r *userRepository) UsernameExists(ctx context.Context, username string) (bool, pkgErrors.AppError) {
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
		return false, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to check username existence")
	}

	r.log.Debug("Username existence check completed",
		logger.String("service", userErrors.ServiceName),
		logger.String("username", username),
		logger.Bool("exists", exists),
	)

	return exists, nil
}
