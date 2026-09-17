package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

const maxItems = 100

var ErrWardrobeFull = errors.New("в гардеробе достигнут лимит вещей")

type itemPhotos interface {
	Confirm(ctx context.Context, userID int64, key string) error
	Discard(ctx context.Context, keys ...string)
}

type Wardrobe struct {
	itemRepository ItemRepository
	photos         itemPhotos
}

func NewWardrobe(itemRepository ItemRepository, photos itemPhotos) *Wardrobe {
	return &Wardrobe{
		itemRepository: itemRepository,
		photos:         photos,
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

	if item.PhotoKey != "" {
		if err := w.photos.Confirm(ctx, params.UserID, item.PhotoKey); err != nil {
			return domain.Item{}, err
		}
	}

	item, err = w.itemRepository.Create(ctx, item)
	if err != nil {
		w.photos.Discard(ctx, params.PhotoKey)
		return domain.Item{}, err
	}
	return item, nil
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
	previousPhotoKey := item.PhotoKey

	item, err = item.Edit(params)
	if err != nil {
		return domain.Item{}, err
	}

	photoChanged := item.PhotoKey != previousPhotoKey
	if photoChanged && item.PhotoKey != "" {
		if err := w.photos.Confirm(ctx, userID, item.PhotoKey); err != nil {
			return domain.Item{}, err
		}
	}

	if err := w.itemRepository.Update(ctx, item); err != nil {
		if photoChanged {
			w.photos.Discard(ctx, item.PhotoKey)
		}
		return domain.Item{}, err
	}

	if photoChanged {
		w.photos.Discard(ctx, previousPhotoKey)
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
	photoKeys, err := w.itemRepository.Delete(ctx, userID, itemIDs)
	if err != nil {
		return fmt.Errorf("удаление вещей: %w", err)
	}
	w.photos.Discard(ctx, photoKeys...)
	return nil
}
