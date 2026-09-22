package miniapp

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func TestGetSettings_DefaultsWithoutCity(t *testing.T) {
	handler := newSettingsHandler(&memorySettings{}, &memoryLocations{})

	response := serve(handler, newRequest(http.MethodGet, "/api/settings", "", signedInitData(testBotToken, 42, time.Now())))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusOK, response.Body)
	}
	body := decodeSettings(t, response.Body.Bytes())
	if !body.MorningEnabled || body.SendAt != "07:00" {
		t.Errorf("ответ = %+v, ожидалась включённая рассылка в 07:00", body)
	}
	if body.City != nil {
		t.Errorf("город = %+v, ожидался null: он ещё не выбран", body.City)
	}
}

func TestSaveSettings(t *testing.T) {
	repository := &memorySettings{}
	locations := &memoryLocations{}
	if err := locations.Save(t.Context(), 42, kazan); err != nil {
		t.Fatalf("Save: %v", err)
	}
	handler := newSettingsHandler(repository, locations)
	initData := signedInitData(testBotToken, 42, time.Now())

	response := serve(handler, newRequest(http.MethodPut, "/api/settings", `{"morning_enabled":false,"send_at":"09:30"}`, initData))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusNoContent, response.Body)
	}

	body := decodeSettings(t, serve(handler, newRequest(http.MethodGet, "/api/settings", "", initData)).Body.Bytes())
	if body.MorningEnabled || body.SendAt != "09:30" {
		t.Errorf("ответ = %+v, ожидалась выключенная рассылка в 09:30", body)
	}
	if body.City == nil || body.City.Name != kazan.Name {
		t.Errorf("город = %+v, ожидалась Казань", body.City)
	}
	if _, saved := repository.saved[7]; saved {
		t.Error("настройки записаны чужому пользователю")
	}
}

func TestSaveSettings_RejectsTimeOutsideDay(t *testing.T) {
	handler := newSettingsHandler(&memorySettings{}, &memoryLocations{})

	response := serve(handler, newRequest(http.MethodPut, "/api/settings", `{"morning_enabled":true,"send_at":"25:00"}`, signedInitData(testBotToken, 42, time.Now())))

	checkError(t, response, http.StatusUnprocessableEntity, "invalid_settings")
}

func newSettingsHandler(repository *memorySettings, locations *memoryLocations) http.Handler {
	return NewHandler(
		testBotToken,
		failingWardrobe{},
		&stubRecommender{},
		service.NewLocations(locations, stubCities{}, nil, nil),
		service.NewSettings(repository),
		&stubPhotos{},
		&stubRecognition{},
	)
}

func decodeSettings(t *testing.T, body []byte) settingsResponse {
	t.Helper()

	var response settingsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	return response
}

type memorySettings struct {
	saved map[int64]domain.Settings
}

func (repository *memorySettings) Get(_ context.Context, userID int64) (domain.Settings, error) {
	if settings, ok := repository.saved[userID]; ok {
		return settings, nil
	}
	return domain.Settings{}, service.ErrSettingsNotSet
}

func (repository *memorySettings) Save(_ context.Context, userID int64, settings domain.Settings) error {
	if repository.saved == nil {
		repository.saved = map[int64]domain.Settings{}
	}
	repository.saved[userID] = settings
	return nil
}
