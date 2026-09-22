package texts_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/texts"
)

func TestTemperature(t *testing.T) {
	tests := []struct {
		low, high float64
		want      string
	}{
		{8.4, 14.6, "+8…+15 °C"},
		{-3.6, 0.2, "−4…0 °C"},
		{-0.4, 0.4, "0 °C"},
	}
	for _, test := range tests {
		if got := texts.Temperature(domain.Weather{TemperatureMin: test.low, TemperatureMax: test.high}); got != test.want {
			t.Errorf("Temperature(%v, %v) = %q, ожидалось %q", test.low, test.high, got, test.want)
		}
	}
}

func TestWeatherDetails(t *testing.T) {
	calm := domain.Weather{TemperatureMin: 10, TemperatureMax: 12, FeelsLikeMin: 10, FeelsLikeMax: 12}
	if got := texts.WeatherDetails(calm); len(got) != 0 {
		t.Errorf("в тихую погоду подробности = %q, ожидалось пусто", got)
	}

	stormy := domain.Weather{TemperatureMin: 10, TemperatureMax: 12, FeelsLikeMin: 6, FeelsLikeMax: 9, PrecipitationChanceMax: 70, WindSpeedMax: 9.6}
	want := []string{"Ощущается как +6…+9 °C", "Дождь, вероятность 70 %", "Ветер до 10 м/с"}
	if got := texts.WeatherDetails(stormy); !slices.Equal(got, want) {
		t.Errorf("подробности = %q, ожидались %q", got, want)
	}
}

func TestNoteCoversAllKinds(t *testing.T) {
	kinds := []domain.NoteKind{
		domain.NoteMissing, domain.NoteTooLight, domain.NoteTooWarm,
		domain.NoteWarmsUp, domain.NoteNoRainProtection, domain.NoteTooWindyForUmbrella,
	}
	for _, kind := range kinds {
		if got := texts.Note(domain.Note{Kind: kind, Category: domain.CategoryShoes}); got == string(kind) {
			t.Errorf("нет текста для заметки %q", kind)
		}
	}
}

func TestMorning(t *testing.T) {
	weather := domain.Weather{TemperatureMin: 3, TemperatureMax: 8, FeelsLikeMin: 3, FeelsLikeMax: 8, PrecipitationChanceMax: 60}
	outfit := domain.Outfit{
		Items: []domain.Item{
			{Name: "Кеды", Category: domain.CategoryShoes},
			{Name: "Куртка", Category: domain.CategoryOuterwear},
		},
		Notes: []domain.Note{{Kind: domain.NoteWarmsUp}},
	}

	message := texts.Morning(domain.Location{Name: "Казань"}, weather, outfit, nil)

	for _, want := range []string{"Казань, +3…+8 °C", "Дождь, вероятность 60 %", "Днём потеплеет"} {
		if !strings.Contains(message, want) {
			t.Errorf("в сообщении нет %q:\n%s", want, message)
		}
	}
	if !strings.Contains(message, "• Куртка\n• Кеды") {
		t.Errorf("вещи не в порядке надевания:\n%s", message)
	}
}

func TestCandidateNote_WornDaysAgo(t *testing.T) {
	for days, want := range map[int]string{
		1: "Надевали вчера",
		2: "Надевали позавчера",
		3: "Надевали 3 дня назад",
		5: "Надевали 5 дней назад",
	} {
		if got := texts.CandidateNote(domain.Note{Kind: domain.NoteWornRecently, DaysAgo: days}); got != want {
			t.Errorf("%d дн.: %q, ожидалось %q", days, got, want)
		}
	}
}
