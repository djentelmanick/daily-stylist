package openmeteo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sethvargo/go-retry"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

const (
	forecastURL      = "https://api.open-meteo.com/v1/forecast"
	geocodingURL     = "https://geocoding-api.open-meteo.com/v1/search"
	requestTimeout   = 3 * time.Second
	maxResponseBytes = 1 << 20
	maxCities        = 8
	forecastCacheTTL = time.Hour

	maxAttempts = 3
	retryPause  = 200 * time.Millisecond
)

var (
	_ service.Forecaster = (*Client)(nil)
	_ service.CitySearch = (*Client)(nil)
	_ service.TimeZones  = (*Client)(nil)
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

type Client struct {
	http         *http.Client
	cache        Cache
	forecastURL  string
	geocodingURL string
}

func NewClient(cache Cache) *Client {
	return &Client{
		http:         &http.Client{Timeout: requestTimeout},
		cache:        cache,
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
	raw, err := client.cachedForecast(ctx, query)
	if err != nil {
		return domain.Weather{}, fmt.Errorf("прогноз погоды: %w", err)
	}
	var body forecastResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		return domain.Weather{}, fmt.Errorf("прогноз погоды: %w", err)
	}
	weather, err := body.Hourly.summary(from, to)
	if err != nil {
		return domain.Weather{}, fmt.Errorf("прогноз погоды: %w", err)
	}
	return weather, nil
}

func (client *Client) cachedForecast(ctx context.Context, query url.Values) ([]byte, error) {
	key := "openmeteo:forecast:" + query.Encode()
	if client.cache != nil {
		raw, found, err := client.cache.Get(ctx, key)
		if err != nil {
			log.Printf("openmeteo: %v", err)
		}
		if found {
			return raw, nil
		}
	}

	raw, err := client.fetch(ctx, client.forecastURL, query)
	if err != nil {
		return nil, err
	}
	if client.cache != nil {
		if err := client.cache.Set(ctx, key, raw, forecastCacheTTL); err != nil {
			log.Printf("openmeteo: %v", err)
		}
	}
	return raw, nil
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
		weather.PrecipitationChanceMax = max(weather.PrecipitationChanceMax, int(hourly.PrecipitationProbability[index]))
		weather.Snow = weather.Snow || hourly.Snowfall[index] > 0
		weather.WindSpeedMax = max(weather.WindSpeedMax, hourly.WindSpeed[index])
		weather.UVIndexMax = max(weather.UVIndexMax, hourly.UVIndex[index])
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

func (client *Client) TimeZoneAt(ctx context.Context, latitude, longitude float64) (string, error) {
	values := url.Values{
		"latitude":      {strconv.FormatFloat(latitude, 'f', -1, 64)},
		"longitude":     {strconv.FormatFloat(longitude, 'f', -1, 64)},
		"timezone":      {"auto"},
		"forecast_days": {"1"},
	}
	var body struct {
		Timezone string `json:"timezone"`
	}
	if err := client.get(ctx, client.forecastURL, values, &body); err != nil {
		return "", fmt.Errorf("часовой пояс: %w", err)
	}
	if body.Timezone == "" {
		return "", errors.New("часовой пояс: пустой ответ")
	}
	return body.Timezone, nil
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
	raw, err := client.fetch(ctx, endpoint, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, body)
}

func (client *Client) fetch(ctx context.Context, endpoint string, query url.Values) ([]byte, error) {
	address := endpoint + "?" + query.Encode()
	backoff := retry.WithMaxRetries(maxAttempts-1, retry.NewExponential(retryPause))

	var body []byte
	attempt := 0
	err := retry.Do(ctx, backoff, func(ctx context.Context) error {
		attempt++
		var err error
		body, err = client.attempt(ctx, address)
		if err == nil || !worthRetry(err) {
			return err
		}
		log.Printf("open-meteo: %s, попытка %d из %d: %v", endpoint, attempt, maxAttempts, err)
		return retry.RetryableError(err)
	})
	return body, err
}

func (client *Client) attempt(ctx context.Context, address string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, statusError{status: response.StatusCode, text: response.Status}
	}
	return io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
}

// Свой неправильный запрос повтором не исправить, а отказ сервиса или обрыв связи - часто да.
func worthRetry(err error) bool {
	var refused statusError
	if errors.As(err, &refused) {
		return refused.status >= http.StatusInternalServerError
	}
	return true
}

type statusError struct {
	status int
	text   string
}

func (err statusError) Error() string {
	return fmt.Sprintf("ответ %s", err.text)
}
