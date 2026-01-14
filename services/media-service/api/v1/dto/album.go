package dto

type CreateAlbumRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=255"`
	Description string `json:"description"`
	AlbumType   string `json:"album_type" validate:"required"`
	Visibility  string `json:"visibility" validate:"required"`
}

type AddFileToAlbumRequest struct {
	FileID       string `json:"file_id" validate:"required"`
	DisplayOrder int    `json:"display_order"`
}
