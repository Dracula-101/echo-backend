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

var SearchIndexName = "users"

var searchIndexSettings = &search.IndexSettings{
	SearchableAttributes: []string{"username", "display_name", "phone"},
	FilterableAttributes: []string{"search_visibility", "deactivated_at"},
}

type SearchService interface {
	CreateIndex(ctx context.Context) error
	AddOrUpdateProfileInIndex(ctx context.Context, profile domain.SearchProfile) error
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

func (s *searchService) CreateIndex(ctx context.Context) error {
	// check if the index already exists
	_, err := s.searchClient.GetIndex(ctx, SearchIndexName)
	if err == nil {
		s.log.Info("Search index already exists",
			logger.String("index_name", SearchIndexName),
		)
		return s.ensureIndexSettings(ctx)
	}

	s.log.Info("Creating search index",
		logger.String("index_name", SearchIndexName),
	)
	err = s.searchClient.CreateIndex(ctx, SearchIndexName, &search.IndexOptions{
		PrimaryKey: domain.SearchProfile{}.SearchID(),
	})
	if err != nil {
		s.log.Error("Failed to create search index",
			logger.String("index_name", SearchIndexName),
			logger.Error(err),
		)
		return err
	}
	s.log.Info("Search index created successfully",
		logger.String("index_name", SearchIndexName),
	)

	return s.ensureIndexSettings(ctx)
}

func (s *searchService) AddOrUpdateProfileInIndex(ctx context.Context, profile domain.SearchProfile) error {
	s.log.Info("Adding/updating profile in search index",
		logger.String("user_id", profile.UserID),
	)

	err := s.searchClient.IndexOne(ctx, SearchIndexName, profile)
	if err != nil {
		s.log.Error("Failed to add/update profile in search index",
			logger.String("user_id", profile.UserID),
			logger.Error(err),
		)
		return err
	}

	s.log.Info("Profile added/updated in search index successfully",
		logger.String("user_id", profile.UserID),
	)

	return nil
}

func (s *searchService) ensureIndexSettings(ctx context.Context) error {
	s.log.Info("Applying search index settings",
		logger.String("index_name", SearchIndexName),
	)

	err := s.searchClient.UpdateSettings(ctx, SearchIndexName, searchIndexSettings)
	if err != nil {
		s.log.Error("Failed to apply search index settings",
			logger.String("index_name", SearchIndexName),
			logger.Error(err),
		)
		return err
	}

	s.log.Info("Search index settings applied",
		logger.String("index_name", SearchIndexName),
	)

	return nil
}

func (s *searchService) SearchProfiles(ctx context.Context, query string, queryType domain.UserQueryType, limit, offset int) ([]*domain.Profile, int, error) {
	s.log.Info("Searching user profiles",
		logger.String("query", query),
		logger.String("query_type", string(queryType)),
		logger.Int("limit", limit),
		logger.Int("offset", offset),
	)

	sha := sha256.Sum256(fmt.Appendf(nil, "%s:%s:%d:%d", query, queryType.String(), limit, offset))
	cacheKey := string(sha[:])

	_, err := s.userCache.GetSearchCache(ctx, cacheKey)
	if err != nil {
		s.log.Error("Failed to retrieve search result from cache",
			logger.String("cache_key", cacheKey),
			logger.Error(err),
		)
	}
	// if result != nil {
	// 	s.log.Info("Search result found in cache",
	// 		logger.String("cache_key", cacheKey),
	// 	)
	// 	return result.Users, int(result.Total), nil
	// }

	// search the meilisearch index for user profiles
	searchResultIds, searchErr := s.searchClient.Search(ctx, SearchIndexName, query, &search.QueryOptions{
		Filter: "search_visibility = true AND deactivated_at IS NULL",
		AttributesToHighlight: func() []string {
			switch queryType {
			case domain.UserQueryTypeName:
				return []string{"display_name"}
			case domain.UserQueryTypePhone:
				return []string{"phone"}
			case domain.UserQueryTypeUsername:
				return []string{"username"}
			case domain.UserQueryTypeAll:
				return []string{"username", "display_name", "phone"}
			default:
				return []string{"username", "display_name", "phone"}
			}
		}(),
		AttributesToRetrieve: []string{"user_id"},
		Limit:                int64(limit),
		Offset:               int64(offset),
		ShowRankingScore:     true,
	})
	if searchErr != nil {
		s.log.Error("Failed to search user profiles in search index",
			logger.String("query", query),
			logger.String("query_type", string(queryType)),
			logger.Int("limit", limit),
			logger.Int("offset", offset),
			logger.Error(searchErr),
		)
	} else {
		s.log.Info("Search result found in search index",
			logger.String("query", query),
			logger.String("query_type", string(queryType)),
			logger.Int("limit", limit),
			logger.Int("offset", offset),
			logger.Int("hits", len(searchResultIds.Hits)),
		)
		var userIds []string
		for _, hit := range searchResultIds.Hits {
			if userID, ok := hit["user_id"].(string); ok {
				userIds = append(userIds, userID)
			}
		}

		profilesMap, err := s.userRepo.GetProfileByUserIDs(ctx, userIds)
		if err != nil {
			s.log.Error("Failed to get profiles by user IDs",
				logger.Strings("user_ids", userIds),
				logger.Error(err),
			)
			return nil, 0, err
		}

		var profiles []*domain.Profile
		for _, userID := range userIds {
			if profile, exists := profilesMap[userID]; exists {
				profiles = append(profiles, profile)
			}
		}

		return profiles, int(searchResultIds.TotalHits), nil
		// TODO: we can return the search results directly without hitting the database, but we need to convert the search result to the profile struct
	}

	var users []*domain.Profile
	var total int64
	switch queryType {
	case domain.UserQueryTypeName:
		users, total, err = s.userRepo.SearchProfilesByFullName(ctx, query, limit, offset)

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
