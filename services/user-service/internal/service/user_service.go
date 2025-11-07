package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"user-service/internal/model"
	repository "user-service/internal/repo"
	"user-service/internal/service/models"

	"shared/pkg/cache"
	"shared/pkg/circuitbreaker"
	dbmodels "shared/pkg/database/postgres/models"
	"shared/pkg/logger"
	"shared/pkg/retry"
)

type UserService struct {
	repo      *repository.UserRepository
	cache     cache.Cache
	log       logger.Logger
	dbCircuit *circuitbreaker.CircuitBreaker
	retryer   *retry.Retryer
}

func NewUserServiceBuilder() *UserServiceBuilder {
	return &UserServiceBuilder{}
}

type UserServiceBuilder struct {
	repo  *repository.UserRepository
	cache cache.Cache
	log   logger.Logger
}

func (b *UserServiceBuilder) WithRepo(repo *repository.UserRepository) *UserServiceBuilder {
	b.repo = repo
	return b
}

func (b *UserServiceBuilder) WithCache(cache cache.Cache) *UserServiceBuilder {
	b.cache = cache
	return b
}

func (b *UserServiceBuilder) WithLogger(log logger.Logger) *UserServiceBuilder {
	b.log = log
	return b
}

func (b *UserServiceBuilder) Build() *UserService {
	if b.repo == nil {
		panic("UserRepository is required")
	}
	if b.log == nil {
		panic("Logger is required")
	}

	b.log.Info("Building UserService",
		logger.String("service", "user-service"),
	)

	dbCircuit := circuitbreaker.New("user-db", circuitbreaker.Config{
		MaxRequests: 2,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from, to circuitbreaker.State) {
			b.log.Info("Circuit breaker state changed",
				logger.String("circuit", name),
				logger.String("from", from.String()),
				logger.String("to", to.String()),
			)
		},
	})

	retryer := retry.New(retry.Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Strategy:     retry.StrategyExponential,
		Multiplier:   2.0,
		OnRetry: func(attempt int, delay time.Duration, err error) {
			b.log.Warn("Retrying operation",
				logger.Int("attempt", attempt),
				logger.Duration("delay", delay),
				logger.Error(err),
			)
		},
	})

	return &UserService{
		repo:      b.repo,
		cache:     b.cache,
		log:       b.log,
		dbCircuit: dbCircuit,
		retryer:   retryer,
	}
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
	var repoProfile *dbmodels.Profile
	var err error

	err = s.dbCircuit.Execute(func() error {
		repoProfile, err = s.repo.GetProfileByUserID(ctx, userID)
		return err
	})

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

func (s *UserService) CreateProfile(ctx context.Context, profile *models.Profile) (*model.User, error) {
	s.log.Info("Creating user profile",
		logger.String("user_id", profile.UserID),
	)

	var createdProfile *models.Profile

	err := s.dbCircuit.Execute(func() error {
		repoInput := toRepoProfile(profile)
		result, err := s.repo.CreateProfile(ctx, repoInput)
		if err != nil {
			return err
		}
		createdProfile = fromRepoProfile(result)
		return nil
	})
	if err != nil {
		s.log.Error("Failed to create profile",
			logger.String("user_id", profile.UserID),
			logger.Error(err),
		)
		return nil, err
	}

	if createdProfile == nil {
		createdProfile = profile
	}

	user := s.profileToUser(createdProfile)

	return user, nil
}

func (s *UserService) profileToUser(profile *models.Profile) *model.User {
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

func toRepoProfile(profile *models.Profile) dbmodels.Profile {
	if profile == nil {
		return dbmodels.Profile{}
	}

	return dbmodels.Profile{
		UserID:       profile.UserID,
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

func fromRepoProfile(profile *dbmodels.Profile) *models.Profile {
	if profile == nil {
		return nil
	}

	return &models.Profile{
		UserID:       profile.UserID,
		Username:     profile.Username,
		DisplayName:  *profile.DisplayName,
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
