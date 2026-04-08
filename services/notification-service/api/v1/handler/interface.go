package handler

import req "shared/server/request"

// NotificationHandlerInterface defines the contract for notification HTTP handlers
type NotificationHandlerInterface interface {
	// Notification endpoints
	ListNotifications(handler *req.RequestHandler)
	GetNotification(handler *req.RequestHandler)
	MarkAsRead(handler *req.RequestHandler)
	MarkAllAsRead(handler *req.RequestHandler)
	DeleteNotification(handler *req.RequestHandler)

	// Preference endpoints
	GetPreferences(handler *req.RequestHandler)
	UpdatePreferences(handler *req.RequestHandler)

	// Internal endpoints
	SendNotification(handler *req.RequestHandler)
	SendBulkNotifications(handler *req.RequestHandler)
}

// Compile-time interface compliance check
var _ NotificationHandlerInterface = (*NotificationHandler)(nil)
