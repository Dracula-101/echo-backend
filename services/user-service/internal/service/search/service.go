package search

import (
	"context"
	"fmt"
	"shared/pkg/logger"
	"shared/pkg/search"
	"user-service/internal/domain"
	repo "user-service/internal/repo"
	"user-service/internal/service/cache"
)

var SearchIndexName = "users"

var SearchIndexSettings = &search.IndexSettings{
	SearchableAttributes: []string{"username", "display_name", "first_name", "last_name"},
	FilterableAttributes: []string{"search_visibility", "deactivated_at"},
}

type SearchService interface {
	CreateIndex(ctx context.Context) error
	AddOrUpdateProfileInIndex(ctx context.Context, profile domain.SearchProfile) error
	SearchProfiles(ctx context.Context, query string, queryType domain.UserQueryType, limit, offset int, requesterUserId *string) ([]*domain.Profile, int, error)
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

	err := s.searchClient.UpdateSettings(ctx, SearchIndexName, SearchIndexSettings)
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

func (s *searchService) SearchProfiles(ctx context.Context, query string, queryType domain.UserQueryType, limit, offset int, requesterUserId *string) ([]*domain.Profile, int, error) {
	s.log.Info("Searching user profiles",
		logger.String("query", query),
		logger.String("query_type", string(queryType)),
		logger.Int("limit", limit),
		logger.Int("offset", offset),
	)

	cacheKey := fmt.Sprintf("%s:%s:%d:%d", queryType, query, limit, offset)
	cached, err := s.userCache.GetSearchCache(ctx, cacheKey)
	if err != nil {
		s.log.Error("Failed to retrieve search result from cache",
			logger.String("cache_key", cacheKey),
			logger.Error(err),
		)
	}
	if cached != nil && len(cached.Users) > 0 {
		s.log.Info("Search result found in cache", logger.String("cache_key", cacheKey))
		users, total := excludeRequesterProfile(cached.Users, int(cached.Total), requesterUserId)
		return users, total, nil
	}

	profiles, total, searchErr := s.meilisearchByType(ctx, query, queryType, limit, offset, requesterUserId)
	if searchErr == nil {
		profiles, total = excludeRequesterProfile(profiles, total, requesterUserId)
		s.userCache.SetSearchCache(ctx, cacheKey, &cache.SearchResult{
			Users:  profiles,
			Total:  int64(total),
			Offset: int64(offset),
		})
		return profiles, total, nil
	}
	s.log.Error("Meilisearch failed, falling back to Postgres",
		logger.String("query", query),
		logger.String("query_type", string(queryType)),
		logger.Error(searchErr),
	)

	var pgUsers []*domain.Profile
	var pgTotal int64
	switch queryType {
	case domain.UserQueryTypeName:
		pgUsers, pgTotal, err = s.userRepo.SearchProfilesByFullName(ctx, query, limit, offset)
	case domain.UserQueryTypeUsername:
		pgUsers, pgTotal, err = s.userRepo.SearchProfilesByUsername(ctx, query, limit, offset)
	case domain.UserQueryTypeAll:
		pgUsers, pgTotal, err = s.userRepo.SearchProfilesByUsername(ctx, query, limit, offset)
	}
	if err != nil {
		s.log.Error("Failed to search user profiles",
			logger.String("query", query),
			logger.String("query_type", string(queryType)),
			logger.Error(err),
		)
		return nil, 0, err
	}

	pgProfiles, pgTotalInt := excludeRequesterProfile(pgUsers, int(pgTotal), requesterUserId)

	s.userCache.SetSearchCache(ctx, cacheKey, &cache.SearchResult{ //nolint:errcheck
		Users:  pgProfiles,
		Total:  int64(pgTotalInt),
		Offset: int64(offset),
	})

	return pgProfiles, pgTotalInt, nil
}

func excludeRequesterProfile(profiles []*domain.Profile, total int, requesterUserId *string) ([]*domain.Profile, int) {
	if requesterUserId == nil || *requesterUserId == "" || len(profiles) == 0 {
		return profiles, total
	}

	filtered := make([]*domain.Profile, 0, len(profiles))
	removed := 0
	for _, p := range profiles {
		if p == nil {
			continue
		}
		if p.UserID == *requesterUserId {
			removed++
			continue
		}
		filtered = append(filtered, p)
	}

	if removed == 0 {
		return profiles, total
	}

	adjustedTotal := total - removed
	if adjustedTotal < 0 {
		adjustedTotal = 0
	}

	return filtered, adjustedTotal
}

// meilisearchByType runs the appropriate Meilisearch query for the given type.
// For "all" it fans out to username + name in a single multi-search call.
func (s *searchService) meilisearchByType(ctx context.Context, query string, queryType domain.UserQueryType, limit, offset int, requesterUserId *string) ([]*domain.Profile, int, error) {
	retrieve := []string{"user_id"}

	switch queryType {
	case domain.UserQueryTypeUsername:
		res, serr := s.searchClient.Search(ctx, SearchIndexName, query, &search.QueryOptions{
			Filter:               "search_visibility = true AND deactivated_at IS NULL",
			AttributesToSearchOn: []string{"username"},
			AttributesToRetrieve: retrieve,
			Limit:                int64(limit),
			Offset:               int64(offset),
			ShowRankingScore:     true,
		})
		if serr != nil {
			return nil, 0, serr
		}
		return s.hitsToProfiles(ctx, res.Hits, int(res.TotalHits), requesterUserId)

	case domain.UserQueryTypeName:
		res, serr := s.searchClient.Search(ctx, SearchIndexName, query, &search.QueryOptions{
			Filter:               "search_visibility = true",
			AttributesToSearchOn: []string{"display_name", "first_name", "last_name"},
			AttributesToRetrieve: retrieve,
			Limit:                int64(limit),
			Offset:               int64(offset),
			ShowRankingScore:     true,
		})
		if serr != nil {
			return nil, 0, serr
		}
		return s.hitsToProfiles(ctx, res.Hits, int(res.TotalHits), requesterUserId)

	case domain.UserQueryTypeAll:
		multi, serr := s.searchClient.SearchMulti(ctx, []search.MultiQuery{
			{
				IndexUID: SearchIndexName,
				Query:    query,
				Opts: &search.QueryOptions{
					Filter:               "search_visibility = true AND deactivated_at IS NULL",
					AttributesToSearchOn: []string{"username"},
					AttributesToRetrieve: retrieve,
					Limit:                int64(limit),
					Offset:               int64(offset),
					ShowRankingScore:     true,
				},
			},
			{
				IndexUID: SearchIndexName,
				Query:    query,
				Opts: &search.QueryOptions{
					Filter:               "search_visibility = true",
					AttributesToSearchOn: []string{"display_name", "first_name", "last_name"},
					AttributesToRetrieve: retrieve,
					Limit:                int64(limit),
					Offset:               int64(offset),
					ShowRankingScore:     true,
				},
			},
		})
		if serr != nil {
			return nil, 0, serr
		}

		// Merge hits from both results, deduplicate by user_id.
		seen := make(map[string]struct{})
		var merged []map[string]interface{}
		total := int64(0)
		for _, r := range multi.Results {
			total += r.TotalHits
			for _, h := range r.Hits {
				uid, _ := h["user_id"].(string)
				if uid == "" {
					continue
				}
				if _, dup := seen[uid]; dup {
					continue
				}
				seen[uid] = struct{}{}
				merged = append(merged, h)
			}
		}
		return s.hitsToProfiles(ctx, merged, int(total), requesterUserId)
	}

	return nil, 0, nil
}

func (s *searchService) hitsToProfiles(ctx context.Context, hits []map[string]interface{}, total int, requesterUserId *string) ([]*domain.Profile, int, error) {
	userIds := make([]string, 0, len(hits))
	for _, hit := range hits {
		if uid, ok := hit["user_id"].(string); ok && uid != "" {
			userIds = append(userIds, uid)
		}
	}
	if len(userIds) == 0 {
		return nil, 0, nil
	}

	profilesMap, err := s.userRepo.GetProfileByUserIDs(ctx, userIds, requesterUserId)
	if err != nil {
		s.log.Error("Failed to get profiles by user IDs",
			logger.Strings("user_ids", userIds),
			logger.Error(err),
		)
		return nil, 0, err
	}

	profiles := make([]*domain.Profile, 0, len(userIds))
	for _, uid := range userIds {
		if p, ok := profilesMap[uid]; ok {
			profiles = append(profiles, p)
		}
	}
	return profiles, total, nil
}
