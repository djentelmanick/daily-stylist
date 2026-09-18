package gigachat

import (
	"strings"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func TestPrompt_ListsWholeVocabulary(t *testing.T) {
	for _, category := range domain.AllCategories() {
		if !strings.Contains(prompt, string(category)) {
			t.Errorf("в запросе нет категории %q", category)
		}
		if categoryHints[category] == "" {
			t.Errorf("у категории %q нет пояснения для модели", category)
		}
	}
	for _, color := range domain.AllColors() {
		if !strings.Contains(prompt, string(color)) {
			t.Errorf("в запросе нет цвета %q", color)
		}
	}
	for _, season := range domain.AllSeasons() {
		if !strings.Contains(prompt, string(season)) {
			t.Errorf("в запросе нет сезона %q", season)
		}
	}
	for _, level := range domain.AllWarmthLevels() {
		if warmthHints[level] == "" {
			t.Errorf("у уровня теплоты %d нет пояснения для модели", level)
		}
	}
}

func TestPrompt_TellsWhereWarmthIsNotAsked(t *testing.T) {
	for _, category := range domain.AllCategories() {
		if category.HasWarmth() {
			continue
		}
		if !strings.Contains(prompt, string(category)+",") && !strings.Contains(prompt, string(category)+" всегда 0") {
			t.Errorf("категория %q без теплоты не перечислена в запросе", category)
		}
	}
}
