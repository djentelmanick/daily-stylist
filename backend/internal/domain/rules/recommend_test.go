package rules

import (
	"slices"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func TestRecommend_WarmDay(t *testing.T) {
	var wardrobe testWardrobe
	wardrobe.add("Свитер", domain.CategoryTop, domain.WarmthLevelWarm, domain.ColorGray)
	wardrobe.add("Футболка", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorWhite)
	wardrobe.add("Джинсы", domain.CategoryBottom, domain.WarmthLevelMedium, domain.ColorBlue)
	wardrobe.add("Шорты", domain.CategoryBottom, domain.WarmthLevelLight, domain.ColorBlack)
	wardrobe.add("Ботинки", domain.CategoryShoes, domain.WarmthLevelWarm, domain.ColorBlack)
	wardrobe.add("Сандалии", domain.CategoryShoes, domain.WarmthLevelLight, domain.ColorBlack)
	wardrobe.add("Куртка", domain.CategoryOuterwear, domain.WarmthLevelWarm, domain.ColorBlack)
	wardrobe.add("Носки", domain.CategorySocks, domain.WarmthLevelLight, domain.ColorWhite)

	outfits, notes := Recommend(Input{Items: wardrobe.items, Weather: feelsLike(25), Season: domain.SeasonSummer})

	if len(outfits) == 0 {
		t.Fatalf("образов нет, заметки: %+v", notes)
	}
	checkNames(t, outfits[0], "Футболка", "Шорты", "Сандалии")
	if len(outfits[0].Notes) != 0 {
		t.Errorf("заметки = %+v, ожидались без заметок", outfits[0].Notes)
	}
}

func TestRecommend_FrostyDay(t *testing.T) {
	var wardrobe testWardrobe
	wardrobe.add("Футболка", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorWhite)
	wardrobe.add("Свитер", domain.CategoryTop, domain.WarmthLevelWarm, domain.ColorGray)
	wardrobe.add("Утеплённые брюки", domain.CategoryBottom, domain.WarmthLevelWarm, domain.ColorBlack)
	wardrobe.add("Пуховик", domain.CategoryOuterwear, domain.WarmthLevelExtreme, domain.ColorBlack)
	wardrobe.add("Ветровка", domain.CategoryOuterwear, domain.WarmthLevelMedium, domain.ColorBlack)
	wardrobe.add("Ботинки на меху", domain.CategoryShoes, domain.WarmthLevelExtreme, domain.ColorBrown)
	wardrobe.add("Шапка", domain.CategoryHat, domain.WarmthLevelHeavy, domain.ColorGray)
	wardrobe.add("Шарф", domain.CategoryScarf, domain.WarmthLevelHeavy, domain.ColorGray)
	wardrobe.add("Термобельё", domain.CategoryThermal, domain.WarmthLevelHeavy, domain.ColorBlack)
	wardrobe.add("Шерстяные носки", domain.CategorySocks, domain.WarmthLevelHeavy, domain.ColorGray)
	wardrobe.add("Сумка", domain.CategoryBag, 0, domain.ColorBlack)

	outfits, _ := Recommend(Input{Items: wardrobe.items, Weather: feelsLike(-15), Season: domain.SeasonWinter})

	if len(outfits) == 0 {
		t.Fatal("образов нет")
	}
	checkNames(t, outfits[0], "Шапка", "Шарф", "Пуховик", "Свитер", "Термобельё", "Утеплённые брюки", "Шерстяные носки", "Ботинки на меху", "Сумка")
	if len(outfits[0].Notes) != 0 {
		t.Errorf("заметки = %+v, ожидались без заметок", outfits[0].Notes)
	}
}

func TestRecommend_UsesOnlyAvailableInSeasonItems(t *testing.T) {
	var wardrobe testWardrobe
	wardrobe.add("Грязная футболка", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorWhite).Status = domain.ItemStatusDirty
	wardrobe.add("Футболка в архиве", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorWhite).Status = domain.ItemStatusArchived
	wardrobe.add("Зимняя футболка", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorWhite).Seasons = []domain.Season{domain.SeasonWinter}
	wardrobe.add("Рубашка", domain.CategoryTop, domain.WarmthLevelMedium, domain.ColorWhite)
	wardrobe.add("Шорты", domain.CategoryBottom, domain.WarmthLevelLight, domain.ColorBlack)

	outfits, _ := Recommend(Input{Items: wardrobe.items, Weather: feelsLike(25), Season: domain.SeasonSummer})

	if len(outfits) != 1 {
		t.Fatalf("образов %d, ожидался 1: %+v", len(outfits), outfits)
	}
	checkNames(t, outfits[0], "Рубашка", "Шорты")
}

func TestRecommend_WarnsWhenWardrobeFallsShort(t *testing.T) {
	var wardrobe testWardrobe
	wardrobe.add("Футболка", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorWhite)
	wardrobe.add("Джинсы", domain.CategoryBottom, domain.WarmthLevelMedium, domain.ColorBlue)
	wardrobe.add("Ветровка", domain.CategoryOuterwear, domain.WarmthLevelMedium, domain.ColorBlack)

	outfits, _ := Recommend(Input{Items: wardrobe.items, Weather: feelsLike(-15), Season: domain.SeasonWinter})

	if len(outfits) == 0 {
		t.Fatal("образов нет: без подходящих вещей нужен образ с предупреждениями")
	}
	checkNames(t, outfits[0], "Ветровка", "Футболка", "Джинсы")
	for _, want := range []domain.Note{
		{Kind: domain.NoteTooLight, Item: wardrobe.named("Ветровка")},
		{Kind: domain.NoteTooLight, Item: wardrobe.named("Футболка")},
		{Kind: domain.NoteMissing, Category: domain.CategoryShoes},
		{Kind: domain.NoteMissing, Category: domain.CategoryHat},
	} {
		if !containsNote(outfits[0].Notes, want) {
			t.Errorf("нет заметки %s %s%s в %+v", want.Kind, want.Category, want.Item.Name, outfits[0].Notes)
		}
	}
}

func TestRecommend_NothingToWear(t *testing.T) {
	var wardrobe testWardrobe
	wardrobe.add("Кеды", domain.CategoryShoes, domain.WarmthLevelMedium, domain.ColorWhite)

	outfits, notes := Recommend(Input{Items: wardrobe.items, Weather: feelsLike(15), Season: domain.SeasonAutumn})

	if len(outfits) != 0 {
		t.Errorf("образы = %+v, ожидалось без образов", outfits)
	}
	if len(notes) != 2 ||
		!containsNote(notes, domain.Note{Kind: domain.NoteMissing, Category: domain.CategoryTop}) ||
		!containsNote(notes, domain.Note{Kind: domain.NoteMissing, Category: domain.CategoryBottom}) {
		t.Errorf("заметки = %+v, ожидались «нет верха» и «нет низа»", notes)
	}
}

func TestRecommend_Rain(t *testing.T) {
	tests := []struct {
		name      string
		wind      float64
		raincoat  bool
		wantNames []string
		wantNote  domain.NoteKind
	}{
		{"зонт", 3, false, []string{"Футболка", "Джинсы", "Кеды", "Зонт"}, ""},
		{"ветер: вместо зонта дождевик", 12, true, []string{"Дождевик", "Футболка", "Джинсы", "Кеды"}, domain.NoteTooWindyForUmbrella},
		{"ветер и защиты нет", 12, false, []string{"Футболка", "Джинсы", "Кеды"}, domain.NoteNoRainProtection},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var wardrobe testWardrobe
			wardrobe.add("Футболка", domain.CategoryTop, domain.WarmthLevelMedium, domain.ColorWhite)
			wardrobe.add("Джинсы", domain.CategoryBottom, domain.WarmthLevelMedium, domain.ColorBlue)
			wardrobe.add("Кеды", domain.CategoryShoes, domain.WarmthLevelMedium, domain.ColorWhite)
			wardrobe.add("Зонт", domain.CategoryUmbrella, 0, domain.ColorBlack)
			if test.raincoat {
				wardrobe.add("Дождевик", domain.CategoryOuterwear, domain.WarmthLevelMedium, domain.ColorYellow).Waterproof = true
			}
			weather := feelsLike(17)
			weather.PrecipitationChanceMax = 80
			weather.WindSpeedMax = test.wind

			outfits, _ := Recommend(Input{Items: wardrobe.items, Weather: weather, Season: domain.SeasonAutumn})

			if len(outfits) == 0 {
				t.Fatal("образов нет")
			}
			checkNames(t, outfits[0], test.wantNames...)
			if test.wantNote != "" && !containsNote(outfits[0].Notes, domain.Note{Kind: test.wantNote}) {
				t.Errorf("нет заметки %s в %+v", test.wantNote, outfits[0].Notes)
			}
			if containsNote(outfits[0].Notes, domain.Note{Kind: domain.NoteNoRainProtection}) != (test.wantNote == domain.NoteNoRainProtection) {
				t.Errorf("заметка о защите от дождя не совпадает с ожиданием: %+v", outfits[0].Notes)
			}
		})
	}
}

