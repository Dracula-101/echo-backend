package pagination

import (
	"fmt"
	"math"
	"net/http"
)

// Pagination represents pagination metadata
type Pagination struct {
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
	TotalItems int64  `json:"total_items"`
	HasNext    bool   `json:"has_next"`
	HasPrev    bool   `json:"has_prev"`
	NextPage   *int   `json:"next_page,omitempty"`
	PrevPage   *int   `json:"prev_page,omitempty"`
	SortBy     string `json:"sort_by,omitempty"`
	SortDir    string `json:"sort_dir,omitempty"`
}

// Links represents pagination links
type Links struct {
	First *string `json:"first,omitempty"`
	Prev  *string `json:"prev,omitempty"`
	Next  *string `json:"next,omitempty"`
	Last  *string `json:"last,omitempty"`
	Self  string  `json:"self"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination *Pagination `json:"pagination,omitempty"`
	Links      *Links      `json:"links,omitempty"`
}

// NewPagination creates pagination metadata
func NewPagination(page, pageSize int, totalItems int64) *Pagination {
	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))

	p := &Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		TotalItems: totalItems,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	if p.HasNext {
		nextPage := page + 1
		p.NextPage = &nextPage
	}

	if p.HasPrev {
		prevPage := page - 1
		p.PrevPage = &prevPage
	}

	return p
}

// WithSort adds sorting information to pagination
func (p *Pagination) WithSort(sortBy, sortDir string) *Pagination {
	p.SortBy = sortBy
	p.SortDir = sortDir
	return p
}

// NewLinks creates pagination links from request
func NewLinks(r *http.Request, page, totalPages int) *Links {
	baseURL := fmt.Sprintf("%s://%s%s", scheme(r), r.Host, r.URL.Path)
	query := r.URL.Query()

	links := &Links{
		Self: buildURL(baseURL, query, page),
	}

	if page > 1 {
		first := buildURL(baseURL, query, 1)
		links.First = &first

		prev := buildURL(baseURL, query, page-1)
		links.Prev = &prev
	}

	if page < totalPages {
		next := buildURL(baseURL, query, page+1)
		links.Next = &next

		last := buildURL(baseURL, query, totalPages)
		links.Last = &last
	}

	return links
}

// buildURL builds a URL with updated page parameter
func buildURL(baseURL string, query map[string][]string, page int) string {
	params := make(map[string][]string)
	for k, v := range query {
		if k != "page" {
			params[k] = v
		}
	}

	url := fmt.Sprintf("%s?page=%d", baseURL, page)
	for k, values := range params {
		for _, v := range values {
			url += fmt.Sprintf("&%s=%s", k, v)
		}
	}

	return url
}

// scheme returns http or https based on request
func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}

// CalculateOffset calculates the database offset for pagination
func CalculateOffset(page, pageSize int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * pageSize
}

// ValidatePagination validates and normalizes pagination parameters
func ValidatePagination(page, pageSize, maxPageSize int) (int, int) {
	if page < 1 {
		page = 1
	}

	if pageSize < 1 {
		pageSize = 20 // default
	}

	if maxPageSize > 0 && pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}

// Cursor represents cursor-based pagination
type Cursor struct {
	Next     *string `json:"next,omitempty"`
	Prev     *string `json:"prev,omitempty"`
	HasMore  bool    `json:"has_more"`
	PageSize int     `json:"page_size"`
}

// CursorPaginatedResponse represents a cursor-paginated API response
type CursorPaginatedResponse struct {
	Data   interface{} `json:"data"`
	Cursor *Cursor     `json:"cursor,omitempty"`
}

// NewCursor creates cursor pagination metadata
func NewCursor(next, prev *string, hasMore bool, pageSize int) *Cursor {
	return &Cursor{
		Next:     next,
		Prev:     prev,
		HasMore:  hasMore,
		PageSize: pageSize,
	}
}
