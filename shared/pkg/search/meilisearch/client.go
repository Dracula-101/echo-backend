package meilisearch

import (
	"context"
	"encoding/json"

	ms "github.com/meilisearch/meilisearch-go"

	"shared/pkg/logger"
	"shared/pkg/search"
)

type client struct {
	ms  ms.ServiceManager
	cfg Config
	log logger.Logger
}

func New(cfg Config, log logger.Logger) (search.Search, error) {
	cfg.setDefaults()
	return &client{
		ms:  ms.New(cfg.Host, ms.WithAPIKey(cfg.APIKey)),
		cfg: cfg,
		log: log,
	}, nil
}

// -- Document operations -----------------------------------------------------

func (c *client) IndexOne(ctx context.Context, indexName string, document search.Document) *search.SearchError {
	return c.IndexMany(ctx, indexName, []search.Document{document})
}

func (c *client) IndexMany(ctx context.Context, indexName string, documents []search.Document) *search.SearchError {
	task, err := c.ms.Index(indexName).AddDocumentsWithContext(ctx, marshalDocuments(documents), nil)
	if err != nil {
		return mapError(err)
	}
	return c.awaitTask(ctx, task.TaskUID)
}

func (c *client) UpdateDocuments(ctx context.Context, indexName string, documents []search.Document) *search.SearchError {
	task, err := c.ms.Index(indexName).UpdateDocumentsWithContext(ctx, marshalDocuments(documents), nil)
	if err != nil {
		return mapError(err)
	}
	return c.awaitTask(ctx, task.TaskUID)
}

func (c *client) DeleteDocuments(ctx context.Context, indexName string, ids ...string) *search.SearchError {
	task, err := c.ms.Index(indexName).DeleteDocumentsWithContext(ctx, ids, nil)
	if err != nil {
		return mapError(err)
	}
	return c.awaitTask(ctx, task.TaskUID)
}

func (c *client) DeleteAllDocuments(ctx context.Context, indexName string) *search.SearchError {
	task, err := c.ms.Index(indexName).DeleteAllDocumentsWithContext(ctx, nil)
	if err != nil {
		return mapError(err)
	}
	return c.awaitTask(ctx, task.TaskUID)
}

func (c *client) GetDocument(ctx context.Context, indexName, id string, dest interface{}) *search.SearchError {
	if err := c.ms.Index(indexName).GetDocumentWithContext(ctx, id, &ms.DocumentQuery{}, dest); err != nil {
		return mapError(err)
	}
	return nil
}

// -- Search operations --------------------------------------------------------

func (c *client) Search(ctx context.Context, indexName, query string, opts *search.QueryOptions) (*search.Result, *search.SearchError) {
	resp, err := c.ms.Index(indexName).SearchWithContext(ctx, query, buildSearchRequest(query, opts))
	if err != nil {
		return nil, mapError(err)
	}
	return toResult(resp), nil
}

func (c *client) SearchMulti(ctx context.Context, queries []search.MultiQuery) (*search.MultiResult, *search.SearchError) {
	msQueries := make([]*ms.SearchRequest, len(queries))
	for i, q := range queries {
		req := buildSearchRequest(q.Query, q.Opts)
		req.IndexUID = q.IndexUID
		msQueries[i] = req
	}

	resp, err := c.ms.MultiSearchWithContext(ctx, &ms.MultiSearchRequest{Queries: msQueries})
	if err != nil {
		return nil, mapError(err)
	}

	results := make([]search.Result, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = *toResult(&r)
	}
	return &search.MultiResult{Results: results}, nil
}

func (c *client) SearchFaceted(ctx context.Context, indexName, _ string, opts *search.FacetOptions) (*search.FacetResult, *search.SearchError) {
	raw, err := c.ms.Index(indexName).FacetSearchWithContext(ctx, &ms.FacetSearchRequest{
		FacetName:  opts.FacetName,
		FacetQuery: opts.FacetQuery,
	})
	if err != nil {
		return nil, mapError(err)
	}

	var resp struct {
		FacetHits []struct {
			Value string `json:"value"`
			Count int64  `json:"count"`
		} `json:"facetHits"`
		FacetQuery       string `json:"facetQuery"`
		ProcessingTimeMs int64  `json:"processingTimeMs"`
	}
	if jsonErr := json.Unmarshal(*raw, &resp); jsonErr != nil {
		return nil, search.ErrInternal("failed to unmarshal facet search response", jsonErr)
	}

	hits := make([]search.FacetHit, len(resp.FacetHits))
	for i, h := range resp.FacetHits {
		hits[i] = search.FacetHit{Value: h.Value, Count: h.Count}
	}
	return &search.FacetResult{
		FacetHits:        hits,
		FacetQuery:       resp.FacetQuery,
		ProcessingTimeMs: resp.ProcessingTimeMs,
	}, nil
}

