package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
	return &userRepository{db: db, log: log}
}

func (r *userRepository) GetProfileByUserID(ctx context.Context, userID string, requesterUserId *string) (*domain.Profile, pkgErrors.AppError) {
	row := r.db.QueryRow(ctx, `SELECT * FROM users.profiles WHERE user_id = $1`, userID)

	var profile dbModels.Profile
	if err := row.ScanModel(&profile); err != nil {
		if postgres.IsNotFoundError(err) {
			return nil, nil
		}
		r.log.Error("Failed to get profile by user ID",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to get profile by user ID")
	}

	var emailVerified, phoneVerified, twoFactorEnabled bool
	var accountStatus string
	authErr := r.db.QueryRow(ctx,
		`SELECT email_verified, phone_verified, two_factor_enabled, account_status FROM auth.users WHERE id = $1`,
		userID,
	).Scan(&emailVerified, &phoneVerified, &twoFactorEnabled, &accountStatus)
	if authErr != nil {
		r.log.Error("Failed to get auth details for profile",
			logger.String("user_id", userID),
			logger.Error(authErr),
		)
		return nil, pkgErrors.FromError(authErr, userErrors.ErrCodeDatabaseError, "Failed to get auth details for profile")
	}

	var isContact *bool
	if requesterUserId != nil && *requesterUserId != userID {
		contactErr := r.db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM users.contacts WHERE user_id = $1 AND contact_user_id = $2)`,
			*requesterUserId, userID,
		).Scan(&isContact)
		if contactErr != nil {
			r.log.Error("Failed to check if profile is contact",
				logger.String("requester_user_id", *requesterUserId),
				logger.String("user_id", userID),
				logger.Error(contactErr),
			)
			isContact = nil
		}
	}

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
		DeactivatedAt:      profile.DeactivatedAt,
		BioLinks: func() *[]string {
			if profile.BioLinks == nil {
				return nil
			}
			var links []string
			if err := json.Unmarshal([]byte(*profile.BioLinks), &links); err != nil {
				r.log.Error("Failed to unmarshal bio links",
					logger.String("user_id", profile.UserID),
					logger.Error(err),
				)
				return nil
			}
			return &links
		}(),
		IsContact: isContact,
	}, nil
}

func (r *userRepository) GetProfileByUserIDs(ctx context.Context, userIDs []string, requesterUserId *string) (map[string]*domain.Profile, pkgErrors.AppError) {
	if len(userIDs) == 0 {
		return map[string]*domain.Profile{}, nil
	}
	rows, err := r.db.Query(ctx, `SELECT * FROM users.profiles WHERE user_id = ANY($1)`, userIDs)
	if err != nil {
		r.log.Error("Failed to get profiles by user IDs",
			logger.Strings("user_ids", userIDs),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to get profiles by user IDs")
	}
	defer rows.Close()

	profiles := make(map[string]*domain.Profile)
	for rows.Next() {
		var profile dbModels.Profile
		if err := rows.ScanModel(&profile); err != nil {
			r.log.Error("Failed to scan profile row",
				logger.Error(err),
			)
			return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to scan profile row")
		}

		profiles[profile.UserID] = &domain.Profile{
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
			DeactivatedAt:      profile.DeactivatedAt,
		}
	}
	if err := rows.Err(); err != nil {
		r.log.Error("Error iterating over profile rows",
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Error iterating over profile rows")
	}
	return profiles, nil
}

func (r *userRepository) GetProfileByUsername(ctx context.Context, username string) (*domain.Profile, pkgErrors.AppError) {
	row := r.db.QueryRow(ctx,
		`SELECT * FROM users.profiles WHERE username = $1 AND deactivated_at IS NULL`,
		username,
	)

	var profile dbModels.Profile
	if err := row.ScanModel(&profile); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.log.Error("Failed to get profile by username",
			logger.String("username", username),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to get profile by username")
	}

	var emailVerified, phoneVerified, twoFactorEnabled bool
	var accountStatus string
	authErr := r.db.QueryRow(ctx,
		`SELECT email_verified, phone_verified, two_factor_enabled, account_status FROM auth.users WHERE id = $1`,
		profile.UserID,
	).Scan(&emailVerified, &phoneVerified, &twoFactorEnabled, &accountStatus)
	if authErr != nil {
		r.log.Error("Failed to get auth details for profile",
			logger.String("user_id", profile.UserID),
			logger.Error(authErr),
		)
		return nil, pkgErrors.FromError(authErr, userErrors.ErrCodeDatabaseError, "Failed to get auth details for profile")
	}

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

func (r *userRepository) GetProfileByPhone(ctx context.Context, phone string) (*domain.Profile, pkgErrors.AppError) {
	row := r.db.QueryRow(ctx,
		`SELECT * FROM users.profiles WHERE phone_visible = true AND deactivated_at IS NULL AND user_id IN (SELECT id FROM auth.users WHERE phone = $1)`,
		phone,
	)
	var profile dbModels.Profile
	err := row.ScanModel(&profile)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.log.Error("Failed to get profile by phone",
			logger.String("phone", phone),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to get profile by phone")
	}
	var emailVerified, phoneVerified, twoFactorEnabled bool
	var accountStatus string

	authErr := r.db.QueryRow(ctx, `SELECT email_verified, phone_verified, two_factor_enabled, account_status FROM auth.users WHERE id = $1`, profile.UserID).Scan(&emailVerified, &phoneVerified, &twoFactorEnabled, &accountStatus)
	if authErr != nil {
		r.log.Error("Failed to get auth details for profile",
			logger.String("user_id", profile.UserID),
			logger.Error(authErr),
		)
		return nil, pkgErrors.FromError(authErr, userErrors.ErrCodeDatabaseError, "Failed to get auth details for profile")
	}
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

type CreateProfileInput struct {
	UserName          string
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

	profileModel := dbModels.Profile{
		UserID:            userId,
		Username:          profile.UserName,
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
			logger.String("user_id", userId),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "Failed to create profile")
	}

	created, err := r.GetProfileByUserID(ctx, userId, nil)
	if err != nil {
		r.log.Error("Failed to retrieve created profile",
			logger.String("user_id", userId),
			logger.String("profile_id", *id),
			logger.Error(err),
		)
		return nil, err
	}
	if created == nil {
		return nil, pkgErrors.FromError(
			fmt.Errorf("created profile not found"),
			userErrors.ErrCodeProfileNotFound,
			"Created profile not found",
		)
	}
	return created, nil
}

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
	setClauses := []string{"updated_at = NOW()"}
	args := []any{}
	argPos := 1

	addField := func(col string, val any) {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, argPos))
		args = append(args, val)
		argPos++
	}

	if params.DisplayName != nil {
		addField("display_name", *params.DisplayName)
	}
	if params.FirstName != nil {
		addField("first_name", *params.FirstName)
	}
	if params.LastName != nil {
		addField("last_name", *params.LastName)
	}
	if params.Bio != nil {
		addField("bio", *params.Bio)
	}
	if params.BioLinks != nil {
		bioLinksJSON, err := json.Marshal(*params.BioLinks)
		if err != nil {
			return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to marshal bio links")
		}
		addField("bio_links", string(bioLinksJSON))
	}
	if params.DateOfBirth != nil {
		addField("date_of_birth", *params.DateOfBirth)
	}
	if params.AvatarURL != nil {
		addField("avatar_url", *params.AvatarURL)
	}
	if params.AvatarThumbnailURL != nil {
		addField("avatar_thumbnail_url", *params.AvatarThumbnailURL)
	}
	if params.LanguageCode != nil {
		addField("language_code", *params.LanguageCode)
	}
	if params.Timezone != nil {
		addField("timezone", *params.Timezone)
	}
	if params.CountryCode != nil {
		addField("country_code", *params.CountryCode)
	}
	if params.PhoneVisible != nil {
		addField("phone_visible", *params.PhoneVisible)
	}
	if params.EmailVisible != nil {
		addField("email_visible", *params.EmailVisible)
	}
	if params.ProfileVisibility != nil {
		addField("profile_visibility", *params.ProfileVisibility)
	}
	if params.SearchVisibility != nil {
		addField("search_visibility", *params.SearchVisibility)
	}

	if len(setClauses) == 1 {
		return r.GetProfileByUserID(ctx, userId, nil)
	}

	args = append(args, userId)
	query := fmt.Sprintf(`
		UPDATE users.profiles SET %s
		WHERE user_id = $%d AND deactivated_at IS NULL
		RETURNING *`,
		strings.Join(setClauses, ", "),
		argPos,
	)

	var profile dbModels.Profile
	if err := r.db.FindOneAndUpdate(ctx, &profile, query, args...); err != nil {
		if postgres.IsNotFoundError(err) {
			return nil, nil
		}
		r.log.Error("Failed to update profile",
			logger.String("user_id", userId),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to update profile")
	}

	var emailVerified, phoneVerified, twoFactorEnabled bool
	var accountStatus string
	authErr := r.db.QueryRow(ctx,
		`SELECT email_verified, phone_verified, two_factor_enabled, account_status FROM auth.users WHERE id = $1 AND deactivated_at IS NULL`,
		profile.UserID,
	).Scan(&emailVerified, &phoneVerified, &twoFactorEnabled, &accountStatus)
	if authErr != nil {
		r.log.Error("Failed to get auth details after profile update",
			logger.String("user_id", profile.UserID),
			logger.Error(authErr),
		)
		return nil, pkgErrors.FromError(authErr, userErrors.ErrCodeDatabaseError, "Failed to get auth details after profile update")
	}

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
		PhoneVerified:      phoneVerified,
		EmailVerified:      emailVerified,
		TwoFactorEnabled:   twoFactorEnabled,
		AccountStatus:      domain.AccountStatus(accountStatus),
	}, nil
}

func (r *userRepository) SearchProfilesByUsername(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int64, pkgErrors.AppError) {
	searchQuery := fmt.Sprintf("%%%s%%", query)
	rows, err := r.db.Query(ctx,
		`SELECT * FROM users.profiles WHERE username ILIKE $1 AND deactivated_at IS NULL AND search_visibility = true ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		searchQuery, limit, offset,
	)
	if err != nil {
		r.log.Error("Failed to search profiles by username",
			logger.String("query", query),
			logger.Int("limit", limit),
			logger.Int("offset", offset),
			logger.Error(err),
		)
		return nil, 0, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to search profiles by username")
	}
	defer rows.Close()

	profiles := []*domain.Profile{}
	for rows.Next() {
		var profile dbModels.Profile
		if err := rows.ScanModel(&profile); err != nil {
			r.log.Error("Failed to scan profile row",
				logger.Error(err),
			)
			return nil, 0, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to scan profile row")
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
			CountryCode:        profile.CountryCode,
			PhoneVisible:       profile.PhoneVisible,
			EmailVisible:       profile.EmailVisible,
			OnlineStatus:       utils.Ptr(domain.OnlineStatus(profile.OnlineStatus)),
			LastSeenAt:         profile.LastSeenAt,
			ProfileVisibility:  domain.ProfileVisibility(profile.ProfileVisibility),
			SearchVisibility:   profile.SearchVisibility,
			IsVerified:         profile.IsVerified,
		})
	}

	var total int64
	countErr := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users.profiles WHERE username ILIKE $1 AND deactivated_at IS NULL`,
		searchQuery,
	).Scan(&total)
	if countErr != nil {
		r.log.Error("Failed to count total profiles for username search",
			logger.String("query", query),
			logger.Error(countErr),
		)
		return nil, 0, pkgErrors.FromError(countErr, userErrors.ErrCodeDatabaseError, "Failed to count total profiles for username search")
	}

	return profiles, total, nil
}

func (r *userRepository) SearchProfilesByFullName(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int64, pkgErrors.AppError) {
	searchQuery := fmt.Sprintf("%%%s%%", query)
	rows, err := r.db.Query(ctx,
		`SELECT * FROM users.profiles WHERE (first_name || ' ' || last_name) ILIKE $1 AND deactivated_at IS NULL AND search_visibility = true ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		searchQuery, limit, offset,
	)
	if err != nil {
		r.log.Error("Failed to search profiles by full name",
			logger.String("query", query),
			logger.Int("limit", limit),
			logger.Int("offset", offset),
			logger.Error(err),
		)
		return nil, 0, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to search profiles by full name")
	}
	defer rows.Close()

	profiles := []*domain.Profile{}
	for rows.Next() {
		var profile dbModels.Profile
		if err := rows.ScanModel(&profile); err != nil {
			r.log.Error("Failed to scan profile row",
				logger.Error(err),
			)
			return nil, 0, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to scan profile row")
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
			CountryCode:        profile.CountryCode,
			PhoneVisible:       profile.PhoneVisible,
			EmailVisible:       profile.EmailVisible,
			OnlineStatus:       utils.Ptr(domain.OnlineStatus(profile.OnlineStatus)),
			LastSeenAt:         profile.LastSeenAt,
			ProfileVisibility:  domain.ProfileVisibility(profile.ProfileVisibility),
			SearchVisibility:   profile.SearchVisibility,
			IsVerified:         profile.IsVerified,
		})
	}

	var total int64
	countErr := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users.profiles WHERE (first_name || ' ' || last_name) ILIKE $1 AND deactivated_at IS NULL`,
		searchQuery,
	).Scan(&total)
	if countErr != nil {
		r.log.Error("Failed to count total profiles for full name search",
			logger.String("query", query),
			logger.Error(countErr),
		)
		return nil, 0, pkgErrors.FromError(countErr, userErrors.ErrCodeDatabaseError, "Failed to count total profiles for full name search")
	}

	return profiles, total, nil
}

func (r *userRepository) SearchProfilesByEmail(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int64, pkgErrors.AppError) {
	rows, err := r.db.Query(ctx,
		`SELECT p.* FROM users.profiles p JOIN auth.users u ON p.user_id = u.id WHERE u.email ILIKE $1 AND p.deactivated_at IS NULL AND p.search_visibility = true ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`,
		fmt.Sprintf("%%%s%%", query), limit, offset,
	)
	if err != nil {
		r.log.Error("Failed to search profiles by email",
			logger.String("query", query),
			logger.Int("limit", limit),
			logger.Int("offset", offset),
			logger.Error(err),
		)
		return nil, 0, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to search profiles by email")
	}
	defer rows.Close()

	profiles := []*domain.Profile{}
	for rows.Next() {
		var profile dbModels.Profile
		if err := rows.ScanModel(&profile); err != nil {
			r.log.Error("Failed to scan profile row",
				logger.Error(err),
			)
			return nil, 0, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to scan profile row")
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
			CountryCode:        profile.CountryCode,
			PhoneVisible:       profile.PhoneVisible,
			EmailVisible:       profile.EmailVisible,
			OnlineStatus:       utils.Ptr(domain.OnlineStatus(profile.OnlineStatus)),
			LastSeenAt:         profile.LastSeenAt,
			ProfileVisibility:  domain.ProfileVisibility(profile.ProfileVisibility),
			SearchVisibility:   profile.SearchVisibility,
			IsVerified:         profile.IsVerified,
		})
	}

	var total int64
	countErr := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users.profiles p JOIN auth.users u ON p.user_id = u.id WHERE u.email ILIKE $1 AND p.deactivated_at IS NULL`,
		fmt.Sprintf("%%%s%%", query),
	).Scan(&total)
	if countErr != nil {
		r.log.Error("Failed to count total profiles for email search",
			logger.String("query", query),
			logger.Error(countErr),
		)
		return nil, 0, pkgErrors.FromError(countErr, userErrors.ErrCodeDatabaseError, "Failed to count total profiles for email search")
	}

	return profiles, total, nil
}

