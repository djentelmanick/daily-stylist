//go:build integration

package postgres_test

import (
	"errors"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/postgres"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func TestLocationRepository_SaveReplacesCity(t *testing.T) {
	pool := newTestPool(t)
	repository := postgres.NewLocationRepository(pool)

	if _, err := repository.Get(t.Context(), 1); !errors.Is(err, service.ErrLocationNotSet) {
		t.Fatalf("Get до выбора города: ошибка = %v, ожидалась ErrLocationNotSet", err)
	}

	moscow := domain.Location{Name: "Москва", Region: "Россия", Latitude: 55.75222, Longitude: 37.61556, TimeZone: "Europe/Moscow"}
	kazan := domain.Location{Name: "Казань", Region: "Татарстан, Россия", Latitude: 55.78874, Longitude: 49.12214, TimeZone: "Europe/Moscow"}
	for _, location := range []domain.Location{moscow, kazan} {
		if err := repository.Save(t.Context(), 1, location); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	got, err := repository.Get(t.Context(), 1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != kazan {
		t.Errorf("Get = %+v, ожидался %+v", got, kazan)
	}
	if _, err := repository.Get(t.Context(), 2); !errors.Is(err, service.ErrLocationNotSet) {
		t.Errorf("город виден другому пользователю: ошибка = %v", err)
	}
}

func TestOutfitRepository_History(t *testing.T) {
	pool := newTestPool(t)
	items := postgres.NewItemRepository(pool)
	outfits := postgres.NewOutfitRepository(pool)

	shirt := createItem(t, items, 1)
	jeans := createItem(t, items, 1)
	dress := createItem(t, items, 1)
	strangers := createItem(t, items, 2)
	monday, tuesday, wednesday := date(2026, 9, 14), date(2026, 9, 15), date(2026, 9, 16)

	save := func(day time.Time, itemIDs ...int64) {
		t.Helper()
		if err := outfits.SaveWorn(t.Context(), 1, day, itemIDs); err != nil {
			t.Fatalf("SaveWorn: %v", err)
		}
	}
	save(monday, shirt.ID, jeans.ID)
	save(tuesday, shirt.ID, strangers.ID)
	// Передумал: во вторник надел платье.
	save(tuesday, dress.ID)

	checkLastWorn(t, outfits, 1, monday, wednesday, map[int64]time.Time{shirt.ID: monday, jeans.ID: monday, dress.ID: tuesday})
	checkLastWorn(t, outfits, 1, tuesday, wednesday, map[int64]time.Time{dress.ID: tuesday})
	checkLastWorn(t, outfits, 2, monday, wednesday, map[int64]time.Time{})
	checkWornOn(t, outfits, 1, monday, shirt.ID, jeans.ID)
	checkWornOn(t, outfits, 1, tuesday, dress.ID)
	checkWornOn(t, outfits, 1, wednesday)
	checkWornOn(t, outfits, 2, tuesday)

	if err := items.Delete(t.Context(), 1, []int64{dress.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	checkLastWorn(t, outfits, 1, monday, wednesday, map[int64]time.Time{shirt.ID: monday, jeans.ID: monday})
	checkWornOn(t, outfits, 1, tuesday)
}

func checkWornOn(t *testing.T, outfits *postgres.OutfitRepository, userID int64, day time.Time, wantIDs ...int64) {
	t.Helper()

	items, err := outfits.WornOn(t.Context(), userID, day)
	if err != nil {
		t.Fatalf("WornOn: %v", err)
	}
	var gotIDs []int64
	for _, item := range items {
		gotIDs = append(gotIDs, item.ID)
	}
	slices.Sort(gotIDs)
	slices.Sort(wantIDs)
	if !slices.Equal(gotIDs, wantIDs) {
		t.Errorf("WornOn(%d, %s) = %v, ожидались %v", userID, day.Format(time.DateOnly), gotIDs, wantIDs)
	}
}

func checkLastWorn(t *testing.T, outfits *postgres.OutfitRepository, userID int64, from, to time.Time, want map[int64]time.Time) {
	t.Helper()

	got, err := outfits.LastWorn(t.Context(), userID, from, to)
	if err != nil {
		t.Fatalf("LastWorn: %v", err)
	}
	if !maps.EqualFunc(got, want, time.Time.Equal) {
		t.Errorf("LastWorn(%d, %s, %s) = %v, ожидалось %v", userID, from.Format(time.DateOnly), to.Format(time.DateOnly), got, want)
	}
}

func createItem(t *testing.T, items *postgres.ItemRepository, userID int64) domain.Item {
	t.Helper()

	item, err := items.Create(t.Context(), newItem(t, userID))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return item
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
