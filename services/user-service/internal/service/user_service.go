package service

import (
	"context"
	"time"

	"user-service/internal/domain"
	repository "user-service/internal/repo"
	"user-service/internal/service/cache"

	"shared/pkg/database/postgres"
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

func (s *userService) GenerateUsername(ctx context.Context, displayName string) (string, error) {
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

func (s *userService) GetProfile(ctx context.Context, userID string) (*domain.Profile, error) {
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
	err = s.cache.SetProfile(ctx, profile.UserID, &domain.Profile{
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
		LastSeenAt:         profile.LastSeenAt,
		OnlineStatus:       profile.OnlineStatus,
		ProfileVisibility:  profile.ProfileVisibility,
		SearchVisibility:   profile.SearchVisibility,
		TwoFactorEnabled:   profile.TwoFactorEnabled,
		AccountStatus:      profile.AccountStatus,
		PhoneVerified:      profile.PhoneVerified,
		EmailVerified:      profile.EmailVerified,
	})

	if err != nil {
		s.log.Error("Failed to cache profile",
			logger.String("user_id", profile.UserID),
			logger.Error(err),
		)
	}

	return profile, nil
}

func (s *userService) CreateProfile(ctx context.Context, input *domain.CreateProfileInput) (*domain.Profile, error) {
	s.log.Info("Creating user profile",
		logger.String("user_id", input.UserID),
	)

	existingProfile, err := s.repo.GetProfileByUserID(ctx, input.UserID)
	if err != nil {
		s.log.Error("Failed to check existing profile",
			logger.String("user_id", input.UserID),
			logger.Error(err),
		)
		return nil, err
	}

	if existingProfile != nil {
		s.log.Info("Profile already exists, updating existing profile",
			logger.String("user_id", input.UserID),
		)
		result, err := s.repo.UpdateProfile(ctx, repository.UpdateProfileParams{
			UserID:       input.UserID,
			Username:     &input.Username,
			DisplayName:  &input.DisplayName,
			FirstName:    input.FirstName,
			LastName:     input.LastName,
			Bio:          input.Bio,
			AvatarURL:    input.AvatarURL,
			LanguageCode: input.LanguageCode,
			Timezone:     input.Timezone,
			CountryCode:  input.CountryCode,
		})
		if err != nil {
			s.log.Error("Failed to update existing profile",
				logger.String("user_id", input.UserID),
				logger.Error(err),
			)
			return nil, err
		}
		return &domain.Profile{
			UserID:             result.UserID,
			Username:           result.Username,
			DisplayName:        result.DisplayName,
			FirstName:          result.FirstName,
			LastName:           result.LastName,
			Bio:                result.Bio,
			AvatarURL:          result.AvatarURL,
			LanguageCode:       result.LanguageCode,
			Timezone:           result.Timezone,
			CountryCode:        result.CountryCode,
			IsVerified:         result.IsVerified,
			AvatarThumbnailURL: result.AvatarThumbnailURL,
			PhoneVisible:       result.PhoneVisible,
			EmailVisible:       result.EmailVisible,
			OnlineStatus:       result.OnlineStatus,
			LastSeenAt:         result.LastSeenAt,
			ProfileVisibility:  result.ProfileVisibility,
			SearchVisibility:   result.SearchVisibility,
			PhoneVerified:      result.PhoneVerified,
			EmailVerified:      result.EmailVerified,
			TwoFactorEnabled:   result.TwoFactorEnabled,
			AccountStatus:      result.AccountStatus,
		}, nil
	}

	now := time.Now()
	languageCode := "en"
	if input.LanguageCode != nil {
		languageCode = *input.LanguageCode
	}

	newProfile := domain.Profile{
		UserID:            input.UserID,
		Username:          input.Username,
		DisplayName:       &input.DisplayName,
		FirstName:         input.FirstName,
		LastName:          input.LastName,
		Bio:               input.Bio,
		AvatarURL:         input.AvatarURL,
		LanguageCode:      languageCode,
		Timezone:          input.Timezone,
		CountryCode:       input.CountryCode,
		PhoneVisible:      false,
		EmailVisible:      false,
		OnlineStatus:      "offline",
		LastSeenAt:        &now,
		ProfileVisibility: "private",
		SearchVisibility:  false,
		IsVerified:        input.IsVerified,
		PhoneVerified:     false,
		EmailVerified:     false,
		TwoFactorEnabled:  false,
		AccountStatus:     domain.AccountStatusActive,
	}

	result, err := s.repo.CreateProfile(ctx, newProfile)
	if err != nil {
		s.log.Error("Failed to create profile",
			logger.String("user_id", input.UserID),
			logger.Error(err),
		)
		return nil, err
	}

	return &domain.Profile{
		UserID:             result.UserID,
		Username:           result.Username,
		DisplayName:        result.DisplayName,
		FirstName:          result.FirstName,
		LastName:           result.LastName,
		Bio:                result.Bio,
		AvatarURL:          result.AvatarURL,
		LanguageCode:       result.LanguageCode,
		Timezone:           result.Timezone,
		CountryCode:        result.CountryCode,
		IsVerified:         result.IsVerified,
		AvatarThumbnailURL: result.AvatarThumbnailURL,
		PhoneVisible:       result.PhoneVisible,
		EmailVisible:       result.EmailVisible,
		LastSeenAt:         result.LastSeenAt,
		OnlineStatus:       result.OnlineStatus,
		ProfileVisibility:  result.ProfileVisibility,
		SearchVisibility:   result.SearchVisibility,
		AccountStatus:      result.AccountStatus,
		PhoneVerified:      result.PhoneVerified,
		EmailVerified:      result.EmailVerified,
		TwoFactorEnabled:   result.TwoFactorEnabled,
	}, nil
}

func (s *userService) AddUserDevice(ctx context.Context, input *domain.UserDevice) error {
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

func (s *userService) AddProfileThumbnail(ctx context.Context, userID string, thumbnailURL string) error {
	s.log.Info("Adding profile thumbnail",
		logger.String("user_id", userID),
		logger.String("thumbnail_url", thumbnailURL),
	)

	_, err := s.repo.UpdateProfile(ctx, repository.UpdateProfileParams{
		UserID:    userID,
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
