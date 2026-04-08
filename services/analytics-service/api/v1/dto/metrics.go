package dto

import "time"

// ============================================================================
// User Metrics
// ============================================================================

// UserMetricsResponse represents metrics for a specific user.
type UserMetricsResponse struct {
	UserID          string    `json:"user_id"`
	TotalEvents     int64     `json:"total_events"`
	TotalSessions   int64     `json:"total_sessions"`
	LastActiveAt    time.Time `json:"last_active_at"`
	FirstSeenAt     time.Time `json:"first_seen_at"`
	AvgSessionDuration float64 `json:"avg_session_duration_seconds"`
	EventsByType    map[string]int64 `json:"events_by_type,omitempty"`
}

// AggregatedUserMetricsResponse represents aggregated user metrics.
type AggregatedUserMetricsResponse struct {
	TotalUsers       int64   `json:"total_users"`
	ActiveUsers      int64   `json:"active_users"`
	NewUsers         int64   `json:"new_users"`
	AvgEventsPerUser float64 `json:"avg_events_per_user"`
	Period           string  `json:"period"`
}

// ============================================================================
// Session Metrics
// ============================================================================

// SessionMetricsResponse represents session-level metrics.
type SessionMetricsResponse struct {
	TotalSessions      int64   `json:"total_sessions"`
	ActiveSessions     int64   `json:"active_sessions"`
	AvgDuration        float64 `json:"avg_duration_seconds"`
	MedianDuration     float64 `json:"median_duration_seconds"`
	BounceRate         float64 `json:"bounce_rate"`
	Period             string  `json:"period"`
}

// ============================================================================
// Overview
// ============================================================================

// OverviewResponse represents the analytics overview dashboard data.
type OverviewResponse struct {
	TotalEvents       int64                    `json:"total_events"`
	TotalUsers        int64                    `json:"total_users"`
	ActiveUsers       int64                    `json:"active_users"`
	TotalSessions     int64                    `json:"total_sessions"`
	TopEventTypes     []EventTypeCount         `json:"top_event_types"`
	RecentActivity    []DailyActivitySummary   `json:"recent_activity"`
}

// EventTypeCount represents a count for a specific event type.
type EventTypeCount struct {
	EventType string `json:"event_type"`
	Count     int64  `json:"count"`
}

// DailyActivitySummary represents activity for a single day.
type DailyActivitySummary struct {
	Date       string `json:"date"`
	Events     int64  `json:"events"`
	Users      int64  `json:"users"`
	Sessions   int64  `json:"sessions"`
}
