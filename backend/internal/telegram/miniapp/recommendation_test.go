package miniapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain/rules"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var kazan = domain.Location{Name: "Казань", Region: "Татарстан, Россия", Latitude: 55.79, Longitude: 49.12, TimeZone: "Europe/Moscow"}

func TestRecommend_ReturnsOutfitsWithTexts(t *testing.T) {
	shoes := domain.Item{ID: 5, Name: "Кеды", Category: domain.CategoryShoes, Colors: domain.Colors{Main: domain.ColorWhite}, Seasons: []domain.Season{domain.SeasonAutumn}, WarmthLevel: 2, Status: domain.ItemStatusAvailable}
	recommender := &stubRecommender{recommendation: service.Recommendation{
		Location: kazan,
		Weather:  domain.Weather{TemperatureMin: 10, TemperatureMax: 12, FeelsLikeMin: 10, FeelsLikeMax: 12},
		Outfits: []domain.Outfit{{
			Items: []domain.Item{shoes},
			Notes: []domain.Note{{Kind: domain.NoteMissing, Category: domain.CategoryTop}},
		}},
	}}
	handler := NewHandler(testBotToken, failingWardrobe{}, recommender, service.NewLocations(&memoryLocations{}, stubCities{}), nil, &stubPhotos{}, &stubRecognition{})

	response := serve(handler, newRequest(http.MethodGet, "/api/recommendation", "", signedInitData(testBotToken, 42, time.Now())))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusOK, response.Body)
	}
	var body recommendationResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if body.City.Name != "Казань" || body.Weather.Temperature != "+10…+12 °C" || body.Notes == nil {
		t.Errorf("ответ = %+v", body)
	}
	if len(body.Outfits) != 1 || len(body.Outfits[0].Items) != 1 || body.Outfits[0].Items[0].Name != "Кеды" {
		t.Fatalf("образы = %+v", body.Outfits)
	}
	if want := []string{"Нет доступной вещи в категории «Верх»"}; !slices.Equal(body.Outfits[0].Notes, want) {
		t.Errorf("заметки = %q, ожидались %q", body.Outfits[0].Notes, want)
	}
}

func TestRecommend_MapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"город не выбран", fmt.Errorf("подбор образа: %w", service.ErrLocationNotSet), http.StatusConflict, "location_not_set"},
		{"погода недоступна", fmt.Errorf("подбор образа: %w: таймаут", service.ErrWeatherUnavailable), http.StatusBadGateway, "weather_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHandler(testBotToken, failingWardrobe{}, &stubRecommender{err: test.err}, service.NewLocations(&memoryLocations{}, stubCities{}), nil, &stubPhotos{}, &stubRecognition{})

			response := serve(handler, newRequest(http.MethodGet, "/api/recommendation", "", signedInitData(testBotToken, 42, time.Now())))

			checkError(t, response, test.wantStatus, test.wantCode)
		})
	}
}

func TestWearToday(t *testing.T) {
	recommender := &stubRecommender{}
	handler := NewHandler(testBotToken, failingWardrobe{}, recommender, service.NewLocations(&memoryLocations{}, stubCities{}), nil, &stubPhotos{}, &stubRecognition{})
	initData := signedInitData(testBotToken, 42, time.Now())

	response := serve(handler, newRequest(http.MethodPut, "/api/outfits/today", `{"item_ids":[3,1]}`, initData))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusNoContent, response.Body)
	}
	if recommender.wornBy != 42 || !slices.Equal(recommender.wornIDs, []int64{3, 1}) {
		t.Errorf("записан образ %v пользователя %d", recommender.wornIDs, recommender.wornBy)
	}

	response = serve(handler, newRequest(http.MethodPut, "/api/outfits/today", `{"item_ids":[]}`, initData))
	if response.Code != http.StatusNoContent || recommender.wornIDs == nil || len(recommender.wornIDs) != 0 {
		t.Errorf("пустой образ: status = %d, записаны %v", response.Code, recommender.wornIDs)
	}

	response = serve(handler, newRequest(http.MethodPut, "/api/outfits/today", `{}`, initData))
	checkError(t, response, http.StatusBadRequest, "bad_request")
}

