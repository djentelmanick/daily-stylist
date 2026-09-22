package miniapp

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/texts"
)

const maxRequestBodyBytes = 1 << 16

type wardrobe interface {
	AddItem(ctx context.Context, params domain.NewItemParams) (domain.Item, error)
	Items(ctx context.Context, userID int64) ([]domain.Item, error)
	EditItem(ctx context.Context, userID, itemID int64, params domain.EditItemParams) (domain.Item, error)
	ChangeItemStatus(ctx context.Context, userID, itemID int64, status domain.ItemStatus) error
	DeleteItems(ctx context.Context, userID int64, itemIDs []int64) error
}

type recommender interface {
	Recommend(ctx context.Context, userID int64) (service.Recommendation, error)
	TodayOutfit(ctx context.Context, userID int64) ([]domain.Item, error)
	WearToday(ctx context.Context, userID int64, itemIDs []int64) error
}

type photos interface {
	RequestUpload(ctx context.Context, userID int64, contentType string, size int64) (service.PhotoUpload, error)
	Link(ctx context.Context, key string) (string, error)
}

type recognition interface {
	FromPhoto(ctx context.Context, userID int64, key string) (service.ItemSuggestion, error)
}

type locations interface {
	SearchCities(ctx context.Context, query string) ([]domain.Location, error)
	SetLocation(ctx context.Context, userID int64, location domain.Location) error
	City(ctx context.Context, userID int64) (domain.Location, error)
}

type settings interface {
	Get(ctx context.Context, userID int64) (domain.Settings, error)
	Save(ctx context.Context, userID int64, settings domain.Settings) error
}

type endpoints struct {
	wardrobe    wardrobe
	recommender recommender
	locations   locations
	settings    settings
	photos      photos
	recognition recognition
}

func NewHandler(
	botToken string,
	wardrobe wardrobe,
	recommender recommender,
	locations locations,
	settings settings,
	photos photos,
	recognition recognition,
) http.Handler {
	api := &endpoints{
		wardrobe:    wardrobe,
		recommender: recommender,
		locations:   locations,
		settings:    settings,
		photos:      photos,
		recognition: recognition,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/options", api.options)
	mux.HandleFunc("GET /api/items", api.listItems)
	mux.HandleFunc("POST /api/items", api.createItem)
	mux.HandleFunc("PUT /api/items/{id}", api.editItem)
	mux.HandleFunc("PUT /api/items/{id}/status", api.changeItemStatus)
	// Не DELETE с телом: тело у DELETE не определено стандартом, и прокси вправе его выбросить.
	mux.HandleFunc("POST /api/items/delete", api.deleteItems)
	mux.HandleFunc("POST /api/photos", api.requestPhotoUpload)
	mux.HandleFunc("POST /api/photos/recognize", api.recognizePhoto)
	mux.HandleFunc("GET /api/recommendation", api.recommend)
	mux.HandleFunc("GET /api/outfits/today", api.todayOutfit)
	mux.HandleFunc("PUT /api/outfits/today", api.wearToday)
	mux.HandleFunc("GET /api/cities", api.searchCities)
	mux.HandleFunc("PUT /api/city", api.setCity)
	mux.HandleFunc("GET /api/settings", api.getSettings)
	mux.HandleFunc("PUT /api/settings", api.saveSettings)

	return requireUser(botToken, mux)
}

type option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type categoryOption struct {
	Value     string `json:"value"`
	Label     string `json:"label"`
	HasWarmth bool   `json:"has_warmth"`
}

type warmthLevelOption struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}

type textLimits struct {
	Name         int      `json:"name"`
	Description  int      `json:"description"`
	PhotoBytes   int64    `json:"photo_bytes"`
	PhotoTypes   []string `json:"photo_types"`
	MinCityQuery int      `json:"min_city_query"`
}

