//go:build integration

package postgres_test

import (
	"slices"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/postgres"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func TestItemRepository_CreateAndCount(t *testing.T) {
	pool := newTestPool(t)
	repository := postgres.NewItemRepository(pool)

	var previousID int64
	for _, userID := range []int64{1, 2, 1} {
		created, err := repository.Create(t.Context(), newItem(t, userID))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if created.ID <= previousID {
			t.Errorf("ID = %d, ожидался больше %d", created.ID, previousID)
		}
		previousID = created.ID
	}

	for userID, wantCount := range map[int64]int{1: 2, 2: 1, 3: 0} {
		count, err := repository.CountByUser(t.Context(), userID)
		if err != nil {
			t.Fatalf("CountByUser(%d): %v", userID, err)
		}
		if count != wantCount {
			t.Errorf("CountByUser(%d) = %d, ожидалось %d", userID, count, wantCount)
		}
	}
}

func TestItemRepository_CreateStoresEveryField(t *testing.T) {
	pool := newTestPool(t)
	repository := postgres.NewItemRepository(pool)

	item, err := domain.NewItem(domain.NewItemParams{
		UserID:      7,
		Name:        "Дождевик",
		Description: "жёлтый, с капюшоном",
		Category:    domain.CategoryOuterwear,
		Colors:      domain.Colors{Main: domain.ColorYellow, Extra: []domain.Color{domain.ColorBlack, domain.ColorWhite}},
		Seasons:     []domain.Season{domain.SeasonSpring, domain.SeasonAutumn},
		WarmthLevel: domain.WarmthLevelMedium,
		Waterproof:  true,
	})
	if err != nil {
		t.Fatalf("NewItem: %v", err)
	}

	created, err := repository.Create(t.Context(), item)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var stored struct {
		userID      int64
		name        string
		description string
		category    string
		mainColor   string
		extraColors []string
		seasons     []string
		warmthLevel int
		waterproof  bool
		status      string
	}
	err = pool.QueryRow(t.Context(), `
		SELECT user_id, name, description, category, main_color, extra_colors, seasons, warmth_level, waterproof, status
		FROM items WHERE id = $1`, created.ID,
	).Scan(&stored.userID, &stored.name, &stored.description, &stored.category, &stored.mainColor,
		&stored.extraColors, &stored.seasons, &stored.warmthLevel, &stored.waterproof, &stored.status)
	if err != nil {
		t.Fatalf("чтение строки: %v", err)
	}

	if stored.userID != 7 ||
		stored.name != "Дождевик" ||
		stored.description != "жёлтый, с капюшоном" ||
		stored.category != "outerwear" ||
		stored.mainColor != "yellow" ||
		!slices.Equal(stored.extraColors, []string{"black", "white"}) ||
		!slices.Equal(stored.seasons, []string{"spring", "autumn"}) ||
		stored.warmthLevel != 2 ||
		!stored.waterproof ||
		stored.status != "available" {
		t.Errorf("в базе = %+v", stored)
	}
}

func newItem(t *testing.T, userID int64) domain.Item {
	t.Helper()

	item, err := domain.NewItem(domain.NewItemParams{
		UserID:      userID,
		Name:        "Синее худи",
		Category:    domain.CategoryTop,
		Colors:      domain.Colors{Main: domain.ColorBlue},
		Seasons:     []domain.Season{domain.SeasonAutumn},
		WarmthLevel: domain.WarmthLevelMedium,
	})
	if err != nil {
		t.Fatalf("NewItem: %v", err)
	}
	return item
}
