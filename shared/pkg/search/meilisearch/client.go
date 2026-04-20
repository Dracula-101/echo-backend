package meilisearch

import (
	"context"
	"encoding/json"
	"time"

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

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

func (c *client) logOp(ctx context.Context, op string) logger.Logger {
	return c.log.WithContext(ctx).With(
		logger.String("component", "meilisearch"),
		logger.String("op", op),
	)
}

// -----------------------------------------------------------------------------
// Document operations
// -----------------------------------------------------------------------------

func (c *client) IndexOne(ctx context.Context, indexName string, document search.Document) *search.SearchError {
	return c.IndexMany(ctx, indexName, []search.Document{document})
}

func (c *client) IndexMany(ctx context.Context, indexName string, documents []search.Document) *search.SearchError {
	log := c.logOp(ctx, "IndexMany").With(
		logger.String("index", indexName),
		logger.Int("doc_count", len(documents)),
	)

	start := time.Now()

	task, err := c.ms.Index(indexName).AddDocumentsWithContext(ctx, marshalDocuments(documents))
	if err != nil {
		log.Error("add documents failed", logger.String("error", err.Error()))
		return mapError(err)
	}

	log.Info("documents submitted", logger.Int("task_uid", int(task.TaskUID)))

	if serr := c.awaitTask(ctx, task.TaskUID); serr != nil {
		log.Error("task failed", logger.String("error", serr.Error()))
		return serr
	}

	log.Info("indexing completed", logger.Int("duration_ms", int(time.Since(start).Milliseconds())))
	return nil
}

func (c *client) UpdateDocuments(ctx context.Context, indexName string, documents []search.Document) *search.SearchError {
	log := c.logOp(ctx, "UpdateDocuments").With(
		logger.String("index", indexName),
		logger.Int("doc_count", len(documents)),
	)

	start := time.Now()

	task, err := c.ms.Index(indexName).UpdateDocumentsWithContext(ctx, marshalDocuments(documents))
	if err != nil {
		log.Error("update documents failed", logger.String("error", err.Error()))
		return mapError(err)
	}

	log.Info("update submitted", logger.Int("task_uid", int(task.TaskUID)))

	if serr := c.awaitTask(ctx, task.TaskUID); serr != nil {
		log.Error("update task failed", logger.String("error", serr.Error()))
		return serr
	}

	log.Info("update completed", logger.Int("duration_ms", int(time.Since(start).Milliseconds())))
	return nil
}

func (c *client) DeleteDocuments(ctx context.Context, indexName string, ids ...string) *search.SearchError {
	log := c.logOp(ctx, "DeleteDocuments").With(
		logger.String("index", indexName),
		logger.Int("id_count", len(ids)),
	)

	start := time.Now()

	task, err := c.ms.Index(indexName).DeleteDocumentsWithContext(ctx, ids)
	if err != nil {
		log.Error("delete failed", logger.String("error", err.Error()))
		return mapError(err)
	}

	log.Info("delete submitted", logger.Int("task_uid", int(task.TaskUID)))

	if serr := c.awaitTask(ctx, task.TaskUID); serr != nil {
		log.Error("delete task failed", logger.String("error", serr.Error()))
		return serr
	}

	log.Info("delete completed", logger.Int("duration_ms", int(time.Since(start).Milliseconds())))
	return nil
}

func (c *client) DeleteAllDocuments(ctx context.Context, indexName string) *search.SearchError {
	log := c.logOp(ctx, "DeleteAllDocuments").With(
		logger.String("index", indexName),
	)

	start := time.Now()

	task, err := c.ms.Index(indexName).DeleteAllDocumentsWithContext(ctx)
	if err != nil {
		log.Error("delete all failed", logger.String("error", err.Error()))
		return mapError(err)
	}

	if serr := c.awaitTask(ctx, task.TaskUID); serr != nil {
		log.Error("delete all task failed", logger.String("error", serr.Error()))
		return serr
	}

	log.Info("delete all completed", logger.Int("duration_ms", int(time.Since(start).Milliseconds())))
	return nil
}

func (c *client) GetDocument(ctx context.Context, indexName, id string, dest interface{}) *search.SearchError {
	log := c.logOp(ctx, "GetDocument").With(
		logger.String("index", indexName),
		logger.String("id", id),
	)

	if err := c.ms.Index(indexName).GetDocumentWithContext(ctx, id, &ms.DocumentQuery{}, dest); err != nil {
		log.Error("get document failed", logger.String("error", err.Error()))
		return mapError(err)
	}

	log.Info("get document success")
	return nil
}

// -----------------------------------------------------------------------------
// Search operations
// -----------------------------------------------------------------------------

func (c *client) Search(ctx context.Context, indexName, query string, opts *search.QueryOptions) (*search.Result, *search.SearchError) {
	log := c.logOp(ctx, "Search").With(
		logger.String("index", indexName),
		logger.String("query", query),
	)

	start := time.Now()

	resp, err := c.ms.Index(indexName).SearchWithContext(ctx, query, buildSearchRequest(query, opts))
	if err != nil {
		log.Error("search failed", logger.String("error", err.Error()))
		return nil, mapError(err)
	}

	log.Info("search success",
		logger.Int("hits", len(resp.Hits)),
		logger.Int("duration_ms", int(time.Since(start).Milliseconds())),
	)

	return toResult(resp, log), nil
}

func (c *client) SearchMulti(ctx context.Context, queries []search.MultiQuery) (*search.MultiResult, *search.SearchError) {
	log := c.logOp(ctx, "SearchMulti").With(
		logger.Int("query_count", len(queries)),
	)

	start := time.Now()

	msQueries := make([]*ms.SearchRequest, len(queries))
	for i, q := range queries {
		req := buildSearchRequest(q.Query, q.Opts)
		req.IndexUID = q.IndexUID
		msQueries[i] = req
	}

	resp, err := c.ms.MultiSearchWithContext(ctx, &ms.MultiSearchRequest{Queries: msQueries})
	if err != nil {
		log.Error("multisearch failed", logger.String("error", err.Error()))
		return nil, mapError(err)
	}

	log.Info("multisearch success", logger.Int("duration_ms", int(time.Since(start).Milliseconds())))

	results := make([]search.Result, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = *toResult(&r, log)
	}

	return &search.MultiResult{Results: results}, nil
}

func (c *client) SearchFaceted(ctx context.Context, indexName, _ string, opts *search.FacetOptions) (*search.FacetResult, *search.SearchError) {
	log := c.logOp(ctx, "SearchFaceted").With(
		logger.String("index", indexName),
		logger.String("facet", opts.FacetName),
	)

	raw, err := c.ms.Index(indexName).FacetSearchWithContext(ctx, &ms.FacetSearchRequest{
		FacetName:  opts.FacetName,
		FacetQuery: opts.FacetQuery,
	})
	if err != nil {
		log.Error("facet search failed", logger.String("error", err.Error()))
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

	if err := json.Unmarshal(*raw, &resp); err != nil {
		log.Error("facet unmarshal failed", logger.String("error", err.Error()))
		return nil, search.ErrInternal("failed to unmarshal facet search response", err)
	}

	hits := make([]search.FacetHit, len(resp.FacetHits))
	for i, h := range resp.FacetHits {
		hits[i] = search.FacetHit{Value: h.Value, Count: h.Count}
	}

	log.Info("facet search success", logger.Int("hits", len(hits)))

	return &search.FacetResult{
		FacetHits:        hits,
		FacetQuery:       resp.FacetQuery,
		ProcessingTimeMs: resp.ProcessingTimeMs,
	}, nil
}

// -----------------------------------------------------------------------------
// Index management
// -----------------------------------------------------------------------------

func (c *client) CreateIndex(ctx context.Context, uid string, opts *search.IndexOptions) *search.SearchError {
	log := c.logOp(ctx, "CreateIndex").With(logger.String("uid", uid))

	task, err := c.ms.CreateIndexWithContext(ctx, &ms.IndexConfig{
		Uid:        uid,
		PrimaryKey: opts.PrimaryKey,
	})
	if err != nil {
		log.Error("create index failed", logger.String("error", err.Error()))
		return mapError(err)
	}

	return c.awaitTask(ctx, task.TaskUID)
}

func (c *client) DeleteIndex(ctx context.Context, uid string) *search.SearchError {
	log := c.logOp(ctx, "DeleteIndex").With(logger.String("uid", uid))

	task, err := c.ms.DeleteIndexWithContext(ctx, uid)
	if err != nil {
		log.Error("delete index failed", logger.String("error", err.Error()))
		return mapError(err)
	}

	return c.awaitTask(ctx, task.TaskUID)
}

func (c *client) GetIndex(ctx context.Context, uid string) (*search.IndexInfo, *search.SearchError) {
	log := c.logOp(ctx, "GetIndex").With(logger.String("uid", uid))

	info, err := c.ms.GetIndexWithContext(ctx, uid)
	if err != nil {
		log.Error("get index failed", logger.String("error", err.Error()))
		return nil, mapError(err)
	}

	return indexResultToInfo(info), nil
}

func (c *client) ListIndexes(ctx context.Context) ([]search.IndexInfo, *search.SearchError) {
	log := c.logOp(ctx, "ListIndexes")

	resp, err := c.ms.ListIndexesWithContext(ctx, &ms.IndexesQuery{})
	if err != nil {
		log.Error("list indexes failed", logger.String("error", err.Error()))
		return nil, mapError(err)
	}

	out := make([]search.IndexInfo, len(resp.Results))
	for i, idx := range resp.Results {
		out[i] = *indexResultToInfo(idx)
	}

	log.Info("list indexes success", logger.Int("count", len(out)))
	return out, nil
}

func (c *client) UpdateSettings(ctx context.Context, uid string, settings *search.IndexSettings) *search.SearchError {
	log := c.logOp(ctx, "UpdateSettings").With(logger.String("uid", uid))

	task, err := c.ms.Index(uid).UpdateSettingsWithContext(ctx, toMSSettings(settings))
	if err != nil {
		log.Error("update settings failed", logger.String("error", err.Error()))
		return mapError(err)
	}

	return c.awaitTask(ctx, task.TaskUID)
}

func (c *client) GetSettings(ctx context.Context, uid string) (*search.IndexSettings, *search.SearchError) {
	log := c.logOp(ctx, "GetSettings").With(logger.String("uid", uid))

	msSettings, err := c.ms.Index(uid).GetSettingsWithContext(ctx)
	if err != nil {
		log.Error("get settings failed", logger.String("error", err.Error()))
		return nil, mapError(err)
	}

	return fromMSSettings(msSettings), nil
}

// -----------------------------------------------------------------------------
// lifecycle
// -----------------------------------------------------------------------------

func (c *client) Ping(ctx context.Context) *search.SearchError {
	log := c.logOp(ctx, "Ping")

	if _, err := c.ms.HealthWithContext(ctx); err != nil {
		log.Error("ping failed", logger.String("error", err.Error()))
		return mapError(err)
	}

	log.Info("ping ok")
	return nil
}

func (c *client) Health(ctx context.Context) (*search.HealthStatus, *search.SearchError) {
	log := c.logOp(ctx, "Health")

	h, err := c.ms.HealthWithContext(ctx)
	if err != nil {
		log.Error("health failed", logger.String("error", err.Error()))
		return nil, mapError(err)
	}

	v, err := c.ms.VersionWithContext(ctx)
	if err != nil {
		log.Error("version fetch failed", logger.String("error", err.Error()))
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
	log := c.logOp(ctx, "Stats").With(logger.String("index", indexName))

	s, err := c.ms.Index(indexName).GetStatsWithContext(ctx)
	if err != nil {
		log.Error("stats failed", logger.String("error", err.Error()))
		return nil, mapError(err)
	}

	return &search.IndexStats{
		NumberOfDocuments: s.NumberOfDocuments,
		IsIndexing:        s.IsIndexing,
		FieldDistribution: s.FieldDistribution,
	}, nil
}

func (c *client) Close() *search.SearchError { return nil }

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

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
