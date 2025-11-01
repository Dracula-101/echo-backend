-- =====================================================
-- LOCATION SCHEMA - IP Tracking & Geolocation
-- =====================================================

-- Create Schema
CREATE SCHEMA IF NOT EXISTS location;

-- IP Address Records
CREATE TABLE location.ip_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip_address INET UNIQUE NOT NULL,
    ip_version INTEGER, -- 4 or 6
    
    -- Geolocation
    country_code VARCHAR(5),
    country_name VARCHAR(100),
    region_code VARCHAR(10),
    region_name VARCHAR(100),
    city VARCHAR(100),
    postal_code VARCHAR(20),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    metro_code VARCHAR(20),
    timezone VARCHAR(100),
    
    -- Network Information
    isp VARCHAR(255),
    organization VARCHAR(255),
    asn VARCHAR(50), -- Autonomous System Number
    as_organization VARCHAR(255),
    
    -- Connection Type
    connection_type VARCHAR(50), -- residential, business, mobile, hosting, education
    user_type VARCHAR(50), -- residential, business, traveler, college, etc
    
    -- Security
    is_proxy BOOLEAN DEFAULT FALSE,
    is_vpn BOOLEAN DEFAULT FALSE,
    is_tor BOOLEAN DEFAULT FALSE,
    is_hosting BOOLEAN DEFAULT FALSE,
    is_anonymous BOOLEAN DEFAULT FALSE,
    threat_level VARCHAR(20), -- low, medium, high
    risk_score INTEGER, -- 0-100
    is_bogon BOOLEAN DEFAULT FALSE, -- Invalid/reserved IP
    
    -- Usage tracking
    first_seen_at TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ DEFAULT NOW(),
    lookup_count INTEGER DEFAULT 1,
    user_count INTEGER DEFAULT 0, -- How many users used this IP
    
    -- Data source
    lookup_provider VARCHAR(50), -- ipapi, maxmind, ipinfo
    last_updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::JSONB
);

-- User Location History
CREATE TABLE location.user_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    ip_address_id UUID REFERENCES location.ip_addresses(id),
    
    -- Location data
    ip_address INET NOT NULL,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    accuracy_meters INTEGER,
    altitude DECIMAL(10, 2),
    
    -- Address
    country VARCHAR(100),
    region VARCHAR(100),
    city VARCHAR(100),
    postal_code VARCHAR(20),
    address TEXT,
    
    -- Location type
    location_type VARCHAR(50), -- ip_based, gps, wifi, cell_tower, manual
    location_source VARCHAR(50), -- device, browser, server
    
    -- Movement detection
    is_new_location BOOLEAN DEFAULT FALSE,
    is_new_country BOOLEAN DEFAULT FALSE,
    is_new_city BOOLEAN DEFAULT FALSE,
    distance_from_previous_km DECIMAL(10, 2),
    
    -- Timezone
    timezone VARCHAR(100),
    timezone_offset INTEGER, -- Minutes from UTC
    
    -- Device context
    device_id VARCHAR(255),
    
    -- Timestamps
    captured_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Location Sharing (live location with other users)
CREATE TABLE location.location_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    shared_with_user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    shared_with_conversation_id UUID REFERENCES messages.conversations(id) ON DELETE CASCADE,
    
    -- Current location
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    accuracy_meters INTEGER,
    altitude DECIMAL(10, 2),
    heading DECIMAL(5, 2), -- Direction in degrees
    speed_mps DECIMAL(10, 2), -- Speed in meters per second
    
    -- Sharing details
    share_type VARCHAR(50) DEFAULT 'temporary', -- temporary, permanent
    duration_minutes INTEGER, -- For temporary shares
    expires_at TIMESTAMPTZ,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    is_live BOOLEAN DEFAULT TRUE, -- Real-time updates
    update_interval_seconds INTEGER DEFAULT 30,
    
    -- Privacy
    show_exact_location BOOLEAN DEFAULT TRUE,
    show_address BOOLEAN DEFAULT FALSE,
    
    started_at TIMESTAMPTZ DEFAULT NOW(),
    stopped_at TIMESTAMPTZ,
    last_updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Location Updates (for live location tracking)
