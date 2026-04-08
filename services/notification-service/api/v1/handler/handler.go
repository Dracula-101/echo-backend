package handler

import (
	"shared/pkg/logger"
)

type NotificationHandler struct {
	log logger.Logger
}

func NewNotificationHandler(log logger.Logger) *NotificationHandler {
	return &NotificationHandler{
		log: log,
	}
}