func TestRecommend_SunglassesOnlyWhenSunny(t *testing.T) {
	for _, test := range []struct {
		name      string
		rain      int
		wantGlass bool
	}{
		{"солнце", 0, true},
		{"солнце и дождь", 70, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var wardrobe testWardrobe
			wardrobe.add("Платье", domain.CategoryDress, domain.WarmthLevelLight, domain.ColorWhite)
			wardrobe.add("Очки", domain.CategorySunglasses, 0, domain.ColorBlack)
			weather := feelsLike(25)
			weather.UVIndexMax = 5
			weather.PrecipitationChanceMax = test.rain

			outfits, _ := Recommend(Input{Items: wardrobe.items, Weather: weather, Season: domain.SeasonSummer})

			if len(outfits) == 0 {
				t.Fatal("образов нет")
			}
			if got := slices.Contains(names(outfits[0]), "Очки"); got != test.wantGlass {
				t.Errorf("очки в образе: %v, ожидалось %v", got, test.wantGlass)
			}
		})
	}
}

func TestRecommend_PrefersMatchingColors(t *testing.T) {
	var wardrobe testWardrobe
	wardrobe.add("Красная футболка", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorRed)
	wardrobe.add("Оранжевые шорты", domain.CategoryBottom, domain.WarmthLevelLight, domain.ColorOrange)
	wardrobe.add("Чёрные шорты", domain.CategoryBottom, domain.WarmthLevelLight, domain.ColorBlack)

	outfits, _ := Recommend(Input{Items: wardrobe.items, Weather: feelsLike(25), Season: domain.SeasonSummer})

	if len(outfits) != 2 {
		t.Fatalf("образов %d, ожидалось 2", len(outfits))
	}
	checkNames(t, outfits[0], "Красная футболка", "Чёрные шорты")
}

