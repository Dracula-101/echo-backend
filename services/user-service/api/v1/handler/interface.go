package handler

import req "shared/server/request"

// ============================================================================
// Handler Interface
// ============================================================================

type UserHandlerInterface interface {
	// Profile endpoints
	GetProfile(handler *req.RequestHandler)
	CreateProfile(handler *req.RequestHandler)
	UpdateProfile(handler *req.RequestHandler)
	DeleteProfile(handler *req.RequestHandler)
	SearchProfiles(handler *req.RequestHandler)

	// Contact endpoints
	ListContacts(handler *req.RequestHandler)
	AddContact(handler *req.RequestHandler)
	UpdateContact(handler *req.RequestHandler)
	RemoveContact(handler *req.RequestHandler)

	// Contact group endpoints
	ListContactGroups(handler *req.RequestHandler)
	CreateContactGroup(handler *req.RequestHandler)
	UpdateContactGroup(handler *req.RequestHandler)
	DeleteContactGroup(handler *req.RequestHandler)

	// Settings endpoints
	GetSettings(handler *req.RequestHandler)
	UpdateSettings(handler *req.RequestHandler)

	// Blocked users endpoints
	ListBlocked(handler *req.RequestHandler)
	BlockUser(handler *req.RequestHandler)
	UnblockUser(handler *req.RequestHandler)

	// Status endpoints
	SetStatus(handler *req.RequestHandler)
	GetStatus(handler *req.RequestHandler)
	DeleteStatus(handler *req.RequestHandler)
	GetStatusViews(handler *req.RequestHandler)

	// Device endpoints
	ListDevices(handler *req.RequestHandler)
	UpdateDevice(handler *req.RequestHandler)
	RemoveDevice(handler *req.RequestHandler)

	// Preferences endpoints
	GetPreferences(handler *req.RequestHandler)
	UpdatePreferences(handler *req.RequestHandler)

	// Achievements endpoints
	ListAchievements(handler *req.RequestHandler)

	// Reports endpoints
	SubmitReport(handler *req.RequestHandler)
}

// Compile-time interface compliance check
var _ UserHandlerInterface = (*UserHandler)(nil)
