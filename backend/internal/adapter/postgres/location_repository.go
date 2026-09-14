package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var _ service.LocationRepository = (*LocationRepository)(nil)

type LocationRepository struct {
	pool *pgxpool.Pool
}

func NewLocationRepository(pool *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{pool: pool}
}

const getLocationQuery = `SELECT name, region, latitude, longitude, timezone FROM locations WHERE user_id = $1`

func (repository *LocationRepository) Get(ctx context.Context, userID int64) (domain.Location, error) {
	var location domain.Location
	err := repository.pool.QueryRow(ctx, getLocationQuery, userID).
		Scan(&location.Name, &location.Region, &location.Latitude, &location.Longitude, &location.TimeZone)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Location{}, service.ErrLocationNotSet
	}
	if err != nil {
		return domain.Location{}, fmt.Errorf("чтение города: %w", err)
	}
	if err := location.Validate(); err != nil {
		return domain.Location{}, fmt.Errorf("город пользователя %d в базе не проходит проверку: %v", userID, err)
	}
	return location, nil
}

const saveLocationQuery = `
INSERT INTO locations (user_id, name, region, latitude, longitude, timezone)
VALUES (@user_id, @name, @region, @latitude, @longitude, @timezone)
ON CONFLICT (user_id) DO UPDATE
SET name = excluded.name, region = excluded.region, latitude = excluded.latitude,
    longitude = excluded.longitude, timezone = excluded.timezone`

func (repository *LocationRepository) Save(ctx context.Context, userID int64, location domain.Location) error {
	_, err := repository.pool.Exec(ctx, saveLocationQuery, pgx.NamedArgs{
		"user_id":   userID,
		"name":      location.Name,
		"region":    location.Region,
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
		"timezone":  location.TimeZone,
	})
	if err != nil {
		return fmt.Errorf("сохранение города: %w", err)
	}
	return nil
}
