package service

import (
	"context"
	"errors"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

var ErrItemNotFound = errors.New("вещь не найдена")

type ItemRepository interface {
	Create(ctx context.Context, item domain.Item) (domain.Item, error)
	CountByUser(ctx context.Context, userID int64) (int, error)
	ListByUser(ctx context.Context, userID int64) ([]domain.Item, error)
	Get(ctx context.Context, userID, itemID int64) (domain.Item, error)
	Update(ctx context.Context, item domain.Item) error
	UpdateStatus(ctx context.Context, userID, itemID int64, status domain.ItemStatus) error
	Delete(ctx context.Context, userID int64, itemIDs []int64) error
}
