package miniapp

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func newHandlerWithRecognition(recognition recognition) http.Handler {
	return NewHandler(testBotToken, failingWardrobe{}, &stubRecommender{}, nil, nil, &stubPhotos{}, recognition)
}

func TestRecognizePhoto_ReturnsFieldsForTheForm(t *testing.T) {
	recognition := &stubRecognition{suggestion: service.ItemSuggestion{
		Name:        "Пуховик",
		Category:    domain.CategoryOuterwear,
		Colors:      domain.Colors{Main: domain.ColorNavy, Extra: []domain.Color{domain.ColorGray}},
		Seasons:     []domain.Season{domain.SeasonWinter},
		WarmthLevel: domain.WarmthLevelExtreme,
		Waterproof:  true,
	}}
	handler := newHandlerWithRecognition(recognition)

	response := serve(handler, newRequest(http.MethodPost, "/api/photos/recognize",
		`{"photo_key":"users/42/photo.jpg"}`, signedInitData(testBotToken, 42, time.Now())))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusOK, response.Body)
	}
	if recognition.key != "users/42/photo.jpg" {
		t.Errorf("распознавали ключ %q", recognition.key)
	}

	var suggestion suggestionResponse
	if err := json.NewDecoder(response.Body).Decode(&suggestion); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	want := suggestionResponse{
		Name:        "Пуховик",
		Category:    "outerwear",
		MainColor:   "navy",
		ExtraColors: []string{"gray"},
		Seasons:     []string{"winter"},
		WarmthLevel: 5,
		Waterproof:  true,
	}
	if suggestion.Name != want.Name || suggestion.Category != want.Category || suggestion.MainColor != want.MainColor {
		t.Errorf("подсказка = %+v, ожидалась %+v", suggestion, want)
	}
	if suggestion.WarmthLevel != want.WarmthLevel || !suggestion.Waterproof {
		t.Errorf("теплота = %d, непромокаемость = %v", suggestion.WarmthLevel, suggestion.Waterproof)
	}
	if len(suggestion.ExtraColors) != 1 || suggestion.ExtraColors[0] != "gray" || len(suggestion.Seasons) != 1 {
		t.Errorf("цвета = %v, сезоны = %v", suggestion.ExtraColors, suggestion.Seasons)
	}
}

func TestRecognizePhoto_NothingRecognizedIsStillAnAnswer(t *testing.T) {
	handler := newHandlerWithRecognition(&stubRecognition{})

	response := serve(handler, newRequest(http.MethodPost, "/api/photos/recognize",
		`{"photo_key":"users/42/photo.jpg"}`, signedInitData(testBotToken, 42, time.Now())))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusOK, response.Body)
	}

	var suggestion suggestionResponse
	if err := json.NewDecoder(response.Body).Decode(&suggestion); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if suggestion.Category != "" || suggestion.WarmthLevel != 0 {
		t.Errorf("подсказка = %+v, ожидалась пустая", suggestion)
	}
	// Пустой список, а не null: форме проще не различать эти два случая.
	if suggestion.ExtraColors == nil || suggestion.Seasons == nil {
		t.Errorf("цвета = %v, сезоны = %v, ожидались пустые списки", suggestion.ExtraColors, suggestion.Seasons)
	}
}

func TestRecognizePhoto_TellsWhyThereIsNoSuggestion(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"сервис распознавания недоступен", service.ErrRecognitionUnavailable, http.StatusBadGateway, "recognition_unavailable"},
		{"чужая или пропавшая фотография", service.ErrPhotoNotUploaded, http.StatusUnprocessableEntity, "photo_not_uploaded"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newHandlerWithRecognition(&stubRecognition{err: test.err})

			response := serve(handler, newRequest(http.MethodPost, "/api/photos/recognize",
				`{"photo_key":"users/42/photo.jpg"}`, signedInitData(testBotToken, 42, time.Now())))

			checkError(t, response, test.status, test.code)
		})
	}
}
