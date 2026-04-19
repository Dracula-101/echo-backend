package service

import (
	"context"
	"time"

	"user-service/internal/domain"
	repository "user-service/internal/repo"
	"user-service/internal/service/cache"

	"shared/pkg/database/postgres"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
)

type userService struct {
	repo  repository.UserRepository
	cache cache.UserCache
	log   logger.Logger
}

func newUserService(repo repository.UserRepository, cache cache.UserCache, log logger.Logger) *userService {
	return &userService{repo: repo, cache: cache, log: log}
}

func (s *userService) GenerateUsername(ctx context.Context, displayName string) (string, pkgErrors.AppError) {
	s.log.Info("Generating username",
		logger.String("display_name", displayName),
	)

	username, err := s.repo.GenerateUniqueUsername(ctx, displayName)
	if err != nil {
		s.log.Error("Failed to generate username",
			logger.String("display_name", displayName),
			logger.Error(err),
		)
		return "", err
	}

	return *username, nil
}

func (s *userService) GetProfile(ctx context.Context, userID string) (*domain.Profile, pkgErrors.AppError) {
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

	profile, err := s.repo.GetProfileByUserID(ctx, userID)
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
		p, err := s.repo.GetProfileByUsername(ctx, username)
		if err != nil {
			return nil, err
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
		p, err := s.repo.GetProfileByUserID(ctx, *subjectID)
		if err != nil {
			return nil, err
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
			ids, err := s.repo.GetBlockedUsers(ctx, viewerID)
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
			ids, err := s.repo.GetBlockedUsers(ctx, *subjectID)
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
			contacts, err := s.repo.GetContacts(ctx, *subjectID)
			if err != nil {
				return nil, err
			}
			if contacts != nil {

				var idStrs []string
				for _, contact := range *contacts {
					idStrs = append(idStrs, contact.UserID)
				}
				subjectContactIDs = &idStrs
				if setErr := s.cache.SetContactIDs(ctx, *subjectID, idStrs); setErr != nil {
					s.log.Error("cache set contact IDs failed", logger.String("user_id", *subjectID), logger.Error(setErr))
				}
			}
		}
		areContacts := utils.ContainsString(*subjectContactIDs, viewerID)

		// step 7: subject settings
		settings, cacheErr := s.cache.GetSettings(ctx, *subjectID)
		if cacheErr != nil {
			s.log.Error("cache get settings failed", logger.String("user_id", *subjectID), logger.Error(cacheErr))
		}
		if settings == nil {
			userSettings, err := s.repo.GetSettings(ctx, *subjectID)
			if err != nil {
				return nil, err
			}
			if userSettings == nil {
				return nil, pkgErrors.New(pkgErrors.CodeNotFound, "settings not found")
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
		override, err := s.repo.GetPrivacyOverrides(ctx, *subjectID, viewerID)
		if err != nil {
			return nil, err
		}
		if override != nil {
			// step 9: build and cache privacy result
			privacy = buildPrivacyResult(areContacts, settings, override)
			if setErr := s.cache.SetPrivacy(ctx, viewerID, *subjectID, privacy); setErr != nil {
				s.log.Error("cache set privacy failed", logger.String("viewer_id", viewerID), logger.String("subject_id", *subjectID), logger.Error(setErr))
			}

			// step 10: apply p	rivacy and return
			if !privacy.CanSeeProfile {
				return nil, pkgErrors.New(pkgErrors.CodeNotFound, "user not found")
			}
			applyPrivacy(profile, privacy)
		}
	}

	return profile, nil
}

func buildPrivacyResult(areContacts bool, s *cache.Settings, override *domain.PrivacyOverride) *cache.PrivacyResult {
	isVisible := func(visibility string) bool {
		switch visibility {
		case string(domain.ProfileVisibilityPrivate):
			return true
		case string(domain.ProfileVisibilityPublic):
			return areContacts
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

func (s *userService) CreateProfile(ctx context.Context, userID string, input *domain.CreateProfileInput) (*domain.Profile, pkgErrors.AppError) {
	s.log.Info("Creating user profile",
		logger.String("user_id", userID),
	)

	result, err := s.repo.CreateProfile(ctx, userID, repository.CreateProfileInput{
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

	result, err := s.repo.UpdateProfile(ctx, userID, repository.UpdateProfileParams{})
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

func (s *userService) AddUserDevice(ctx context.Context, input *domain.UserDevice) pkgErrors.AppError {
	s.log.Info("Adding user device",
		logger.String("user_id", input.UserID),
		logger.String("device_id", input.DeviceID),
	)

	devices, err := s.repo.GetUserDevices(ctx, input.UserID)
	if err != nil && !postgres.IsNotFoundError(err) {
		s.log.Error("Failed to get user devices",
			logger.String("user_id", input.UserID),
			logger.Error(err),
		)
		return err
	}

	alreadyRegistered := false
	for _, device := range devices {
		if device.DeviceID == input.DeviceID {
			s.log.Info("Device already registered for user",
				logger.String("user_id", input.UserID),
				logger.String("device_id", input.DeviceID),
			)
			alreadyRegistered = true
			break
		}
	}

	if !alreadyRegistered {
		err = s.repo.AddUserDevice(ctx, input, true)
		if err != nil {
			s.log.Error("Failed to add user device",
				logger.String("user_id", input.UserID),
				logger.String("device_id", input.DeviceID),
				logger.Error(err),
			)
			return err
		}
		s.log.Info("User device added successfully",
			logger.String("user_id", input.UserID),
			logger.String("device_id", input.DeviceID),
		)
	} else {
		for _, device := range devices {
			updateInput := &domain.UpdateUserDevice{
				IsActive:   utils.PtrBool(device.DeviceID == input.DeviceID),
				AppVersion: input.AppVersion,
			}
			s.repo.UpdateUserDevice(ctx, updateInput, input.UserID, device.DeviceID)
		}
	}

	return nil
}

func (s *userService) AddProfileThumbnail(ctx context.Context, userID string, thumbnailURL string) pkgErrors.AppError {
	s.log.Info("Adding profile thumbnail",
		logger.String("user_id", userID),
		logger.String("thumbnail_url", thumbnailURL),
	)

	_, err := s.repo.UpdateProfile(ctx, userID, repository.UpdateProfileParams{
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
