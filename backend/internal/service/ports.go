package service

import (
	"context"
	"errors"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

var (
	ErrItemNotFound   = errors.New("вещь не найдена")
	ErrLocationNotSet = errors.New("город не выбран")
)

type ItemRepository interface {
	Create(ctx context.Context, item domain.Item) (domain.Item, error)
	CountByUser(ctx context.Context, userID int64) (int, error)
	ListByUser(ctx context.Context, userID int64) ([]domain.Item, error)
	Get(ctx context.Context, userID, itemID int64) (domain.Item, error)
	Update(ctx context.Context, item domain.Item) error
	UpdateStatus(ctx context.Context, userID, itemID int64, status domain.ItemStatus) error
	Delete(ctx context.Context, userID int64, itemIDs []int64) error
}

type LocationRepository interface {
	Get(ctx context.Context, userID int64) (domain.Location, error)
	Save(ctx context.Context, userID int64, location domain.Location) error
}

// Дни - даты из domain.DateOf.
type OutfitRepository interface {
	// SaveWorn заменяет образ дня: если человек передумал, в истории останется последний.
	SaveWorn(ctx context.Context, userID int64, day time.Time, itemIDs []int64) error
	WornOn(ctx context.Context, userID int64, day time.Time) ([]domain.Item, error)
	// LastWorn - последний день, когда надевали каждую вещь, среди дней [from, to).
	LastWorn(ctx context.Context, userID int64, from, to time.Time) (map[int64]time.Time, error)
}

type Forecaster interface {
	Forecast(ctx context.Context, location domain.Location, from, to time.Time) (domain.Weather, error)
}

type CitySearch interface {
	SearchCities(ctx context.Context, query string) ([]domain.Location, error)
}
