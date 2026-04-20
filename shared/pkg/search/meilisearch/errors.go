package meilisearch

import (
	"strings"

	ms "github.com/meilisearch/meilisearch-go"

	"shared/pkg/search"
)

func mapError(err error) *search.SearchError {
	if err == nil {
		return nil
	}

	msErr, ok := err.(*ms.Error)
	if !ok {
		return search.ErrInternal("unexpected error", err)
	}

	code := msErr.MeilisearchApiError.Code
	msg := msErr.MeilisearchApiError.Message

	switch {
	case msErr.StatusCode == 404 || containsAny(code, "not_found", "index_not_found", "document_not_found"):
		return search.ErrNotFound(msg, err)

	case msErr.StatusCode == 401 || msErr.StatusCode == 403:
		return search.NewError(search.ErrCodeUnauthorized, msg, err)

	case containsAny(code, "invalid_request", "bad_request", "invalid_content_type"):
		return search.ErrInvalidRequest(msg, err)

	case containsAny(code, "index_already_exists"):
		return search.NewError(search.ErrCodeIndexExists, msg, err)

	default:
		return search.ErrInternal(msg, err)
	}
}

func containsAny(s string, substrings ...string) bool {
	s = strings.ToLower(s)
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