type optionsResponse struct {
	Categories    []categoryOption    `json:"categories"`
	Colors        []option            `json:"colors"`
	Seasons       []option            `json:"seasons"`
	CurrentSeason string              `json:"current_season"`
	WarmthLevels  []warmthLevelOption `json:"warmth_levels"`
	Statuses      []option            `json:"statuses"`
	Limits        textLimits          `json:"limits"`
}

func (api *endpoints) options(writer http.ResponseWriter, request *http.Request) {
	response := optionsResponse{
		// TODO: По времени сервера: пока нет часового пояса пользователя, ошибиться можно
		// только на несколько часов в ночь смены сезона.
		CurrentSeason: string(domain.SeasonAt(time.Now())),
		Limits: textLimits{
			Name:         domain.MaxNameLength,
			Description:  domain.MaxDescriptionLength,
			PhotoBytes:   service.MaxPhotoBytes,
			PhotoTypes:   service.PhotoTypes(),
			MinCityQuery: service.MinCityQueryLength,
		},
	}
	for _, category := range domain.AllCategories() {
		response.Categories = append(response.Categories, categoryOption{
			Value:     string(category),
			Label:     texts.Category(category),
			HasWarmth: category.HasWarmth(),
		})
	}
	for _, color := range domain.AllColors() {
		response.Colors = append(response.Colors, option{Value: string(color), Label: texts.Color(color)})
	}
	for _, season := range domain.AllSeasons() {
		response.Seasons = append(response.Seasons, option{Value: string(season), Label: texts.Season(season)})
	}
	for _, level := range domain.AllWarmthLevels() {
		response.WarmthLevels = append(response.WarmthLevels, warmthLevelOption{Value: int(level), Label: texts.WarmthLevel(level)})
	}
	for _, status := range domain.AllItemStatuses() {
		response.Statuses = append(response.Statuses, option{Value: string(status), Label: texts.ItemStatus(status)})
	}
	writeJSON(writer, http.StatusOK, response)
}

type itemRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	MainColor   string   `json:"main_color"`
	ExtraColors []string `json:"extra_colors"`
	Seasons     []string `json:"seasons"`
	WarmthLevel int      `json:"warmth_level"`
	Waterproof  bool     `json:"waterproof"`
	PhotoKey    string   `json:"photo_key"`
}

type itemResponse struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	MainColor   string   `json:"main_color"`
	ExtraColors []string `json:"extra_colors"`
	Seasons     []string `json:"seasons"`
	WarmthLevel int      `json:"warmth_level"`
	Waterproof  bool     `json:"waterproof"`
	PhotoKey    string   `json:"photo_key"`
	PhotoURL    string   `json:"photo_url"`
	Status      string   `json:"status"`
}

type itemsResponse struct {
	Items []itemResponse `json:"items"`
}

func (api *endpoints) listItems(writer http.ResponseWriter, request *http.Request) {
	items, err := api.wardrobe.Items(request.Context(), userIDFrom(request.Context()))
	if err != nil {
		writeFailure(writer, err)
		return
	}

	response := itemsResponse{Items: make([]itemResponse, len(items))}
	for index, item := range items {
		response.Items[index] = api.toItemResponse(request.Context(), item)
	}
	writeJSON(writer, http.StatusOK, response)
}

