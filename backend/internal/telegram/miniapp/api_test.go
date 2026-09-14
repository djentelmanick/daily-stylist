package miniapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/jsonfile"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

const validItemBody = `{"name":"Синее худи","description":"С капюшоном, из Uniqlo","category":"top","main_color":"blue","seasons":["autumn"],"warmth_level":2}`

func TestCreateItem_SavesItem(t *testing.T) {
	repository := jsonfile.NewItemRepository(filepath.Join(t.TempDir(), "items.json"))
	handler := NewHandler(testBotToken, service.NewWardrobe(repository))

	response := serve(handler, newRequest(http.MethodPost, "/api/items", validItemBody, signedInitData(testBotToken, 42, time.Now())))

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusCreated, response.Body)
	}
	var item itemResponse
	if err := json.NewDecoder(response.Body).Decode(&item); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if item.ID != 1 || item.Name != "Синее худи" || item.Description != "С капюшоном, из Uniqlo" || item.Status != string(domain.ItemStatusAvailable) {
		t.Errorf("ответ = %+v", item)
	}

	count, err := repository.CountByUser(t.Context(), 42)
	if err != nil {
		t.Fatalf("CountByUser: %v", err)
	}
	if count != 1 {
		t.Errorf("у пользователя %d вещей, ожидалась 1", count)
	}
}

func TestCreateItem_MapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"невалидная вещь", fmt.Errorf("добавление вещи: %w", domain.ErrInvalidItem), http.StatusUnprocessableEntity, "invalid_item"},
		{"гардероб полон", fmt.Errorf("добавление вещи: %w", service.ErrWardrobeFull), http.StatusConflict, "wardrobe_full"},
		{"сбой хранилища", errors.New("диск недоступен"), http.StatusInternalServerError, "internal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHandler(testBotToken, failingAdder{err: test.err})

			response := serve(handler, newRequest(http.MethodPost, "/api/items", validItemBody, signedInitData(testBotToken, 42, time.Now())))

			checkError(t, response, test.wantStatus, test.wantCode)
		})
	}
}

func TestCreateItem_RejectsBadRequests(t *testing.T) {
	handler := NewHandler(testBotToken, failingAdder{err: errors.New("сервис не должен вызываться")})
	now := time.Now()

	tests := []struct {
		name       string
		body       string
		initData   string
		wantStatus int
		wantCode   string
	}{
		{"без initData", validItemBody, "", http.StatusUnauthorized, "unauthorized"},
		{"подпись другим токеном", validItemBody, signedInitData("999:OTHER-TOKEN", 42, now), http.StatusUnauthorized, "unauthorized"},
		{"неизвестное поле", `{"name":"Худи","colour":"red"}`, signedInitData(testBotToken, 42, now), http.StatusBadRequest, "bad_request"},
		{"не JSON", `name=Худи`, signedInitData(testBotToken, 42, now), http.StatusBadRequest, "bad_request"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := serve(handler, newRequest(http.MethodPost, "/api/items", test.body, test.initData))

			checkError(t, response, test.wantStatus, test.wantCode)
		})
	}
}

func TestOptions_ListsDomainValues(t *testing.T) {
	handler := NewHandler(testBotToken, failingAdder{})

	response := serve(handler, newRequest(http.MethodGet, "/api/options", "", signedInitData(testBotToken, 42, time.Now())))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, ожидался %d", response.Code, http.StatusOK)
	}
	var options optionsResponse
	if err := json.NewDecoder(response.Body).Decode(&options); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if len(options.Categories) != len(domain.AllCategories()) ||
		len(options.Colors) != len(domain.AllColors()) ||
		len(options.Seasons) != len(domain.AllSeasons()) ||
		len(options.WarmthLevels) != len(domain.AllWarmthLevels()) {
		t.Errorf("списки не совпадают с доменом: %+v", options)
	}
	if options.Limits.Name != domain.MaxNameLength || options.Limits.Description != domain.MaxDescriptionLength {
		t.Errorf("лимиты = %+v, ожидались %d и %d", options.Limits, domain.MaxNameLength, domain.MaxDescriptionLength)
	}
	for _, category := range options.Categories {
		if category.Value == string(domain.CategoryUmbrella) && category.HasWarmth {
			t.Errorf("у зонта не должно быть уровня теплоты")
		}
	}
}

type failingAdder struct {
	err error
}

func (adder failingAdder) AddItem(context.Context, domain.NewItemParams) (domain.Item, error) {
	return domain.Item{}, adder.err
}

func newRequest(method, path, body, initData string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if initData != "" {
		request.Header.Set("Authorization", "tma "+initData)
	}
	return request
}

func serve(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func checkError(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	if response.Code != wantStatus {
		t.Errorf("status = %d, ожидался %d; тело: %s", response.Code, wantStatus, response.Body)
	}
	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if body["error"] != wantCode {
		t.Errorf("код ошибки = %q, ожидался %q", body["error"], wantCode)
	}
}
