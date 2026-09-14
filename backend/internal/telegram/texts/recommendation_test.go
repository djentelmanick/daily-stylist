package texts_test

import (
	"slices"
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

	stormy := domain.Weather{TemperatureMin: 10, TemperatureMax: 12, FeelsLikeMin: 6, FeelsLikeMax: 9, PrecipitationChance: 70, WindSpeed: 9.6}
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
