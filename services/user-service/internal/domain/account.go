package domain

type AccountStatus string

const (
	AccountStatusActive      AccountStatus = "active"
	AccountStatusPending     AccountStatus = "pending"
	AccountStatusSuspended   AccountStatus = "suspended"
	AccountStatusLocked      AccountStatus = "locked"
	AccountStatusDeactivated AccountStatus = "deactivated"
	AccountStatusDeleted     AccountStatus = "deleted"
)

func (s AccountStatus) String() string {
	return string(s)
}

type OnlineStatus string

const (
	OnlineStatusOnline  OnlineStatus = "online"
	OnlineStatusOffline OnlineStatus = "offline"
	OnlineStatusAway    OnlineStatus = "away"
	OnlineStatusBusy    OnlineStatus = "busy"
)

func (s OnlineStatus) String() string {
	return string(s)
}

type ProfileVisibility string

const (
	ProfileVisibilityPublic  ProfileVisibility = "public"
	ProfileVisibilityPrivate ProfileVisibility = "private"
)

func (s ProfileVisibility) String() string {
	return string(s)
}
