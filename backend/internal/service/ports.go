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
}
