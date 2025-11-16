package handler

import "net/http"

type PresenceHandlerInterface interface {
	UpdatePresence(w http.ResponseWriter, r *http.Request)
	GetPresence(w http.ResponseWriter, r *http.Request)
	GetBulkPresence(w http.ResponseWriter, r *http.Request)
	Heartbeat(w http.ResponseWriter, r *http.Request)
	GetActiveDevices(w http.ResponseWriter, r *http.Request)
	SetTypingIndicator(w http.ResponseWriter, r *http.Request)
	GetTypingIndicators(w http.ResponseWriter, r *http.Request)
}

var _ PresenceHandlerInterface = (*PresenceHandler)(nil)
