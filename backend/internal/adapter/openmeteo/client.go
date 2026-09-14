// Package openmeteo - погода и поиск городов через Open-Meteo: бесплатно и без ключа.
package openmeteo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

const (
	forecastURL      = "https://api.open-meteo.com/v1/forecast"
	geocodingURL     = "https://geocoding-api.open-meteo.com/v1/search"
	requestTimeout   = 5 * time.Second
	maxResponseBytes = 1 << 20
	maxCities        = 8
)

var (
	_ service.Forecaster = (*Client)(nil)
	_ service.CitySearch = (*Client)(nil)
)

type Client struct {
	http         *http.Client
	forecastURL  string
	geocodingURL string
}

func NewClient() *Client {
	return &Client{
		http:         &http.Client{Timeout: requestTimeout},
		forecastURL:  forecastURL,
		geocodingURL: geocodingURL,
	}
}

type forecastResponse struct {
	Hourly hourlyForecast `json:"hourly"`
}

type hourlyForecast struct {
	Time                     []int64   `json:"time"`
	Temperature              []float64 `json:"temperature_2m"`
	ApparentTemperature      []float64 `json:"apparent_temperature"`
	PrecipitationProbability []float64 `json:"precipitation_probability"`
	Snowfall                 []float64 `json:"snowfall"`
	WindSpeed                []float64 `json:"wind_speed_10m"`
	UVIndex                  []float64 `json:"uv_index"`
}

func (client *Client) Forecast(ctx context.Context, location domain.Location, from, to time.Time) (domain.Weather, error) {
	query := url.Values{
		"latitude":        {strconv.FormatFloat(location.Latitude, 'f', -1, 64)},
		"longitude":       {strconv.FormatFloat(location.Longitude, 'f', -1, 64)},
		"hourly":          {"temperature_2m,apparent_temperature,precipitation_probability,snowfall,wind_speed_10m,uv_index"},
		"wind_speed_unit": {"ms"},
		"timeformat":      {"unixtime"},
		// Сутки прогноза начинаются в полночь по времени города, а поздно вечером окно
		// заходит на следующие сутки - поэтому двое.
		"timezone":      {location.TimeZone},
		"forecast_days": {"2"},
	}
	var body forecastResponse
	if err := client.get(ctx, client.forecastURL, query, &body); err != nil {
		return domain.Weather{}, fmt.Errorf("прогноз погоды: %w", err)
	}
	weather, err := body.Hourly.summary(from, to)
	if err != nil {
		return domain.Weather{}, fmt.Errorf("прогноз погоды: %w", err)
	}
	return weather, nil
}

func (hourly hourlyForecast) summary(from, to time.Time) (domain.Weather, error) {
	for _, series := range [][]float64{hourly.Temperature, hourly.ApparentTemperature, hourly.PrecipitationProbability, hourly.Snowfall, hourly.WindSpeed, hourly.UVIndex} {
		if len(series) != len(hourly.Time) {
			return domain.Weather{}, errors.New("в прогнозе ряды разной длины")
		}
	}

	var weather domain.Weather
	hours := 0
	for index, unixTime := range hourly.Time {
		hour := time.Unix(unixTime, 0)
		if hour.Before(from) || !hour.Before(to) {
			continue
		}
		temperature, feelsLike := hourly.Temperature[index], hourly.ApparentTemperature[index]
		if hours == 0 {
			weather.TemperatureMin, weather.TemperatureMax = temperature, temperature
			weather.FeelsLikeMin, weather.FeelsLikeMax = feelsLike, feelsLike
		}
		hours++

		weather.TemperatureMin = min(weather.TemperatureMin, temperature)
		weather.TemperatureMax = max(weather.TemperatureMax, temperature)
		weather.FeelsLikeMin = min(weather.FeelsLikeMin, feelsLike)
		weather.FeelsLikeMax = max(weather.FeelsLikeMax, feelsLike)
		weather.PrecipitationChance = max(weather.PrecipitationChance, int(hourly.PrecipitationProbability[index]))
		weather.Snow = weather.Snow || hourly.Snowfall[index] > 0
		weather.WindSpeed = max(weather.WindSpeed, hourly.WindSpeed[index])
		weather.UVIndex = max(weather.UVIndex, hourly.UVIndex[index])
	}
	if hours == 0 {
		return domain.Weather{}, fmt.Errorf("в прогнозе нет часов с %s по %s", from.Format(time.RFC3339), to.Format(time.RFC3339))
	}
	return weather, nil
}

type searchResponse struct {
	Results []struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Timezone  string  `json:"timezone"`
		Country   string  `json:"country"`
		Admin1    string  `json:"admin1"`
	} `json:"results"`
}

func (client *Client) SearchCities(ctx context.Context, query string) ([]domain.Location, error) {
	values := url.Values{
		"name":     {query},
		"count":    {strconv.Itoa(maxCities)},
		"language": {"ru"},
		"format":   {"json"},
	}
	var body searchResponse
	if err := client.get(ctx, client.geocodingURL, values, &body); err != nil {
		return nil, fmt.Errorf("поиск города: %w", err)
	}

	cities := []domain.Location{}
	for _, result := range body.Results {
		city := domain.Location{
			Name:      result.Name,
			Region:    region(result.Name, result.Admin1, result.Country),
			Latitude:  result.Latitude,
			Longitude: result.Longitude,
			TimeZone:  result.Timezone,
		}
		// Бывают места без часового пояса: без него не понять, когда у человека утро.
		if city.Validate() == nil {
			cities = append(cities, city)
		}
	}
	return cities, nil
}

// region отличает одноимённые города. Область с названием города
// («Москва, Москва») ничего не добавляет и пропускается.
func region(name string, parts ...string) string {
	var kept []string
	for _, part := range parts {
		if part != "" && part != name {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, ", ")
}

func (client *Client) get(ctx context.Context, endpoint string, query url.Values, body any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	response, err := client.http.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("ответ %s", response.Status)
	}
	return json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(body)
}
