package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func TestSeasonAt_MonthBoundaries(t *testing.T) {
	tests := []struct {
		date string
		want domain.Season
	}{
		{"2026-02-28", domain.SeasonWinter},
		{"2026-03-01", domain.SeasonSpring},
		{"2026-05-31", domain.SeasonSpring},
		{"2026-06-01", domain.SeasonSummer},
		{"2026-08-31", domain.SeasonSummer},
		{"2026-09-01", domain.SeasonAutumn},
		{"2026-11-30", domain.SeasonAutumn},
		{"2026-12-01", domain.SeasonWinter},
		{"2027-01-01", domain.SeasonWinter},
	}
	for _, test := range tests {
		date, err := time.Parse(time.DateOnly, test.date)
		if err != nil {
			t.Fatalf("разбор даты %s: %v", test.date, err)
		}
		if got := domain.SeasonAt(date); got != test.want {
			t.Errorf("SeasonAt(%s) = %q, ожидался %q", test.date, got, test.want)
		}
	}
}

func TestItem_Edit(t *testing.T) {
	item, err := domain.NewItem(domain.NewItemParams{
		UserID:      1,
		Name:        "Худи",
		Category:    domain.CategoryTop,
		Colors:      domain.Colors{Main: domain.ColorBlue},
		Seasons:     []domain.Season{domain.SeasonAutumn},
		WarmthLevel: domain.WarmthLevelMedium,
	})
	if err != nil {
		t.Fatalf("NewItem: %v", err)
	}
	item.ID = 5
	item.Status = domain.ItemStatusDirty

	edited, err := item.Edit(domain.EditItemParams{
		Name:     "  Зонт  ",
		Category: domain.CategoryUmbrella,
		Colors:   domain.Colors{Main: domain.ColorBlack},
		Seasons:  []domain.Season{domain.SeasonSpring},
	})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if edited.ID != 5 || edited.UserID != 1 || edited.Status != domain.ItemStatusDirty {
		t.Errorf("Edit поменял ID, владельца или статус: %+v", edited)
	}
	if edited.Name != "Зонт" || edited.Category != domain.CategoryUmbrella {
		t.Errorf("Edit не применил поля: %+v", edited)
	}

	_, err = item.Edit(domain.EditItemParams{Name: "Зонт", Category: domain.CategoryUmbrella})
	if !errors.Is(err, domain.ErrInvalidItem) {
		t.Errorf("вещь без цвета и сезонов: ошибка = %v, ожидалась ErrInvalidItem", err)
	}
}

// Лимиты заполняются буквой «я»: она занимает два байта, поэтому тест упадёт,
// если длину начнут считать в байтах, а не в символах.
func TestNewItem_TextLimits(t *testing.T) {
	longestName := strings.Repeat("я", domain.MaxNameLength)
	longestDescription := strings.Repeat("я", domain.MaxDescriptionLength)

	tests := []struct {
		name        string
		itemName    string
		description string
		wantErr     bool
	}{
		{"название ровно на лимите", longestName, "", false},
		{"название длиннее лимита", longestName + "я", "", true},
		{"пробелы по краям названия не считаются", "  " + longestName + "  ", "", false},
		{"пустое название", "   ", "", true},
		{"описание ровно на лимите", "Худи", longestDescription, false},
		{"описание длиннее лимита", "Худи", longestDescription + "я", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := domain.NewItem(domain.NewItemParams{
				UserID:      1,
				Name:        test.itemName,
				Description: test.description,
				Category:    domain.CategoryTop,
				Colors:      domain.Colors{Main: domain.ColorBlue},
				Seasons:     []domain.Season{domain.SeasonAutumn},
				WarmthLevel: domain.WarmthLevelMedium,
			})

			if gotErr := err != nil; gotErr != test.wantErr {
				t.Fatalf("ошибка = %v, ожидалась ошибка: %t", err, test.wantErr)
			}
			if err != nil && !errors.Is(err, domain.ErrInvalidItem) {
				t.Errorf("ошибка = %v, ожидалась ErrInvalidItem", err)
			}
		})
	}
}
