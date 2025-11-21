package service

import (
	"context"

	"media-service/internal/errors"
	"media-service/internal/service/models"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

func (s *MediaService) CreateAlbum(ctx context.Context, input models.CreateAlbumInput) (*models.CreateAlbumOutput, pkgErrors.AppError) {
	s.log.Info("Creating album",
		logger.String("user_id", input.UserID),
		logger.String("title", input.Title),
	)

	album := &dbModels.Album{
		UserID:      input.UserID,
		Title:       input.Title,
		Description: &input.Description,
		AlbumType:   dbModels.AlbumType(input.AlbumType),
		Visibility:  dbModels.MediaVisibility(input.Visibility),
	}
	albumID, err := s.repo.CreateAlbum(ctx, album)

	if err != nil {
		s.log.Error("Failed to create album", logger.Error(err))
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create album").
			WithDetail("user_id", input.UserID).
			WithDetail("title", input.Title).
			WithService("media_service")
	}

	return &models.CreateAlbumOutput{
		AlbumID:   albumID,
		Title:     input.Title,
		AlbumType: input.AlbumType,
	}, nil
}

func (s *MediaService) GetAlbum(ctx context.Context, input models.GetAlbumInput) (*models.GetAlbumOutput, pkgErrors.AppError) {
	album, err := s.repo.GetAlbumByID(ctx, input.AlbumID)
	if err != nil || album == nil {
		return nil, pkgErrors.New(errors.CodeAlbumNotFound, "album not found").
			WithDetail("album_id", input.AlbumID).
			WithService("media_service")
	}

	if album.UserID != input.UserID {
		return nil, pkgErrors.New(pkgErrors.CodeNotAcceptable, "access denied").
			WithDetail("album_id", input.AlbumID).
			WithDetail("user_id", input.UserID).
			WithService("media_service")
	}

	description := ""
	if album.Description != nil {
		description = *album.Description
	}

	coverFileID := ""
	if album.CoverFileID != nil {
		coverFileID = *album.CoverFileID
	}

	return &models.GetAlbumOutput{
		AlbumID:     album.ID,
		UserID:      album.UserID,
		Title:       album.Title,
		Description: description,
		CoverFileID: coverFileID,
		AlbumType:   string(album.AlbumType),
		FileCount:   album.FileCount,
		Visibility:  string(album.Visibility),
		UpdatedAt:   album.UpdatedAt,
	}, nil
}

func (s *MediaService) ListAlbums(ctx context.Context, input models.ListAlbumsInput) ([]*models.GetAlbumOutput, pkgErrors.AppError) {
	albums, err := s.repo.ListAlbumsByUser(ctx, input.UserID, input.Limit, input.Offset)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to list albums").
			WithDetail("user_id", input.UserID).
			WithService("media_service")
	}

	var result []*models.GetAlbumOutput
	for _, album := range albums {
		description := ""
		if album.Description != nil {
			description = *album.Description
		}

		coverFileID := ""
		if album.CoverFileID != nil {
			coverFileID = *album.CoverFileID
		}

		result = append(result, &models.GetAlbumOutput{
			AlbumID:     album.ID,
			UserID:      album.UserID,
			Title:       album.Title,
			Description: description,
			CoverFileID: coverFileID,
			AlbumType:   string(album.AlbumType),
			FileCount:   album.FileCount,
			Visibility:  string(album.Visibility),
			UpdatedAt:   album.UpdatedAt,
		})
	}

	return result, nil
}

func (s *MediaService) AddFileToAlbum(ctx context.Context, input models.AddFileToAlbumInput) pkgErrors.AppError {
	album, err := s.repo.GetAlbumByID(ctx, input.AlbumID)
	if err != nil || album == nil {
		return pkgErrors.New(errors.CodeAlbumNotFound, "album not found").
			WithDetail("album_id", input.AlbumID).
			WithService("media_service")
	}

	if album.UserID != input.UserID {
		return pkgErrors.New(pkgErrors.CodeNotAcceptable, "access denied").
			WithDetail("album_id", input.AlbumID).
			WithDetail("user_id", input.UserID).
			WithService("media_service")
	}

	albumFile := &dbModels.AlbumFile{
		AlbumID:      input.AlbumID,
		FileID:       input.FileID,
		DisplayOrder: &input.DisplayOrder,
	}

	return s.repo.AddFileToAlbum(ctx, albumFile)
}

func (s *MediaService) RemoveFileFromAlbum(ctx context.Context, input models.RemoveFileFromAlbumInput) pkgErrors.AppError {
	album, err := s.repo.GetAlbumByID(ctx, input.AlbumID)
	if err != nil || album == nil {
		return pkgErrors.New(errors.CodeAlbumNotFound, "album not found").
			WithDetail("album_id", input.AlbumID).
			WithService("media_service")
	}

	if album.UserID != input.UserID {
		return pkgErrors.New(pkgErrors.CodeNotAcceptable, "access denied").
			WithDetail("album_id", input.AlbumID).
			WithDetail("user_id", input.UserID).
			WithService("media_service")
	}

	return s.repo.RemoveFileFromAlbum(ctx, input.AlbumID, input.FileID)
}
