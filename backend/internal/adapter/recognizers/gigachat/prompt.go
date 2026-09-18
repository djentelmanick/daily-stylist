package gigachat

import (
	"fmt"
	"strings"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

// Без примеров модель путает границы: пиджак уезжает в верхнюю одежду, колготки - в носки.
var categoryHints = map[domain.Category]string{
	domain.CategoryTop:        "верх: футболка, рубашка, свитер, худи, пиджак",
	domain.CategoryBottom:     "низ: брюки, джинсы, юбка, шорты",
	domain.CategoryDress:      "платье",
	domain.CategoryJumpsuit:   "комбинезон",
	domain.CategoryOuterwear:  "верхняя одежда: куртка, пальто, пуховик, дождевик",
	domain.CategoryShoes:      "обувь",
	domain.CategorySocks:      "носки, колготки",
	domain.CategoryHat:        "головной убор",
	domain.CategoryScarf:      "шарф, снуд, платок",
	domain.CategoryThermal:    "термобельё",
	domain.CategoryUmbrella:   "зонт",
	domain.CategorySunglasses: "солнцезащитные очки",
	domain.CategoryBag:        "сумка, рюкзак",
	domain.CategoryAccessory:  "прочее: ремень, часы, перчатки",
}

var warmthHints = map[domain.WarmthLevel]string{
	domain.WarmthLevelLight:   "лёгкая: футболка, шорты, сандалии, от +20",
	domain.WarmthLevelMedium:  "средняя: джинсы, лонгслив, ветровка, от +10 до +20",
	domain.WarmthLevelWarm:    "тёплая: свитер, пальто, ботинки, от 0 до +10",
	domain.WarmthLevelHeavy:   "зимняя: зимняя куртка, шапка, от -10 до 0",
	domain.WarmthLevelExtreme: "для мороза: пуховик, ботинки на меху, ниже -10",
}

var prompt = buildPrompt()

func buildPrompt() string {
	var text strings.Builder

	text.WriteString(`Ты помогаешь заполнить карточку вещи в гардеробе по фотографии.

Верни ТОЛЬКО JSON без пояснений и без markdown, строго такого вида:
{"name": "", "category": "", "main_color": "", "extra_colors": [],
 "seasons": [], "warmth_level": 1, "waterproof": false}

`)
	fmt.Fprintf(&text, "name - короткое название вещи по-русски, до %d символов.\n\n", domain.MaxNameLength)

	text.WriteString("category - ровно одно значение из списка:\n")
	for _, category := range domain.AllCategories() {
		fmt.Fprintf(&text, "- %s: %s\n", category, categoryHints[category])
	}

	fmt.Fprintf(&text, "\nmain_color - доминирующий цвет вещи, extra_colors - до %d других её цветов,\n", maxSuggestedColors)
	text.WriteString("без повтора главного. Только эти значения: ")
	text.WriteString(strings.Join(values(domain.AllColors()), ", "))

	text.WriteString(".\n\nseasons - в какие сезоны вещь носят, одно или несколько: ")
	text.WriteString(strings.Join(values(domain.AllSeasons()), ", "))

	text.WriteString(".\n\nwarmth_level - насколько вещь греет, когда она внешний слой:\n")
	for _, level := range domain.AllWarmthLevels() {
		fmt.Fprintf(&text, "- %d: %s\n", level, warmthHints[level])
	}
	fmt.Fprintf(&text, "Для категорий %s всегда 0.\n", strings.Join(values(withoutWarmth()), ", "))

	text.WriteString("\nwaterproof - true только если вещь действительно не промокает: дождевик, резиновые сапоги, зонт.\n")
	text.WriteString("\nЕсли на фотографии нет одежды, верни {}.")

	return text.String()
}

func withoutWarmth() []domain.Category {
	var categories []domain.Category
	for _, category := range domain.AllCategories() {
		if !category.HasWarmth() {
			categories = append(categories, category)
		}
	}
	return categories
}

func values[T ~string](items []T) []string {
	result := make([]string, len(items))
	for index, item := range items {
		result[index] = string(item)
	}
	return result
}
