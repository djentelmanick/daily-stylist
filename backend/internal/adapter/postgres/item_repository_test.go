//go:build integration

package postgres_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/postgres"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
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

func TestItemRepository_ReadsBackWhatWasCreated(t *testing.T) {
	pool := newTestPool(t)
	repository := postgres.NewItemRepository(pool)

	withExtras := newItem(t, 1)
	withExtras.Colors.Extra = []domain.Color{domain.ColorWhite, domain.ColorGray}
	withExtras.Seasons = []domain.Season{domain.SeasonWinter, domain.SeasonSpring}
	older, err := repository.Create(t.Context(), withExtras)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	newer, err := repository.Create(t.Context(), newItem(t, 1))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repository.Get(t.Context(), 1, older.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, older) {
		t.Errorf("Get = %+v\nожидалась %+v", got, older)
	}

	items, err := repository.ListByUser(t.Context(), 1)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	newer.Colors.Extra = []domain.Color{}
	if want := []domain.Item{newer, older}; !reflect.DeepEqual(items, want) {
		t.Errorf("ListByUser = %+v\nожидались сначала новые: %+v", items, want)
	}
}

func TestItemRepository_HidesOtherUsersItems(t *testing.T) {
	pool := newTestPool(t)
	repository := postgres.NewItemRepository(pool)

	item, err := repository.Create(t.Context(), newItem(t, 1))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	stranger := item
	stranger.UserID = 2

	if _, err := repository.Get(t.Context(), 2, item.ID); !errors.Is(err, service.ErrItemNotFound) {
		t.Errorf("Get чужой вещи: ошибка = %v, ожидалась ErrItemNotFound", err)
	}
	if err := repository.Update(t.Context(), stranger); !errors.Is(err, service.ErrItemNotFound) {
		t.Errorf("Update чужой вещи: ошибка = %v, ожидалась ErrItemNotFound", err)
	}
	if err := repository.UpdateStatus(t.Context(), 2, item.ID, domain.ItemStatusDirty); !errors.Is(err, service.ErrItemNotFound) {
		t.Errorf("UpdateStatus чужой вещи: ошибка = %v, ожидалась ErrItemNotFound", err)
	}
	if err := repository.Delete(t.Context(), 2, []int64{item.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repository.Get(t.Context(), 1, item.ID)
	if err != nil {
		t.Fatalf("Get своей вещи: %v", err)
	}
	if got.Status != domain.ItemStatusAvailable {
		t.Errorf("чужой пользователь поменял статус: %q", got.Status)
	}
}

func TestItemRepository_UpdateStatusAndDelete(t *testing.T) {
	pool := newTestPool(t)
	repository := postgres.NewItemRepository(pool)

	first, err := repository.Create(t.Context(), newItem(t, 1))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	second, err := repository.Create(t.Context(), newItem(t, 1))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repository.UpdateStatus(t.Context(), 1, first.ID, domain.ItemStatusDirty); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	edited, err := first.Edit(domain.EditItemParams{
		Name:     "Зонт",
		Category: domain.CategoryUmbrella,
		Colors:   domain.Colors{Main: domain.ColorBlack},
		Seasons:  []domain.Season{domain.SeasonSpring},
	})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	// У edited старый статус available: Update не должен вернуть его в базу.
	if err := repository.Update(t.Context(), edited); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repository.Get(t.Context(), 1, first.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Зонт" || got.WarmthLevel != 0 || got.Status != domain.ItemStatusDirty {
		t.Errorf("после изменения = %+v", got)
	}

	if err := repository.Delete(t.Context(), 1, []int64{first.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	items, err := repository.ListByUser(t.Context(), 1)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(items) != 1 || items[0].ID != second.ID {
		t.Errorf("после удаления осталось %+v, ожидалась только вещь %d", items, second.ID)
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
