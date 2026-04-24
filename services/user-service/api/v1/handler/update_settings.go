package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"shared/pkg/logger"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"user-service/api/v1/dto"
	repoSettings "user-service/internal/repo/settings"
)

var reHHMM = regexp.MustCompile(`^\d{2}:\d{2}$`)

var acceptedLanguageCodes = map[string]struct{}{
	"af": {}, "sq": {}, "am": {}, "ar": {}, "hy": {}, "az": {}, "eu": {},
	"be": {}, "bn": {}, "bs": {}, "bg": {}, "ca": {}, "zh": {}, "hr": {},
	"cs": {}, "da": {}, "nl": {}, "en": {}, "et": {}, "fi": {}, "fr": {},
	"gl": {}, "ka": {}, "de": {}, "el": {}, "gu": {}, "ht": {}, "ha": {},
	"he": {}, "hi": {}, "hu": {}, "is": {}, "ig": {}, "id": {}, "ga": {},
	"it": {}, "ja": {}, "kn": {}, "kk": {}, "km": {}, "ko": {}, "ku": {},
	"ky": {}, "lo": {}, "lv": {}, "lt": {}, "lb": {}, "mk": {}, "mg": {},
	"ms": {}, "ml": {}, "mt": {}, "mi": {}, "mr": {}, "mn": {}, "my": {},
	"ne": {}, "nb": {}, "or": {}, "ps": {}, "fa": {}, "pl": {}, "pt": {},
	"pa": {}, "ro": {}, "ru": {}, "sm": {}, "sr": {}, "sn": {}, "sd": {},
	"si": {}, "sk": {}, "sl": {}, "so": {}, "es": {}, "su": {}, "sw": {},
	"sv": {}, "tl": {}, "tg": {}, "ta": {}, "tt": {}, "te": {}, "th": {},
	"tr": {}, "tk": {}, "uk": {}, "ur": {}, "ug": {}, "uz": {}, "vi": {},
	"cy": {}, "xh": {}, "yi": {}, "yo": {}, "zu": {},
}

func validateHHMM(value string) error {
	if !reHHMM.MatchString(value) {
		return fmt.Errorf("must be in HH:MM format")
	}
	parts := strings.SplitN(value, ":", 2)
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	if h > 23 || m > 59 {
		return fmt.Errorf("must be a valid time (HH:MM)")
	}
	return nil
}

