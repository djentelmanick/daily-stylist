package texts

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain/rules"
)

// О дожде и ветре сообщение предупреждает раньше, чем на них реагирует подбор.
const (
	showPrecipitationFrom = 30
	showWindFrom          = 5.0
)

func Temperature(weather domain.Weather) string {
	return temperatureRange(weather.TemperatureMin, weather.TemperatureMax)
}

func WeatherDetails(weather domain.Weather) []string {
	details := []string{}
	if feelsLike := temperatureRange(weather.FeelsLikeMin, weather.FeelsLikeMax); feelsLike != Temperature(weather) {
		details = append(details, "Ощущается как "+feelsLike)
	}
	switch {
	case weather.Snow:
		details = append(details, fmt.Sprintf("Снег, вероятность %d %%", weather.PrecipitationChanceMax))
	case weather.PrecipitationChanceMax >= showPrecipitationFrom:
		details = append(details, fmt.Sprintf("Дождь, вероятность %d %%", weather.PrecipitationChanceMax))
	}
	if weather.WindSpeedMax >= showWindFrom {
		details = append(details, fmt.Sprintf("Ветер до %d м/с", int(math.Round(weather.WindSpeedMax))))
	}
	if weather.UVIndexMax >= rules.StrongUVIndex {
		details = append(details, "Сильное солнце")
	}
	return details
}

func Note(note domain.Note) string {
	switch note.Kind {
	case domain.NoteMissing:
		return fmt.Sprintf("Нет доступной вещи в категории «%s»", Category(note.Category))
	case domain.NoteTooLight:
		return fmt.Sprintf("«%s» легче, чем нужно по погоде", note.Item.Name)
	case domain.NoteTooWarm:
		return fmt.Sprintf("«%s» теплее, чем нужно по погоде", note.Item.Name)
	case domain.NoteWarmsUp:
		return "Днём потеплеет: верхнюю одежду можно будет снять"
	case domain.NoteNoRainProtection:
		return "Обещают дождь, а зонта и непромокаемой верхней одежды нет"
	case domain.NoteTooWindyForUmbrella:
		return "Сильный ветер: зонт не спасёт, лучше непромокаемая куртка"
	default:
		return string(note.Kind)
	}
}

func temperatureRange(low, high float64) string {
	if degrees(low) == degrees(high) {
		return degrees(low) + " °C"
	}
	return degrees(low) + "…" + degrees(high) + " °C"
}

func degrees(value float64) string {
	rounded := int(math.Round(value))
	switch {
	case rounded > 0:
		return fmt.Sprintf("+%d", rounded)
	case rounded < 0:
		return fmt.Sprintf("−%d", -rounded)
	default:
		return "0"
	}
}

func Morning(location domain.Location, weather domain.Weather, outfit domain.Outfit, notes []domain.Note) string {
	lines := []string{"Доброе утро!", "", location.Name + ", " + Temperature(weather)}
	lines = append(lines, WeatherDetails(weather)...)

	items := slices.Clone(outfit.Items)
	domain.SortForWearing(items)
	lines = append(lines, "", "Что надеть:")
	for _, item := range items {
		lines = append(lines, "• "+item.Name)
	}

	for _, note := range slices.Concat(notes, outfit.Notes) {
		lines = append(lines, "", Note(note))
	}
	return strings.Join(lines, "\n")
}
