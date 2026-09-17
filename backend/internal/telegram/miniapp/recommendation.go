package miniapp

import (
	"net/http"

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

func (api *endpoints) wearToday(writer http.ResponseWriter, request *http.Request) {
	var body wearRequest
	if !decodeBody(writer, request, &body) {
		return
	}
	if len(body.ItemIDs) == 0 || len(body.ItemIDs) > maxOutfitItems {
		writeError(writer, http.StatusBadRequest, "bad_request")
		return
	}

	if err := api.recommender.WearToday(request.Context(), userIDFrom(request.Context()), body.ItemIDs); err != nil {
		writeFailure(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

type citiesResponse struct {
	Cities []cityBody `json:"cities"`
}

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