func (r *userRepository) SearchProfilesByPhone(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int64, pkgErrors.AppError) {
	rows, err := r.db.Query(ctx,
		`SELECT p.* FROM users.profiles p JOIN auth.users u ON p.user_id = u.id WHERE u.phone ILIKE $1 AND p.deactivated_at IS NULL AND p.search_visibility = true ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`,
		fmt.Sprintf("%%%s%%", query), limit, offset,
	)
	if err != nil {
		r.log.Error("Failed to search profiles by phone",
			logger.String("query", query),
			logger.Int("limit", limit),
			logger.Int("offset", offset),
			logger.Error(err),
		)
		return nil, 0, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to search profiles by phone")
	}
	defer rows.Close()

	profiles := []*domain.Profile{}
	for rows.Next() {
		var profile dbModels.Profile
		if err := rows.ScanModel(&profile); err != nil {
			r.log.Error("Failed to scan profile row",
				logger.Error(err),
			)
			return nil, 0, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to scan profile row")
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
			CountryCode:        profile.CountryCode,
			PhoneVisible:       profile.PhoneVisible,
			EmailVisible:       profile.EmailVisible,
			OnlineStatus:       utils.Ptr(domain.OnlineStatus(profile.OnlineStatus)),
			LastSeenAt:         profile.LastSeenAt,
			ProfileVisibility:  domain.ProfileVisibility(profile.ProfileVisibility),
			SearchVisibility:   profile.SearchVisibility,
			IsVerified:         profile.IsVerified,
		})
	}

	var total int64
	countErr := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users.profiles p JOIN auth.users u ON p.user_id = u.id WHERE u.phone ILIKE $1 AND p.deactivated_at IS NULL`,
		fmt.Sprintf("%%%s%%", query),
	).Scan(&total)
	if countErr != nil {
		r.log.Error("Failed to count total profiles for phone search",
			logger.String("query", query),
			logger.Error(countErr),
		)
		return nil, 0, pkgErrors.FromError(countErr, userErrors.ErrCodeDatabaseError, "Failed to count total profiles for phone search")
	}

	return profiles, total, nil
}

func (r *userRepository) UsernameExists(ctx context.Context, username string) (bool, pkgErrors.AppError) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users.profiles WHERE username = $1 AND deactivated_at IS NULL)`,
		username,
	).Scan(&exists)
	if err != nil {
		r.log.Error("Failed to check username existence",
			logger.String("username", username),
			logger.Error(err),
		)
		return false, pkgErrors.FromError(err, userErrors.ErrCodeDatabaseError, "Failed to check username existence")
	}
	return exists, nil
}
