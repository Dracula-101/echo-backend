package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"user-service/internal/model"
	repository "user-service/internal/repo"

	"shared/pkg/cache"
	"shared/pkg/circuitbreaker"
	"shared/pkg/database/postgres/models"
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

// GetProfile retrieves a user profile by user ID
func (s *UserService) GetProfile(ctx context.Context, userID string) (*model.User, error) {
	s.log.Info("Getting user profile",
		logger.String("user_id", userID),
	)

	// Try to get from cache first
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

	// Get from database with circuit breaker
	var profile *models.Profile
	var err error

	err = s.dbCircuit.Execute(func() error {
		profile, err = s.repo.GetProfileByUserID(ctx, userID)
		return err
	})

	if err != nil {
		s.log.Error("Failed to get profile",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, err
	}

	if profile == nil {
		return nil, fmt.Errorf("profile not found")
	}

	// Convert to domain model
	user := s.profileToUser(profile)

	// Cache the result
	if s.cache != nil {
		cacheKey := fmt.Sprintf("user:profile:%s", userID)
		if data, err := json.Marshal(user); err == nil {
			_ = s.cache.Set(ctx, cacheKey, data, 5*time.Minute)
		}
	}

	return user, nil
}

// UpdateProfile updates a user profile
func (s *UserService) UpdateProfile(ctx context.Context, userID string, update *model.ProfileUpdate) (*model.User, error) {
	s.log.Info("Updating user profile",
		logger.String("user_id", userID),
	)

	// Check if username is being changed and if it's available
	if update.Username != nil {
		exists, err := s.repo.UsernameExists(ctx, *update.Username)
		if err != nil {
			s.log.Error("Failed to check username availability",
				logger.String("username", *update.Username),
				logger.Error(err),
			)
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("username already taken")
		}
	}

	// Update profile with circuit breaker
	params := repository.UpdateProfileParams{
		UserID:       userID,
		Username:     update.Username,
		DisplayName:  update.DisplayName,
		FirstName:    update.FirstName,
		LastName:     update.LastName,
		Bio:          update.Bio,
		AvatarURL:    update.AvatarURL,
		LanguageCode: update.LanguageCode,
		Timezone:     update.Timezone,
		CountryCode:  update.CountryCode,
	}

	var profile *models.Profile
	var err error

	err = s.dbCircuit.Execute(func() error {
		profile, err = s.repo.UpdateProfile(ctx, params)
		return err
	})

	if err != nil {
		s.log.Error("Failed to update profile",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return nil, err
	}

	// Convert to domain model
	user := s.profileToUser(profile)

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("user:profile:%s", userID)
		_ = s.cache.Delete(ctx, cacheKey)
	}

	return user, nil
}

// SearchUsers searches for users by query
func (s *UserService) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*model.User, int, error) {
	s.log.Info("Searching users",
		logger.String("query", query),
		logger.Int("limit", limit),
		logger.Int("offset", offset),
	)

	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var profiles []*models.Profile
	var totalCount int
	var err error

	err = s.dbCircuit.Execute(func() error {
		profiles, totalCount, err = s.repo.SearchProfiles(ctx, query, limit, offset)
		return err
	})

	if err != nil {
		s.log.Error("Failed to search users",
			logger.String("query", query),
			logger.Error(err),
		)
		return nil, 0, err
	}

	// Convert to domain models
	users := make([]*model.User, 0, len(profiles))
	for _, profile := range profiles {
		users = append(users, s.profileToUser(profile))
	}

	s.log.Info("User search completed",
		logger.String("query", query),
		logger.Int("results", len(users)),
		logger.Int("total_count", totalCount),
	)

	return users, totalCount, nil
}

// Helper function to convert Profile model to User domain model
func (s *UserService) profileToUser(profile *models.Profile) *model.User {
	return &model.User{
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
		CreatedAt:    profile.CreatedAt,
		UpdatedAt:    profile.UpdatedAt,
	}
}
