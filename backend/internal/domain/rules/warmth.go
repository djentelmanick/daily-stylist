package rules

import "github.com/djentelmanick/daily-stylist/backend/internal/domain"

const (
	outerwearBelow   = 15.0
	outerwearHotFrom = 20.0
	rainChance       = 50
	// При таком ветре зонт выворачивает.
	strongWind    = 11.0
	sunnyUVIndex  = 3.0
	strongUVIndex = 6.0
)

type conditions struct {
	level         domain.WarmthLevel
	needOuterwear bool
	precipitation bool
	snow          bool
	windy         bool
	uvIndex       float64
	warmsUp       bool
}

func conditionsFor(weather domain.Weather) conditions {
	return conditions{
		level:         warmthFor(weather.FeelsLikeMin),
		needOuterwear: weather.FeelsLikeMin < outerwearBelow,
		precipitation: weather.PrecipitationChanceMax >= rainChance,
		snow:          weather.Snow,
		windy:         weather.WindSpeedMax >= strongWind,
		uvIndex:       weather.UVIndexMax,
		warmsUp:       weather.FeelsLikeMax >= outerwearHotFrom,
	}
}

func (c conditions) rain() bool {
	return c.precipitation && !c.snow
}

func (c conditions) sunny() bool {
	return c.uvIndex >= sunnyUVIndex && !c.precipitation
}

func warmthFor(feelsLike float64) domain.WarmthLevel {
	switch {
	case feelsLike >= 20:
		return domain.WarmthLevelLight
	case feelsLike >= 10:
		return domain.WarmthLevelMedium
	case feelsLike >= 0:
		return domain.WarmthLevelWarm
	case feelsLike >= -10:
		return domain.WarmthLevelHeavy
	default:
		return domain.WarmthLevelExtreme
	}
}

// idealLevel - какой теплоты должна быть вещь в образе. Под верхней одеждой верх
// берётся на уровень легче, а тёплый низ и верх выше «тёплого» не нужны:
// в мороз греют верхняя одежда и термобельё.
func (c conditions) idealLevel(item domain.Item, withOuterwear bool) domain.WarmthLevel {
	switch item.Category {
	case domain.CategoryTop, domain.CategoryDress, domain.CategoryJumpsuit:
		if withOuterwear {
			return clampLevel(c.level-1, domain.WarmthLevelWarm)
		}
		return clampLevel(c.level, domain.WarmthLevelWarm)
	case domain.CategoryBottom:
		return clampLevel(c.level, domain.WarmthLevelHeavy)
	default:
		return c.level
	}
}

func clampLevel(level, highest domain.WarmthLevel) domain.WarmthLevel {
	return max(domain.WarmthLevelLight, min(level, highest))
}

func distance(item domain.Item, ideal domain.WarmthLevel) int {
	if !item.Category.HasWarmth() {
		return 0
	}
	difference := int(item.WarmthLevel) - int(ideal)
	return max(difference, -difference)
}

// near - вещи, которые отличаются от нужной теплоты не больше чем на уровень.
func near(items []domain.Item, ideal domain.WarmthLevel) []domain.Item {
	return within(items, ideal, 1)
}

// closest - как near, а если таких вещей нет, то самые близкие по теплоте:
// образ без куртки хуже, чем с лёгкой курткой и предупреждением.
func closest(items []domain.Item, ideal domain.WarmthLevel) []domain.Item {
	nearest := 1
	if found := near(items, ideal); len(found) == 0 && len(items) > 0 {
		nearest = distance(items[0], ideal)
		for _, item := range items {
			nearest = min(nearest, distance(item, ideal))
		}
	}
	return within(items, ideal, nearest)
}

func within(items []domain.Item, ideal domain.WarmthLevel, limit int) []domain.Item {
	var result []domain.Item
	for _, item := range items {
		if distance(item, ideal) <= limit {
			result = append(result, item)
		}
	}
	return result
}
