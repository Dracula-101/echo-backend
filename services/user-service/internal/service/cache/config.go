package cache

import "time"

const (
	ProfilePrefix         = "profile:"
	ProfileUsernamePrefix = "profile:username:"
	ProfilePhonePrefix    = "profile:phone:"
	SettingsPrefix        = "settings:"
	PresencePrefix        = "presence:"
	PresenceContactPrefix = "presence:contacts:"
	BlockedPrefix         = "blocked:"
	ContactIDsPrefix      = "contact_ids:"
	SearchCachePrefix     = "search:cache:"
	StatusFeedPrefix      = "status_feed:"
)

const (
	TTLProfile         = 300 * time.Second
	TTLProfileUsername = 600 * time.Second
	TTLProfilePhone    = 3600 * time.Second
	TTLSettings        = 300 * time.Second
	TTLPresence        = 35 * time.Second
	TTLPresenceContact = 30 * time.Second
	TTLBlocked         = 300 * time.Second
	TTLContactIDs      = 120 * time.Second
	TTLSearchCache     = 30 * time.Second
	TTLStatusFeed      = 60 * time.Second
)
