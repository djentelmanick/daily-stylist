package openmeteo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

var moscow = domain.Location{Name: "Москва", Latitude: 55.75, Longitude: 37.62, TimeZone: "Europe/Moscow"}

func TestForecast_AsksOnceForTheSameCity(t *testing.T) {
	requests := 0
	client, from := clientWithForecast(t, &requests, &memoryCache{})

	for _, window := range []time.Duration{0, time.Hour} {
		if _, err := client.Forecast(t.Context(), moscow, from.Add(window), from.Add(window+time.Hour)); err != nil {
			t.Fatalf("Forecast: %v", err)
		}
	}

	if requests != 1 {
		t.Errorf("запросов к Open-Meteo %d, ожидался один: второе окно считается из кэша", requests)
	}
}

func TestForecast_CacheLivesAnHour(t *testing.T) {
	requests := 0
	cache := &memoryCache{}
	client, from := clientWithForecast(t, &requests, cache)

	if _, err := client.Forecast(t.Context(), moscow, from, from.Add(time.Hour)); err != nil {
		t.Fatalf("Forecast: %v", err)
	}

	if cache.ttl != forecastCacheTTL {
		t.Errorf("срок записи %s, ожидался %s", cache.ttl, forecastCacheTTL)
	}
}

func TestForecast_WorksWhenCacheIsDown(t *testing.T) {
	requests := 0
	client, from := clientWithForecast(t, &requests, &memoryCache{err: errors.New("соединение сброшено")})

	for range 2 {
		if _, err := client.Forecast(t.Context(), moscow, from, from.Add(time.Hour)); err != nil {
			t.Fatalf("Forecast без кэша: %v", err)
		}
	}

	if requests != 2 {
		t.Errorf("запросов %d, ожидалось по одному на каждый подбор", requests)
	}
}

func clientWithForecast(t *testing.T, requests *int, cache Cache) (*Client, time.Time) {
	t.Helper()

	from := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	body := `{"hourly":{
		"time":[` + unix(from) + `,` + unix(from.Add(time.Hour)) + `],
		"temperature_2m":[10, 12],
		"apparent_temperature":[9, 11],
		"precipitation_probability":[0, 0],
		"snowfall":[0, 0],
		"wind_speed_10m":[2, 3],
		"uv_index":[1, 2]
	}}`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		*requests++
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return &Client{http: server.Client(), cache: cache, forecastURL: server.URL}, from
}

type memoryCache struct {
	values map[string][]byte
	ttl    time.Duration
	err    error
}

func (cache *memoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	if cache.err != nil {
		return nil, false, cache.err
	}
	value, found := cache.values[key]
	return value, found, nil
}

func (cache *memoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if cache.err != nil {
		return cache.err
	}
	if cache.values == nil {
		cache.values = map[string][]byte{}
	}
	cache.values[key] = value
	cache.ttl = ttl
	return nil
}
