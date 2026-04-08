package handler

import (
	"shared/pkg/logger"
)

type AnalyticsHandler struct {
	log logger.Logger
}

func NewAnalyticsHandler(log logger.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		log: log,
	}
}
