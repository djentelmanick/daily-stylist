package openmeteo

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func TestForecast_SummarizesHoursInWindow(t *testing.T) {
	from := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	to := from.Add(2 * time.Hour)
	var query map[string][]string
	// Час до окна и час на его границе - с крайними значениями: в сводку они не должны попасть.
	body := `{"hourly":{
		"time":[` + unix(from.Add(-time.Hour)) + `,` + unix(from) + `,` + unix(from.Add(time.Hour)) + `,` + unix(to) + `],
		"temperature_2m":[-30, 10, 14, 40],
		"apparent_temperature":[-35, 8, 12, 45],
		"precipitation_probability":[100, 20, 60, null],
		"snowfall":[5, 0, 0, 5],
		"wind_speed_10m":[30, 4, 6, 30],
		"uv_index":[11, 2, 3, 11]
	}}`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query = request.URL.Query()
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	client := &Client{http: server.Client(), forecastURL: server.URL}

	weather, err := client.Forecast(t.Context(), domain.Location{Latitude: 43.1, Longitude: 131.9, TimeZone: "Asia/Vladivostok"}, from, to)
	if err != nil {
		t.Fatalf("Forecast: %v", err)
	}

	want := domain.Weather{
		TemperatureMin: 10, TemperatureMax: 14,
		FeelsLikeMin: 8, FeelsLikeMax: 12,
		PrecipitationChanceMax: 60,
		WindSpeedMax:           6,
		UVIndexMax:             3,
	}
	if weather != want {
		t.Errorf("сводка = %+v\nожидалась %+v", weather, want)
	}
	for name, value := range map[string]string{"latitude": "43.1", "timezone": "Asia/Vladivostok", "wind_speed_unit": "ms", "timeformat": "unixtime"} {
		if got := query[name]; len(got) != 1 || got[0] != value {
			t.Errorf("параметр %s = %v, ожидалось %q", name, got, value)
		}
	}
}

func TestForecast_FailsWithoutHoursInWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"hourly":{"time":[],"temperature_2m":[],"apparent_temperature":[],"precipitation_probability":[],"snowfall":[],"wind_speed_10m":[],"uv_index":[]}}`))
	}))
	t.Cleanup(server.Close)
	client := &Client{http: server.Client(), forecastURL: server.URL}

	now := time.Now()
	if _, err := client.Forecast(t.Context(), domain.Location{TimeZone: "UTC"}, now, now.Add(time.Hour)); err == nil {
		t.Error("пустой прогноз принят за погоду")
	}
}

func TestSearchCities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"results":[
			{"name":"Москва","latitude":55.75,"longitude":37.62,"timezone":"Europe/Moscow","country":"Россия","admin1":"Москва"},
			{"name":"Москва","latitude":46.73,"longitude":-116.99,"timezone":"America/Los_Angeles","country":"США","admin1":"Айдахо"},
			{"name":"Москва","latitude":1,"longitude":1,"country":"Нигде"}
		]}`))
	}))
	t.Cleanup(server.Close)
	client := &Client{http: server.Client(), geocodingURL: server.URL}

	cities, err := client.SearchCities(t.Context(), "Москва")
	if err != nil {
		t.Fatalf("SearchCities: %v", err)
	}

	want := []domain.Location{
		{Name: "Москва", Region: "Россия", Latitude: 55.75, Longitude: 37.62, TimeZone: "Europe/Moscow"},
		{Name: "Москва", Region: "Айдахо, США", Latitude: 46.73, Longitude: -116.99, TimeZone: "America/Los_Angeles"},
	}
	if len(cities) != len(want) {
		t.Fatalf("города = %+v, ожидались %+v", cities, want)
	}
	for index := range want {
		if cities[index] != want[index] {
			t.Errorf("город %d = %+v, ожидался %+v", index, cities[index], want[index])
		}
	}
}

func TestSearchCities_NoResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"generationtime_ms":0.5}`))
	}))
	t.Cleanup(server.Close)
	client := &Client{http: server.Client(), geocodingURL: server.URL}

	cities, err := client.SearchCities(t.Context(), "Ыыы")
	if err != nil {
		t.Fatalf("SearchCities: %v", err)
	}
	if cities == nil || len(cities) != 0 {
		t.Errorf("города = %#v, ожидался пустой список", cities)
	}
}

func unix(moment time.Time) string {
	return strconv.FormatInt(moment.Unix(), 10)
}
