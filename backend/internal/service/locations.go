package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

const MinCityQueryLength = 2

// До сотых, около километра: точное место человека нам знать незачем.
const coordinateSteps = 100

type Locations struct {
	repository LocationRepository
	cities     CitySearch
	places     PlaceNames
	zones      TimeZones
}

func NewLocations(repository LocationRepository, cities CitySearch, places PlaceNames, zones TimeZones) *Locations {
	return &Locations{repository: repository, cities: cities, places: places, zones: zones}
}

func (locations *Locations) SearchCities(ctx context.Context, query string) ([]domain.Location, error) {
	query = strings.TrimSpace(query)
	length := utf8.RuneCountInString(query)
	if length < MinCityQueryLength || length > domain.MaxLocationNameLength {
		return nil, nil
	}

	cities, err := locations.cities.SearchCities(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("поиск города: %w: %w", ErrWeatherUnavailable, err)
	}
	return cities, nil
}

func (locations *Locations) CityAt(ctx context.Context, latitude, longitude float64) (domain.Location, error) {
	location := domain.Location{Latitude: roundCoordinate(latitude), Longitude: roundCoordinate(longitude)}
	if err := location.ValidateCoordinates(); err != nil {
		return domain.Location{}, fmt.Errorf("город по координатам: %w", err)
	}

	name, region, err := locations.places.PlaceAt(ctx, location.Latitude, location.Longitude)
	if errors.Is(err, ErrPlaceNotFound) {
		return domain.Location{}, fmt.Errorf("город по координатам: %w", err)
	}
	if err != nil {
		return domain.Location{}, fmt.Errorf("город по координатам: %w: %w", ErrWeatherUnavailable, err)
	}
	zone, err := locations.zones.TimeZoneAt(ctx, location.Latitude, location.Longitude)
	if err != nil {
		return domain.Location{}, fmt.Errorf("часовой пояс по координатам: %w: %w", ErrWeatherUnavailable, err)
	}

	location.Name, location.Region, location.TimeZone = strings.TrimSpace(name), strings.TrimSpace(region), zone
	if err := location.Validate(); err != nil {
		return domain.Location{}, fmt.Errorf("город по координатам: %w", err)
	}
	return location, nil
}

func roundCoordinate(value float64) float64 {
	// Деление, а не умножение на 0.01: так получается ровно 55.79, а не 55.790000000000006.
	return math.Round(value*coordinateSteps) / coordinateSteps
}

func (locations *Locations) SetLocation(ctx context.Context, userID int64, location domain.Location) error {
	location.Name = strings.TrimSpace(location.Name)
	location.Region = strings.TrimSpace(location.Region)
	if err := location.Validate(); err != nil {
		return fmt.Errorf("выбор города: %w", err)
	}
	if err := locations.repository.Save(ctx, userID, location); err != nil {
		return fmt.Errorf("выбор города: %w", err)
	}
	return nil
}

func (locations *Locations) City(ctx context.Context, userID int64) (domain.Location, error) {
	location, err := locations.repository.Get(ctx, userID)
	if err != nil {
		return domain.Location{}, fmt.Errorf("город пользователя: %w", err)
	}
	return location, nil
}
