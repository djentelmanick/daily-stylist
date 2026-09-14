package domain

import (
	"cmp"
	"slices"
)

type Outfit struct {
	Items []Item
	Notes []Note
}

type NoteKind string

const (
	// Нет доступной вещи нужной категории: Category.
	NoteMissing NoteKind = "missing"
	// Вещь из образа легче или теплее, чем нужно: Item.
	NoteTooLight NoteKind = "too_light"
	NoteTooWarm  NoteKind = "too_warm"

	NoteWarmsUp             NoteKind = "warms_up"
	NoteNoRainProtection    NoteKind = "no_rain_protection"
	NoteTooWindyForUmbrella NoteKind = "too_windy_for_umbrella"
)

type Note struct {
	Kind     NoteKind
	Category Category
	Item     Item
}

// SortForWearing ставит вещи сверху вниз, как они надеты, а то, что в руках, - в конце.
func SortForWearing(items []Item) {
	slices.SortStableFunc(items, func(a, b Item) int {
		return cmp.Compare(slices.Index(wearingOrder, a.Category), slices.Index(wearingOrder, b.Category))
	})
}

var wearingOrder = []Category{
	CategoryHat, CategorySunglasses, CategoryScarf, CategoryOuterwear,
	CategoryTop, CategoryDress, CategoryJumpsuit, CategoryThermal,
	CategoryBottom, CategorySocks, CategoryShoes, CategoryAccessory, CategoryBag, CategoryUmbrella,
}