CREATE TABLE location.location_updates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_share_id UUID NOT NULL REFERENCES location.location_shares(id) ON DELETE CASCADE,
    
    -- Location
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    accuracy_meters INTEGER,
    altitude DECIMAL(10, 2),
    heading DECIMAL(5, 2),
    speed_mps DECIMAL(10, 2),
    
    -- Battery (optional)
    battery_level INTEGER, -- 0-100
    is_charging BOOLEAN,
    
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Places/Points of Interest
CREATE TABLE location.places (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Place details
    place_name VARCHAR(255) NOT NULL,
    place_type VARCHAR(100), -- restaurant, cafe, park, business, etc
    place_category VARCHAR(100),
    
    -- Location
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    address TEXT,
    city VARCHAR(100),
    region VARCHAR(100),
    country VARCHAR(100),
    postal_code VARCHAR(20),
    
    -- Contact
    phone_number VARCHAR(20),
    website_url TEXT,
    
    -- External IDs
    google_place_id VARCHAR(255) UNIQUE,
    foursquare_id VARCHAR(255),
    
    -- Metadata
    rating DECIMAL(3, 2),
    review_count INTEGER DEFAULT 0,
    price_level INTEGER, -- 1-4
    
    -- Usage
    check_in_count INTEGER DEFAULT 0,
    share_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User Check-ins
CREATE TABLE location.check_ins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    place_id UUID REFERENCES location.places(id) ON DELETE SET NULL,
    
    -- Location
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    place_name VARCHAR(255),
    
    -- Check-in details
    caption TEXT,
    photo_url TEXT,
    
    -- Privacy
    visibility VARCHAR(50) DEFAULT 'friends', -- public, friends, private
    
    -- Social
    like_count INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    
    checked_in_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Geofences (location-based triggers)
CREATE TABLE location.geofences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Geofence details
    fence_name VARCHAR(255) NOT NULL,
    fence_type VARCHAR(50), -- circle, polygon
    
    -- Circle geofence
    center_latitude DECIMAL(10, 8),
    center_longitude DECIMAL(11, 8),
    radius_meters INTEGER,
    
    -- Polygon geofence
    polygon_coordinates JSONB, -- Array of [lat, lng] pairs
    
    -- Trigger
    trigger_on_enter BOOLEAN DEFAULT TRUE,
    trigger_on_exit BOOLEAN DEFAULT FALSE,
    trigger_on_dwell BOOLEAN DEFAULT FALSE,
    dwell_time_seconds INTEGER,
    
    -- Actions
    action_type VARCHAR(100), -- notify, auto_message, change_status
    action_config JSONB,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Geofence Events
CREATE TABLE location.geofence_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    geofence_id UUID NOT NULL REFERENCES location.geofences(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    event_type VARCHAR(50) NOT NULL, -- enter, exit, dwell
    
    -- Location
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    
    -- Action taken
    action_executed BOOLEAN DEFAULT FALSE,
    action_result JSONB,
    
    occurred_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Nearby Users (for proximity-based features)
CREATE TABLE location.nearby_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    nearby_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Distance
    distance_meters INTEGER NOT NULL,
    
    -- Location context
    location_name VARCHAR(255), -- Name of place if available
    
    -- Interaction
    has_interacted BOOLEAN DEFAULT FALSE,
    interaction_type VARCHAR(50), -- viewed, messaged, added
    
    -- Timestamps
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    last_nearby_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(user_id, nearby_user_id, detected_at)
);

-- Location-based Recommendations
CREATE TABLE location.location_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    recommendation_type VARCHAR(100), -- nearby_friends, suggested_places, events
    
    -- Recommended entity
    recommended_user_id UUID REFERENCES auth.users(id),
    recommended_place_id UUID REFERENCES location.places(id),
    
    -- Scoring
    relevance_score DECIMAL(5, 2), -- 0-100
    distance_meters INTEGER,
    
    -- Status
    is_shown BOOLEAN DEFAULT FALSE,
    shown_at TIMESTAMPTZ,
    is_acted_upon BOOLEAN DEFAULT FALSE,
    action_type VARCHAR(50),
    acted_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

-- Country/Region Statistics
CREATE TABLE location.region_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region_type VARCHAR(50) NOT NULL, -- country, region, city
    region_code VARCHAR(10),
    region_name VARCHAR(255) NOT NULL,
    
    -- User counts
    total_users INTEGER DEFAULT 0,
    active_users_daily INTEGER DEFAULT 0,
    active_users_monthly INTEGER DEFAULT 0,
    new_users_today INTEGER DEFAULT 0,
    
    -- Activity
    messages_sent_today BIGINT DEFAULT 0,
    calls_made_today INTEGER DEFAULT 0,
    
    -- Growth
    growth_rate_percentage DECIMAL(5, 2),
    
    date DATE DEFAULT CURRENT_DATE,
    last_updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(region_type, region_code, date)
);