func (h *UserHandler) UpdateSettings(handler *req.RequestHandler) {
	ctx := handler.Context()
	requestID := handler.GetRequestID()
	correlationID := handler.GetCorrelationID()

	h.log.Info("Received request to update settings",
		logger.String("request_id", requestID),
		logger.String("correlation_id", correlationID),
	)

	body := dto.NewUpdateSettingsRequest()
	if ok := handler.ParseValidateAndSend(body); !ok {
		return
	}

	// Require at least one field.
	if !hasAnySettingsField(body) {
		response.Error().
			WithContext(ctx).
			WithRequest(handler.Request()).
			WithMessage("At least one field must be provided").
			UnprocessableEntity(handler.Writer())
		return
	}

	userID, ok := request.GetUserIDFromContext(ctx)
	if !ok {
		response.UnauthorizedError(ctx, handler.Request(), handler.Writer(), "User is not authenticated", nil)
		return
	}

	// Validate HH:MM fields when provided.
	if body.QuietHoursStart != nil {
		if err := validateHHMM(*body.QuietHoursStart); err != nil {
			response.Error().
				WithContext(ctx).
				WithRequest(handler.Request()).
				WithMessage("quiet_hours_start: " + err.Error()).
				UnprocessableEntity(handler.Writer())
			return
		}
	}
	if body.QuietHoursEnd != nil {
		if err := validateHHMM(*body.QuietHoursEnd); err != nil {
			response.Error().
				WithContext(ctx).
				WithRequest(handler.Request()).
				WithMessage("quiet_hours_end: " + err.Error()).
				UnprocessableEntity(handler.Writer())
			return
		}
	}

	// If quiet_hours_enabled is being turned on, both start and end must be present.
	if body.QuietHoursEnabled != nil && *body.QuietHoursEnabled {
		if body.QuietHoursStart == nil || body.QuietHoursEnd == nil {
			response.Error().
				WithContext(ctx).
				WithRequest(handler.Request()).
				WithMessage("quiet_hours_start and quiet_hours_end are required when quiet_hours_enabled is true").
				UnprocessableEntity(handler.Writer())
			return
		}
	}

	// Validate language_code against accepted list.
	if body.LanguageCode != nil {
		code := strings.ToLower(*body.LanguageCode)
		if _, ok := acceptedLanguageCodes[code]; !ok {
			response.Error().
				WithContext(ctx).
				WithRequest(handler.Request()).
				WithMessage("language_code is not supported").
				UnprocessableEntity(handler.Writer())
			return
		}
	}

	privacyChanged := body.ProfileVisibility != nil ||
		body.LastSeenVisibility != nil ||
		body.OnlineStatusVisibility != nil ||
		body.ProfilePhotoVisibility != nil ||
		body.AboutVisibility != nil

	params := repoSettings.UpdateSettingsParams{
		ProfileVisibility:         body.ProfileVisibility,
		LastSeenVisibility:        body.LastSeenVisibility,
		OnlineStatusVisibility:    body.OnlineStatusVisibility,
		ProfilePhotoVisibility:    body.ProfilePhotoVisibility,
		AboutVisibility:           body.AboutVisibility,
		ReadReceiptsEnabled:       body.ReadReceiptsEnabled,
		TypingIndicatorsEnabled:   body.TypingIndicatorsEnabled,
		PushNotificationsEnabled:  body.PushNotificationsEnabled,
		EmailNotificationsEnabled: body.EmailNotificationsEnabled,
		SmsNotificationsEnabled:   body.SmsNotificationsEnabled,
		MessageNotifications:      body.MessageNotifications,
		GroupMessageNotifications: body.GroupMessageNotifications,
		MentionNotifications:      body.MentionNotifications,
		ReactionNotifications:     body.ReactionNotifications,
		CallNotifications:         body.CallNotifications,
		NotificationSound:         body.NotificationSound,
		VibrationEnabled:          body.VibrationEnabled,
		NotificationPreview:       body.NotificationPreview,
		QuietHoursEnabled:         body.QuietHoursEnabled,
		QuietHoursStart:           body.QuietHoursStart,
		QuietHoursEnd:             body.QuietHoursEnd,
		EnterKeyToSend:            body.EnterKeyToSend,
		AutoDownloadPhotos:        body.AutoDownloadPhotos,
		AutoDownloadVideos:        body.AutoDownloadVideos,
		AutoDownloadDocuments:     body.AutoDownloadDocuments,
		AutoDownloadOnWifiOnly:    body.AutoDownloadOnWifiOnly,
		CompressImages:            body.CompressImages,
		SaveToGallery:             body.SaveToGallery,
		Theme:                     body.Theme,
		FontSize:                  body.FontSize,
		LanguageCode:              body.LanguageCode,
	}

	updated, err := h.settingsRepo.UpdateSettings(ctx, userID, params)
	if err != nil {
		h.log.Error("Failed to update settings",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.InternalServerError(ctx, handler.Request(), handler.Writer(), "Failed to update settings", nil)
		return
	}

	// Invalidate both settings cache entries.
	if cacheErr := h.cache.DeleteSettings(ctx, userID); cacheErr != nil {
		h.log.Error("Failed to invalidate settings cache",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(cacheErr),
		)
	}
	if cacheErr := h.cache.DeleteFullSettings(ctx, userID); cacheErr != nil {
		h.log.Error("Failed to invalidate full settings cache",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(cacheErr),
		)
	}

	if privacyChanged {
		if _, cacheErr := h.cache.IncrPrivacyVersion(ctx, userID); cacheErr != nil {
			h.log.Error("Failed to increment privacy version",
				logger.String("request_id", requestID),
				logger.String("user_id", userID),
				logger.Error(cacheErr),
			)
		}
		// TODO: publish user.settings.privacy_changed to Kafka (ws-service needs to update visible data for connected contacts)
	}

	h.log.Info("Settings updated successfully",
		logger.String("request_id", requestID),
		logger.String("user_id", userID),
	)

	response.JSONWithMessage(ctx, handler.Request(), handler.Writer(), http.StatusOK, "Settings updated successfully", updated)
}

func hasAnySettingsField(r *dto.UpdateSettingsRequest) bool {
	return r.ProfileVisibility != nil ||
		r.LastSeenVisibility != nil ||
		r.OnlineStatusVisibility != nil ||
		r.ProfilePhotoVisibility != nil ||
		r.AboutVisibility != nil ||
		r.ReadReceiptsEnabled != nil ||
		r.TypingIndicatorsEnabled != nil ||
		r.PushNotificationsEnabled != nil ||
		r.EmailNotificationsEnabled != nil ||
		r.SmsNotificationsEnabled != nil ||
		r.MessageNotifications != nil ||
		r.GroupMessageNotifications != nil ||
		r.MentionNotifications != nil ||
		r.ReactionNotifications != nil ||
		r.CallNotifications != nil ||
		r.NotificationSound != nil ||
		r.VibrationEnabled != nil ||
		r.NotificationPreview != nil ||
		r.QuietHoursEnabled != nil ||
		r.QuietHoursStart != nil ||
		r.QuietHoursEnd != nil ||
		r.EnterKeyToSend != nil ||
		r.AutoDownloadPhotos != nil ||
		r.AutoDownloadVideos != nil ||
		r.AutoDownloadDocuments != nil ||
		r.AutoDownloadOnWifiOnly != nil ||
		r.CompressImages != nil ||
		r.SaveToGallery != nil ||
		r.Theme != nil ||
		r.FontSize != nil ||
		r.LanguageCode != nil
}
