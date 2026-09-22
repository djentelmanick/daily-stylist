package rules

import (
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func TestReview_HandPickedOutfit(t *testing.T) {
	var wardrobe testWardrobe
	sweater := *wardrobe.add("Свитер", domain.CategoryTop, domain.WarmthLevelHeavy, domain.ColorGray)
	shorts := *wardrobe.add("Шорты", domain.CategoryBottom, domain.WarmthLevelLight, domain.ColorBlack)
	bag := *wardrobe.add("Сумка", domain.CategoryBag, 0, domain.ColorBlack)

	notes := Review(feelsLike(25), []domain.Item{sweater, shorts, bag})

	for _, want := range []domain.Note{
		{Kind: domain.NoteTooWarm, Item: sweater},
		{Kind: domain.NoteMissing, Category: domain.CategoryShoes},
	} {
		if !containsNote(notes, want) {
			t.Errorf("нет заметки %s %s%s в %+v", want.Kind, want.Category, want.Item.Name, notes)
		}
	}
	if len(notes) != 2 {
		t.Errorf("заметки = %+v, ожидались только две", notes)
	}
}

func TestReview_Rain(t *testing.T) {
	var wardrobe testWardrobe
	top := *wardrobe.add("Футболка", domain.CategoryTop, domain.WarmthLevelMedium, domain.ColorWhite)
	shoes := *wardrobe.add("Кеды", domain.CategoryShoes, domain.WarmthLevelMedium, domain.ColorWhite)
	umbrella := *wardrobe.add("Зонт", domain.CategoryUmbrella, 0, domain.ColorBlack)

	tests := []struct {
		name  string
		wind  float64
		items []domain.Item
		want  []domain.NoteKind
	}{
		{"с зонтом", 3, []domain.Item{top, shoes, umbrella}, nil},
		{"без зонта", 3, []domain.Item{top, shoes}, []domain.NoteKind{domain.NoteNoRainProtection}},
		{"зонт в ветер", 12, []domain.Item{top, shoes, umbrella}, []domain.NoteKind{domain.NoteTooWindyForUmbrella, domain.NoteNoRainProtection}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			weather := feelsLike(18)
			weather.PrecipitationChanceMax = 80
			weather.WindSpeedMax = test.wind

			notes := Review(weather, test.items)

			if len(notes) != len(test.want) {
				t.Fatalf("заметки = %+v, ожидались %v", notes, test.want)
			}
			for _, kind := range test.want {
				if !containsNote(notes, domain.Note{Kind: kind}) {
					t.Errorf("нет заметки %s в %+v", kind, notes)
				}
			}
		})
	}
}

func TestReview_FrostWithoutHat(t *testing.T) {
	var wardrobe testWardrobe
	coat := *wardrobe.add("Пуховик", domain.CategoryOuterwear, domain.WarmthLevelExtreme, domain.ColorBlack)
	boots := *wardrobe.add("Ботинки", domain.CategoryShoes, domain.WarmthLevelExtreme, domain.ColorBlack)

	notes := Review(feelsLike(-15), []domain.Item{coat, boots})

	if !containsNote(notes, domain.Note{Kind: domain.NoteMissing, Category: domain.CategoryHat}) {
		t.Errorf("нет заметки о шапке: %+v", notes)
	}
}

