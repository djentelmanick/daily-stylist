package miniapp

import (
	"errors"
	"net/http"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

type settingsRequest struct {
	MorningEnabled bool   `json:"morning_enabled"`
	SendAt         string `json:"send_at"`
}

type settingsResponse struct {
	MorningEnabled bool      `json:"morning_enabled"`
	SendAt         string    `json:"send_at"`
	City           *cityBody `json:"city"`
}

// @Summary  Настройки рассылки и выбранный город
// @Description Пока пользователь ничего не менял, возвращаются значения по умолчанию из домена: рассылка включена, время 7:00.
// @Tags     Город и настройки
// @Produce  json
// @Security initData
// @Success  200 {object} settingsResponse
// @Failure  401 {object} errorResponse "unauthorized"
// @Router   /api/settings [get]
func (api *endpoints) getSettings(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	userID := userIDFrom(ctx)

	current, err := api.settings.Get(ctx, userID)
	if err != nil {
		writeFailure(writer, err)
		return
	}
	response := settingsResponse{MorningEnabled: current.MorningEnabled, SendAt: current.SendAt.String()}

	city, err := api.locations.City(ctx, userID)
	switch {
	case err == nil:
		body := toCityBody(city)
		response.City = &body
	case !errors.Is(err, service.ErrLocationNotSet):
		writeFailure(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

// @Summary  Сохранить настройки рассылки
// @Tags     Город и настройки
// @Accept   json
// @Produce  json
// @Security initData
// @Param    settings body settingsRequest true "Настройки"
// @Success  204 "Настройки сохранены"
// @Failure  400 {object} errorResponse "bad_request"
// @Failure  401 {object} errorResponse "unauthorized"
// @Failure  422 {object} errorResponse "invalid_settings"
// @Router   /api/settings [put]
func (api *endpoints) saveSettings(writer http.ResponseWriter, request *http.Request) {
	var body settingsRequest
	if !decodeBody(writer, request, &body) {
		return
	}

	sendAt, err := domain.ParseDayTime(body.SendAt)
	if err != nil {
		writeFailure(writer, err)
		return
	}

	settings := domain.Settings{MorningEnabled: body.MorningEnabled, SendAt: sendAt}
	if err := api.settings.Save(request.Context(), userIDFrom(request.Context()), settings); err != nil {
		writeFailure(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
