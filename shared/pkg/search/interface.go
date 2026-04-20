package search

import "context"

type Search interface {
	// Document operations
	IndexOne(ctx context.Context, indexName string, document Document) *SearchError
	IndexMany(ctx context.Context, indexName string, documents []Document) *SearchError
	UpdateDocuments(ctx context.Context, indexName string, documents []Document) *SearchError
	DeleteDocuments(ctx context.Context, indexName string, ids ...string) *SearchError
	DeleteAllDocuments(ctx context.Context, indexName string) *SearchError
	GetDocument(ctx context.Context, indexName string, id string, dest interface{}) *SearchError

	// Search operations
	Search(ctx context.Context, indexName string, query string, opts *QueryOptions) (*Result, *SearchError)
	SearchMulti(ctx context.Context, queries []MultiQuery) (*MultiResult, *SearchError)
	SearchFaceted(ctx context.Context, indexName string, query string, opts *FacetOptions) (*FacetResult, *SearchError)

	// Index management
	CreateIndex(ctx context.Context, uid string, opts *IndexOptions) *SearchError
	DeleteIndex(ctx context.Context, uid string) *SearchError
	GetIndex(ctx context.Context, uid string) (*IndexInfo, *SearchError)
	ListIndexes(ctx context.Context) ([]IndexInfo, *SearchError)
	UpdateSettings(ctx context.Context, uid string, settings *IndexSettings) *SearchError
	GetSettings(ctx context.Context, uid string) (*IndexSettings, *SearchError)

	// Lifecycle
	Ping(ctx context.Context) *SearchError
	Health(ctx context.Context) (*HealthStatus, *SearchError)
	Stats(ctx context.Context, indexName string) (*IndexStats, *SearchError)
	Close() *SearchError
}

type Document interface {
	SearchID() string
	SearchIndex() string
}
