package miniapp

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/texts"
)

const maxRequestBodyBytes = 1 << 16

type itemAdder interface {
	AddItem(ctx context.Context, params domain.NewItemParams) (domain.Item, error)
}

type endpoints struct {
	wardrobe itemAdder
}

func NewHandler(botToken string, wardrobe itemAdder) http.Handler {
	api := &endpoints{wardrobe: wardrobe}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/options", api.options)
	mux.HandleFunc("POST /api/items", api.createItem)

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
	Name        int `json:"name"`
	Description int `json:"description"`
}

type optionsResponse struct {
	Categories   []categoryOption    `json:"categories"`
	Colors       []option            `json:"colors"`
	Seasons      []option            `json:"seasons"`
	WarmthLevels []warmthLevelOption `json:"warmth_levels"`
	Limits       textLimits          `json:"limits"`
}

func (api *endpoints) options(writer http.ResponseWriter, request *http.Request) {
	response := optionsResponse{
		Limits: textLimits{Name: domain.MaxNameLength, Description: domain.MaxDescriptionLength},
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
	writeJSON(writer, http.StatusOK, response)
}

type createItemRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	MainColor   string   `json:"main_color"`
	ExtraColors []string `json:"extra_colors"`
	Seasons     []string `json:"seasons"`
	WarmthLevel int      `json:"warmth_level"`
	Waterproof  bool     `json:"waterproof"`
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
	Status      string   `json:"status"`
}

func (api *endpoints) createItem(writer http.ResponseWriter, request *http.Request) {
	var body createItemRequest
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, maxRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeError(writer, http.StatusBadRequest, "bad_request")
		return
	}

	item, err := api.wardrobe.AddItem(request.Context(), domain.NewItemParams{
		UserID:      userIDFrom(request.Context()),
		Name:        body.Name,
		Description: body.Description,
		Category:    domain.Category(body.Category),
		Colors: domain.Colors{
			Main:  domain.Color(body.MainColor),
			Extra: fromStrings[domain.Color](body.ExtraColors),
		},
		Seasons:     fromStrings[domain.Season](body.Seasons),
		WarmthLevel: domain.WarmthLevel(body.WarmthLevel),
		Waterproof:  body.Waterproof,
	})

	switch {
	case errors.Is(err, domain.ErrInvalidItem):
		log.Printf("miniapp: %v", err)
		writeError(writer, http.StatusUnprocessableEntity, "invalid_item")
	case errors.Is(err, service.ErrWardrobeFull):
		writeError(writer, http.StatusConflict, "wardrobe_full")
	case err != nil:
		log.Printf("miniapp: %v", err)
		writeError(writer, http.StatusInternalServerError, "internal")
	default:
		writeJSON(writer, http.StatusCreated, toItemResponse(item))
	}
}

func toItemResponse(item domain.Item) itemResponse {
	return itemResponse{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Category:    string(item.Category),
		MainColor:   string(item.Colors.Main),
		ExtraColors: toStrings(item.Colors.Extra),
		Seasons:     toStrings(item.Seasons),
		WarmthLevel: int(item.WarmthLevel),
		Waterproof:  item.Waterproof,
		Status:      string(item.Status),
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
