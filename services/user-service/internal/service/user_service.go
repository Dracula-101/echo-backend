package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"user-service/internal/domain"
	repository "user-service/internal/repo"

	"shared/pkg/cache"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	"shared/pkg/utils"
)

type UserService struct {
	repo  repository.UserRepositoryInterface
	cache cache.Cache
	log   logger.Logger
}

func newUserService(repo repository.UserRepositoryInterface, cache cache.Cache, log logger.Logger) *UserService {
	return &UserService{repo: repo, cache: cache, log: log}
}

func (s *UserService) GenerateUsername(ctx context.Context, displayName string) (string, error) {
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

func (s *UserService) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	s.log.Info("Getting user profile",
		logger.String("user_id", userID),
	)
	if s.cache != nil {
		cacheKey := fmt.Sprintf("user:profile:%s", userID)
		cachedData, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cachedData != nil {
			var cachedProfile domain.User
			if err := json.Unmarshal(cachedData, &cachedProfile); err == nil {
				s.log.Debug("Profile found in cache",
					logger.String("user_id", userID),
				)
				return &cachedProfile, nil
			}
		}
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

	user := &domain.User{
		ID:           profile.UserID,
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
		CreatedAt:    *profile.CreatedAt,
		UpdatedAt:    *profile.UpdatedAt,
	}

	if s.cache != nil {
		cacheKey := fmt.Sprintf("user:profile:%s", userID)
		if data, err := json.Marshal(user); err == nil {
			_ = s.cache.Set(ctx, cacheKey, data, 5*time.Minute)
		}
	}

	return user, nil
}

func (s *UserService) CreateProfile(ctx context.Context, input *domain.CreateProfileInput) (*domain.User, error) {
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
		return &domain.User{
			ID:           result.UserID,
			Username:     result.Username,
			DisplayName:  result.DisplayName,
			FirstName:    result.FirstName,
			LastName:     result.LastName,
			Bio:          result.Bio,
			AvatarURL:    result.AvatarURL,
			LanguageCode: result.LanguageCode,
			Timezone:     result.Timezone,
			CountryCode:  result.CountryCode,
			IsVerified:   result.IsVerified,
			CreatedAt:    *result.CreatedAt,
			UpdatedAt:    *result.UpdatedAt,
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
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}

	result, err := s.repo.CreateProfile(ctx, newProfile)
	if err != nil {
		s.log.Error("Failed to create profile",
			logger.String("user_id", input.UserID),
			logger.Error(err),
		)
		return nil, err
	}

	return &domain.User{
		ID:           result.UserID,
		Username:     result.Username,
		DisplayName:  result.DisplayName,
		FirstName:    result.FirstName,
		LastName:     result.LastName,
		Bio:          result.Bio,
		AvatarURL:    result.AvatarURL,
		LanguageCode: result.LanguageCode,
		Timezone:     result.Timezone,
		CountryCode:  result.CountryCode,
		IsVerified:   result.IsVerified,
		CreatedAt:    *result.CreatedAt,
		UpdatedAt:    *result.UpdatedAt,
	}, nil
}

func (s *UserService) AddUserDevice(ctx context.Context, input *domain.UserDevice) error {
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
