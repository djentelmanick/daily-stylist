package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

const maxItems = 100

var ErrWardrobeFull = errors.New("в гардеробе достигнут лимит вещей")

type Wardrobe struct {
	itemRepository ItemRepository
}

func NewWardrobe(itemRepository ItemRepository) *Wardrobe {
	return &Wardrobe{
		itemRepository: itemRepository,
	}
}

func (w *Wardrobe) AddItem(ctx context.Context, params domain.NewItemParams) (item domain.Item, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("добавление вещи: %w", err)
		}
	}()

	// TODO: гонка. Два одновременных добавления при 99 вещах оба увидят 99,
	// и вещей станет 101. Строгий лимит - ограничение в базе или транзакция.
	itemCount, err := w.itemRepository.CountByUser(ctx, params.UserID)
	if err != nil {
		return domain.Item{}, err
	}
	if itemCount >= maxItems {
		return domain.Item{}, ErrWardrobeFull
	}

	item, err = domain.NewItem(params)
	if err != nil {
		return domain.Item{}, err
	}

	return w.itemRepository.Create(ctx, item)
}
