package session

import (
	"auth-service/internal/domain"
	"shared/pkg/database/postgres/models"
)

func toDBSessionType(t domain.SessionType) models.SessionType {
	switch t {
	case domain.SessionTypeMobile:
		return models.SessionTypeMobile
	default:
		return models.SessionTypeWeb
	}
}
