package meilisearch

import (
	"encoding/json"

	ms "github.com/meilisearch/meilisearch-go"

	"shared/pkg/logger"
	"shared/pkg/search"
)

func buildSearchRequest(query string, opts *search.QueryOptions) *ms.SearchRequest {
	req := &ms.SearchRequest{Query: query}
	if opts == nil {
		return req
	}

	req.Filter = opts.Filter
	req.Facets = opts.Facets
	req.AttributesToRetrieve = opts.AttributesToRetrieve
	req.AttributesToHighlight = opts.AttributesToHighlight
	req.AttributesToSearchOn = opts.AttributesToSearchOn
	req.HighlightPreTag = opts.HighlightPreTag
	req.HighlightPostTag = opts.HighlightPostTag
	req.AttributesToCrop = opts.AttributesToCrop
	req.CropLength = opts.CropLength
	req.ShowMatchesPosition = opts.ShowMatchesPosition
	req.Sort = opts.Sort
	req.MatchingStrategy = ms.MatchingStrategy(opts.MatchingStrategy)
	req.Limit = opts.Limit
	req.Offset = opts.Offset
	req.HitsPerPage = opts.HitsPerPage
	req.Page = opts.Page
	req.Distinct = opts.Distinct
	req.ShowRankingScore = opts.ShowRankingScore

	return req
}

func toResult(r *ms.SearchResponse, log logger.Logger) *search.Result {
	log.Debug("Converting Meilisearch response to search.Result",
		logger.Any("raw_response", r),
		logger.Int("hits_count", len(r.Hits)),
	)
	hits := make([]map[string]interface{}, len(r.Hits))
	for i, h := range r.Hits {
		decoded := make(map[string]interface{})
		switch hit := h.(type) {
		case map[string]interface{}:
			for k, v := range hit {
				decoded[k] = v
			}
		case map[string]json.RawMessage:
			for k, raw := range hit {
				var v interface{}
				if err := json.Unmarshal(raw, &v); err == nil {
					decoded[k] = v
					continue
				}

				decoded[k] = string(raw)
			}
		default:
			// Keep compatibility across SDK versions that return hits as generic interfaces.
			b, err := json.Marshal(hit)
			if err == nil {
				_ = json.Unmarshal(b, &decoded)
			}
		}
		hits[i] = decoded
	}

	result := &search.Result{
		Hits:               hits,
		Query:              r.Query,
		ProcessingTimeMs:   r.ProcessingTimeMs,
		Limit:              r.Limit,
		Offset:             r.Offset,
		EstimatedTotalHits: r.EstimatedTotalHits,
		TotalHits:          r.TotalHits,
		TotalPages:         r.TotalPages,
		HitsPerPage:        r.HitsPerPage,
		Page:               r.Page,
	}

	return result
}

func toMSSettings(s *search.IndexSettings) *ms.Settings {
	if s == nil {
		return nil
	}
	settings := &ms.Settings{
		RankingRules:         s.RankingRules,
		DistinctAttribute:    s.DistinctAttribute,
		SearchableAttributes: s.SearchableAttributes,
		DisplayedAttributes:  s.DisplayedAttributes,
		StopWords:            s.StopWords,
		Synonyms:             s.Synonyms,
		FilterableAttributes: s.FilterableAttributes,
		SortableAttributes:   s.SortableAttributes,
	}

	if s.TypoTolerance != nil {
		settings.TypoTolerance = &ms.TypoTolerance{
			Enabled:             s.TypoTolerance.Enabled,
			DisableOnWords:      s.TypoTolerance.DisableOnWords,
			DisableOnAttributes: s.TypoTolerance.DisableOnAttributes,
		}
		if s.TypoTolerance.MinWordSizeForTypos != nil {
			settings.TypoTolerance.MinWordSizeForTypos = ms.MinWordSizeForTypos{
				OneTypo:  int64(s.TypoTolerance.MinWordSizeForTypos.OneTypo),
				TwoTypos: int64(s.TypoTolerance.MinWordSizeForTypos.TwoTypos),
			}
		}
	}

	if s.Pagination != nil {
		settings.Pagination = &ms.Pagination{MaxTotalHits: int64(s.Pagination.MaxTotalHits)}
	}

	if s.Faceting != nil {
		settings.Faceting = &ms.Faceting{MaxValuesPerFacet: int64(s.Faceting.MaxValuesPerFacet)}
	}

	return settings
}

func fromMSSettings(s *ms.Settings) *search.IndexSettings {
	if s == nil {
		return nil
	}
	settings := &search.IndexSettings{
		RankingRules:         s.RankingRules,
		DistinctAttribute:    s.DistinctAttribute,
		SearchableAttributes: s.SearchableAttributes,
		DisplayedAttributes:  s.DisplayedAttributes,
		StopWords:            s.StopWords,
		Synonyms:             s.Synonyms,
		FilterableAttributes: s.FilterableAttributes,
		SortableAttributes:   s.SortableAttributes,
	}

	if s.TypoTolerance != nil {
		settings.TypoTolerance = &search.TypoTolerance{
			Enabled:             s.TypoTolerance.Enabled,
			DisableOnWords:      s.TypoTolerance.DisableOnWords,
			DisableOnAttributes: s.TypoTolerance.DisableOnAttributes,
		}
	}

	if s.Pagination != nil {
		settings.Pagination = &search.PaginationSettings{MaxTotalHits: int(s.Pagination.MaxTotalHits)}
	}

	if s.Faceting != nil {
		settings.Faceting = &search.FacetingSettings{MaxValuesPerFacet: int(s.Faceting.MaxValuesPerFacet)}
	}

	return settings
}