func TestCandidates(t *testing.T) {
	recommender := &stubRecommender{candidates: []rules.Candidate{
		{Item: domain.Item{ID: 9}, Notes: []domain.Note{{Kind: domain.NoteWornRecently, DaysAgo: 1}}},
		{Item: domain.Item{ID: 4}},
	}}
	handler := NewHandler(testBotToken, failingWardrobe{}, recommender, service.NewLocations(&memoryLocations{}, stubCities{}), nil, &stubPhotos{}, &stubRecognition{})
	initData := signedInitData(testBotToken, 42, time.Now())

	response := serve(handler, newRequest(http.MethodGet, "/api/outfits/candidates?items=3,5&replace=5", "", initData))
	want := `{"candidates":[{"item_id":9,"notes":["Надевали вчера"]},{"item_id":4,"notes":[]}]}`
	if got := strings.TrimSpace(response.Body.String()); got != want {
		t.Errorf("тело = %s, ожидалось %s", got, want)
	}
	if !slices.Equal(recommender.outfitIDs, []int64{3, 5}) || recommender.replaceID != 5 {
		t.Errorf("спросили образ %v, замена %d", recommender.outfitIDs, recommender.replaceID)
	}

	response = serve(handler, newRequest(http.MethodGet, "/api/outfits/candidates", "", initData))
	if response.Code != http.StatusOK || len(recommender.outfitIDs) != 0 || recommender.replaceID != 0 {
		t.Errorf("пустой образ без замены: status = %d, образ %v, замена %d", response.Code, recommender.outfitIDs, recommender.replaceID)
	}

	for _, query := range []string{"items=1,x", "items=0", "replace=-1", "items=1,,2"} {
		response = serve(handler, newRequest(http.MethodGet, "/api/outfits/candidates?"+query, "", initData))
		checkError(t, response, http.StatusBadRequest, "bad_request")
	}
}

func TestReview(t *testing.T) {
	recommender := &stubRecommender{notes: []domain.Note{{Kind: domain.NoteMissing, Category: domain.CategoryShoes}}}
	handler := NewHandler(testBotToken, failingWardrobe{}, recommender, service.NewLocations(&memoryLocations{}, stubCities{}), nil, &stubPhotos{}, &stubRecognition{})

	response := serve(handler, newRequest(http.MethodGet, "/api/outfits/review?items=2", "", signedInitData(testBotToken, 42, time.Now())))

	if got := strings.TrimSpace(response.Body.String()); got != `{"notes":["Нет доступной вещи в категории «Обувь»"]}` {
		t.Errorf("тело = %s", got)
	}
	if !slices.Equal(recommender.outfitIDs, []int64{2}) {
		t.Errorf("разобран образ %v", recommender.outfitIDs)
	}
}

func TestTodayOutfit(t *testing.T) {
	recommender := &stubRecommender{today: []domain.Item{{ID: 7}, {ID: 3}}}
	handler := NewHandler(testBotToken, failingWardrobe{}, recommender, service.NewLocations(&memoryLocations{}, stubCities{}), nil, &stubPhotos{}, &stubRecognition{})
	initData := signedInitData(testBotToken, 42, time.Now())

	response := serve(handler, newRequest(http.MethodGet, "/api/outfits/today", "", initData))
	if got := strings.TrimSpace(response.Body.String()); got != `{"item_ids":[7,3]}` {
		t.Errorf("тело = %s, ожидались ID в порядке образа", got)
	}

	recommender.today = nil
	response = serve(handler, newRequest(http.MethodGet, "/api/outfits/today", "", initData))
	if got := strings.TrimSpace(response.Body.String()); got != `{"item_ids":[]}` {
		t.Errorf("тело = %s, ожидался пустой список", got)
	}
}

