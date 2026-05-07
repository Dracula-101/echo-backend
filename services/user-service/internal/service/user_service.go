package service

import (
	"context"
	"time"

	"user-service/internal/domain"
	repository "user-service/internal/repo"
	"user-service/internal/repo/blocked_users"
	"user-service/internal/repo/contacts"
	"user-service/internal/repo/privacy_overrides"
	"user-service/internal/repo/settings"
	"user-service/internal/service/cache"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
)

type userService struct {
	userRepo     repository.UserRepository
	blockedRepo  blocked_users.BlockedRepository
	contactRepo  contacts.ContactRepository
	settingsRepo settings.SettingsRepository
	privacyRepo  privacy_overrides.PrivacyOverrideRepository
	cache        cache.UserCache
	log          logger.Logger
}

func NewUserService(
	userRepo repository.UserRepository,
	blockedRepo blocked_users.BlockedRepository,
	contactRepo contacts.ContactRepository,
	settingsRepo settings.SettingsRepository,
	privacyRepo privacy_overrides.PrivacyOverrideRepository,
	cache cache.UserCache,
	log logger.Logger,
) *userService {
	return &userService{
		userRepo:     userRepo,
		blockedRepo:  blockedRepo,
		contactRepo:  contactRepo,
		settingsRepo: settingsRepo,
		privacyRepo:  privacyRepo,
		cache:        cache,
		log:          log,
	}
}

func (s *userService) UsernameExists(ctx context.Context, username string) (bool, pkgErrors.AppError) {
	s.log.Info("Checking if username exists",
		logger.String("username", username),
	)

	// check cache first
	cachedUserID, cacheErr := s.cache.GetUserIDByUsername(ctx, username)
	if cacheErr != nil {
		s.log.Error("Cache error while checking username existence",
			logger.String("username", username),
			logger.Error(cacheErr),
		)
	} else if cachedUserID != nil {
		// cachedUserID == "" indicates a cached negative (non-existent) result
		if *cachedUserID == "" {
			s.log.Info("Username cached as non-existent",
				logger.String("username", username),
			)
		} else {
			s.log.Info("Username exists in cache",
				logger.String("username", username),
				logger.String("user_id", *cachedUserID),
			)
			return true, nil
		}
	}

	exists, err := s.userRepo.UsernameExists(ctx, username)
	if err != nil {
		s.log.Error("Failed to check if username exists",
			logger.String("username", username),
			logger.Error(err),
		)
		return false, err
	}

	// Cache the negative result to prevent cache penetration, with a short TTL
	if !exists {
		cacheErr := s.cache.SetUserIDByUsername(ctx, username, "")
		if cacheErr != nil {
			s.log.Error("Cache error while setting non-existent username",
				logger.String("username", username),
				logger.Error(cacheErr),
			)
		}
	}

	s.log.Info("Username existence check completed",
		logger.String("username", username),
		logger.Bool("exists", exists),
	)

	return exists, nil
}

