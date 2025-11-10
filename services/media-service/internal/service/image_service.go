package service

import (
	"media-service/internal/repo"
	"shared/pkg/logger"
)

type ImageService struct {
	fileRepo *repo.FileRepository
	log      logger.Logger
}

func NewImageService(fileRepo *repo.FileRepository, log logger.Logger) ImageService {
	return ImageService{
		fileRepo: fileRepo,
		log:      log,
	}
}