func TestRecommend_AvoidsRecentlyWornClothes(t *testing.T) {
	var wardrobe testWardrobe
	worn := wardrobe.add("Вчерашняя футболка", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorWhite)
	wardrobe.add("Футболка", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorWhite)
	wardrobe.add("Шорты", domain.CategoryBottom, domain.WarmthLevelLight, domain.ColorBlack)

	outfits, _ := Recommend(Input{
		Items:       wardrobe.items,
		Weather:     feelsLike(25),
		Season:      domain.SeasonSummer,
		WornDaysAgo: map[int64]int{worn.ID: 1},
	})

	if len(outfits) != 2 {
		t.Fatalf("образов %d, ожидалось 2: повтор - не запрет", len(outfits))
	}
	checkNames(t, outfits[0], "Футболка", "Шорты")
	checkNames(t, outfits[1], "Вчерашняя футболка", "Шорты")
}

func TestRecommend_WarnsThatItWarmsUp(t *testing.T) {
	var wardrobe testWardrobe
	wardrobe.add("Лонгслив", domain.CategoryTop, domain.WarmthLevelMedium, domain.ColorWhite)
	wardrobe.add("Джинсы", domain.CategoryBottom, domain.WarmthLevelMedium, domain.ColorBlue)
	wardrobe.add("Куртка", domain.CategoryOuterwear, domain.WarmthLevelWarm, domain.ColorBlack)
	weather := feelsLike(8)
	weather.FeelsLikeMax = 22

	outfits, _ := Recommend(Input{Items: wardrobe.items, Weather: weather, Season: domain.SeasonAutumn})

	if len(outfits) == 0 {
		t.Fatal("образов нет")
	}
	if !containsNote(outfits[0].Notes, domain.Note{Kind: domain.NoteWarmsUp}) {
		t.Errorf("нет заметки о потеплении: %+v", outfits[0].Notes)
	}
}

