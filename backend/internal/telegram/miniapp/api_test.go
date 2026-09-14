package miniapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

const validItemBody = `{"name":"Синее худи","description":"С капюшоном, из Uniqlo","category":"top","main_color":"blue","seasons":["autumn"],"warmth_level":2}`

func TestCreateItem_SavesItem(t *testing.T) {
	repository := &memoryRepository{}
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

func TestListItems_ReturnsOnlyOwnItems(t *testing.T) {
	handler := NewHandler(testBotToken, service.NewWardrobe(&memoryRepository{}))
	createAs(t, handler, 42)
	createAs(t, handler, 7)
	createAs(t, handler, 42)

	if ids := listIDs(t, handler, 42); !slices.Equal(ids, []int64{1, 3}) {
		t.Errorf("вещи пользователя 42 = %v, ожидались [1 3]", ids)
	}
}

func TestListItems_EmptyWardrobeIsEmptyArray(t *testing.T) {
	handler := NewHandler(testBotToken, service.NewWardrobe(&memoryRepository{}))

	response := serve(handler, newRequest(http.MethodGet, "/api/items", "", signedInitData(testBotToken, 42, time.Now())))

	if got := strings.TrimSpace(response.Body.String()); got != `{"items":[]}` {
		t.Errorf("тело = %s, ожидался пустой массив, а не null", got)
	}
}

func TestEditItem_ChangesFieldsAndKeepsStatus(t *testing.T) {
	handler := NewHandler(testBotToken, service.NewWardrobe(&memoryRepository{}))
	itemID := createAs(t, handler, 42)
	initData := signedInitData(testBotToken, 42, time.Now())
	serve(handler, newRequest(http.MethodPut, itemPath(itemID)+"/status", `{"status":"dirty"}`, initData))

	body := `{"name":"  Зонт  ","category":"umbrella","main_color":"black","seasons":["spring","autumn"],"waterproof":true}`
	response := serve(handler, newRequest(http.MethodPut, itemPath(itemID), body, initData))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusOK, response.Body)
	}
	var item itemResponse
	if err := json.NewDecoder(response.Body).Decode(&item); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if item.ID != itemID || item.Name != "Зонт" || item.Category != "umbrella" || !item.Waterproof || item.Status != "dirty" {
		t.Errorf("ответ = %+v", item)
	}
}

func TestChangeItemStatus(t *testing.T) {
	repository := &memoryRepository{}
	handler := NewHandler(testBotToken, service.NewWardrobe(repository))
	itemID := createAs(t, handler, 42)
	initData := signedInitData(testBotToken, 42, time.Now())

	response := serve(handler, newRequest(http.MethodPut, itemPath(itemID)+"/status", `{"status":"archived"}`, initData))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusNoContent, response.Body)
	}
	if status := repository.items[0].Status; status != domain.ItemStatusArchived {
		t.Errorf("статус в хранилище = %q", status)
	}

	response = serve(handler, newRequest(http.MethodPut, itemPath(itemID)+"/status", `{"status":"lost"}`, initData))
	checkError(t, response, http.StatusUnprocessableEntity, "invalid_item")
}

func TestItemRoutes_HideOtherUsersItems(t *testing.T) {
	handler := NewHandler(testBotToken, service.NewWardrobe(&memoryRepository{}))
	itemID := createAs(t, handler, 42)
	stranger := signedInitData(testBotToken, 7, time.Now())

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"изменение чужой вещи", http.MethodPut, itemPath(itemID), validItemBody},
		{"смена статуса чужой вещи", http.MethodPut, itemPath(itemID) + "/status", `{"status":"dirty"}`},
		{"несуществующий ID", http.MethodPut, itemPath(999), validItemBody},
		{"ID не число", http.MethodPut, "/api/items/abc", validItemBody},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := serve(handler, newRequest(test.method, test.path, test.body, stranger))

			checkError(t, response, http.StatusNotFound, "not_found")
		})
	}
}

func TestDeleteItems_DeletesOnlyOwnItems(t *testing.T) {
	handler := NewHandler(testBotToken, service.NewWardrobe(&memoryRepository{}))
	first := createAs(t, handler, 42)
	second := createAs(t, handler, 42)
	strangers := createAs(t, handler, 7)
	kept := createAs(t, handler, 42)

	body := fmt.Sprintf(`{"ids":[%d,%d,%d]}`, first, second, strangers)
	response := serve(handler, newRequest(http.MethodPost, "/api/items/delete", body, signedInitData(testBotToken, 42, time.Now())))

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusNoContent, response.Body)
	}
	if ids := listIDs(t, handler, 42); !slices.Equal(ids, []int64{kept}) {
		t.Errorf("у пользователя 42 остались %v, ожидалась [%d]", ids, kept)
	}
	if ids := listIDs(t, handler, 7); !slices.Equal(ids, []int64{strangers}) {
		t.Errorf("у пользователя 7 остались %v, ожидалась [%d]", ids, strangers)
	}
}

