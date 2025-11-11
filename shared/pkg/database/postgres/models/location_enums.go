package models

import (
	"database/sql/driver"
	"fmt"
)

// LocationType represents the type of location data
type LocationType string

const (
	LocationTypeIPBased   LocationType = "ip_based"
	LocationTypeGPS       LocationType = "gps"
	LocationTypeWiFi      LocationType = "wifi"
	LocationTypeCellTower LocationType = "cell_tower"
	LocationTypeManual    LocationType = "manual"
)

func (l LocationType) IsValid() bool {
	switch l {
	case LocationTypeIPBased, LocationTypeGPS, LocationTypeWiFi,
		LocationTypeCellTower, LocationTypeManual:
		return true
	}
	return false
}

func (l LocationType) Value() (driver.Value, error) {
	if !l.IsValid() {
		return nil, fmt.Errorf("invalid location type: %s", l)
	}
	return string(l), nil
}

func (l *LocationType) Scan(value interface{}) error {
	if value == nil {
		*l = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan LocationType: expected string, got %T", value)
	}
	*l = LocationType(str)
	if !l.IsValid() {
		return fmt.Errorf("invalid location type value: %s", str)
	}
	return nil
}

// LocationSource represents the source of location data
type LocationSource string

const (
	LocationSourceDevice  LocationSource = "device"
	LocationSourceBrowser LocationSource = "browser"
	LocationSourceServer  LocationSource = "server"
)

func (l LocationSource) IsValid() bool {
	switch l {
	case LocationSourceDevice, LocationSourceBrowser, LocationSourceServer:
		return true
	}
	return false
}

func (l LocationSource) Value() (driver.Value, error) {
	if !l.IsValid() {
		return nil, fmt.Errorf("invalid location source: %s", l)
	}
	return string(l), nil
}

func (l *LocationSource) Scan(value interface{}) error {
	if value == nil {
		*l = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan LocationSource: expected string, got %T", value)
	}
	*l = LocationSource(str)
	if !l.IsValid() {
		return fmt.Errorf("invalid location source value: %s", str)
	}
	return nil
}

// LocationShareType represents the type of location share
type LocationShareType string

const (
	LocationShareTypeTemporary LocationShareType = "temporary"
	LocationShareTypePermanent LocationShareType = "permanent"
)

func (l LocationShareType) IsValid() bool {
	switch l {
	case LocationShareTypeTemporary, LocationShareTypePermanent:
		return true
	}
	return false
}

func (l LocationShareType) Value() (driver.Value, error) {
	if !l.IsValid() {
		return nil, fmt.Errorf("invalid location share type: %s", l)
	}
	return string(l), nil
}

func (l *LocationShareType) Scan(value interface{}) error {
	if value == nil {
		*l = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan LocationShareType: expected string, got %T", value)
	}
	*l = LocationShareType(str)
	if !l.IsValid() {
		return fmt.Errorf("invalid location share type value: %s", str)
	}
	return nil
}

// GeofenceType represents the type of geofence
type GeofenceType string

const (
	GeofenceTypeCircle  GeofenceType = "circle"
	GeofenceTypePolygon GeofenceType = "polygon"
)

func (g GeofenceType) IsValid() bool {
	switch g {
	case GeofenceTypeCircle, GeofenceTypePolygon:
		return true
	}
	return false
}

func (g GeofenceType) Value() (driver.Value, error) {
	if !g.IsValid() {
		return nil, fmt.Errorf("invalid geofence type: %s", g)
	}
	return string(g), nil
}

func (g *GeofenceType) Scan(value interface{}) error {
	if value == nil {
		*g = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan GeofenceType: expected string, got %T", value)
	}
	*g = GeofenceType(str)
	if !g.IsValid() {
		return fmt.Errorf("invalid geofence type value: %s", str)
	}
	return nil
}

// GeofenceEventType represents the type of geofence event
type GeofenceEventType string

const (
	GeofenceEventTypeEnter GeofenceEventType = "enter"
	GeofenceEventTypeExit  GeofenceEventType = "exit"
	GeofenceEventTypeDwell GeofenceEventType = "dwell"
)

