package search

import (
	"context"
	"crypto/sha256"
	"fmt"
	"shared/pkg/logger"
	"shared/pkg/search"
	"user-service/internal/domain"
	repo "user-service/internal/repo"
	"user-service/internal/service/cache"
)

type SearchService interface {
	SearchProfiles(ctx context.Context, query string, queryType domain.UserQueryType, limit, offset int) ([]*domain.Profile, int, error)
}

// Compile-time interface compliance check
var _ SearchService = (*searchService)(nil)

type searchService struct {
	userRepo     repo.UserRepository
	userCache    cache.UserCache
	searchClient search.Search
	log          logger.Logger
}

func NewSearchService(userRepo repo.UserRepository, searchClient search.Search, userCache cache.UserCache, log logger.Logger) SearchService {
	return &searchService{
		userRepo:     userRepo,
		searchClient: searchClient,
		userCache:    userCache,
		log:          log,
	}
}

func (s *searchService) SearchProfiles(ctx context.Context, query string, queryType domain.UserQueryType, limit, offset int) ([]*domain.Profile, int, error) {
	s.log.Info("Searching user profiles",
		logger.String("query", query),
		logger.String("query_type", string(queryType)),
		logger.Int("limit", limit),
		logger.Int("offset", offset),
	)

	sha := sha256.Sum256(fmt.Appendf(nil, "%s:%s:%d:%d", query, queryType, limit, offset))
	cacheKey := string(sha[:])

	result, err := s.userCache.GetSearchCache(ctx, cacheKey)
	if err != nil {
		s.log.Error("Failed to retrieve search result from cache",
			logger.String("cache_key", cacheKey),
			logger.Error(err),
		)
	}
	if result != nil {
		s.log.Info("Search result found in cache",
			logger.String("cache_key", cacheKey),
		)
		return result.Users, int(result.Total), nil
	}

	// search the meilisearch index for user profiles
	searchResult, searchErr := s.searchClient.Search(ctx, "users", query, &search.QueryOptions{
		Filter:           fmt.Sprintf("type = %q", queryType),
		Limit:            int64(limit),
		Offset:           int64(offset),
		ShowRankingScore: true,
	})
	if searchErr != nil {
		s.log.Error("Failed to search user profiles in search index",
			logger.String("query", query),
			logger.String("query_type", string(queryType)),
			logger.Int("limit", limit),
			logger.Int("offset", offset),
			logger.Error(searchErr),
		)
		// we can continue to search the database even if the search index is down, but we should log the error and monitor it
	} else if len(searchResult.Hits) != 0 {
		s.log.Info("Search result found in search index",
			logger.String("query", query),
			logger.String("query_type", string(queryType)),
			logger.Int("limit", limit),
			logger.Int("offset", offset),
			logger.Int("hits", len(searchResult.Hits)),
		)
		// TODO: we can return the search results directly without hitting the database, but we need to convert the search result to the profile struct
	}

	var users []*domain.Profile
	var total int64
	switch queryType {
	case domain.UserQueryTypeName:
		users, total, err = s.userRepo.SearchProfilesByUsername(ctx, query, limit, offset)

	case domain.UserQueryTypePhone:
		users, total, err = s.userRepo.SearchProfilesByPhone(ctx, query, limit, offset)

	case domain.UserQueryTypeUsername:
		users, total, err = s.userRepo.SearchProfilesByUsername(ctx, query, limit, offset)

	case domain.UserQueryTypeAll:
		users, total, err = s.userRepo.SearchProfilesByUsername(ctx, query, limit, offset)
	}
	if err != nil {
		s.log.Error("Failed to search user profiles",
			logger.String("query", query),
			logger.String("query_type", string(queryType)),
			logger.Int("limit", limit),
			logger.Int("offset", offset),
			logger.Error(err),
		)
		return nil, 0, err
	}

	cacheErr := s.userCache.SetSearchCache(ctx, cacheKey, &cache.SearchResult{
		Users:  users,
		Total:  int64(total),
		Offset: int64(offset),
	})
	if cacheErr != nil {
		s.log.Error("Failed to set search result in cache",
			logger.String("cache_key", cacheKey),
			logger.Error(cacheErr),
		)
	}

	return users, int(total), nil
}
