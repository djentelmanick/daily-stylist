package rules

import (
	"cmp"
	"slices"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

const (
	HistoryDays = 7
	MaxOutfits  = 5
)

const (
	warmthStepPenalty    = 3
	notWaterproofPenalty = 2
)

type Input struct {
	Items       []domain.Item
	Weather     domain.Weather
	Season      domain.Season
	WornDaysAgo map[int64]int
}

// Recommend возвращает до MaxOutfits образов, лучший первым. Если собрать
// нечего, образов нет, а заметки говорят, чего не хватает.
func Recommend(input Input) ([]domain.Outfit, []domain.Note) {
	recommender := recommender{
		conditions:  conditionsFor(input.Weather),
		wardrobe:    wearable(input.Items, input.Season),
		wornDaysAgo: input.WornDaysAgo,
	}

	looks := recommender.looks()
	if len(looks) == 0 {
		return nil, recommender.missingBase()
	}
	outfits := make([]domain.Outfit, 0, min(len(looks), MaxOutfits))
	for _, look := range looks[:min(len(looks), MaxOutfits)] {
		outfits = append(outfits, recommender.outfit(look))
	}
	return outfits, nil
}

type byCategory map[domain.Category][]domain.Item

func wearable(items []domain.Item, season domain.Season) byCategory {
	result := byCategory{}
	for _, item := range items {
		if item.Status == domain.ItemStatusAvailable && slices.Contains(item.Seasons, season) {
			result[item.Category] = append(result[item.Category], item)
		}
	}
	return result
}

type recommender struct {
	conditions  conditions
	wardrobe    byCategory
	wornDaysAgo map[int64]int
}

type rainPlan int

const (
	rainNothing rainPlan = iota
	rainUmbrella
	rainCoat
)

// look - основа образа: одежда, от которой зависят теплота и цвета.
// Аксессуары подбираются к ней потом.
type look struct {
	// Верх и низ либо платье или комбинезон.
	base      []domain.Item
	outerwear *domain.Item
	shoes     *domain.Item
	rain      rainPlan
	score     int
}

func (look look) items() []domain.Item {
	items := slices.Clone(look.base)
	for _, item := range []*domain.Item{look.outerwear, look.shoes} {
		if item != nil {
			items = append(items, *item)
		}
	}
	return items
}

// looks перебирает сочетания и оставляет лучшее для каждой основы: варианты
// должны отличаться одеждой и обувью. В дождь основа удваивается - с зонтом и с
// дождевиком, - поэтому способ защиты входит в ключ.
func (recommender recommender) looks() []look {
	type baseKey struct {
		top, bottom int64
		rain        rainPlan
	}
	var order []baseKey
	best := map[baseKey]look{}

	shoes := options(closest(recommender.wardrobe[domain.CategoryShoes], recommender.conditions.level))
	for _, plan := range recommender.rainPlans() {
		for _, outerwear := range recommender.outerwearOptions(plan) {
			for _, base := range recommender.bases(outerwear != nil) {
				for _, pair := range shoes {
					candidate := look{base: base, outerwear: outerwear, shoes: pair, rain: plan}
					candidate.score = recommender.score(candidate)

					key := baseKey{top: base[0].ID, rain: plan}
					if len(base) > 1 {
						key.bottom = base[1].ID
					}
					current, seen := best[key]
					if !seen {
						order = append(order, key)
					}
					if !seen || candidate.score < current.score {
						best[key] = candidate
					}
				}
			}
		}
	}

	looks := make([]look, len(order))
	for index, key := range order {
		looks[index] = best[key]
	}
	slices.SortStableFunc(looks, func(a, b look) int { return cmp.Compare(a.score, b.score) })
	return looks
}

func (recommender recommender) bases(withOuterwear bool) [][]domain.Item {
	conditions, wardrobe := recommender.conditions, recommender.wardrobe
	topLevel := conditions.idealLevel(domain.Item{Category: domain.CategoryTop}, withOuterwear)
	bottomLevel := conditions.idealLevel(domain.Item{Category: domain.CategoryBottom}, withOuterwear)

	var bases [][]domain.Item
	for _, top := range closest(wardrobe[domain.CategoryTop], topLevel) {
		for _, bottom := range closest(wardrobe[domain.CategoryBottom], bottomLevel) {
			bases = append(bases, []domain.Item{top, bottom})
		}
	}
	for _, one := range closest(slices.Concat(wardrobe[domain.CategoryDress], wardrobe[domain.CategoryJumpsuit]), topLevel) {
		bases = append(bases, []domain.Item{one})
	}
	return bases
}

func (recommender recommender) outerwearOptions(plan rainPlan) []*domain.Item {
	conditions := recommender.conditions

	if plan == rainCoat {
		raincoats := recommender.raincoats()
		if fitting := near(raincoats, conditions.level); len(fitting) > 0 {
			return options(fitting)
		}
		return options(closest(raincoats, conditions.level))
	}
	if conditions.needOuterwear {
		return options(closest(recommender.wardrobe[domain.CategoryOuterwear], conditions.level))
	}
	return []*domain.Item{nil}
}

func (recommender recommender) rainPlans() []rainPlan {
	if !recommender.conditions.precipitation {
		return []rainPlan{rainNothing}
	}

	var plans []rainPlan
	if recommender.umbrellaHelps() {
		plans = append(plans, rainUmbrella)
	}
	if len(recommender.raincoats()) > 0 {
		plans = append(plans, rainCoat)
	}
	if len(plans) == 0 {
		return []rainPlan{rainNothing}
	}
	return plans
}

func (recommender recommender) raincoats() []domain.Item {
	coats := slices.Clone(recommender.wardrobe[domain.CategoryOuterwear])
	return slices.DeleteFunc(coats, func(coat domain.Item) bool { return !coat.Waterproof })
}

func (recommender recommender) umbrellaHelps() bool {
	conditions := recommender.conditions
	return conditions.rain() && !conditions.windy && len(recommender.wardrobe[domain.CategoryUmbrella]) > 0
}

func (recommender recommender) score(look look) int {
	items := look.items()
	score := colorPenalty(mainColors(items))
	for _, item := range items {
		score += warmthStepPenalty * distance(item, recommender.conditions.idealLevel(item, look.outerwear != nil))
		score += recommender.historyPenalty(item)
		// Под зонтом непромокаемая куртка не нужна, а вот обувь он не прикрывает.
		protects := item.Category == domain.CategoryShoes ||
			(item.Category == domain.CategoryOuterwear && look.rain != rainUmbrella)
		if recommender.conditions.precipitation && protects && !item.Waterproof {
			score += notWaterproofPenalty
		}
	}
	return score
}

// historyPenalty: чем недавнее надевали, тем больше штраф. Не запрет: при
// маленьком гардеробе повтор лучше, чем никакого образа. Обувь, верхнюю одежду
// и аксессуары носят каждый день, поэтому история считается только для одежды.
func (recommender recommender) historyPenalty(item domain.Item) int {
	switch item.Category {
	case domain.CategoryTop, domain.CategoryBottom, domain.CategoryDress, domain.CategoryJumpsuit:
	default:
		return 0
	}
	days, worn := recommender.wornDaysAgo[item.ID]
	if !worn || days < 1 || days > HistoryDays {
		return 0
	}
	return HistoryDays + 1 - days
}

func (recommender recommender) outfit(look look) domain.Outfit {
	conditions, wardrobe := recommender.conditions, recommender.wardrobe
	items := look.items()
	withOuterwear := look.outerwear != nil
	var notes []domain.Note

	for _, item := range items {
		ideal := conditions.idealLevel(item, withOuterwear)
		switch {
		case item.WarmthLevel < ideal-1:
			notes = append(notes, domain.Note{Kind: domain.NoteTooLight, Item: item})
		case item.WarmthLevel > ideal+1:
			notes = append(notes, domain.Note{Kind: domain.NoteTooWarm, Item: item})
		}
	}
	if conditions.needOuterwear && !withOuterwear {
		notes = append(notes, domain.Note{Kind: domain.NoteMissing, Category: domain.CategoryOuterwear})
	}
	if look.shoes == nil {
		notes = append(notes, domain.Note{Kind: domain.NoteMissing, Category: domain.CategoryShoes})
	}
	if conditions.needOuterwear && withOuterwear && conditions.warmsUp {
		notes = append(notes, domain.Note{Kind: domain.NoteWarmsUp})
	}

	colors := mainColors(items)
	add := func(candidates []domain.Item, ideal domain.WarmthLevel) bool {
		item, found := pick(candidates, ideal, colors)
		if found {
			items = append(items, item)
			colors = append(colors, item.Colors.Main)
		}
		return found
	}

	level := conditions.level
	if level == domain.WarmthLevelExtreme {
		add(near(wardrobe[domain.CategoryThermal], level), level)
	}
	if level >= domain.WarmthLevelMedium {
		add(near(wardrobe[domain.CategorySocks], level), level)
	}
	if level >= domain.WarmthLevelHeavy {
		if !add(near(wardrobe[domain.CategoryHat], level), level) {
			notes = append(notes, domain.Note{Kind: domain.NoteMissing, Category: domain.CategoryHat})
		}
		add(near(wardrobe[domain.CategoryScarf], level), level)
	} else if conditions.uvIndex >= strongUVIndex && !conditions.precipitation {
		add(near(wardrobe[domain.CategoryHat], domain.WarmthLevelLight), domain.WarmthLevelLight)
	}

	umbrella := false
	if look.rain == rainUmbrella {
		umbrella = add(wardrobe[domain.CategoryUmbrella], level)
	} else if conditions.rain() && conditions.windy && len(wardrobe[domain.CategoryUmbrella]) > 0 {
		notes = append(notes, domain.Note{Kind: domain.NoteTooWindyForUmbrella})
	}
	raincoat := withOuterwear && look.outerwear.Waterproof
	if conditions.rain() && !umbrella && !raincoat {
		notes = append(notes, domain.Note{Kind: domain.NoteNoRainProtection})
	}
	if conditions.sunny() {
		add(wardrobe[domain.CategorySunglasses], level)
	}
	add(wardrobe[domain.CategoryBag], level)

	domain.SortForWearing(items)
	return domain.Outfit{Items: items, Notes: notes}
}

// pick выбирает вещь, ближайшую по теплоте и лучше всего подходящую по цвету к уже выбранным.
func pick(candidates []domain.Item, ideal domain.WarmthLevel, colors []domain.Color) (domain.Item, bool) {
	var best domain.Item
	bestScore, found := 0, false
	for _, candidate := range candidates {
		score := warmthStepPenalty*distance(candidate, ideal) + colorPenalty(append(slices.Clip(colors), candidate.Colors.Main))
		if !found || score < bestScore {
			best, bestScore, found = candidate, score, true
		}
	}
	return best, found
}

func (recommender recommender) missingBase() []domain.Note {
	var notes []domain.Note
	for _, category := range []domain.Category{domain.CategoryTop, domain.CategoryBottom} {
		if len(recommender.wardrobe[category]) == 0 {
			notes = append(notes, domain.Note{Kind: domain.NoteMissing, Category: category})
		}
	}
	return notes
}

func options(items []domain.Item) []*domain.Item {
	if len(items) == 0 {
		return []*domain.Item{nil}
	}
	result := make([]*domain.Item, len(items))
	for index := range items {
		result[index] = &items[index]
	}
	return result
}
