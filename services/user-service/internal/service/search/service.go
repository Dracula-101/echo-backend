package search

import (
	"context"
	"shared/pkg/logger"
	"user-service/internal/domain"
	repo "user-service/internal/repo"
)

type SearchService interface {
	SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int, error)
}

// Compile-time interface compliance check
var _ SearchService = (*searchService)(nil)

type searchService struct {
	userRepo repo.UserRepository
	log      logger.Logger
}

func NewSearchService(userRepo repo.UserRepository, log logger.Logger) SearchService {
	return &searchService{userRepo: userRepo, log: log}
}

func (s *searchService) SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int, error) {
	s.log.Info("Searching user profiles",
		logger.String("query", query),
		logger.Int("limit", limit),
		logger.Int("offset", offset),
	)

	users, total, err := s.userRepo.SearchProfiles(ctx, query, limit, offset)
	if err != nil {
		s.log.Error("Failed to search user profiles",
			logger.String("query", query),
			logger.Int("limit", limit),
			logger.Int("offset", offset),
			logger.Error(err),
		)
		return nil, 0, err
	}

	return users, total, nil
}
