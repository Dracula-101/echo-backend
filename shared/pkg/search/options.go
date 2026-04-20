package search

// QueryOptions controls search behaviour for a single query.
type QueryOptions struct {
	Filter                interface{} // string or []string / [][]string
	Facets                []string
	AttributesToRetrieve  []string
	AttributesToHighlight []string
	HighlightPreTag       string
	HighlightPostTag      string
	AttributesToCrop      []string
	CropLength            int64
	ShowMatchesPosition   bool
	Sort                  []string
	MatchingStrategy      string // "last" | "all" | "frequency"
	Limit                 int64
	Offset                int64
	HitsPerPage           int64
	Page                  int64
	Distinct              string
	ShowRankingScore      bool
}

// FacetOptions extends QueryOptions with facet-specific controls.
type FacetOptions struct {
	QueryOptions
	FacetName  string
	FacetQuery string
}

// IndexOptions configures index creation.
type IndexOptions struct {
	PrimaryKey string
}

// IndexSettings holds the full Meilisearch settings payload.
type IndexSettings struct {
	RankingRules         []string            `json:"rankingRules,omitempty"`
	DistinctAttribute    *string             `json:"distinctAttribute,omitempty"`
	SearchableAttributes []string            `json:"searchableAttributes,omitempty"`
	DisplayedAttributes  []string            `json:"displayedAttributes,omitempty"`
	StopWords            []string            `json:"stopWords,omitempty"`
	Synonyms             map[string][]string `json:"synonyms,omitempty"`
	FilterableAttributes []string            `json:"filterableAttributes,omitempty"`
	SortableAttributes   []string            `json:"sortableAttributes,omitempty"`
	TypoTolerance        *TypoTolerance      `json:"typoTolerance,omitempty"`
	Pagination           *PaginationSettings `json:"pagination,omitempty"`
	Faceting             *FacetingSettings   `json:"faceting,omitempty"`
}

type TypoTolerance struct {
	Enabled             bool `json:"enabled"`
	MinWordSizeForTypos *struct {
		OneTypo  int `json:"oneTypo"`
		TwoTypos int `json:"twoTypos"`
	} `json:"minWordSizeForTypos,omitempty"`
	DisableOnWords      []string `json:"disableOnWords,omitempty"`
	DisableOnAttributes []string `json:"disableOnAttributes,omitempty"`
}

type PaginationSettings struct {
	MaxTotalHits int `json:"maxTotalHits"`
}

type FacetingSettings struct {
	MaxValuesPerFacet int `json:"maxValuesPerFacet"`
}

// MultiQuery represents a single search within a multi-search request.
type MultiQuery struct {
	IndexUID string
	Query    string
	Opts     *QueryOptions
}

// Result is the response for a single search request.
type Result struct {
	Hits               []map[string]interface{}
	Query              string
	ProcessingTimeMs   int64
	Limit              int64
	Offset             int64
	EstimatedTotalHits int64
	TotalHits          int64
	TotalPages         int64
	HitsPerPage        int64
	Page               int64
	FacetDistribution  map[string]map[string]int64
	FacetStats         map[string]FacetStat
}

type FacetStat struct {
	Min float64
	Max float64
}

// MultiResult holds responses for each query in a multi-search.
type MultiResult struct {
	Results []Result
}

// FacetResult holds the response for a facet search.
type FacetResult struct {
	FacetHits        []FacetHit
	FacetQuery       string
	ProcessingTimeMs int64
}

type FacetHit struct {
	Value string
	Count int64
}

// IndexInfo describes a Meilisearch index.
type IndexInfo struct {
	UID        string
	PrimaryKey string
	CreatedAt  string
	UpdatedAt  string
}

// IndexStats holds document and field statistics for an index.
type IndexStats struct {
	NumberOfDocuments int64
	IsIndexing        bool
	FieldDistribution map[string]int64
}
