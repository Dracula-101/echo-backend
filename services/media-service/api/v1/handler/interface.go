package handler

import req "shared/server/request"

// HandlerInterface defines the contract for media HTTP handlers
type HandlerInterface interface {
	// File operations
	Upload(handler *req.RequestHandler)
	GetFile(handler *req.RequestHandler)
	DeleteFile(handler *req.RequestHandler)

	// Album operations
	CreateAlbum(handler *req.RequestHandler)
	GetAlbum(handler *req.RequestHandler)
	ListAlbums(handler *req.RequestHandler)
	AddFileToAlbum(handler *req.RequestHandler)
	RemoveFileFromAlbum(handler *req.RequestHandler)

	// Message media operations
	UploadMessageMedia(handler *req.RequestHandler)

	// Profile operations
	UploadProfilePhoto(handler *req.RequestHandler)
	// Share operations
	CreateShare(handler *req.RequestHandler)
	RevokeShare(handler *req.RequestHandler)

	// Stats operations
	GetStorageStats(handler *req.RequestHandler)
}

// Ensure Handler implements HandlerInterface
var _ HandlerInterface = (*Handler)(nil)