func (s *userService) GetProfile(ctx context.Context, userID string, requesterUserId *string) (*domain.Profile, pkgErrors.AppError) {
	s.log.Info("Getting user profile",
		logger.String("user_id", userID),
	)
	cachedProfile, cacheErr := s.cache.GetProfile(ctx, userID)
	if cacheErr != nil {
		s.log.Error("Failed to get cached profile",
			logger.String("user_id", userID),
			logger.Error(cacheErr),
		)
	}
	if cachedProfile != nil {
		return cachedProfile, nil
	}

	profile, err := s.userRepo.GetProfileByUserID(ctx, userID, requesterUserId)
	if err != nil {
		s.log.Error("Failed to get profile",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, err
	}

	if profile == nil {
		return nil, nil
	}

	// Cache the profile for future requests
	err = s.cache.SetProfile(ctx, profile.UserID, profile)
	if err != nil {
		s.log.Error("Failed to cache profile",
			logger.String("user_id", profile.UserID),
			logger.Error(err),
		)
	}

	err = s.cache.SetUserIDByUsername(ctx, *profile.DisplayName, profile.UserID)
	if err != nil {
		s.log.Error("Failed to cache username to userID mapping",
			logger.String("user_id", profile.UserID),
			logger.String("username", *profile.DisplayName),
			logger.Error(err),
		)
	}

	return profile, nil
}
func (s *userService) GetProfileByUsername(ctx context.Context, viewerID, username string) (*domain.Profile, pkgErrors.AppError) {
	s.log.Info("getting profile by username", logger.String("username", username))

	// step 1: username → userID
	subjectID, cacheErr := s.cache.GetUserIDByUsername(ctx, username)
	if cacheErr != nil {
		s.log.Error("cache get username mapping failed", logger.String("username", username), logger.Error(cacheErr))
	}
	if subjectID == nil {
		p, err := s.userRepo.GetProfileByUsername(ctx, username)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, pkgErrors.New(pkgErrors.CodeNotFound, "user not found")
		}
		if setErr := s.cache.SetUserIDByUsername(ctx, username, p.UserID); setErr != nil {
			s.log.Error("cache set username mapping failed", logger.String("username", username), logger.Error(setErr))
		}
		subjectID = &p.UserID
	}

	// step 2: load profile
	profile, cacheErr := s.cache.GetProfile(ctx, *subjectID)
	if cacheErr != nil {
		s.log.Error("cache get profile failed", logger.String("user_id", *subjectID), logger.Error(cacheErr))
	}
	if profile == nil {
		p, err := s.userRepo.GetProfileByUserID(ctx, *subjectID, &viewerID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, pkgErrors.New(pkgErrors.CodeNotFound, "user not found")
		}
		profile = p
		if setErr := s.cache.SetProfile(ctx, *subjectID, profile); setErr != nil {
			s.log.Error("cache set profile failed", logger.String("user_id", *subjectID), logger.Error(setErr))
		}
	}

	// step 3: privacy cache
	privacy, cacheErr := s.cache.GetPrivacy(ctx, viewerID, *subjectID)
	if cacheErr != nil {
		s.log.Error("cache get privacy failed", logger.String("viewer_id", viewerID), logger.String("subject_id", *subjectID), logger.Error(cacheErr))
	}

	if privacy == nil {
		// step 4: viewer block list
		viewerBlockedIDs, cacheErr := s.cache.GetBlockedIDs(ctx, viewerID)
		if cacheErr != nil {
			s.log.Error("cache get blocked IDs failed", logger.String("user_id", viewerID), logger.Error(cacheErr))
		}
		if viewerBlockedIDs == nil {
			ids, err := s.blockedRepo.GetBlockedUserIDs(ctx, viewerID)
			if err != nil {
				return nil, err
			}
			if ids != nil {
				viewerBlockedIDs = &ids
				if setErr := s.cache.SetBlockedIDs(ctx, viewerID, ids); setErr != nil {
					s.log.Error("cache set blocked IDs failed", logger.String("user_id", viewerID), logger.Error(setErr))
				}
			}
		}

		// step 5: subject block list
		subjectBlockedIDs, cacheErr := s.cache.GetBlockedIDs(ctx, *subjectID)
		if cacheErr != nil {
			s.log.Error("cache get blocked IDs failed", logger.String("user_id", *subjectID), logger.Error(cacheErr))
		}
		if subjectBlockedIDs == nil {
			ids, err := s.blockedRepo.GetBlockedUserIDs(ctx, *subjectID)
			if err != nil {
				return nil, err
			}
			subjectBlockedIDs = &ids
			if setErr := s.cache.SetBlockedIDs(ctx, *subjectID, ids); setErr != nil {
				s.log.Error("cache set blocked IDs failed", logger.String("user_id", *subjectID), logger.Error(setErr))
			}
		}

		if viewerBlockedIDs != nil && utils.ContainsString(*viewerBlockedIDs, *subjectID) {
			return nil, pkgErrors.New(pkgErrors.CodeNotFound, "user not found")
		}
		if subjectBlockedIDs != nil && utils.ContainsString(*subjectBlockedIDs, viewerID) {
			return nil, pkgErrors.New(pkgErrors.CodeNotFound, "user not found")
		}

		// step 6: subject contact IDs (check if viewer is in subject's contacts)
		subjectContactIDs, cacheErr := s.cache.GetContactIDs(ctx, *subjectID)
		if cacheErr != nil {
			s.log.Error("cache get contact IDs failed", logger.String("user_id", *subjectID), logger.Error(cacheErr))
		}
		if subjectContactIDs == nil {
			ids, err := s.contactRepo.GetContactIDs(ctx, *subjectID)
			if err != nil {
				return nil, err
			}
			subjectContactIDs = &ids
			if setErr := s.cache.SetContactIDs(ctx, *subjectID, ids); setErr != nil {
				s.log.Error("cache set contact IDs failed", logger.String("user_id", *subjectID), logger.Error(setErr))
			}
		}
		areContacts := utils.ContainsString(*subjectContactIDs, viewerID)

		// step 7: subject settings
		settings, cacheErr := s.cache.GetSettings(ctx, *subjectID)
		if cacheErr != nil {
			s.log.Error("cache get settings failed", logger.String("user_id", *subjectID), logger.Error(cacheErr))
		}
		if settings == nil {
			userSettings, err := s.settingsRepo.GetSettings(ctx, *subjectID)
			if err != nil {
				return nil, err
			}
			if userSettings == nil {
				userSettings, err = s.settingsRepo.CreateSettings(ctx, *subjectID)
				if err != nil {
					return nil, err
				}
			}
			settings = &cache.Settings{
				UserID:                  *subjectID,
				ProfileVisibility:       string(userSettings.ProfileVisibility),
				LastSeenVisibility:      string(userSettings.LastSeenVisibility),
				OnlineStatusVisibility:  string(userSettings.OnlineStatusVisibility),
				ProfilePhotoVisibility:  string(userSettings.ProfilePhotoVisibility),
				AboutVisibility:         string(userSettings.AboutVisibility),
				ReadReceiptsEnabled:     userSettings.ReadReceiptsEnabled,
				TypingIndicatorsEnabled: userSettings.TypingIndicatorsEnabled,
				UpdatedAt:               time.Now().Unix(),
			}
			if setErr := s.cache.SetSettings(ctx, *subjectID, settings); setErr != nil {
				s.log.Error("cache set settings failed", logger.String("user_id", *subjectID), logger.Error(setErr))
			}
		}

		// step 8: privacy overrides (no cache)
		override, err := s.privacyRepo.GetPrivacyOverride(ctx, *subjectID, viewerID)
		if err != nil {
			return nil, err
		}

		// step 9: build and cache privacy result (always, override may be nil)
		privacy = buildPrivacyResult(areContacts, settings, override)
		if setErr := s.cache.SetPrivacy(ctx, viewerID, *subjectID, privacy); setErr != nil {
			s.log.Error("cache set privacy failed", logger.String("viewer_id", viewerID), logger.String("subject_id", *subjectID), logger.Error(setErr))
		}
	}

	// step 10: apply privacy and return
	if !privacy.CanSeeProfile {
		return nil, pkgErrors.New(pkgErrors.CodeNotFound, "user not found")
	}
	applyPrivacy(profile, privacy)

	return profile, nil
}