func TestRecommend_LimitsOutfits(t *testing.T) {
	var wardrobe testWardrobe
	for range MaxOutfits + 2 {
		wardrobe.add("Футболка", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorWhite)
	}
	wardrobe.add("Шорты", domain.CategoryBottom, domain.WarmthLevelLight, domain.ColorBlack)

	outfits, _ := Recommend(Input{Items: wardrobe.items, Weather: feelsLike(25), Season: domain.SeasonSummer})

	if len(outfits) != MaxOutfits {
		t.Errorf("образов %d, ожидалось %d", len(outfits), MaxOutfits)
	}
}

func TestWarmthFor_Boundaries(t *testing.T) {
	tests := []struct {
		feelsLike float64
		want      domain.WarmthLevel
	}{
		{20, domain.WarmthLevelLight},
		{19.9, domain.WarmthLevelMedium},
		{10, domain.WarmthLevelMedium},
		{0, domain.WarmthLevelWarm},
		{-0.1, domain.WarmthLevelHeavy},
		{-10, domain.WarmthLevelHeavy},
		{-10.1, domain.WarmthLevelExtreme},
	}
	for _, test := range tests {
		if got := warmthFor(test.feelsLike); got != test.want {
			t.Errorf("warmthFor(%v) = %d, ожидался %d", test.feelsLike, got, test.want)
		}
	}
}

func TestColorPenalty(t *testing.T) {
	tests := []struct {
		name   string
		colors []domain.Color
		want   int
	}{
		{"только нейтральные", []domain.Color{domain.ColorBlack, domain.ColorWhite, domain.ColorNavy}, 0},
		{"оттенки одной семьи", []domain.Color{domain.ColorBlue, domain.ColorLightBlue, domain.ColorGray}, 0},
		{"два акцента", []domain.Color{domain.ColorBlue, domain.ColorBrown}, twoAccentsPenalty},
		{"спорящие акценты", []domain.Color{domain.ColorGreen, domain.ColorBurgundy}, clashPenalty},
		{"три акцента", []domain.Color{domain.ColorBlue, domain.ColorBrown, domain.ColorYellow}, manyAccentsPenalty},
	}
	for _, test := range tests {
		if got := colorPenalty(test.colors); got != test.want {
			t.Errorf("%s: colorPenalty = %d, ожидался %d", test.name, got, test.want)
		}
	}
}

type testWardrobe struct {
	items []domain.Item
}

// add возвращает указатель, чтобы тест мог поменять статус или сезоны.
func (wardrobe *testWardrobe) add(name string, category domain.Category, level domain.WarmthLevel, color domain.Color) *domain.Item {
	wardrobe.items = append(wardrobe.items, domain.Item{
		ID:          int64(len(wardrobe.items) + 1),
		UserID:      1,
		Name:        name,
		Category:    category,
		Colors:      domain.Colors{Main: color},
		Seasons:     domain.AllSeasons(),
		WarmthLevel: level,
		Status:      domain.ItemStatusAvailable,
	})
	return &wardrobe.items[len(wardrobe.items)-1]
}

func (wardrobe *testWardrobe) named(name string) domain.Item {
	index := slices.IndexFunc(wardrobe.items, func(item domain.Item) bool { return item.Name == name })
	return wardrobe.items[index]
}

func feelsLike(temperature float64) domain.Weather {
	return domain.Weather{
		TemperatureMin: temperature,
		TemperatureMax: temperature,
		FeelsLikeMin:   temperature,
		FeelsLikeMax:   temperature,
	}
}

func names(outfit domain.Outfit) []string {
	result := make([]string, len(outfit.Items))
	for index, item := range outfit.Items {
		result[index] = item.Name
	}
	return result
}

func checkNames(t *testing.T, outfit domain.Outfit, want ...string) {
	t.Helper()
	if got := names(outfit); !slices.Equal(got, want) {
		t.Errorf("образ = %q, ожидался %q", got, want)
	}
}

func containsNote(notes []domain.Note, want domain.Note) bool {
	return slices.ContainsFunc(notes, func(note domain.Note) bool {
		return note.Kind == want.Kind && note.Category == want.Category && note.Item.ID == want.Item.ID
	})
}
