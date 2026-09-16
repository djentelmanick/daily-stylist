package service_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var vladivostok = domain.Location{Name: "Владивосток", Latitude: 43.1, Longitude: 131.9, TimeZone: "Asia/Vladivostok"}

var utcNow = time.Date(2026, 9, 15, 23, 30, 0, 0, time.UTC)

func TestRecommender_CountsDayInUserTimeZone(t *testing.T) {
	forecaster := &fakeForecaster{weather: domain.Weather{FeelsLikeMin: 25, FeelsLikeMax: 25}}
	yesterday := date(2026, 9, 15)
	outfits := &fakeOutfits{lastWorn: map[int64]time.Time{1: yesterday}}
	items := fakeItems{items: []domain.Item{
		testItem(1, "Вчерашняя футболка", domain.CategoryTop),
		testItem(2, "Футболка", domain.CategoryTop),
		testItem(3, "Шорты", domain.CategoryBottom),
	}}
	recommender := service.NewRecommender(items, outfits, fakeLocations{location: vladivostok}, forecaster, fixedNow)

	recommendation, err := recommender.Recommend(t.Context(), 42)
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}

	zone := vladivostok.Zone()
	if want := time.Date(2026, 9, 16, 9, 0, 0, 0, zone); !forecaster.from.Equal(want) {
		t.Errorf("прогноз с %v, ожидалось с %v", forecaster.from, want)
	}
	if want := time.Date(2026, 9, 16, 22, 0, 0, 0, zone); !forecaster.to.Equal(want) {
		t.Errorf("прогноз до %v, ожидалось до %v", forecaster.to, want)
	}
	if !outfits.from.Equal(date(2026, 9, 9)) || !outfits.to.Equal(date(2026, 9, 16)) {
		t.Errorf("история с %v до %v, ожидалась неделя до 16 сентября", outfits.from, outfits.to)
	}
	if len(recommendation.Outfits) != 2 || recommendation.Outfits[0].Items[0].Name != "Футболка" {
		t.Errorf("образы = %+v, первым ожидался образ без вчерашней футболки", recommendation.Outfits)
	}
}

func TestRecommender_RequiresLocation(t *testing.T) {
	forecaster := &fakeForecaster{}
	recommender := service.NewRecommender(fakeItems{}, &fakeOutfits{}, fakeLocations{err: service.ErrLocationNotSet}, forecaster, fixedNow)

	_, err := recommender.Recommend(t.Context(), 42)

	if !errors.Is(err, service.ErrLocationNotSet) {
		t.Errorf("ошибка = %v, ожидалась ErrLocationNotSet", err)
	}
	if forecaster.called {
		t.Error("без города погоду запрашивать не за чем")
	}
}

func TestRecommender_WeatherFailure(t *testing.T) {
	forecaster := &fakeForecaster{err: errors.New("таймаут")}
	recommender := service.NewRecommender(fakeItems{}, &fakeOutfits{}, fakeLocations{location: vladivostok}, forecaster, fixedNow)

	_, err := recommender.Recommend(t.Context(), 42)

	if !errors.Is(err, service.ErrWeatherUnavailable) {
		t.Errorf("ошибка = %v, ожидалась ErrWeatherUnavailable", err)
	}
}

func TestRecommender_WearTodaySavesLocalDate(t *testing.T) {
	outfits := &fakeOutfits{}
	recommender := service.NewRecommender(fakeItems{}, outfits, fakeLocations{location: vladivostok}, &fakeForecaster{}, fixedNow)

	if err := recommender.WearToday(t.Context(), 42, []int64{1, 2}); err != nil {
		t.Fatalf("WearToday: %v", err)
	}

	if !outfits.savedDay.Equal(date(2026, 9, 16)) {
		t.Errorf("образ записан на %v, ожидалось 16 сентября", outfits.savedDay)
	}
}

func TestRecommender_TodayOutfit(t *testing.T) {
	outfits := &fakeOutfits{worn: []domain.Item{
		testItem(3, "Кеды", domain.CategoryShoes),
		testItem(1, "Футболка", domain.CategoryTop),
		testItem(2, "Куртка", domain.CategoryOuterwear),
	}}
	recommender := service.NewRecommender(fakeItems{}, outfits, fakeLocations{location: vladivostok}, &fakeForecaster{}, fixedNow)

	items, err := recommender.TodayOutfit(t.Context(), 42)
	if err != nil {
		t.Fatalf("TodayOutfit: %v", err)
	}

	if !outfits.wornDay.Equal(date(2026, 9, 16)) {
		t.Errorf("образ прочитан за %v, ожидалось 16 сентября", outfits.wornDay)
	}
	var names []string
	for _, item := range items {
		names = append(names, item.Name)
	}
	if want := []string{"Куртка", "Футболка", "Кеды"}; !slices.Equal(names, want) {
		t.Errorf("образ дня = %q, ожидался %q: сверху вниз", names, want)
	}
}

func TestRecommender_TodayOutfitWithoutLocation(t *testing.T) {
	recommender := service.NewRecommender(fakeItems{}, &fakeOutfits{}, fakeLocations{err: service.ErrLocationNotSet}, &fakeForecaster{}, fixedNow)

	items, err := recommender.TodayOutfit(t.Context(), 42)

	if err != nil || len(items) != 0 {
		t.Errorf("без города: образ %+v, ошибка %v; ожидалось пусто без ошибки", items, err)
	}
}

func fixedNow() time.Time {
	return utcNow
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func testItem(id int64, name string, category domain.Category) domain.Item {
	return domain.Item{
		ID:          id,
		UserID:      42,
		Name:        name,
		Category:    category,
		Colors:      domain.Colors{Main: domain.ColorWhite},
		Seasons:     domain.AllSeasons(),
		WarmthLevel: domain.WarmthLevelLight,
		Status:      domain.ItemStatusAvailable,
	}
}

// Подбору нужен только список вещей, остальные методы не вызываются.
type fakeItems struct {
	service.ItemRepository
	items []domain.Item
}

func (items fakeItems) ListByUser(context.Context, int64) ([]domain.Item, error) {
	return items.items, nil
}

type fakeLocations struct {
	location domain.Location
	err      error
}

func (locations fakeLocations) Get(context.Context, int64) (domain.Location, error) {
	return locations.location, locations.err
}

func (locations fakeLocations) Save(context.Context, int64, domain.Location) error {
	return locations.err
}

type fakeForecaster struct {
	weather  domain.Weather
	err      error
	called   bool
	from, to time.Time
}

func (forecaster *fakeForecaster) Forecast(_ context.Context, _ domain.Location, from, to time.Time) (domain.Weather, error) {
	forecaster.called, forecaster.from, forecaster.to = true, from, to
	return forecaster.weather, forecaster.err
}

type fakeOutfits struct {
	lastWorn map[int64]time.Time
	from, to time.Time
	savedDay time.Time
	worn     []domain.Item
	wornDay  time.Time
}

func (outfits *fakeOutfits) WornOn(_ context.Context, _ int64, day time.Time) ([]domain.Item, error) {
	outfits.wornDay = day
	return outfits.worn, nil
}

func (outfits *fakeOutfits) SaveWorn(_ context.Context, _ int64, day time.Time, _ []int64) error {
	outfits.savedDay = day
	return nil
}

func (outfits *fakeOutfits) LastWorn(_ context.Context, _ int64, from, to time.Time) (map[int64]time.Time, error) {
	outfits.from, outfits.to = from, to
	return outfits.lastWorn, nil
}