-- IP Blacklist
CREATE TABLE location.ip_blacklist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip_address INET NOT NULL,
    ip_range CIDR, -- For blocking IP ranges
    
    -- Reason
    blacklist_reason VARCHAR(255) NOT NULL,
    blacklist_type VARCHAR(50), -- spam, abuse, bot, security_threat
    severity VARCHAR(20) DEFAULT 'medium',
    
    -- Evidence
    incident_count INTEGER DEFAULT 1,
    evidence_urls TEXT[],
    notes TEXT,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Source
    blacklisted_by_user_id UUID REFERENCES auth.users(id),
    source VARCHAR(100), -- manual, auto_detected, external_list
    
    blacklisted_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    removed_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- VPN/Proxy Detection Log
CREATE TABLE location.vpn_detection_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES auth.sessions(id) ON DELETE SET NULL,
    
    ip_address INET NOT NULL,
    
    -- Detection results
    is_vpn BOOLEAN DEFAULT FALSE,
    is_proxy BOOLEAN DEFAULT FALSE,
    is_tor BOOLEAN DEFAULT FALSE,
    is_hosting BOOLEAN DEFAULT FALSE,
    
    vpn_provider VARCHAR(255),
    proxy_type VARCHAR(50),
    
    -- Confidence
    confidence_score DECIMAL(5, 2), -- 0-100
    
    -- Action taken
    action_taken VARCHAR(100), -- allowed, blocked, flagged, challenged
    
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_ip_addresses_ip ON location.ip_addresses(ip_address);
CREATE INDEX idx_ip_addresses_country ON location.ip_addresses(country_code);
CREATE INDEX idx_user_locations_user ON location.user_locations(user_id);
CREATE INDEX idx_user_locations_created ON location.user_locations(created_at);
CREATE INDEX idx_user_locations_country ON location.user_locations(country);
CREATE INDEX idx_location_shares_user ON location.location_shares(user_id);
CREATE INDEX idx_location_shares_active ON location.location_shares(is_active);
CREATE INDEX idx_places_location ON location.places(latitude, longitude);
CREATE INDEX idx_places_google_id ON location.places(google_place_id);
CREATE INDEX idx_check_ins_user ON location.check_ins(user_id);
CREATE INDEX idx_check_ins_place ON location.check_ins(place_id);
CREATE INDEX idx_geofences_user ON location.geofences(user_id);
CREATE INDEX idx_geofence_events_geofence ON location.geofence_events(geofence_id);
CREATE INDEX idx_nearby_users_user ON location.nearby_users(user_id);
CREATE INDEX idx_nearby_users_detected ON location.nearby_users(detected_at);
CREATE INDEX idx_ip_blacklist_ip ON location.ip_blacklist(ip_address);
CREATE INDEX idx_ip_blacklist_active ON location.ip_blacklist(is_active);