// -- Index management ---------------------------------------------------------

func (c *client) CreateIndex(ctx context.Context, uid string, opts *search.IndexOptions) *search.SearchError {
	req := &ms.IndexConfig{Uid: uid}
	if opts != nil {
		req.PrimaryKey = opts.PrimaryKey
	}
	task, err := c.ms.CreateIndexWithContext(ctx, req)
	if err != nil {
		return mapError(err)
	}
	return c.awaitTask(ctx, task.TaskUID)
}

func (c *client) DeleteIndex(ctx context.Context, uid string) *search.SearchError {
	task, err := c.ms.DeleteIndexWithContext(ctx, uid)
	if err != nil {
		return mapError(err)
	}
	return c.awaitTask(ctx, task.TaskUID)
}

func (c *client) GetIndex(ctx context.Context, uid string) (*search.IndexInfo, *search.SearchError) {
	info, err := c.ms.GetIndexWithContext(ctx, uid)
	if err != nil {
		return nil, mapError(err)
	}
	return indexResultToInfo(info), nil
}

func (c *client) ListIndexes(ctx context.Context) ([]search.IndexInfo, *search.SearchError) {
	resp, err := c.ms.ListIndexesWithContext(ctx, &ms.IndexesQuery{})
	if err != nil {
		return nil, mapError(err)
	}
	infos := make([]search.IndexInfo, len(resp.Results))
	for i, idx := range resp.Results {
		infos[i] = *indexResultToInfo(idx)
	}
	return infos, nil
}

func (c *client) UpdateSettings(ctx context.Context, uid string, settings *search.IndexSettings) *search.SearchError {
	task, err := c.ms.Index(uid).UpdateSettingsWithContext(ctx, toMSSettings(settings))
	if err != nil {
		return mapError(err)
	}
	return c.awaitTask(ctx, task.TaskUID)
}

func (c *client) GetSettings(ctx context.Context, uid string) (*search.IndexSettings, *search.SearchError) {
	msSettings, err := c.ms.Index(uid).GetSettingsWithContext(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return fromMSSettings(msSettings), nil
}

// -- Lifecycle ----------------------------------------------------------------

func (c *client) Ping(ctx context.Context) *search.SearchError {
	if _, err := c.ms.HealthWithContext(ctx); err != nil {
		return mapError(err)
	}
	return nil
}

func (c *client) Health(ctx context.Context) (*search.HealthStatus, *search.SearchError) {
	h, err := c.ms.HealthWithContext(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	v, err := c.ms.VersionWithContext(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return &search.HealthStatus{
		Status: h.Status,
		Version: search.VersionInfo{
			CommitSHA:  v.CommitSha,
			CommitDate: v.CommitDate,
			PkgVersion: v.PkgVersion,
		},
	}, nil
}

func (c *client) Stats(ctx context.Context, indexName string) (*search.IndexStats, *search.SearchError) {
	s, err := c.ms.Index(indexName).GetStatsWithContext(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return &search.IndexStats{
		NumberOfDocuments: s.NumberOfDocuments,
		IsIndexing:        s.IsIndexing,
		FieldDistribution: s.FieldDistribution,
	}, nil
}

func (c *client) Close() *search.SearchError { return nil }

// -- Private ------------------------------------------------------------------

func marshalDocuments(docs []search.Document) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(docs))
	for _, d := range docs {
		b, err := json.Marshal(d)
		if err != nil {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

func indexResultToInfo(r *ms.IndexResult) *search.IndexInfo {
	return &search.IndexInfo{
		UID:        r.UID,
		PrimaryKey: r.PrimaryKey,
		CreatedAt:  r.CreatedAt.String(),
		UpdatedAt:  r.UpdatedAt.String(),
	}
}