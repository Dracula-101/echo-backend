package handler

import (
	"net/http"
	"shared/server/request"
	"user-service/api/dto"
)

func (h *UserHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	handler := request.NewHandler(r, w)
	createProfileRequest := dto.NewCreateProfileRequest()
	if !handler.ParseValidateAndSend(createProfileRequest) {
		return
	}

}