func TestDeleteItems_RequiresIDs(t *testing.T) {
	handler := NewHandler(testBotToken, failingWardrobe{err: errors.New("сервис не должен вызываться")})

	response := serve(handler, newRequest(http.MethodPost, "/api/items/delete", `{"ids":[]}`, signedInitData(testBotToken, 42, time.Now())))

	checkError(t, response, http.StatusBadRequest, "bad_request")
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
		{"сбой хранилища", errors.New("база недоступна"), http.StatusInternalServerError, "internal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHandler(testBotToken, failingWardrobe{err: test.err})

			response := serve(handler, newRequest(http.MethodPost, "/api/items", validItemBody, signedInitData(testBotToken, 42, time.Now())))

			checkError(t, response, test.wantStatus, test.wantCode)
		})
	}
}

func TestCreateItem_RejectsBadRequests(t *testing.T) {
	handler := NewHandler(testBotToken, failingWardrobe{err: errors.New("сервис не должен вызываться")})
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
	handler := NewHandler(testBotToken, failingWardrobe{})

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
		len(options.WarmthLevels) != len(domain.AllWarmthLevels()) ||
		len(options.Statuses) != len(domain.AllItemStatuses()) {
		t.Errorf("списки не совпадают с доменом: %+v", options)
	}
	if !domain.Season(options.CurrentSeason).Valid() {
		t.Errorf("текущий сезон = %q, ожидался один из сезонов домена", options.CurrentSeason)
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

type memoryRepository struct {
	items  []domain.Item
	lastID int64
}

func (repository *memoryRepository) Create(_ context.Context, item domain.Item) (domain.Item, error) {
	repository.lastID++
	item.ID = repository.lastID
	repository.items = append(repository.items, item)
	return item, nil
}

func (repository *memoryRepository) CountByUser(ctx context.Context, userID int64) (int, error) {
	items, err := repository.ListByUser(ctx, userID)
	return len(items), err
}

func (repository *memoryRepository) ListByUser(_ context.Context, userID int64) ([]domain.Item, error) {
	var items []domain.Item
	for _, item := range repository.items {
		if item.UserID == userID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (repository *memoryRepository) Get(_ context.Context, userID, itemID int64) (domain.Item, error) {
	index := repository.find(userID, itemID)
	if index == -1 {
		return domain.Item{}, service.ErrItemNotFound
	}
	return repository.items[index], nil
}

func (repository *memoryRepository) Update(_ context.Context, item domain.Item) error {
	index := repository.find(item.UserID, item.ID)
	if index == -1 {
		return service.ErrItemNotFound
	}
	item.Status = repository.items[index].Status
	repository.items[index] = item
	return nil
}

func (repository *memoryRepository) UpdateStatus(_ context.Context, userID, itemID int64, status domain.ItemStatus) error {
	index := repository.find(userID, itemID)
	if index == -1 {
		return service.ErrItemNotFound
	}
	repository.items[index].Status = status
	return nil
}

func (repository *memoryRepository) Delete(_ context.Context, userID int64, itemIDs []int64) error {
	repository.items = slices.DeleteFunc(repository.items, func(item domain.Item) bool {
		return item.UserID == userID && slices.Contains(itemIDs, item.ID)
	})
	return nil
}

func (repository *memoryRepository) find(userID, itemID int64) int {
	return slices.IndexFunc(repository.items, func(item domain.Item) bool {
		return item.UserID == userID && item.ID == itemID
	})
}

type failingWardrobe struct {
	err error
}

func (wardrobe failingWardrobe) AddItem(context.Context, domain.NewItemParams) (domain.Item, error) {
	return domain.Item{}, wardrobe.err
}

func (wardrobe failingWardrobe) Items(context.Context, int64) ([]domain.Item, error) {
	return nil, wardrobe.err
}

func (wardrobe failingWardrobe) EditItem(context.Context, int64, int64, domain.EditItemParams) (domain.Item, error) {
	return domain.Item{}, wardrobe.err
}

func (wardrobe failingWardrobe) ChangeItemStatus(context.Context, int64, int64, domain.ItemStatus) error {
	return wardrobe.err
}

func (wardrobe failingWardrobe) DeleteItems(context.Context, int64, []int64) error {
	return wardrobe.err
}

func createAs(t *testing.T, handler http.Handler, userID int64) int64 {
	t.Helper()

	response := serve(handler, newRequest(http.MethodPost, "/api/items", validItemBody, signedInitData(testBotToken, userID, time.Now())))
	if response.Code != http.StatusCreated {
		t.Fatalf("создание вещи: status = %d; тело: %s", response.Code, response.Body)
	}
	var item itemResponse
	if err := json.NewDecoder(response.Body).Decode(&item); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	return item.ID
}

func listIDs(t *testing.T, handler http.Handler, userID int64) []int64 {
	t.Helper()

	response := serve(handler, newRequest(http.MethodGet, "/api/items", "", signedInitData(testBotToken, userID, time.Now())))
	if response.Code != http.StatusOK {
		t.Fatalf("список вещей: status = %d; тело: %s", response.Code, response.Body)
	}
	var body itemsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	ids := []int64{}
	for _, item := range body.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

func itemPath(itemID int64) string {
	return fmt.Sprintf("/api/items/%d", itemID)
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
