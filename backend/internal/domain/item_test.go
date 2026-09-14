package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

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
