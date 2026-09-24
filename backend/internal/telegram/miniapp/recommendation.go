package miniapp

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/texts"
)

const maxOutfitItems = 20

type cityBody struct {
	Name      string  `json:"name"`
	Region    string  `json:"region"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	TimeZone  string  `json:"timezone"`
}

type weatherResponse struct {
	Temperature string   `json:"temperature"`
	Details     []string `json:"details"`
}

type outfitResponse struct {
	Items []itemResponse `json:"items"`
	Notes []string       `json:"notes"`
}

type recommendationResponse struct {
	City    cityBody         `json:"city"`
	Weather weatherResponse  `json:"weather"`
	Outfits []outfitResponse `json:"outfits"`
	Notes   []string         `json:"notes"`
}

// @Summary  Подобрать образы на сегодня
// @Description Несколько вариантов, лучший первым. Заметки - готовые строки по-русски.
// @Tags     Образ
// @Produce  json
// @Security initData
// @Success  200 {object} recommendationResponse
// @Failure  401 {object} errorResponse "unauthorized"
// @Failure  409 {object} errorResponse "location_not_set"
// @Failure  502 {object} errorResponse "weather_unavailable"
// @Router   /api/recommendation [get]
func (api *endpoints) recommend(writer http.ResponseWriter, request *http.Request) {
	recommendation, err := api.recommender.Recommend(request.Context(), userIDFrom(request.Context()))
	if err != nil {
		writeFailure(writer, err)
		return
	}

	response := recommendationResponse{
		City: toCityBody(recommendation.Location),
		Weather: weatherResponse{
			Temperature: texts.Temperature(recommendation.Weather),
			Details:     texts.WeatherDetails(recommendation.Weather),
		},
		Outfits: make([]outfitResponse, len(recommendation.Outfits)),
		Notes:   noteTexts(recommendation.Notes),
	}
	for index, outfit := range recommendation.Outfits {
		items := make([]itemResponse, len(outfit.Items))
		for itemIndex, item := range outfit.Items {
			items[itemIndex] = api.toItemResponse(request.Context(), item)
		}
		response.Outfits[index] = outfitResponse{Items: items, Notes: noteTexts(outfit.Notes)}
	}
	writeJSON(writer, http.StatusOK, response)
}

// Только ID: вещи у приложения уже есть, и так карточка покажет их после правки без перезагрузки.
type todayOutfitResponse struct {
	ItemIDs []int64 `json:"item_ids"`
}

// @Summary  Образ дня
// @Description Только идентификаторы: сами вещи у приложения уже загружены.
// @Tags     Образ
// @Produce  json
// @Security initData
// @Success  200 {object} todayOutfitResponse
// @Failure  401 {object} errorResponse "unauthorized"
// @Router   /api/outfits/today [get]
func (api *endpoints) todayOutfit(writer http.ResponseWriter, request *http.Request) {
	items, err := api.recommender.TodayOutfit(request.Context(), userIDFrom(request.Context()))
	if err != nil {
		writeFailure(writer, err)
		return
	}

	response := todayOutfitResponse{ItemIDs: make([]int64, len(items))}
	for index, item := range items {
		response.ItemIDs[index] = item.ID
	}
	writeJSON(writer, http.StatusOK, response)
}

type wearRequest struct {
	ItemIDs []int64 `json:"item_ids"`
}

// @Summary  Записать образ дня
// @Description Запись заменяет образ целиком, пустой список его очищает.
// @Tags     Образ
// @Accept   json
// @Produce  json
// @Security initData
// @Param    outfit body wearRequest true "Вещи образа"
// @Success  204 "Образ записан"
// @Failure  400 {object} errorResponse "bad_request"
// @Failure  401 {object} errorResponse "unauthorized"
// @Failure  404 {object} errorResponse "not_found"
// @Router   /api/outfits/today [put]
func (api *endpoints) wearToday(writer http.ResponseWriter, request *http.Request) {
	var body wearRequest
	if !decodeBody(writer, request, &body) {
		return
	}
	if body.ItemIDs == nil || len(body.ItemIDs) > maxOutfitItems {
		writeError(writer, http.StatusBadRequest, "bad_request")
		return
	}

	if err := api.recommender.WearToday(request.Context(), userIDFrom(request.Context()), body.ItemIDs); err != nil {
		writeFailure(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

type candidateResponse struct {
	ItemID int64    `json:"item_id"`
	Notes  []string `json:"notes"`
}

type candidatesResponse struct {
	Candidates []candidateResponse `json:"candidates"`
}

// @Summary  Чем заменить или дополнить образ
// @Description Кандидаты отсортированы по тому, насколько вещь подходит: теплота, цвета, дождь, недавнее ношение. Вещи не по сезону уходят в конец списка.
// @Tags     Образ
// @Produce  json
// @Security initData
// @Param    items query string true "Идентификаторы вещей образа через запятую" example(12,15,31)
// @Param    replace query int false "Какую вещь заменяем. Без неё - добавление вещи в образ"
// @Success  200 {object} candidatesResponse
// @Failure  400 {object} errorResponse "bad_request"
// @Failure  401 {object} errorResponse "unauthorized"
// @Failure  404 {object} errorResponse "not_found"
// @Failure  409 {object} errorResponse "location_not_set"
// @Failure  502 {object} errorResponse "weather_unavailable"
// @Router   /api/outfits/candidates [get]
func (api *endpoints) candidates(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	outfitIDs, ok := parseIDs(query.Get("items"))
	var replaceID int64
	if value := query.Get("replace"); ok && value != "" {
		id, err := strconv.ParseInt(value, 10, 64)
		ok = err == nil && id > 0
		replaceID = id
	}
	if !ok {
		writeError(writer, http.StatusBadRequest, "bad_request")
		return
	}

	candidates, err := api.recommender.Candidates(request.Context(), userIDFrom(request.Context()), outfitIDs, replaceID)
	if err != nil {
		writeFailure(writer, err)
		return
	}

	response := candidatesResponse{Candidates: make([]candidateResponse, len(candidates))}
	for index, candidate := range candidates {
		notes := make([]string, len(candidate.Notes))
		for noteIndex, note := range candidate.Notes {
			notes[noteIndex] = texts.CandidateNote(note)
		}
		response.Candidates[index] = candidateResponse{ItemID: candidate.Item.ID, Notes: notes}
	}
	writeJSON(writer, http.StatusOK, response)
}

type reviewResponse struct {
	Notes []string `json:"notes"`
}

// @Summary  Разобрать образ по погоде
// @Description Те же заметки, что пишет подбор: чего не хватает и что не по погоде.
// @Tags     Образ
// @Produce  json
// @Security initData
// @Param    items query string true "Идентификаторы вещей образа через запятую" example(12,15,31)
// @Success  200 {object} reviewResponse
// @Failure  400 {object} errorResponse "bad_request"
// @Failure  401 {object} errorResponse "unauthorized"
// @Failure  409 {object} errorResponse "location_not_set"
// @Failure  502 {object} errorResponse "weather_unavailable"
// @Router   /api/outfits/review [get]
func (api *endpoints) review(writer http.ResponseWriter, request *http.Request) {
	itemIDs, ok := parseIDs(request.URL.Query().Get("items"))
	if !ok {
		writeError(writer, http.StatusBadRequest, "bad_request")
		return
	}

	notes, err := api.recommender.Review(request.Context(), userIDFrom(request.Context()), itemIDs)
	if err != nil {
		writeFailure(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, reviewResponse{Notes: noteTexts(notes)})
}

func parseIDs(value string) ([]int64, bool) {
	if value == "" {
		return []int64{}, true
	}
	parts := strings.Split(value, ",")
	if len(parts) > maxOutfitItems {
		return nil, false
	}
	ids := make([]int64, len(parts))
	for index, part := range parts {
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, false
		}
		ids[index] = id
	}
	return ids, true
}

type citiesResponse struct {
	Cities []cityBody `json:"cities"`
}

// @Summary  Найти город по названию
// @Tags     Город и настройки
// @Produce  json
// @Security initData
// @Param    query query string true "Часть названия, не короче limits.min_city_query из /api/options"
// @Success  200 {object} citiesResponse
// @Failure  401 {object} errorResponse "unauthorized"
// @Router   /api/cities [get]
func (api *endpoints) searchCities(writer http.ResponseWriter, request *http.Request) {
	cities, err := api.locations.SearchCities(request.Context(), request.URL.Query().Get("query"))
	if err != nil {
		writeFailure(writer, err)
		return
	}

	response := citiesResponse{Cities: make([]cityBody, len(cities))}
	for index, city := range cities {
		response.Cities[index] = toCityBody(city)
	}
	writeJSON(writer, http.StatusOK, response)
}

// @Summary  Определить город по координатам
// @Description Координаты округляются до сотых, около километра: погоде точнее не нужно.
// @Tags     Город и настройки
// @Produce  json
// @Security initData
// @Param    latitude query number true "Широта"
// @Param    longitude query number true "Долгота"
// @Success  200 {object} cityBody
// @Failure  400 {object} errorResponse "bad_request"
// @Failure  401 {object} errorResponse "unauthorized"
// @Failure  404 {object} errorResponse "place_not_found"
// @Router   /api/cities/at [get]
func (api *endpoints) cityAt(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	latitude, latitudeErr := strconv.ParseFloat(query.Get("latitude"), 64)
	longitude, longitudeErr := strconv.ParseFloat(query.Get("longitude"), 64)
	if latitudeErr != nil || longitudeErr != nil {
		writeError(writer, http.StatusBadRequest, "bad_request")
		return
	}

	city, err := api.locations.CityAt(request.Context(), latitude, longitude)
	if err != nil {
		writeFailure(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, toCityBody(city))
}

// @Summary  Сохранить город пользователя
// @Tags     Город и настройки
// @Accept   json
// @Produce  json
// @Security initData
// @Param    city body cityBody true "Город"
// @Success  204 "Город сохранён"
// @Failure  400 {object} errorResponse "bad_request"
// @Failure  401 {object} errorResponse "unauthorized"
// @Failure  422 {object} errorResponse "invalid_location"
// @Router   /api/city [put]
func (api *endpoints) setCity(writer http.ResponseWriter, request *http.Request) {
	var body cityBody
	if !decodeBody(writer, request, &body) {
		return
	}

	err := api.locations.SetLocation(request.Context(), userIDFrom(request.Context()), domain.Location{
		Name:      body.Name,
		Region:    body.Region,
		Latitude:  body.Latitude,
		Longitude: body.Longitude,
		TimeZone:  body.TimeZone,
	})
	if err != nil {
		writeFailure(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func toCityBody(location domain.Location) cityBody {
	return cityBody{
		Name:      location.Name,
		Region:    location.Region,
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
		TimeZone:  location.TimeZone,
	}
}

func noteTexts(notes []domain.Note) []string {
	result := make([]string, len(notes))
	for index, note := range notes {
		result[index] = texts.Note(note)
	}
	return result
}
