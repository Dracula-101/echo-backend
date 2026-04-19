package handler

import (
	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
	"user-service/internal/domain"
)

func (h *UserHandler) UpdateMyProfile(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestId := handler.GetRequestID()
	correlationId := handler.GetCorrelationID()

	h.log.Debug("Received UpdateMyProfile request",
		logger.String("request_id", requestId),
		logger.String("correlation_id", correlationId),
	)

	updateProfileRequest := dto.NewUpdateProfileRequest()
	if ok := handler.ParseValidateAndSend(updateProfileRequest); !ok {
		h.log.Error("Failed to parse and validate UpdateMyProfile request",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
		)
		return
	}

	userId, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		h.log.Error("User ID not found in context",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
		)
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	location, _ := request.GetIPAddressInfoFromContext(ctx)

	profile, err := h.service.UpdateProfile(ctx, userId, &domain.UpdateProfileInput{
		DisplayName:  updateProfileRequest.DisplayName,
		FirstName:    updateProfileRequest.FirstName,
		LastName:     updateProfileRequest.LastName,
		Bio:          updateProfileRequest.Bio,
		BioLinks:     updateProfileRequest.BioLinks,
		DateOfBirth:  updateProfileRequest.DateOfBirth,
		Gender:       updateProfileRequest.Gender,
		Pronouns:     updateProfileRequest.Pronouns,
		LanguageCode: updateProfileRequest.LanguageCode,
		Timezone:     &location.Timezone,
		CountryCode:  &location.CountryCode,
		City:         &location.City,
		WebsiteURL:   updateProfileRequest.WebsiteURL,
		SocialLinks:  updateProfileRequest.SocialLinks,
		Interests:    updateProfileRequest.Interests,
		PhoneVisible: updateProfileRequest.PhoneVisible,
		EmailVisible: updateProfileRequest.EmailVisible,
		ProfileVisibility: func() *domain.ProfileVisibility {
			if updateProfileRequest.ProfileVisibility == nil {
				return nil
			}
			v := domain.ProfileVisibility(*updateProfileRequest.ProfileVisibility)
			return &v
		}(),
		SearchVisibility: updateProfileRequest.SearchVisibility,
	})
	if err != nil {
		h.log.Error("Failed to update profile",
			logger.String("request_id", requestId),
			logger.String("correlation_id", correlationId),
			logger.Error(err),
		)
		return
	}

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), response.StatusOK, "Profile updated successfully", dto.NewUpdateProfileResponseFromDomain(profile))
}
