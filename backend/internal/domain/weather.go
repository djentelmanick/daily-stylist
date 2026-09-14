package domain

import "time"

// Weather - сводка за часы, которые человек проведёт в образе.
type Weather struct {
	TemperatureMin float64
	TemperatureMax float64
	FeelsLikeMin   float64
	FeelsLikeMax   float64
	// Наибольшая за эти часы, в процентах.
	PrecipitationChance int
	Snow                bool
	// Наибольшая, м/с.
	WindSpeed float64
	UVIndex   float64
}

const (
	dayEndHour       = 22
	minForecastHours = 3
)

// ForecastWindow - часы, на которые подбирается образ: с текущего до вечера,
// а поздно вечером - хотя бы несколько ближайших.
func ForecastWindow(now time.Time) (from, to time.Time) {
	from = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	to = time.Date(now.Year(), now.Month(), now.Day(), dayEndHour, 0, 0, 0, now.Location())
	if to.Sub(from) < minForecastHours*time.Hour {
		to = from.Add(minForecastHours * time.Hour)
	}
	return from, to
}

// DateOf - календарная дата момента в его часовом поясе, записанная полночью UTC:
// так дата не сдвигается при переводе между поясами.
func DateOf(moment time.Time) time.Time {
	return time.Date(moment.Year(), moment.Month(), moment.Day(), 0, 0, 0, 0, time.UTC)
}
