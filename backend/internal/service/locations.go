package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

const minCityQueryLength = 2

type Locations struct {
	repository LocationRepository
	cities     CitySearch
}

func NewLocations(repository LocationRepository, cities CitySearch) *Locations {
	return &Locations{repository: repository, cities: cities}
}

func (locations *Locations) SearchCities(ctx context.Context, query string) ([]domain.Location, error) {
	query = strings.TrimSpace(query)
	length := utf8.RuneCountInString(query)
	if length < minCityQueryLength || length > domain.MaxLocationNameLength {
		return nil, nil
	}

	cities, err := locations.cities.SearchCities(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("поиск города: %w: %w", ErrWeatherUnavailable, err)
	}
	return cities, nil
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
