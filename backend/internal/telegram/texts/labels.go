package texts

import (
	"strconv"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func Category(category domain.Category) string {
	return labelOrDefault(categoryLabels, category, string(category))
}

func Color(color domain.Color) string {
	return labelOrDefault(colorLabels, color, string(color))
}

func Season(season domain.Season) string {
	return labelOrDefault(seasonLabels, season, string(season))
}

func WarmthLevel(level domain.WarmthLevel) string {
	return labelOrDefault(warmthLevelLabels, level, strconv.Itoa(int(level)))
}

func ItemStatus(status domain.ItemStatus) string {
	return labelOrDefault(itemStatusLabels, status, string(status))
}

func labelOrDefault[K comparable](labels map[K]string, key K, fallback string) string {
	if label, ok := labels[key]; ok {
		return label
	}
	return fallback
}

var categoryLabels = map[domain.Category]string{
	domain.CategoryTop:        "Верх",
	domain.CategoryBottom:     "Низ",
	domain.CategoryDress:      "Платье",
	domain.CategoryJumpsuit:   "Комбинезон",
	domain.CategoryOuterwear:  "Верхняя одежда",
	domain.CategoryShoes:      "Обувь",
	domain.CategorySocks:      "Носки",
	domain.CategoryHat:        "Головной убор",
	domain.CategoryScarf:      "Шарф",
	domain.CategoryThermal:    "Термобельё",
	domain.CategoryUmbrella:   "Зонт",
	domain.CategorySunglasses: "Солнцезащитные очки",
	domain.CategoryBag:        "Сумка",
	domain.CategoryAccessory:  "Другой аксессуар",
}

var colorLabels = map[domain.Color]string{
	domain.ColorRed:       "Красный",
	domain.ColorBurgundy:  "Бордовый",
	domain.ColorBlue:      "Синий",
	domain.ColorLightBlue: "Голубой",
	domain.ColorNavy:      "Тёмно-синий",
	domain.ColorGreen:     "Зелёный",
	domain.ColorKhaki:     "Хаки",
	domain.ColorYellow:    "Жёлтый",
	domain.ColorBlack:     "Чёрный",
	domain.ColorWhite:     "Белый",
	domain.ColorGray:      "Серый",
	domain.ColorBrown:     "Коричневый",
	domain.ColorBeige:     "Бежевый",
	domain.ColorOrange:    "Оранжевый",
	domain.ColorPurple:    "Фиолетовый",
	domain.ColorPink:      "Розовый",
}

var seasonLabels = map[domain.Season]string{
	domain.SeasonSpring: "Весна",
	domain.SeasonSummer: "Лето",
	domain.SeasonAutumn: "Осень",
	domain.SeasonWinter: "Зима",
}

var itemStatusLabels = map[domain.ItemStatus]string{
	domain.ItemStatusAvailable: "Доступна",
	domain.ItemStatusDirty:     "Грязная",
	domain.ItemStatusArchived:  "В архиве",
}

var warmthLevelLabels = map[domain.WarmthLevel]string{
	domain.WarmthLevelLight:   "Для жары, от +20 °C",
	domain.WarmthLevelMedium:  "Для тепла, от +10 до +20 °C",
	domain.WarmthLevelWarm:    "Для прохлады, от 0 до +10 °C",
	domain.WarmthLevelHeavy:   "Для холода, от −10 до 0 °C",
	domain.WarmthLevelExtreme: "Для мороза, ниже −10 °C",
}