func (api *endpoints) createItem(writer http.ResponseWriter, request *http.Request) {
	var body itemRequest
	if !decodeBody(writer, request, &body) {
		return
	}

	item, err := api.wardrobe.AddItem(request.Context(), domain.NewItemParams{
		UserID:      userIDFrom(request.Context()),
		Name:        body.Name,
		Description: body.Description,
		Category:    domain.Category(body.Category),
		Colors:      body.colors(),
		Seasons:     fromStrings[domain.Season](body.Seasons),
		WarmthLevel: domain.WarmthLevel(body.WarmthLevel),
		Waterproof:  body.Waterproof,
		PhotoKey:    body.PhotoKey,
	})
	if err != nil {
		writeFailure(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, api.toItemResponse(request.Context(), item))
}

func (api *endpoints) editItem(writer http.ResponseWriter, request *http.Request) {
	itemID, ok := itemIDFrom(writer, request)
	if !ok {
		return
	}
	var body itemRequest
	if !decodeBody(writer, request, &body) {
		return
	}

	item, err := api.wardrobe.EditItem(request.Context(), userIDFrom(request.Context()), itemID, domain.EditItemParams{
		Name:        body.Name,
		Description: body.Description,
		Category:    domain.Category(body.Category),
		Colors:      body.colors(),
		Seasons:     fromStrings[domain.Season](body.Seasons),
		WarmthLevel: domain.WarmthLevel(body.WarmthLevel),
		Waterproof:  body.Waterproof,
		PhotoKey:    body.PhotoKey,
	})
	if err != nil {
		writeFailure(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, api.toItemResponse(request.Context(), item))
}

type statusRequest struct {
	Status string `json:"status"`
}

func (api *endpoints) changeItemStatus(writer http.ResponseWriter, request *http.Request) {
	itemID, ok := itemIDFrom(writer, request)
	if !ok {
		return
	}
	var body statusRequest
	if !decodeBody(writer, request, &body) {
		return
	}

	err := api.wardrobe.ChangeItemStatus(request.Context(), userIDFrom(request.Context()), itemID, domain.ItemStatus(body.Status))
	if err != nil {
		writeFailure(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

type deleteItemsRequest struct {
	IDs []int64 `json:"ids"`
}

func (api *endpoints) deleteItems(writer http.ResponseWriter, request *http.Request) {
	var body deleteItemsRequest
	if !decodeBody(writer, request, &body) {
		return
	}
	if len(body.IDs) == 0 {
		writeError(writer, http.StatusBadRequest, "bad_request")
		return
	}

	if err := api.wardrobe.DeleteItems(request.Context(), userIDFrom(request.Context()), body.IDs); err != nil {
		writeFailure(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (body itemRequest) colors() domain.Colors {
	return domain.Colors{
		Main:  domain.Color(body.MainColor),
		Extra: fromStrings[domain.Color](body.ExtraColors),
	}
}

// Ссылка на фотографию подписана и живёт недолго, поэтому выдаётся вместе
// с вещью, а не отдельным запросом.
func (api *endpoints) toItemResponse(ctx context.Context, item domain.Item) itemResponse {
	response := itemResponse{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Category:    string(item.Category),
		MainColor:   string(item.Colors.Main),
		ExtraColors: toStrings(item.Colors.Extra),
		Seasons:     toStrings(item.Seasons),
		WarmthLevel: int(item.WarmthLevel),
		Waterproof:  item.Waterproof,
		PhotoKey:    item.PhotoKey,
		Status:      string(item.Status),
	}
	if item.PhotoKey == "" {
		return response
	}

	url, err := api.photos.Link(ctx, item.PhotoKey)
	if err != nil {
		// Без ссылки вещь показывается без фотографии - это лучше, чем пустой экран.
		log.Printf("miniapp: %v", err)
		return response
	}
	response.PhotoURL = url
	return response
}

type photoUploadRequest struct {
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type photoUploadResponse struct {
	Key       string `json:"key"`
	UploadURL string `json:"upload_url"`
	ViewURL   string `json:"view_url"`
}

func (api *endpoints) requestPhotoUpload(writer http.ResponseWriter, request *http.Request) {
	var body photoUploadRequest
	if !decodeBody(writer, request, &body) {
		return
	}

	upload, err := api.photos.RequestUpload(request.Context(), userIDFrom(request.Context()), body.ContentType, body.Size)
	if err != nil {
		writeFailure(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, photoUploadResponse{Key: upload.Key, UploadURL: upload.URL, ViewURL: upload.ViewURL})
}

type recognizeRequest struct {
	PhotoKey string `json:"photo_key"`
}

type suggestionResponse struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	MainColor   string   `json:"main_color"`
	ExtraColors []string `json:"extra_colors"`
	Seasons     []string `json:"seasons"`
	WarmthLevel int      `json:"warmth_level"`
	Waterproof  bool     `json:"waterproof"`
}

func (api *endpoints) recognizePhoto(writer http.ResponseWriter, request *http.Request) {
	var body recognizeRequest
	if !decodeBody(writer, request, &body) {
		return
	}

	suggestion, err := api.recognition.FromPhoto(request.Context(), userIDFrom(request.Context()), body.PhotoKey)
	if err != nil {
		writeFailure(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, suggestionResponse{
		Name:        suggestion.Name,
		Category:    string(suggestion.Category),
		MainColor:   string(suggestion.Colors.Main),
		ExtraColors: toStrings(suggestion.Colors.Extra),
		Seasons:     toStrings(suggestion.Seasons),
		WarmthLevel: int(suggestion.WarmthLevel),
		Waterproof:  suggestion.Waterproof,
	})
}

func decodeBody(writer http.ResponseWriter, request *http.Request, body any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, maxRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(body); err != nil {
		writeError(writer, http.StatusBadRequest, "bad_request")
		return false
	}
	return true
}

func itemIDFrom(writer http.ResponseWriter, request *http.Request) (int64, bool) {
	itemID, err := strconv.ParseInt(request.PathValue("id"), 10, 64)
	if err != nil || itemID <= 0 {
		writeError(writer, http.StatusNotFound, "not_found")
		return 0, false
	}
	return itemID, true
}

func writeFailure(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidItem):
		log.Printf("miniapp: %v", err)
		writeError(writer, http.StatusUnprocessableEntity, "invalid_item")
	case errors.Is(err, service.ErrItemNotFound):
		writeError(writer, http.StatusNotFound, "not_found")
	case errors.Is(err, service.ErrWardrobeFull):
		writeError(writer, http.StatusConflict, "wardrobe_full")
	case errors.Is(err, domain.ErrInvalidLocation):
		log.Printf("miniapp: %v", err)
		writeError(writer, http.StatusUnprocessableEntity, "invalid_location")
	case errors.Is(err, service.ErrLocationNotSet):
		writeError(writer, http.StatusConflict, "location_not_set")
	case errors.Is(err, domain.ErrInvalidSettings):
		log.Printf("miniapp: %v", err)
		writeError(writer, http.StatusUnprocessableEntity, "invalid_settings")
	case errors.Is(err, service.ErrPhotoTooLarge):
		log.Printf("miniapp: %v", err)
		writeError(writer, http.StatusUnprocessableEntity, "photo_too_large")
	case errors.Is(err, service.ErrPhotoTypeUnsupported):
		log.Printf("miniapp: %v", err)
		writeError(writer, http.StatusUnprocessableEntity, "photo_type_unsupported")
	case errors.Is(err, service.ErrPhotoNotUploaded):
		log.Printf("miniapp: %v", err)
		writeError(writer, http.StatusUnprocessableEntity, "photo_not_uploaded")
	case errors.Is(err, service.ErrRecognitionLimit):
		writeError(writer, http.StatusTooManyRequests, "recognition_limit")
	case errors.Is(err, service.ErrRecognitionUnavailable):
		log.Printf("miniapp: %v", err)
		writeError(writer, http.StatusBadGateway, "recognition_unavailable")
	case errors.Is(err, service.ErrWeatherUnavailable):
		log.Printf("miniapp: %v", err)
		writeError(writer, http.StatusBadGateway, "weather_unavailable")
	default:
		log.Printf("miniapp: %v", err)
		writeError(writer, http.StatusInternalServerError, "internal")
	}
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(body); err != nil {
		log.Printf("miniapp: запись ответа: %v", err)
	}
}

func writeError(writer http.ResponseWriter, status int, code string) {
	writeJSON(writer, status, map[string]string{"error": code})
}

func fromStrings[T ~string](values []string) []T {
	result := make([]T, len(values))
	for index, value := range values {
		result[index] = T(value)
	}
	return result
}

func toStrings[T ~string](values []T) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = string(value)
	}
	return result
}
