//go:build integration

package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/postgres"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var vladivostok = domain.Location{Name: "Владивосток", Region: "Россия", Latitude: 43.11, Longitude: 131.87, TimeZone: "Asia/Vladivostok"}

func TestSettingsRepository_DefaultsUntilSaved(t *testing.T) {
	pool := newTestPool(t)
	repository := postgres.NewSettingsRepository(pool)

	if _, err := repository.Get(t.Context(), 1); !errors.Is(err, service.ErrSettingsNotSet) {
		t.Fatalf("Get до сохранения: ошибка = %v, ожидалась ErrSettingsNotSet", err)
	}

	evening := domain.Settings{MorningEnabled: false, SendAt: domain.DayTime{Hour: 21, Minute: 45}}
	if err := repository.Save(t.Context(), 1, evening); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got, err := repository.Get(t.Context(), 1); err != nil || got != evening {
		t.Errorf("Get = %+v (%v), ожидались %+v", got, err, evening)
	}

	if _, err := repository.Get(t.Context(), 2); !errors.Is(err, service.ErrSettingsNotSet) {
		t.Errorf("чужие настройки видны: ошибка = %v, ожидалась ErrSettingsNotSet", err)
	}
}

// Семь утра во Владивостоке - это 21:00 предыдущего дня по UTC.
func TestDeliveryRepository_DueCountsTimeInUserZone(t *testing.T) {
	pool := newTestPool(t)
	locations := postgres.NewLocationRepository(pool)
	deliveries := postgres.NewDeliveryRepository(pool)
	if err := locations.Save(t.Context(), 1, vladivostok); err != nil {
		t.Fatalf("Save: %v", err)
	}

	early := due(t, deliveries, time.Date(2026, 9, 15, 20, 30, 0, 0, time.UTC))
	if len(early) != 0 {
		t.Errorf("в 6:30 по местному времени рассылка = %+v, ожидалось пусто", early)
	}

	got := due(t, deliveries, time.Date(2026, 9, 15, 21, 0, 0, 0, time.UTC))
	if len(got) != 1 || got[0].UserID != 1 {
		t.Fatalf("рассылка = %+v, ожидался пользователь 1", got)
	}
	if want := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC); !got[0].Day.Equal(want) {
		t.Errorf("день = %v, ожидался местный день %v", got[0].Day, want)
	}
}

func TestDeliveryRepository_DueOnlyWithinWindow(t *testing.T) {
	pool := newTestPool(t)
	locations := postgres.NewLocationRepository(pool)
	deliveries := postgres.NewDeliveryRepository(pool)
	if err := locations.Save(t.Context(), 1, vladivostok); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if got := due(t, deliveries, time.Date(2026, 9, 15, 21, 59, 0, 0, time.UTC)); len(got) != 1 {
		t.Errorf("через 59 минут после 7:00 рассылка = %+v, ожидался пользователь 1", got)
	}
	if got := due(t, deliveries, time.Date(2026, 9, 15, 22, 1, 0, 0, time.UTC)); len(got) != 0 {
		t.Errorf("через час с минутами рассылка = %+v, ожидалось пусто", got)
	}
}

func TestDeliveryRepository_DueRespectsSettings(t *testing.T) {
	pool := newTestPool(t)
	locations := postgres.NewLocationRepository(pool)
	settings := postgres.NewSettingsRepository(pool)
	deliveries := postgres.NewDeliveryRepository(pool)
	for _, userID := range []int64{1, 2} {
		if err := locations.Save(t.Context(), userID, vladivostok); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	if err := settings.Save(t.Context(), 1, domain.Settings{MorningEnabled: false, SendAt: domain.DefaultSendAt}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := settings.Save(t.Context(), 2, domain.Settings{MorningEnabled: true, SendAt: domain.DayTime{Hour: 9}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if got := due(t, deliveries, time.Date(2026, 9, 15, 21, 0, 0, 0, time.UTC)); len(got) != 0 {
		t.Errorf("в 7:00 рассылка = %+v, ожидалось пусто: у одного она выключена, у другого время своё", got)
	}
	got := due(t, deliveries, time.Date(2026, 9, 15, 23, 0, 0, 0, time.UTC))
	if len(got) != 1 || got[0].UserID != 2 {
		t.Errorf("в 9:00 рассылка = %+v, ожидался пользователь 2", got)
	}
}

func TestDeliveryRepository_ClaimTakesDayOnce(t *testing.T) {
	pool := newTestPool(t)
	locations := postgres.NewLocationRepository(pool)
	deliveries := postgres.NewDeliveryRepository(pool)
	if err := locations.Save(t.Context(), 1, vladivostok); err != nil {
		t.Fatalf("Save: %v", err)
	}
	now := time.Date(2026, 9, 15, 21, 0, 0, 0, time.UTC)
	delivery := due(t, deliveries, now)[0]

	claimed, err := deliveries.Claim(t.Context(), delivery)
	if err != nil || !claimed {
		t.Fatalf("Claim = %v (%v), ожидалось true", claimed, err)
	}
	if claimed, err = deliveries.Claim(t.Context(), delivery); err != nil || claimed {
		t.Errorf("повторный Claim = %v (%v), ожидалось false", claimed, err)
	}
	if got := due(t, deliveries, now); len(got) != 0 {
		t.Errorf("занятый день снова в рассылке: %+v", got)
	}

	if err := deliveries.Release(t.Context(), delivery); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if got := due(t, deliveries, now); len(got) != 1 {
		t.Errorf("после Release рассылка = %+v, ожидался пользователь 1", got)
	}
}

func due(t *testing.T, repository *postgres.DeliveryRepository, now time.Time) []service.MorningDelivery {
	t.Helper()

	got, err := repository.Due(t.Context(), service.DueParams{
		Now:      now,
		Window:   service.MorningWindow,
		Defaults: domain.DefaultSettings(),
	})
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	return got
}

func TestDeliveryRepository_ClaimRemovesOldMarks(t *testing.T) {
	pool := newTestPool(t)
	locations := postgres.NewLocationRepository(pool)
	deliveries := postgres.NewDeliveryRepository(pool)
	if err := locations.Save(t.Context(), 1, vladivostok); err != nil {
		t.Fatalf("Save: %v", err)
	}

	week := time.Date(2026, 9, 15, 21, 0, 0, 0, time.UTC)
	for day := range 7 {
		delivery := due(t, deliveries, week.AddDate(0, 0, day))[0]
		if _, err := deliveries.Claim(t.Context(), delivery); err != nil {
			t.Fatalf("Claim: %v", err)
		}
	}

	var marks int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM morning_deliveries WHERE user_id = 1`).Scan(&marks); err != nil {
		t.Fatalf("подсчёт отметок: %v", err)
	}
	if marks != 3 {
		t.Errorf("отметок за неделю осталось %d, ожидалось 3: сегодняшняя и пара дней в запасе", marks)
	}
}
