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
