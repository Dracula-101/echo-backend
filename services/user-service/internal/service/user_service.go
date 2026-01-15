package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"user-service/internal/model"
	repository "user-service/internal/repo"
	repoModel "user-service/internal/repo/model"
	svcmodel "user-service/internal/service/model"

	"shared/pkg/cache"
	"shared/pkg/logger"
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

func (s *UserService) GetProfile(ctx context.Context, userID string) (*model.User, error) {
	s.log.Info("Getting user profile",
		logger.String("user_id", userID),
	)
	if s.cache != nil {
		cacheKey := fmt.Sprintf("user:profile:%s", userID)
		cachedData, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cachedData != nil {
			var cachedProfile model.User
			if err := json.Unmarshal(cachedData, &cachedProfile); err == nil {
				s.log.Debug("Profile found in cache",
					logger.String("user_id", userID),
				)
				return &cachedProfile, nil
			}
		}
	}
	var repoProfile *repoModel.Profile
	var err error

	repoProfile, err = s.repo.GetProfileByUserID(ctx, userID)

	if err != nil {
		s.log.Error("Failed to get profile",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, err
	}

	if repoProfile == nil {
		return nil, nil
	}
	profile := fromRepoProfile(repoProfile)
	if profile == nil {
		return nil, nil
	}
	user := s.profileToUser(profile)

	if s.cache != nil {
		cacheKey := fmt.Sprintf("user:profile:%s", userID)
		if data, err := json.Marshal(user); err == nil {
			_ = s.cache.Set(ctx, cacheKey, data, 5*time.Minute)
		}
	}

	return user, nil
}

func (s *UserService) CreateProfile(ctx context.Context, profile *svcmodel.Profile) (*model.User, error) {
	s.log.Info("Creating user profile",
		logger.String("user_id", profile.UserID),
	)

	var createdProfile *svcmodel.Profile

	repoInput := toRepoProfile(profile)
	existingProfile, err := s.repo.GetProfileByUserID(ctx, profile.UserID)
	if err != nil {
		s.log.Error("Failed to check existing profile",
			logger.String("user_id", profile.UserID),
			logger.Error(err),
		)
		return nil, err
	}
	if existingProfile != nil {
		s.log.Info("Profile already exists, Updating existing profile",
			logger.String("user_id", profile.UserID),
		)
		result, err := s.repo.UpdateProfile(ctx, repository.UpdateProfileParams{
			UserID:       profile.UserID,
			Username:     &repoInput.Username,
			DisplayName:  repoInput.DisplayName,
			FirstName:    repoInput.FirstName,
			LastName:     repoInput.LastName,
			Bio:          repoInput.Bio,
			AvatarURL:    repoInput.AvatarURL,
			LanguageCode: &repoInput.LanguageCode,
			Timezone:     repoInput.Timezone,
			CountryCode:  repoInput.CountryCode,
		})
		if err != nil {
			s.log.Error("Failed to update existing profile",
				logger.String("user_id", profile.UserID),
				logger.Error(err),
			)
			return nil, err
		}
		createdProfile = fromRepoProfile(result)
		return s.profileToUser(createdProfile), nil
	}

	result, err := s.repo.CreateProfile(ctx, repoInput)
	if err != nil {
		s.log.Error("Failed to create profile",
			logger.String("user_id", profile.UserID),
			logger.Error(err),
		)
		return nil, err
	}
	createdProfile = fromRepoProfile(result)

	if createdProfile == nil {
		createdProfile = profile
	}

	user := s.profileToUser(createdProfile)

	return user, nil
}

func (s *UserService) profileToUser(profile *svcmodel.Profile) *model.User {
	return &model.User{
		ID:           profile.UserID,
		Username:     profile.Username,
		DisplayName:  &profile.DisplayName,
		FirstName:    profile.FirstName,
		LastName:     profile.LastName,
		Bio:          profile.Bio,
		AvatarURL:    profile.AvatarURL,
		LanguageCode: *profile.LanguageCode,
		Timezone:     profile.Timezone,
		CountryCode:  profile.CountryCode,
		IsVerified:   profile.IsVerified,
	}
}

func toRepoProfile(profile *svcmodel.Profile) repoModel.Profile {
	if profile == nil {
		return repoModel.Profile{}
	}

	now := time.Now()
	return repoModel.Profile{
		UserID:            profile.UserID,
		Username:          profile.Username,
		DisplayName:       &profile.DisplayName,
		FirstName:         profile.FirstName,
		LastName:          profile.LastName,
		Bio:               profile.Bio,
		AvatarURL:         profile.AvatarURL,
		LanguageCode:      valueOrDefault(profile.LanguageCode),
		Timezone:          profile.Timezone,
		CountryCode:       profile.CountryCode,
		PhoneVisible:      false,
		EmailVisible:      false,
		OnlineStatus:      "offline",
		LastSeenAt:        &now,
		ProfileVisibility: "private",
		SearchVisibility:  false,
		IsVerified:        profile.IsVerified,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func fromRepoProfile(profile *repoModel.Profile) *svcmodel.Profile {
	if profile == nil {
		return nil
	}

	return &svcmodel.Profile{
		UserID:       profile.UserID,
		Username:     profile.Username,
		DisplayName:  derefString(profile.DisplayName),
		FirstName:    profile.FirstName,
		LastName:     profile.LastName,
		Bio:          profile.Bio,
		AvatarURL:    profile.AvatarURL,
		LanguageCode: &profile.LanguageCode,
		Timezone:     profile.Timezone,
		CountryCode:  profile.CountryCode,
		IsVerified:   profile.IsVerified,
	}
}

func valueOrDefault(val *string) string {
	if val != nil {
		return *val
	}
	return ""
}

func derefString(val *string) string {
	if val == nil {
		return ""
	}
	return *val
}