func TestReplacements_RankedForWeather(t *testing.T) {
	var wardrobe testWardrobe
	jacket := *wardrobe.add("Куртка", domain.CategoryOuterwear, domain.WarmthLevelWarm, domain.ColorBlack)
	jeans := *wardrobe.add("Джинсы", domain.CategoryBottom, domain.WarmthLevelWarm, domain.ColorBlue)
	current := *wardrobe.add("Свитшот", domain.CategoryTop, domain.WarmthLevelMedium, domain.ColorGray)
	wardrobe.add("Майка", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorWhite)
	wardrobe.add("Вчерашняя рубашка", domain.CategoryTop, domain.WarmthLevelMedium, domain.ColorWhite)
	wardrobe.add("Рубашка", domain.CategoryTop, domain.WarmthLevelMedium, domain.ColorWhite)
	wardrobe.add("Оранжевый лонгслив", domain.CategoryTop, domain.WarmthLevelMedium, domain.ColorOrange)
	wardrobe.add("Летняя рубашка", domain.CategoryTop, domain.WarmthLevelMedium, domain.ColorWhite).Seasons = []domain.Season{domain.SeasonSummer}
	wardrobe.add("Грязная рубашка", domain.CategoryTop, domain.WarmthLevelMedium, domain.ColorWhite).Status = domain.ItemStatusDirty
	wardrobe.add("Кеды", domain.CategoryShoes, domain.WarmthLevelWarm, domain.ColorWhite)

	input := Input{
		Items:       wardrobe.items,
		Weather:     feelsLike(5),
		Season:      domain.SeasonAutumn,
		WornDaysAgo: map[int64]int{wardrobe.named("Вчерашняя рубашка").ID: 1},
	}
	candidates := Replacements(input, []domain.Item{jacket, current, jeans}, current)

	var got []string
	for _, candidate := range candidates {
		got = append(got, candidate.Item.Name)
	}
	want := []string{"Рубашка", "Оранжевый лонгслив", "Майка", "Вчерашняя рубашка", "Летняя рубашка"}
	if len(got) != len(want) {
		t.Fatalf("кандидаты = %q, ожидались %q", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("кандидаты = %q, ожидались %q", got, want)
		}
	}

	notesOf := func(name string) []domain.Note {
		for _, candidate := range candidates {
			if candidate.Item.Name == name {
				return candidate.Notes
			}
		}
		return nil
	}
	if notes := notesOf("Вчерашняя рубашка"); len(notes) != 1 || notes[0].Kind != domain.NoteWornRecently || notes[0].DaysAgo != 1 {
		t.Errorf("заметки вчерашней рубашки = %+v", notes)
	}
	if !containsNote(notesOf("Летняя рубашка"), domain.Note{Kind: domain.NoteOutOfSeason}) {
		t.Errorf("нет заметки о сезоне: %+v", notesOf("Летняя рубашка"))
	}
}

func TestAdditions_AnyCategoryExceptChosen(t *testing.T) {
	var wardrobe testWardrobe
	top := *wardrobe.add("Футболка", domain.CategoryTop, domain.WarmthLevelLight, domain.ColorWhite)
	wardrobe.add("Кеды", domain.CategoryShoes, domain.WarmthLevelLight, domain.ColorWhite)
	wardrobe.add("Сумка", domain.CategoryBag, 0, domain.ColorBlack)

	candidates := Additions(Input{Items: wardrobe.items, Weather: feelsLike(25), Season: domain.SeasonSummer}, []domain.Item{top})

	if len(candidates) != 2 {
		t.Errorf("кандидаты = %+v, ожидались кеды и сумка", candidates)
	}
}

func TestReplacements_WarnAboutRain(t *testing.T) {
	var wardrobe testWardrobe
	current := *wardrobe.add("Туфли", domain.CategoryShoes, domain.WarmthLevelMedium, domain.ColorBlack)
	wardrobe.add("Кеды", domain.CategoryShoes, domain.WarmthLevelMedium, domain.ColorWhite)
	wardrobe.add("Резиновые сапоги", domain.CategoryShoes, domain.WarmthLevelMedium, domain.ColorBlack).Waterproof = true
	weather := feelsLike(18)
	weather.PrecipitationChanceMax = 80

	candidates := Replacements(Input{Items: wardrobe.items, Weather: weather, Season: domain.SeasonAutumn}, []domain.Item{current}, current)

	if len(candidates) != 2 || candidates[0].Item.Name != "Резиновые сапоги" {
		t.Fatalf("кандидаты = %+v, первыми ожидались сапоги", candidates)
	}
	if !containsNote(candidates[1].Notes, domain.Note{Kind: domain.NoteNotWaterproof}) {
		t.Errorf("нет заметки о дожде у кед: %+v", candidates[1].Notes)
	}
}