func buildPrivacyResult(areContacts bool, s *cache.Settings, override *domain.PrivacyOverride) *cache.PrivacyResult {
	isVisible := func(visibility string) bool {
		switch visibility {
		case string(domain.ProfileVisibilityPublic):
			return true
		case string(domain.ProfileVisibilityFriends):
			return areContacts
		case string(domain.ProfileVisibilityPrivate):
			return false
		default:
			return false
		}
	}

	r := &cache.PrivacyResult{
		CanSeeProfile:      isVisible(s.ProfileVisibility),
		CanSeeAvatar:       isVisible(s.ProfilePhotoVisibility),
		CanSeeLastSeen:     isVisible(s.LastSeenVisibility),
		CanSeeOnlineStatus: isVisible(s.OnlineStatusVisibility),
		CanSeeAbout:        isVisible(s.AboutVisibility),
		AreContacts:        areContacts,
	}

	if override != nil {
		if override.ProfilePhotoVisible != nil {
			r.CanSeeAvatar = *override.ProfilePhotoVisible
		}
		if override.LastSeenVisible != nil {
			r.CanSeeLastSeen = *override.LastSeenVisible
		}
		if override.OnlineStatusVisible != nil {
			r.CanSeeOnlineStatus = *override.OnlineStatusVisible
		}
		if override.AboutVisible != nil {
			r.CanSeeAbout = *override.AboutVisible
		}
	}

	return r
}

func applyPrivacy(p *domain.Profile, r *cache.PrivacyResult) {
	if !r.CanSeeAvatar {
		p.AvatarURL = nil
		p.AvatarThumbnailURL = nil
	}
	if !r.CanSeeLastSeen {
		p.LastSeenAt = nil
	}
	if !r.CanSeeOnlineStatus {
		p.OnlineStatus = nil
	}
	if !r.CanSeeAbout {
		p.Bio = nil
		p.BioLinks = nil
	}
	p.IsContact = utils.PtrBool(r.AreContacts)
}

func (s *userService) GetProfileByPhone(ctx context.Context, viewerID, phone string) (*domain.Profile, pkgErrors.AppError) {
	s.log.Info("getting profile by phone", logger.String("phone", phone))

	profile, err := s.userRepo.GetProfileByPhone(ctx, phone)
	if err != nil {
		s.log.Error("failed to get profile by phone", logger.String("phone", phone), logger.Error(err))
		return nil, err
	}
	if profile == nil {
		return nil, nil
	}

	// reuse existing logic to apply privacy etc
	return s.GetProfileByUsername(ctx, viewerID, *profile.DisplayName)
}

