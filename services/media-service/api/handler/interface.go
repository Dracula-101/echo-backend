package handler

import (
	"net/http"
)

// HandlerInterface defines the contract for media HTTP handlers
type HandlerInterface interface {
	// File operations
	Upload(w http.ResponseWriter, r *http.Request)
	GetFile(w http.ResponseWriter, r *http.Request)
	DeleteFile(w http.ResponseWriter, r *http.Request)

	// Album operations
	CreateAlbum(w http.ResponseWriter, r *http.Request)
	GetAlbum(w http.ResponseWriter, r *http.Request)
	ListAlbums(w http.ResponseWriter, r *http.Request)
	AddFileToAlbum(w http.ResponseWriter, r *http.Request)
	RemoveFileFromAlbum(w http.ResponseWriter, r *http.Request)

	// Message media operations
	UploadMessageMedia(w http.ResponseWriter, r *http.Request)

	// Profile operations
	UploadProfilePhoto(w http.ResponseWriter, r *http.Request)

	// Share operations
	CreateShare(w http.ResponseWriter, r *http.Request)
	RevokeShare(w http.ResponseWriter, r *http.Request)

	// Stats operations
	GetStorageStats(w http.ResponseWriter, r *http.Request)
}

// Ensure Handler implements HandlerInterface
var _ HandlerInterface = (*Handler)(nil)
