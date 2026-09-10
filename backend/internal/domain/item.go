package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var ErrInvalidItem = errors.New("невалидная вещь")

type Item struct {
	ID          int64
	UserID      int64
	Name        string
	Category    Category
	Colors      Colors
	Seasons     []Season
	WarmthLevel WarmthLevel
	Waterproof  bool
	Status      ItemStatus
}

type NewItemParams struct {
	UserID      int64
	Name        string
	Category    Category
	Colors      Colors
	Seasons     []Season
	WarmthLevel WarmthLevel
	Waterproof  bool
}

func NewItem(params NewItemParams) (Item, error) {
	item := Item{
		UserID:      params.UserID,
		Name:        strings.TrimSpace(params.Name),
		Category:    params.Category,
		Colors:      params.Colors,
		Seasons:     params.Seasons,
		WarmthLevel: params.WarmthLevel,
		Waterproof:  params.Waterproof,
		Status:      ItemStatusAvailable,
	}
	if err := item.Validate(); err != nil {
		return Item{}, err
	}
	return item, nil
}

func (item Item) Validate() error {
	var problems []string

	if item.UserID <= 0 {
		problems = append(problems, "не указан владелец")
	}
	if strings.TrimSpace(item.Name) == "" {
		problems = append(problems, "пустое название")
	}
	if !item.Category.Valid() {
		problems = append(problems, fmt.Sprintf("неизвестная категория %q", item.Category))
	}

	problems = append(problems, item.Colors.problems()...)

	if len(item.Seasons) == 0 {
		problems = append(problems, "не указан ни один сезон")
	}
	for _, season := range item.Seasons {
		if !season.Valid() {
			problems = append(problems, fmt.Sprintf("неизвестный сезон %q", season))
		}
	}
	if hasDuplicates(item.Seasons) {
		problems = append(problems, "сезоны повторяются")
	}

	if item.Category.Valid() {
		switch {
		case item.Category.HasWarmth() && !item.WarmthLevel.Valid():
			problems = append(problems, fmt.Sprintf("уровень теплоты %d вне диапазона 1–5", item.WarmthLevel))
		case !item.Category.HasWarmth() && item.WarmthLevel != 0:
			problems = append(problems, fmt.Sprintf("у категории %q не бывает уровня теплоты", item.Category))
		}
	}

	if !item.Status.Valid() {
		problems = append(problems, fmt.Sprintf("неизвестный статус %q", item.Status))
	}

	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidItem, strings.Join(problems, "; "))
	}
	return nil
}

type Colors struct {
	Main  Color
	Extra []Color
}

func (colors Colors) problems() []string {
	var problems []string

	if !colors.Main.Valid() {
		problems = append(problems, fmt.Sprintf("неизвестный основной цвет %q", colors.Main))
	}
	for _, color := range colors.Extra {
		if !color.Valid() {
			problems = append(problems, fmt.Sprintf("неизвестный дополнительный цвет %q", color))
		}
	}
	if slices.Contains(colors.Extra, colors.Main) {
		problems = append(problems, "основной цвет повторяется среди дополнительных")
	}
	if hasDuplicates(colors.Extra) {
		problems = append(problems, "дополнительные цвета повторяются")
	}

	return problems
}

type Color string

const (
	ColorRed       Color = "red"
	ColorBurgundy  Color = "burgundy"
	ColorBlue      Color = "blue"
	ColorLightBlue Color = "light_blue"
	ColorNavy      Color = "navy"
	ColorGreen     Color = "green"
	ColorKhaki     Color = "khaki"
	ColorYellow    Color = "yellow"
	ColorBlack     Color = "black"
	ColorWhite     Color = "white"
	ColorGray      Color = "gray"
	ColorBrown     Color = "brown"
	ColorBeige     Color = "beige"
	ColorOrange    Color = "orange"
	ColorPurple    Color = "purple"
	ColorPink      Color = "pink"
)

var allColors = []Color{
	ColorRed, ColorBurgundy, ColorBlue, ColorLightBlue, ColorNavy, ColorGreen, ColorKhaki, ColorYellow,
	ColorBlack, ColorWhite, ColorGray, ColorBrown, ColorBeige, ColorOrange, ColorPurple, ColorPink,
}

func (color Color) Valid() bool {
	return slices.Contains(allColors, color)
}

type Category string

const (
	CategoryTop        Category = "top"
	CategoryBottom     Category = "bottom"
	CategoryDress      Category = "dress"
	CategoryJumpsuit   Category = "jumpsuit"
	CategoryOuterwear  Category = "outerwear"
	CategoryShoes      Category = "shoes"
	CategorySocks      Category = "socks"
	CategoryHat        Category = "hat"
	CategoryScarf      Category = "scarf"
	CategoryThermal    Category = "thermal"
	CategoryUmbrella   Category = "umbrella"
	CategorySunglasses Category = "sunglasses"
	CategoryBag        Category = "bag"
	CategoryAccessory  Category = "accessory"
)

var allCategories = []Category{
	CategoryTop, CategoryBottom, CategoryDress, CategoryJumpsuit, CategoryOuterwear, CategoryShoes, CategorySocks,
	CategoryHat, CategoryScarf, CategoryThermal, CategoryUmbrella, CategorySunglasses, CategoryBag, CategoryAccessory,
}

func (category Category) Valid() bool {
	return slices.Contains(allCategories, category)
}

func (category Category) HasWarmth() bool {
	switch category {
	case CategoryUmbrella, CategorySunglasses, CategoryBag, CategoryAccessory:
		return false
	}
	return true
}

type Season string

const (
	SeasonSpring Season = "spring"
	SeasonSummer Season = "summer"
	SeasonAutumn Season = "autumn"
	SeasonWinter Season = "winter"
)

var allSeasons = []Season{SeasonSpring, SeasonSummer, SeasonAutumn, SeasonWinter}

func (season Season) Valid() bool {
	return slices.Contains(allSeasons, season)
}

type WarmthLevel int

const (
	// Футболка, шорты, льняная рубашка, сандалии. Внешним слоем — от +20 °C.
	WarmthLevelLight WarmthLevel = 1
	// Лонгслив, джинсы, лёгкая толстовка, ветровка, кеды. Внешним слоем — от +10 до +20 °C.
	WarmthLevelMedium WarmthLevel = 2
	// Свитер, утеплённое худи, демисезонная куртка, ботинки. Внешним слоем — от 0 до +10 °C.
	WarmthLevelWarm WarmthLevel = 3
	// Зимняя куртка, тёплые ботинки, шерстяная шапка. Внешним слоем — от −10 до 0 °C.
	WarmthLevelHeavy WarmthLevel = 4
	// Пуховик, ботинки на меху, меховая шапка. Внешним слоем — ниже −10 °C.
	WarmthLevelExtreme WarmthLevel = 5
)

func (level WarmthLevel) Valid() bool {
	return level >= WarmthLevelLight && level <= WarmthLevelExtreme
}

type ItemStatus string

const (
	ItemStatusAvailable ItemStatus = "available"
	ItemStatusDirty     ItemStatus = "dirty"
	ItemStatusOffSeason ItemStatus = "off_season"
	ItemStatusArchived  ItemStatus = "archived"
)

var allItemStatuses = []ItemStatus{ItemStatusAvailable, ItemStatusDirty, ItemStatusOffSeason, ItemStatusArchived}

func (status ItemStatus) Valid() bool {
	return slices.Contains(allItemStatuses, status)
}

func hasDuplicates[T comparable](values []T) bool {
	seen := make(map[T]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