func (s *userService) CreateProfile(ctx context.Context, userID string, input *domain.CreateProfileInput) (*domain.Profile, pkgErrors.AppError) {
	s.log.Info("Creating user profile",
		logger.String("user_id", userID),
	)

	result, err := s.userRepo.CreateProfile(ctx, userID, repository.CreateProfileInput{
		UserName:          input.UserName,
		DisplayName:       utils.PtrString(input.DisplayName),
		FirstName:         utils.PtrString(input.FirstName),
		LastName:          utils.PtrString(input.LastName),
		Bio:               utils.PtrString(input.Bio),
		LanguageCode:      utils.PtrString(input.LanguageCode),
		Timezone:          utils.PtrString(input.Timezone),
		CountryCode:       utils.PtrString(input.CountryCode),
		City:              utils.PtrString(input.City),
		PhoneVisible:      utils.PtrBool(input.PhoneVisible),
		EmailVisible:      utils.PtrBool(input.EmailVisible),
		ProfileVisibility: utils.PtrString(input.ProfileVisibility.String()),
		SearchVisibility:  utils.PtrBool(input.SearchVisibility),
	})
	if err != nil {
		s.log.Error("Failed to create profile",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, err
	}

	err = s.cache.SetProfile(ctx, result.UserID, result)
	if err != nil {
		s.log.Error("Failed to cache profile after creation",
			logger.String("user_id", result.UserID),
			logger.Error(err),
		)
	}
	err = s.cache.SetUserIDByUsername(ctx, *result.DisplayName, result.UserID)
	if err != nil {
		s.log.Error("Failed to cache username to userID mapping after profile creation",
			logger.String("user_id", result.UserID),
			logger.String("username", *result.DisplayName),
			logger.Error(err),
		)
	}
	err = s.cache.SetSettings(ctx, result.UserID, &cache.Settings{
		UserID:                  result.UserID,
		ProfileVisibility:       string(input.ProfileVisibility),
		LastSeenVisibility:      string(input.ProfileVisibility), // default to private
		OnlineStatusVisibility:  string(input.ProfileVisibility), // default to private
		ProfilePhotoVisibility:  string(input.ProfileVisibility), // default to private
		AboutVisibility:         string(input.ProfileVisibility), // default to private
		ReadReceiptsEnabled:     true,                            // default to true
		TypingIndicatorsEnabled: true,                            // default to true
		UpdatedAt:               time.Now().Unix(),
	})
	if err != nil {
		s.log.Error("Failed to cache settings after profile creation",
			logger.String("user_id", result.UserID),
			logger.Error(err),
		)
	}

	// TODO:
	// INSERT into users.outbox: user.profile.updated event with {changed_fields: [...], new_values: {...searchable fields only}}.

	return result, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID string, input *domain.UpdateProfileInput) (*domain.Profile, pkgErrors.AppError) {
	s.log.Info("Updating user profile",
		logger.String("user_id", userID),
	)

	result, err := s.userRepo.UpdateProfile(ctx, userID, repository.UpdateProfileParams{})
	if err != nil {
		s.log.Error("Failed to update profile",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, err
	}

	// Invalidate cache
	if input.DisplayName != nil {
		s.cache.DeleteUserIDByUsername(ctx, *input.DisplayName)
		s.cache.DeleteStatusFeed(ctx, userID)
	}
	if input.ProfileVisibility != nil || input.SearchVisibility != nil {
		s.cache.DeleteSettings(ctx, userID)
	}
	if input.ProfileVisibility != nil {
		s.cache.IncrPrivacyVersion(ctx, userID)
	}

	return result, nil
}

func (s *userService) AddProfileThumbnail(ctx context.Context, userID string, thumbnailURL string) pkgErrors.AppError {
	s.log.Info("Adding profile thumbnail",
		logger.String("user_id", userID),
		logger.String("thumbnail_url", thumbnailURL),
	)

	_, err := s.userRepo.UpdateProfile(ctx, userID, repository.UpdateProfileParams{
		AvatarURL: &thumbnailURL,
	})
	if err != nil {
		s.log.Error("Failed to add profile thumbnail",
			logger.String("user_id", userID),
			logger.String("thumbnail_url", thumbnailURL),
			logger.Error(err),
		)
		return err
	}

	s.log.Info("Profile thumbnail added successfully",
		logger.String("user_id", userID),
		logger.String("thumbnail_url", thumbnailURL),
	)
	return nil
}
