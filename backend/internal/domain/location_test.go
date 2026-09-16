package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func TestLocation_Validate(t *testing.T) {
	valid := domain.Location{Name: "Казань", Region: "Татарстан, Россия", Latitude: 55.79, Longitude: 49.12, TimeZone: "Europe/Moscow"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate валидного места: %v", err)
	}

	tests := []struct {
		name   string
		change func(location *domain.Location)
	}{
		{"пустое название", func(location *domain.Location) { location.Name = " " }},
		{"широта больше 90", func(location *domain.Location) { location.Latitude = 91 }},
		{"долгота меньше -180", func(location *domain.Location) { location.Longitude = -181 }},
		{"без часового пояса", func(location *domain.Location) { location.TimeZone = "" }},
		{"пояс сервера", func(location *domain.Location) { location.TimeZone = "Local" }},
		{"неизвестный пояс", func(location *domain.Location) { location.TimeZone = "Mars/Olympus" }},
	}
	for _, test := range tests {
		location := valid
		test.change(&location)
		if err := location.Validate(); !errors.Is(err, domain.ErrInvalidLocation) {
			t.Errorf("%s: ошибка = %v, ожидалась ErrInvalidLocation", test.name, err)
		}
	}
}

func TestForecastWindow(t *testing.T) {
	zone := time.FixedZone("UTC+3", 3*60*60)
	tests := []struct {
		name     string
		now      time.Time
		from, to time.Time
	}{
		{"утро", time.Date(2026, 9, 15, 7, 40, 0, 0, zone), time.Date(2026, 9, 15, 7, 0, 0, 0, zone), time.Date(2026, 9, 15, 22, 0, 0, 0, zone)},
		{"поздний вечер", time.Date(2026, 9, 15, 21, 15, 0, 0, zone), time.Date(2026, 9, 15, 21, 0, 0, 0, zone), time.Date(2026, 9, 16, 0, 0, 0, 0, zone)},
	}
	for _, test := range tests {
		from, to := domain.ForecastWindow(test.now)
		if !from.Equal(test.from) || !to.Equal(test.to) {
			t.Errorf("%s: окно %v - %v, ожидалось %v - %v", test.name, from, to, test.from, test.to)
		}
	}
}
