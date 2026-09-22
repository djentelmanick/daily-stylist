package rules

import (
	"cmp"
	"slices"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

const outOfSeasonPenalty = 10

func Review(weather domain.Weather, items []domain.Item) []domain.Note {
	return conditionsFor(weather).review(items)
}

func (c conditions) review(items []domain.Item) []domain.Note {
	withOuterwear := hasCategory(items, domain.CategoryOuterwear)
	umbrella := hasCategory(items, domain.CategoryUmbrella)
	umbrellaHelps := umbrella && !c.windy
	raincoat := slices.ContainsFunc(items, func(item domain.Item) bool {
		return item.Category == domain.CategoryOuterwear && item.Waterproof
	})

	var notes []domain.Note
	for _, item := range items {
		if note, found := c.warmthNote(item, withOuterwear); found {
			notes = append(notes, note)
		}
	}
	if c.needOuterwear && !withOuterwear {
		notes = append(notes, domain.Note{Kind: domain.NoteMissing, Category: domain.CategoryOuterwear})
	}
	if !hasCategory(items, domain.CategoryShoes) {
		notes = append(notes, domain.Note{Kind: domain.NoteMissing, Category: domain.CategoryShoes})
	}
	if c.needOuterwear && withOuterwear && c.warmsUp {
		notes = append(notes, domain.Note{Kind: domain.NoteWarmsUp})
	}
	if c.level >= domain.WarmthLevelHeavy && !hasCategory(items, domain.CategoryHat) {
		notes = append(notes, domain.Note{Kind: domain.NoteMissing, Category: domain.CategoryHat})
	}
	if c.rain() && umbrella && c.windy {
		notes = append(notes, domain.Note{Kind: domain.NoteTooWindyForUmbrella})
	}
	if c.rain() && !umbrellaHelps && !raincoat {
		notes = append(notes, domain.Note{Kind: domain.NoteNoRainProtection})
	}
	return notes
}

func (c conditions) warmthNote(item domain.Item, withOuterwear bool) (domain.Note, bool) {
	if !item.Category.HasWarmth() {
		return domain.Note{}, false
	}
	ideal := c.idealLevel(item, withOuterwear)
	switch {
	case item.WarmthLevel < ideal-1:
		return domain.Note{Kind: domain.NoteTooLight, Item: item}, true
	case item.WarmthLevel > ideal+1:
		return domain.Note{Kind: domain.NoteTooWarm, Item: item}, true
	}
	return domain.Note{}, false
}

type Candidate struct {
	Item  domain.Item
	Notes []domain.Note
}

// Вещи не по сезону не прячутся, а уходят в конец: выбирает человек.
func Replacements(input Input, outfit []domain.Item, replaced domain.Item) []Candidate {
	rest := slices.DeleteFunc(slices.Clone(outfit), func(item domain.Item) bool { return item.ID == replaced.ID })
	return candidates(input, rest, replaced)
}

func Additions(input Input, outfit []domain.Item) []Candidate {
	return candidates(input, outfit, domain.Item{})
}

// Пустая категория у replaced - любая вещь.
func candidates(input Input, outfit []domain.Item, replaced domain.Item) []Candidate {
	c := conditionsFor(input.Weather)
	history := recommender{wornDaysAgo: input.WornDaysAgo}
	colors := mainColors(outfit)
	umbrella := hasCategory(outfit, domain.CategoryUmbrella)

	type scored struct {
		candidate Candidate
		score     int
	}
	var ranked []scored
	for _, item := range input.Items {
		if item.Status != domain.ItemStatusAvailable ||
			item.ID == replaced.ID ||
			(replaced.Category != "" && item.Category != replaced.Category) ||
			slices.ContainsFunc(outfit, func(chosen domain.Item) bool { return chosen.ID == item.ID }) {
			continue
		}

		withOuterwear := hasCategory(outfit, domain.CategoryOuterwear) || item.Category == domain.CategoryOuterwear
		score := warmthStepPenalty*distance(item, c.idealLevel(item, withOuterwear)) +
			colorPenalty(append(slices.Clip(colors), item.Colors.Main))
		var notes []domain.Note
		if note, found := c.warmthNote(item, withOuterwear); found {
			notes = append(notes, note)
		}
		protects := item.Category == domain.CategoryShoes || (item.Category == domain.CategoryOuterwear && !umbrella)
		if c.precipitation && protects && !item.Waterproof {
			score += notWaterproofPenalty
			notes = append(notes, domain.Note{Kind: domain.NoteNotWaterproof})
		}
		if penalty := history.historyPenalty(item); penalty > 0 {
			score += penalty
			notes = append(notes, domain.Note{Kind: domain.NoteWornRecently, DaysAgo: input.WornDaysAgo[item.ID]})
		}
		if !slices.Contains(item.Seasons, input.Season) {
			score += outOfSeasonPenalty
			notes = append(notes, domain.Note{Kind: domain.NoteOutOfSeason})
		}
		ranked = append(ranked, scored{candidate: Candidate{Item: item, Notes: notes}, score: score})
	}

	slices.SortStableFunc(ranked, func(a, b scored) int { return cmp.Compare(a.score, b.score) })
	candidates := make([]Candidate, len(ranked))
	for index, entry := range ranked {
		candidates[index] = entry.candidate
	}
	return candidates
}

func hasCategory(items []domain.Item, category domain.Category) bool {
	return slices.ContainsFunc(items, func(item domain.Item) bool { return item.Category == category })
}
