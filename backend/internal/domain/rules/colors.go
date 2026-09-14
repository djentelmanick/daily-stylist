package rules

import (
	"slices"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

// Нейтральные цвета - чёрный, белый, серый, бежевый, тёмно-синий - сочетаются
// со всем, поэтому семьи у них нет. Соседние оттенки - одна семья.
var colorFamilies = map[domain.Color]string{
	domain.ColorRed:       "red",
	domain.ColorBurgundy:  "red",
	domain.ColorPink:      "pink",
	domain.ColorOrange:    "orange",
	domain.ColorYellow:    "yellow",
	domain.ColorGreen:     "green",
	domain.ColorKhaki:     "green",
	domain.ColorBlue:      "blue",
	domain.ColorLightBlue: "blue",
	domain.ColorBrown:     "brown",
	domain.ColorPurple:    "purple",
}

var clashingFamilies = [][2]string{
	{"red", "orange"},
	{"red", "pink"},
	{"orange", "pink"},
	{"red", "green"},
	{"orange", "purple"},
}

const (
	twoAccentsPenalty  = 1
	clashPenalty       = 6
	manyAccentsPenalty = 10
)

// colorPenalty оценивает главные цвета вещей: один акцент - хорошо,
// два - допустимо, если они не спорят, три и больше - пестро.
func colorPenalty(colors []domain.Color) int {
	var families []string
	for _, color := range colors {
		if family, ok := colorFamilies[color]; ok && !slices.Contains(families, family) {
			families = append(families, family)
		}
	}

	switch {
	case len(families) <= 1:
		return 0
	case len(families) > 2:
		return manyAccentsPenalty
	case slices.Contains(clashingFamilies, [2]string{families[0], families[1]}),
		slices.Contains(clashingFamilies, [2]string{families[1], families[0]}):
		return clashPenalty
	default:
		return twoAccentsPenalty
	}
}

func mainColors(items []domain.Item) []domain.Color {
	colors := make([]domain.Color, len(items))
	for index, item := range items {
		colors[index] = item.Colors.Main
	}
	return colors
}
