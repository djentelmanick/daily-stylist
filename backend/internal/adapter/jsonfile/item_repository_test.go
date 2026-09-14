package jsonfile_test

import (
	"path/filepath"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/jsonfile"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func TestItemRepository_CreateAndCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "items.json")
	repository := jsonfile.NewItemRepository(path)

	for index, userID := range []int64{1, 2, 1} {
		created, err := repository.Create(t.Context(), newItem(t, userID))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if wantID := int64(index + 1); created.ID != wantID {
			t.Errorf("ID = %d, ожидался %d", created.ID, wantID)
		}
	}

	reopened := jsonfile.NewItemRepository(path)
	for userID, wantCount := range map[int64]int{1: 2, 2: 1, 3: 0} {
		count, err := reopened.CountByUser(t.Context(), userID)
		if err != nil {
			t.Fatalf("CountByUser(%d): %v", userID, err)
		}
		if count != wantCount {
			t.Errorf("CountByUser(%d) = %d, ожидалось %d", userID, count, wantCount)
		}
	}
}

func TestItemRepository_CountWithoutFile(t *testing.T) {
	repository := jsonfile.NewItemRepository(filepath.Join(t.TempDir(), "items.json"))

	count, err := repository.CountByUser(t.Context(), 1)
	if err != nil {
		t.Fatalf("CountByUser: %v", err)
	}
	if count != 0 {
		t.Errorf("CountByUser = %d, ожидалось 0", count)
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
