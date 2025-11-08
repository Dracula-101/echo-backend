package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type IPAddress struct {
	ID        string  `db:"id" json:"id" pk:"true"`
	IPAddress string  `db:"ip_address" json:"ip_address"`
	IPVersion *int    `db:"ip_version" json:"ip_version,omitempty"`

	// Geolocation
	CountryCode *string  `db:"country_code" json:"country_code,omitempty"`
	CountryName *string  `db:"country_name" json:"country_name,omitempty"`
	RegionCode  *string  `db:"region_code" json:"region_code,omitempty"`
	RegionName  *string  `db:"region_name" json:"region_name,omitempty"`
	City        *string  `db:"city" json:"city,omitempty"`
	PostalCode  *string  `db:"postal_code" json:"postal_code,omitempty"`
	Latitude    *float64 `db:"latitude" json:"latitude,omitempty"`
	Longitude   *float64 `db:"longitude" json:"longitude,omitempty"`
	MetroCode   *string  `db:"metro_code" json:"metro_code,omitempty"`
	Timezone    *string  `db:"timezone" json:"timezone,omitempty"`

	// Network Information
	ISP            *string `db:"isp" json:"isp,omitempty"`
	Organization   *string `db:"organization" json:"organization,omitempty"`
	ASN            *string `db:"asn" json:"asn,omitempty"`
	ASOrganization *string `db:"as_organization" json:"as_organization,omitempty"`

	// Connection Type
	ConnectionType *string `db:"connection_type" json:"connection_type,omitempty"`
	UserType       *string `db:"user_type" json:"user_type,omitempty"`

	// Security
	IsProxy     bool         `db:"is_proxy" json:"is_proxy"`
	IsVPN       bool         `db:"is_vpn" json:"is_vpn"`
	IsTor       bool         `db:"is_tor" json:"is_tor"`
	IsHosting   bool         `db:"is_hosting" json:"is_hosting"`
	IsAnonymous bool         `db:"is_anonymous" json:"is_anonymous"`
	ThreatLevel *ThreatLevel `db:"threat_level" json:"threat_level,omitempty"`
	RiskScore   *int         `db:"risk_score" json:"risk_score,omitempty"`
	IsBogon     bool         `db:"is_bogon" json:"is_bogon"`

	// Usage tracking
	FirstSeenAt  time.Time `db:"first_seen_at" json:"first_seen_at"`
	LastSeenAt   time.Time `db:"last_seen_at" json:"last_seen_at"`
	LookupCount  int       `db:"lookup_count" json:"lookup_count"`
	UserCount    int       `db:"user_count" json:"user_count"`

	// Data source
	LookupProvider *string   `db:"lookup_provider" json:"lookup_provider,omitempty"`
	LastUpdatedAt  time.Time `db:"last_updated_at" json:"last_updated_at"`

	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	Metadata  json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (i *IPAddress) TableName() string {
	return "location.ip_addresses"
}

func (i *IPAddress) PrimaryKey() interface{} {
	return i.ID
}

type UserLocation struct {
	ID            string  `db:"id" json:"id" pk:"true"`
	UserID        string  `db:"user_id" json:"user_id"`
	SessionID     *string `db:"session_id" json:"session_id,omitempty"`
	IPAddressID   *string `db:"ip_address_id" json:"ip_address_id,omitempty"`

	// Location data
	IPAddress      string   `db:"ip_address" json:"ip_address"`
	Latitude       *float64 `db:"latitude" json:"latitude,omitempty"`
	Longitude      *float64 `db:"longitude" json:"longitude,omitempty"`
	AccuracyMeters *int     `db:"accuracy_meters" json:"accuracy_meters,omitempty"`
	Altitude       *float64 `db:"altitude" json:"altitude,omitempty"`

	// Address
	Country    *string `db:"country" json:"country,omitempty"`
	Region     *string `db:"region" json:"region,omitempty"`
	City       *string `db:"city" json:"city,omitempty"`
	PostalCode *string `db:"postal_code" json:"postal_code,omitempty"`
	Address    *string `db:"address" json:"address,omitempty"`

	// Location type
	LocationType   *LocationType   `db:"location_type" json:"location_type,omitempty"`
	LocationSource *LocationSource `db:"location_source" json:"location_source,omitempty"`

	// Movement detection
	IsNewLocation          bool     `db:"is_new_location" json:"is_new_location"`
	IsNewCountry           bool     `db:"is_new_country" json:"is_new_country"`
	IsNewCity              bool     `db:"is_new_city" json:"is_new_city"`
	DistanceFromPreviousKM *float64 `db:"distance_from_previous_km" json:"distance_from_previous_km,omitempty"`

	// Timezone
	Timezone       *string `db:"timezone" json:"timezone,omitempty"`
	TimezoneOffset *int    `db:"timezone_offset" json:"timezone_offset,omitempty"`

	// Device context
	DeviceID *string `db:"device_id" json:"device_id,omitempty"`

	// Timestamps
	CapturedAt time.Time `db:"captured_at" json:"captured_at"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`

	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (u *UserLocation) TableName() string {
	return "location.user_locations"
}

func (u *UserLocation) PrimaryKey() interface{} {
	return u.ID
}

type LocationShare struct {
	ID                      string  `db:"id" json:"id" pk:"true"`
	UserID                  string  `db:"user_id" json:"user_id"`
	SharedWithUserID        *string `db:"shared_with_user_id" json:"shared_with_user_id,omitempty"`
	SharedWithConversationID *string `db:"shared_with_conversation_id" json:"shared_with_conversation_id,omitempty"`

	// Current location
	Latitude       float64  `db:"latitude" json:"latitude"`
	Longitude      float64  `db:"longitude" json:"longitude"`
	AccuracyMeters *int     `db:"accuracy_meters" json:"accuracy_meters,omitempty"`
	Altitude       *float64 `db:"altitude" json:"altitude,omitempty"`
	Heading        *float64 `db:"heading" json:"heading,omitempty"`
	SpeedMPS       *float64 `db:"speed_mps" json:"speed_mps,omitempty"`

	// Sharing details
	ShareType       LocationShareType `db:"share_type" json:"share_type"`
	DurationMinutes *int              `db:"duration_minutes" json:"duration_minutes,omitempty"`
	ExpiresAt       *time.Time        `db:"expires_at" json:"expires_at,omitempty"`

	// Status
	IsActive             bool `db:"is_active" json:"is_active"`
	IsLive               bool `db:"is_live" json:"is_live"`
	UpdateIntervalSeconds int  `db:"update_interval_seconds" json:"update_interval_seconds"`

	// Privacy
	ShowExactLocation bool `db:"show_exact_location" json:"show_exact_location"`
	ShowAddress       bool `db:"show_address" json:"show_address"`

	StartedAt     time.Time  `db:"started_at" json:"started_at"`
	StoppedAt     *time.Time `db:"stopped_at" json:"stopped_at,omitempty"`
	LastUpdatedAt time.Time  `db:"last_updated_at" json:"last_updated_at"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (l *LocationShare) TableName() string {
	return "location.location_shares"
}

func (l *LocationShare) PrimaryKey() interface{} {
	return l.ID
}

type LocationUpdate struct {
	ID              string `db:"id" json:"id" pk:"true"`
	LocationShareID string `db:"location_share_id" json:"location_share_id"`

	// Location
	Latitude       float64  `db:"latitude" json:"latitude"`
	Longitude      float64  `db:"longitude" json:"longitude"`
	AccuracyMeters *int     `db:"accuracy_meters" json:"accuracy_meters,omitempty"`
	Altitude       *float64 `db:"altitude" json:"altitude,omitempty"`
	Heading        *float64 `db:"heading" json:"heading,omitempty"`
	SpeedMPS       *float64 `db:"speed_mps" json:"speed_mps,omitempty"`

	// Battery (optional)
	BatteryLevel *int  `db:"battery_level" json:"battery_level,omitempty"`
	IsCharging   *bool `db:"is_charging" json:"is_charging,omitempty"`

	Timestamp time.Time `db:"timestamp" json:"timestamp"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (l *LocationUpdate) TableName() string {
	return "location.location_updates"
}

func (l *LocationUpdate) PrimaryKey() interface{} {
	return l.ID
}

type Place struct {
	ID string `db:"id" json:"id" pk:"true"`

	// Place details
	PlaceName     string  `db:"place_name" json:"place_name"`
	PlaceType     *string `db:"place_type" json:"place_type,omitempty"`
	PlaceCategory *string `db:"place_category" json:"place_category,omitempty"`

	// Location
	Latitude   float64 `db:"latitude" json:"latitude"`
	Longitude  float64 `db:"longitude" json:"longitude"`
	Address    *string `db:"address" json:"address,omitempty"`
	City       *string `db:"city" json:"city,omitempty"`
	Region     *string `db:"region" json:"region,omitempty"`
	Country    *string `db:"country" json:"country,omitempty"`
	PostalCode *string `db:"postal_code" json:"postal_code,omitempty"`

	// Contact
	PhoneNumber *string `db:"phone_number" json:"phone_number,omitempty"`
	WebsiteURL  *string `db:"website_url" json:"website_url,omitempty"`

	// External IDs
	GooglePlaceID *string `db:"google_place_id" json:"google_place_id,omitempty"`
	FoursquareID  *string `db:"foursquare_id" json:"foursquare_id,omitempty"`

	// Metadata
	Rating      *float64 `db:"rating" json:"rating,omitempty"`
	ReviewCount int      `db:"review_count" json:"review_count"`
	PriceLevel  *int     `db:"price_level" json:"price_level,omitempty"`

	// Usage
	CheckInCount int `db:"check_in_count" json:"check_in_count"`
	ShareCount   int `db:"share_count" json:"share_count"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (p *Place) TableName() string {
	return "location.places"
}

func (p *Place) PrimaryKey() interface{} {
	return p.ID
}

type CheckIn struct {
	ID      string  `db:"id" json:"id" pk:"true"`
	UserID  string  `db:"user_id" json:"user_id"`
	PlaceID *string `db:"place_id" json:"place_id,omitempty"`

	// Location
	Latitude  *float64 `db:"latitude" json:"latitude,omitempty"`
	Longitude *float64 `db:"longitude" json:"longitude,omitempty"`
	PlaceName *string  `db:"place_name" json:"place_name,omitempty"`

	// Check-in details
	Caption  *string `db:"caption" json:"caption,omitempty"`
	PhotoURL *string `db:"photo_url" json:"photo_url,omitempty"`

	// Privacy
	Visibility CheckInVisibility `db:"visibility" json:"visibility"`

	// Social
	LikeCount    int `db:"like_count" json:"like_count"`
	CommentCount int `db:"comment_count" json:"comment_count"`

	CheckedInAt time.Time `db:"checked_in_at" json:"checked_in_at"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

func (c *CheckIn) TableName() string {
	return "location.check_ins"
}

func (c *CheckIn) PrimaryKey() interface{} {
	return c.ID
}

type Geofence struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	// Geofence details
	FenceName string        `db:"fence_name" json:"fence_name"`
	FenceType *GeofenceType `db:"fence_type" json:"fence_type,omitempty"`

	// Circle geofence
	CenterLatitude  *float64 `db:"center_latitude" json:"center_latitude,omitempty"`
	CenterLongitude *float64 `db:"center_longitude" json:"center_longitude,omitempty"`
	RadiusMeters    *int     `db:"radius_meters" json:"radius_meters,omitempty"`

	// Polygon geofence
	PolygonCoordinates json.RawMessage `db:"polygon_coordinates" json:"polygon_coordinates,omitempty"`

	// Trigger
	TriggerOnEnter bool `db:"trigger_on_enter" json:"trigger_on_enter"`
	TriggerOnExit  bool `db:"trigger_on_exit" json:"trigger_on_exit"`
	TriggerOnDwell bool `db:"trigger_on_dwell" json:"trigger_on_dwell"`
	DwellTimeSeconds *int `db:"dwell_time_seconds" json:"dwell_time_seconds,omitempty"`

	// Actions
	ActionType   *string         `db:"action_type" json:"action_type,omitempty"`
	ActionConfig json.RawMessage `db:"action_config" json:"action_config,omitempty"`

	// Status
	IsActive bool `db:"is_active" json:"is_active"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (g *Geofence) TableName() string {
	return "location.geofences"
}

func (g *Geofence) PrimaryKey() interface{} {
	return g.ID
}

type GeofenceEvent struct {
	ID         string `db:"id" json:"id" pk:"true"`
	GeofenceID string `db:"geofence_id" json:"geofence_id"`
	UserID     string `db:"user_id" json:"user_id"`

	EventType GeofenceEventType `db:"event_type" json:"event_type"`

	// Location
	Latitude  *float64 `db:"latitude" json:"latitude,omitempty"`
	Longitude *float64 `db:"longitude" json:"longitude,omitempty"`

	// Action taken
	ActionExecuted bool            `db:"action_executed" json:"action_executed"`
	ActionResult   json.RawMessage `db:"action_result" json:"action_result,omitempty"`

	OccurredAt time.Time `db:"occurred_at" json:"occurred_at"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

func (g *GeofenceEvent) TableName() string {
	return "location.geofence_events"
}

func (g *GeofenceEvent) PrimaryKey() interface{} {
	return g.ID
}

type NearbyUser struct {
	ID           string `db:"id" json:"id" pk:"true"`
	UserID       string `db:"user_id" json:"user_id"`
	NearbyUserID string `db:"nearby_user_id" json:"nearby_user_id"`

	// Distance
	DistanceMeters int `db:"distance_meters" json:"distance_meters"`

	// Location context
	LocationName *string `db:"location_name" json:"location_name,omitempty"`

	// Interaction
	HasInteracted   bool    `db:"has_interacted" json:"has_interacted"`
	InteractionType *string `db:"interaction_type" json:"interaction_type,omitempty"`

	// Timestamps
	DetectedAt   time.Time `db:"detected_at" json:"detected_at"`
	LastNearbyAt time.Time `db:"last_nearby_at" json:"last_nearby_at"`
}

func (n *NearbyUser) TableName() string {
	return "location.nearby_users"
}

func (n *NearbyUser) PrimaryKey() interface{} {
	return n.ID
}

type LocationRecommendation struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	RecommendationType *string `db:"recommendation_type" json:"recommendation_type,omitempty"`

	// Recommended entity
	RecommendedUserID  *string `db:"recommended_user_id" json:"recommended_user_id,omitempty"`
	RecommendedPlaceID *string `db:"recommended_place_id" json:"recommended_place_id,omitempty"`

	// Scoring
	RelevanceScore *float64 `db:"relevance_score" json:"relevance_score,omitempty"`
	DistanceMeters *int     `db:"distance_meters" json:"distance_meters,omitempty"`

	// Status
	IsShown       bool       `db:"is_shown" json:"is_shown"`
	ShownAt       *time.Time `db:"shown_at" json:"shown_at,omitempty"`
	IsActedUpon   bool       `db:"is_acted_upon" json:"is_acted_upon"`
	ActionType    *string    `db:"action_type" json:"action_type,omitempty"`
	ActedAt       *time.Time `db:"acted_at" json:"acted_at,omitempty"`

	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	ExpiresAt *time.Time `db:"expires_at" json:"expires_at,omitempty"`
}

func (l *LocationRecommendation) TableName() string {
	return "location.location_recommendations"
}

func (l *LocationRecommendation) PrimaryKey() interface{} {
	return l.ID
}

type RegionStat struct {
	ID         string `db:"id" json:"id" pk:"true"`
	RegionType string `db:"region_type" json:"region_type"`
	RegionCode *string `db:"region_code" json:"region_code,omitempty"`
	RegionName string `db:"region_name" json:"region_name"`

	// User counts
	TotalUsers         int `db:"total_users" json:"total_users"`
	ActiveUsersDaily   int `db:"active_users_daily" json:"active_users_daily"`
	ActiveUsersMonthly int `db:"active_users_monthly" json:"active_users_monthly"`
	NewUsersToday      int `db:"new_users_today" json:"new_users_today"`

	// Activity
	MessagesSentToday int64 `db:"messages_sent_today" json:"messages_sent_today"`
	CallsMadeToday    int   `db:"calls_made_today" json:"calls_made_today"`

	// Growth
	GrowthRatePercentage *float64 `db:"growth_rate_percentage" json:"growth_rate_percentage,omitempty"`

	Date          time.Time `db:"date" json:"date"`
	LastUpdatedAt time.Time `db:"last_updated_at" json:"last_updated_at"`
}

func (r *RegionStat) TableName() string {
	return "location.region_stats"
}

func (r *RegionStat) PrimaryKey() interface{} {
	return r.ID
}

type IPBlacklist struct {
	ID       string  `db:"id" json:"id" pk:"true"`
	IPAddress string  `db:"ip_address" json:"ip_address"`
	IPRange  *string `db:"ip_range" json:"ip_range,omitempty"`

	// Reason
	BlacklistReason string                `db:"blacklist_reason" json:"blacklist_reason"`
	BlacklistType   *string               `db:"blacklist_type" json:"blacklist_type,omitempty"`
	Severity        IPBlacklistSeverity   `db:"severity" json:"severity"`

	// Evidence
	IncidentCount int            `db:"incident_count" json:"incident_count"`
	EvidenceURLs  pq.StringArray `db:"evidence_urls" json:"evidence_urls,omitempty"`
	Notes         *string        `db:"notes" json:"notes,omitempty"`

	// Status
	IsActive bool `db:"is_active" json:"is_active"`

	// Source
	BlacklistedByUserID *string `db:"blacklisted_by_user_id" json:"blacklisted_by_user_id,omitempty"`
	Source              *string `db:"source" json:"source,omitempty"`

	BlacklistedAt time.Time  `db:"blacklisted_at" json:"blacklisted_at"`
	ExpiresAt     *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	RemovedAt     *time.Time `db:"removed_at" json:"removed_at,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (i *IPBlacklist) TableName() string {
	return "location.ip_blacklist"
}

func (i *IPBlacklist) PrimaryKey() interface{} {
	return i.ID
}

type VPNDetectionLog struct {
	ID        string  `db:"id" json:"id" pk:"true"`
	UserID    *string `db:"user_id" json:"user_id,omitempty"`
	SessionID *string `db:"session_id" json:"session_id,omitempty"`

	IPAddress string `db:"ip_address" json:"ip_address"`

	// Detection results
	IsVPN     bool    `db:"is_vpn" json:"is_vpn"`
	IsProxy   bool    `db:"is_proxy" json:"is_proxy"`
	IsTor     bool    `db:"is_tor" json:"is_tor"`
	IsHosting bool    `db:"is_hosting" json:"is_hosting"`

	VPNProvider *string `db:"vpn_provider" json:"vpn_provider,omitempty"`
	ProxyType   *string `db:"proxy_type" json:"proxy_type,omitempty"`

	// Confidence
	ConfidenceScore *float64 `db:"confidence_score" json:"confidence_score,omitempty"`

	// Action taken
	ActionTaken *string `db:"action_taken" json:"action_taken,omitempty"`

	DetectedAt time.Time `db:"detected_at" json:"detected_at"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

func (v *VPNDetectionLog) TableName() string {
	return "location.vpn_detection_log"
}

func (v *VPNDetectionLog) PrimaryKey() interface{} {
	return v.ID
}