func TestSearchCities(t *testing.T) {
	cities := stubCities{cities: []domain.Location{kazan}}
	handler := NewHandler(testBotToken, failingWardrobe{}, &stubRecommender{}, service.NewLocations(&memoryLocations{}, cities), nil, &stubPhotos{}, &stubRecognition{})
	initData := signedInitData(testBotToken, 42, time.Now())

	response := serve(handler, newRequest(http.MethodGet, "/api/cities?query="+url.QueryEscape("Каз"), "", initData))
	var body citiesResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if len(body.Cities) != 1 || body.Cities[0].Name != "Казань" || body.Cities[0].TimeZone != "Europe/Moscow" {
		t.Errorf("города = %+v", body.Cities)
	}

	response = serve(handler, newRequest(http.MethodGet, "/api/cities?query="+url.QueryEscape("К"), "", initData))
	if got := strings.TrimSpace(response.Body.String()); got != `{"cities":[]}` {
		t.Errorf("тело = %s, ожидался пустой список", got)
	}
}

func TestSetCity(t *testing.T) {
	repository := &memoryLocations{}
	handler := NewHandler(testBotToken, failingWardrobe{}, &stubRecommender{}, service.NewLocations(repository, stubCities{}), nil, &stubPhotos{}, &stubRecognition{})
	initData := signedInitData(testBotToken, 42, time.Now())

	valid := `{"name":"Казань","region":"Татарстан, Россия","latitude":55.79,"longitude":49.12,"timezone":"Europe/Moscow"}`
	response := serve(handler, newRequest(http.MethodPut, "/api/city", valid, initData))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusNoContent, response.Body)
	}
	if repository.saved[42] != kazan {
		t.Errorf("сохранён город %+v", repository.saved[42])
	}

	invalid := `{"name":"Казань","region":"","latitude":55.79,"longitude":49.12,"timezone":"Mars/Olympus"}`
	response = serve(handler, newRequest(http.MethodPut, "/api/city", invalid, initData))
	checkError(t, response, http.StatusUnprocessableEntity, "invalid_location")
}

func newHandler(wardrobe wardrobe) http.Handler {
	return NewHandler(testBotToken, wardrobe, &stubRecommender{err: errors.New("подбор не должен вызываться")}, service.NewLocations(&memoryLocations{}, stubCities{}), nil, &stubPhotos{}, &stubRecognition{})
}

type stubRecommender struct {
	recommendation service.Recommendation
	today          []domain.Item
	err            error
	wornBy         int64
	wornIDs        []int64
	candidates     []rules.Candidate
	notes          []domain.Note
	outfitIDs      []int64
	replaceID      int64
}

func (recommender *stubRecommender) Recommend(context.Context, int64) (service.Recommendation, error) {
	return recommender.recommendation, recommender.err
}

func (recommender *stubRecommender) TodayOutfit(context.Context, int64) ([]domain.Item, error) {
	return recommender.today, recommender.err
}

func (recommender *stubRecommender) WearToday(_ context.Context, userID int64, itemIDs []int64) error {
	recommender.wornBy, recommender.wornIDs = userID, itemIDs
	return recommender.err
}

func (recommender *stubRecommender) Candidates(_ context.Context, _ int64, outfitIDs []int64, replaceID int64) ([]rules.Candidate, error) {
	recommender.outfitIDs, recommender.replaceID = outfitIDs, replaceID
	return recommender.candidates, recommender.err
}

func (recommender *stubRecommender) Review(_ context.Context, _ int64, itemIDs []int64) ([]domain.Note, error) {
	recommender.outfitIDs = itemIDs
	return recommender.notes, recommender.err
}

type stubCities struct {
	cities []domain.Location
}

func (cities stubCities) SearchCities(context.Context, string) ([]domain.Location, error) {
	return cities.cities, nil
}

type memoryLocations struct {
	saved map[int64]domain.Location
}

func (locations *memoryLocations) Get(_ context.Context, userID int64) (domain.Location, error) {
	location, ok := locations.saved[userID]
	if !ok {
		return domain.Location{}, service.ErrLocationNotSet
	}
	return location, nil
}

func (locations *memoryLocations) Save(_ context.Context, userID int64, location domain.Location) error {
	if locations.saved == nil {
		locations.saved = map[int64]domain.Location{}
	}
	locations.saved[userID] = location
	return nil
}
