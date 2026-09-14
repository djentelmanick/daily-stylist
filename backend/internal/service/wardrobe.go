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

func (w *Wardrobe) Items(ctx context.Context, userID int64) ([]domain.Item, error) {
	items, err := w.itemRepository.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("список вещей: %w", err)
	}
	return items, nil
}

func (w *Wardrobe) EditItem(ctx context.Context, userID, itemID int64, params domain.EditItemParams) (item domain.Item, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("изменение вещи %d: %w", itemID, err)
		}
	}()

	item, err = w.itemRepository.Get(ctx, userID, itemID)
	if err != nil {
		return domain.Item{}, err
	}
	item, err = item.Edit(params)
	if err != nil {
		return domain.Item{}, err
	}
	if err := w.itemRepository.Update(ctx, item); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func (w *Wardrobe) ChangeItemStatus(ctx context.Context, userID, itemID int64, status domain.ItemStatus) error {
	if !status.Valid() {
		return fmt.Errorf("смена статуса вещи %d: %w: неизвестный статус %q", itemID, domain.ErrInvalidItem, status)
	}
	if err := w.itemRepository.UpdateStatus(ctx, userID, itemID, status); err != nil {
		return fmt.Errorf("смена статуса вещи %d: %w", itemID, err)
	}
	return nil
}

func (w *Wardrobe) DeleteItems(ctx context.Context, userID int64, itemIDs []int64) error {
	if len(itemIDs) == 0 {
		return nil
	}
	if err := w.itemRepository.Delete(ctx, userID, itemIDs); err != nil {
		return fmt.Errorf("удаление вещей: %w", err)
	}
	return nil
}
