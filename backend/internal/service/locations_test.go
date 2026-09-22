package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func TestLocations_CityAt(t *testing.T) {
	places := &fakePlaces{name: "Казань", region: "Татарстан, Россия", zone: "Europe/Moscow"}
	locations := service.NewLocations(fakeLocations{}, nil, places, places)

	city, err := locations.CityAt(t.Context(), 55.794_649, 49.111_502)
	if err != nil {
		t.Fatalf("CityAt: %v", err)
	}

	want := domain.Location{Name: "Казань", Region: "Татарстан, Россия", Latitude: 55.79, Longitude: 49.11, TimeZone: "Europe/Moscow"}
	if city != want {
		t.Errorf("город = %+v, ожидался %+v", city, want)
	}
	if places.latitude != 55.79 || places.longitude != 49.11 {
		t.Errorf("спросили %v, %v: точные координаты не должны уходить наружу", places.latitude, places.longitude)
	}
}

func TestLocations_CityAtFailures(t *testing.T) {
	tests := []struct {
		name      string
		latitude  float64
		places    *fakePlaces
		wantError error
	}{
		{"вне диапазона", 91, &fakePlaces{name: "Где-то", zone: "Europe/Moscow"}, domain.ErrInvalidLocation},
		{"посреди моря", 43, &fakePlaces{placeErr: service.ErrPlaceNotFound}, service.ErrPlaceNotFound},
		{"сервис недоступен", 43, &fakePlaces{placeErr: errors.New("таймаут")}, service.ErrWeatherUnavailable},
		{"нет пояса", 43, &fakePlaces{name: "Где-то", zoneErr: errors.New("таймаут")}, service.ErrWeatherUnavailable},
		{"неизвестный пояс", 43, &fakePlaces{name: "Где-то", zone: "Mars/Olympus"}, domain.ErrInvalidLocation},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			locations := service.NewLocations(fakeLocations{}, nil, test.places, test.places)

			if _, err := locations.CityAt(t.Context(), test.latitude, 30); !errors.Is(err, test.wantError) {
				t.Errorf("ошибка = %v, ожидалась %v", err, test.wantError)
			}
		})
	}
}

type fakePlaces struct {
	name, region        string
	zone                string
	placeErr, zoneErr   error
	latitude, longitude float64
}

func (places *fakePlaces) PlaceAt(_ context.Context, latitude, longitude float64) (string, string, error) {
	places.latitude, places.longitude = latitude, longitude
	return places.name, places.region, places.placeErr
}

func (places *fakePlaces) TimeZoneAt(context.Context, float64, float64) (string, error) {
	return places.zone, places.zoneErr
}