func (g GeofenceEventType) IsValid() bool {
	switch g {
	case GeofenceEventTypeEnter, GeofenceEventTypeExit, GeofenceEventTypeDwell:
		return true
	}
	return false
}

func (g GeofenceEventType) Value() (driver.Value, error) {
	if !g.IsValid() {
		return nil, fmt.Errorf("invalid geofence event type: %s", g)
	}
	return string(g), nil
}

func (g *GeofenceEventType) Scan(value interface{}) error {
	if value == nil {
		*g = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan GeofenceEventType: expected string, got %T", value)
	}
	*g = GeofenceEventType(str)
	if !g.IsValid() {
		return fmt.Errorf("invalid geofence event type value: %s", str)
	}
	return nil
}

// CheckInVisibility represents the visibility level of a check-in
type CheckInVisibility string

const (
	CheckInVisibilityPublic  CheckInVisibility = "public"
	CheckInVisibilityFriends CheckInVisibility = "friends"
	CheckInVisibilityPrivate CheckInVisibility = "private"
)

func (c CheckInVisibility) IsValid() bool {
	switch c {
	case CheckInVisibilityPublic, CheckInVisibilityFriends, CheckInVisibilityPrivate:
		return true
	}
	return false
}

func (c CheckInVisibility) Value() (driver.Value, error) {
	if !c.IsValid() {
		return nil, fmt.Errorf("invalid check-in visibility: %s", c)
	}
	return string(c), nil
}

func (c *CheckInVisibility) Scan(value interface{}) error {
	if value == nil {
		*c = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan CheckInVisibility: expected string, got %T", value)
	}
	*c = CheckInVisibility(str)
	if !c.IsValid() {
		return fmt.Errorf("invalid check-in visibility value: %s", str)
	}
	return nil
}

// ThreatLevel represents the threat level of an IP address
type ThreatLevel string

const (
	ThreatLevelLow    ThreatLevel = "low"
	ThreatLevelMedium ThreatLevel = "medium"
	ThreatLevelHigh   ThreatLevel = "high"
)

func (t ThreatLevel) IsValid() bool {
	switch t {
	case ThreatLevelLow, ThreatLevelMedium, ThreatLevelHigh:
		return true
	}
	return false
}

func (t ThreatLevel) Value() (driver.Value, error) {
	if !t.IsValid() {
		return nil, fmt.Errorf("invalid threat level: %s", t)
	}
	return string(t), nil
}

func (t *ThreatLevel) Scan(value interface{}) error {
	if value == nil {
		*t = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ThreatLevel: expected string, got %T", value)
	}
	*t = ThreatLevel(str)
	if !t.IsValid() {
		return fmt.Errorf("invalid threat level value: %s", str)
	}
	return nil
}

// IPBlacklistSeverity represents the severity of an IP blacklist entry
type IPBlacklistSeverity string

const (
	IPBlacklistSeverityLow      IPBlacklistSeverity = "low"
	IPBlacklistSeverityMedium   IPBlacklistSeverity = "medium"
	IPBlacklistSeverityHigh     IPBlacklistSeverity = "high"
	IPBlacklistSeverityCritical IPBlacklistSeverity = "critical"
)

func (i IPBlacklistSeverity) IsValid() bool {
	switch i {
	case IPBlacklistSeverityLow, IPBlacklistSeverityMedium,
		IPBlacklistSeverityHigh, IPBlacklistSeverityCritical:
		return true
	}
	return false
}

func (i IPBlacklistSeverity) Value() (driver.Value, error) {
	if !i.IsValid() {
		return nil, fmt.Errorf("invalid IP blacklist severity: %s", i)
	}
	return string(i), nil
}

func (i *IPBlacklistSeverity) Scan(value interface{}) error {
	if value == nil {
		*i = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan IPBlacklistSeverity: expected string, got %T", value)
	}
	*i = IPBlacklistSeverity(str)
	if !i.IsValid() {
		return fmt.Errorf("invalid IP blacklist severity value: %s", str)
	}
	return nil
}
