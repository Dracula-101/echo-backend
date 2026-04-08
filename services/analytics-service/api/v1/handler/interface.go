package handler

import req "shared/server/request"

// AnalyticsHandlerInterface defines the contract for analytics HTTP handlers
type AnalyticsHandlerInterface interface {
	// Event endpoints
	TrackEvent(handler *req.RequestHandler)
	TrackBatchEvents(handler *req.RequestHandler)
	QueryEvents(handler *req.RequestHandler)

	// Metrics endpoints
	GetUserMetrics(handler *req.RequestHandler)
	GetAggregatedUserMetrics(handler *req.RequestHandler)
	GetSessionMetrics(handler *req.RequestHandler)
	GetOverview(handler *req.RequestHandler)
}

// Compile-time interface compliance check
var _ AnalyticsHandlerInterface = (*AnalyticsHandler)(nil)
