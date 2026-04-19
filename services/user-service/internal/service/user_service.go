package service

import (
	"context"

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
	err = s.cache.SetProfile(ctx, profile.UserID, profile)

	if err != nil {
		s.log.Error("Failed to cache profile",
			logger.String("user_id", profile.UserID),
			logger.Error(err),
		)
	}

	return profile, nil
}

func (s *userService) CreateProfile(ctx context.Context, userID string, input *domain.CreateProfileInput) (*domain.Profile, error) {
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

	return result, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID string, input *domain.UpdateProfileInput) (*domain.Profile, error) {
	s.log.Info("Updating user profile",
		logger.String("user_id", userID),
	)

	result, err := s.repo.UpdateProfile(ctx, userID, repository.UpdateProfileParams{
	})
	if err != nil {
		s.log.Error("Failed to update profile",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, err
	}

	err = s.cache.SetProfile(ctx, result.UserID, result)
	if err != nil {
		s.log.Error("Failed to cache profile after update",
			logger.String("user_id", result.UserID),
			logger.Error(err),
		)
	}

	return result, nil
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
