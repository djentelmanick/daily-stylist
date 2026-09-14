package texts_test

import (
	"strconv"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/texts"
)

func TestLabelsCoverAllDomainValues(t *testing.T) {
	for _, category := range domain.AllCategories() {
		if texts.Category(category) == string(category) {
			t.Errorf("нет подписи для категории %q", category)
		}
	}
	for _, color := range domain.AllColors() {
		if texts.Color(color) == string(color) {
			t.Errorf("нет подписи для цвета %q", color)
		}
	}
	for _, season := range domain.AllSeasons() {
		if texts.Season(season) == string(season) {
			t.Errorf("нет подписи для сезона %q", season)
		}
	}
	for _, status := range domain.AllItemStatuses() {
		if texts.ItemStatus(status) == string(status) {
			t.Errorf("нет подписи для статуса %q", status)
		}
	}
	for _, level := range domain.AllWarmthLevels() {
		if texts.WarmthLevel(level) == strconv.Itoa(int(level)) {
			t.Errorf("нет подписи для уровня теплоты %d", level)
		}
	}
}
