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

var _ service.ItemRepository = (*ItemRepository)(nil)

type ItemRepository struct {
	pool *pgxpool.Pool
}

func NewItemRepository(pool *pgxpool.Pool) *ItemRepository {
	return &ItemRepository{pool: pool}
}

const createItemQuery = `
INSERT INTO items (user_id, name, description, category, main_color, extra_colors, seasons, warmth_level, waterproof, status)
VALUES (@user_id, @name, @description, @category, @main_color, @extra_colors, @seasons, @warmth_level, @waterproof, @status)
RETURNING id`

func (repository *ItemRepository) Create(ctx context.Context, item domain.Item) (domain.Item, error) {
	err := repository.pool.QueryRow(ctx, createItemQuery, pgx.NamedArgs{
		"user_id":      item.UserID,
		"name":         item.Name,
		"description":  item.Description,
		"category":     item.Category,
		"main_color":   item.Colors.Main,
		"extra_colors": extraColors(item),
		"seasons":      item.Seasons,
		"warmth_level": item.WarmthLevel,
		"waterproof":   item.Waterproof,
		"status":       item.Status,
	}).Scan(&item.ID)
	if err != nil {
		return domain.Item{}, fmt.Errorf("вставка вещи: %w", err)
	}
	return item, nil
}

const countItemsByUserQuery = `SELECT count(*) FROM items WHERE user_id = $1`

func (repository *ItemRepository) CountByUser(ctx context.Context, userID int64) (int, error) {
	var count int
	if err := repository.pool.QueryRow(ctx, countItemsByUserQuery, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("подсчёт вещей: %w", err)
	}
	return count, nil
}

const itemColumns = `id, user_id, name, description, category, main_color, extra_colors, seasons, warmth_level, waterproof, status`

const listItemsByUserQuery = `SELECT ` + itemColumns + ` FROM items WHERE user_id = $1 ORDER BY id DESC`

func (repository *ItemRepository) ListByUser(ctx context.Context, userID int64) ([]domain.Item, error) {
	rows, err := repository.pool.Query(ctx, listItemsByUserQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("чтение вещей: %w", err)
	}
	items, err := pgx.CollectRows(rows, scanItem)
	if err != nil {
		return nil, fmt.Errorf("чтение вещей: %w", err)
	}
	return items, nil
}

const getItemQuery = `SELECT ` + itemColumns + ` FROM items WHERE id = $1 AND user_id = $2`

func (repository *ItemRepository) Get(ctx context.Context, userID, itemID int64) (domain.Item, error) {
	rows, err := repository.pool.Query(ctx, getItemQuery, itemID, userID)
	if err != nil {
		return domain.Item{}, fmt.Errorf("чтение вещи: %w", err)
	}
	item, err := pgx.CollectExactlyOneRow(rows, scanItem)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Item{}, service.ErrItemNotFound
	}
	if err != nil {
		return domain.Item{}, fmt.Errorf("чтение вещи: %w", err)
	}
	return item, nil
}

const updateItemQuery = `
UPDATE items
SET name = @name, description = @description, category = @category, main_color = @main_color,
    extra_colors = @extra_colors, seasons = @seasons, warmth_level = @warmth_level, waterproof = @waterproof
WHERE id = @id AND user_id = @user_id`

func (repository *ItemRepository) Update(ctx context.Context, item domain.Item) error {
	tag, err := repository.pool.Exec(ctx, updateItemQuery, pgx.NamedArgs{
		"id":           item.ID,
		"user_id":      item.UserID,
		"name":         item.Name,
		"description":  item.Description,
		"category":     item.Category,
		"main_color":   item.Colors.Main,
		"extra_colors": extraColors(item),
		"seasons":      item.Seasons,
		"warmth_level": item.WarmthLevel,
		"waterproof":   item.Waterproof,
	})
	if err != nil {
		return fmt.Errorf("обновление вещи: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return service.ErrItemNotFound
	}
	return nil
}

const updateItemStatusQuery = `UPDATE items SET status = $1 WHERE id = $2 AND user_id = $3`

func (repository *ItemRepository) UpdateStatus(ctx context.Context, userID, itemID int64, status domain.ItemStatus) error {
	tag, err := repository.pool.Exec(ctx, updateItemStatusQuery, status, itemID, userID)
	if err != nil {
		return fmt.Errorf("смена статуса: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return service.ErrItemNotFound
	}
	return nil
}

const deleteItemsQuery = `DELETE FROM items WHERE user_id = $1 AND id = ANY($2)`

func (repository *ItemRepository) Delete(ctx context.Context, userID int64, itemIDs []int64) error {
	if _, err := repository.pool.Exec(ctx, deleteItemsQuery, userID, itemIDs); err != nil {
		return fmt.Errorf("удаление вещей: %w", err)
	}
	return nil
}

func scanItem(row pgx.CollectableRow) (domain.Item, error) {
	var item domain.Item
	err := row.Scan(&item.ID, &item.UserID, &item.Name, &item.Description, &item.Category, &item.Colors.Main,
		&item.Colors.Extra, &item.Seasons, &item.WarmthLevel, &item.Waterproof, &item.Status)
	if err != nil {
		return domain.Item{}, err
	}
	// Ошибка без %w: испорченная строка в базе - сбой хранилища, а не
	// невалидный ввод, и снаружи её не должны принять за ErrInvalidItem.
	if err := item.Validate(); err != nil {
		return domain.Item{}, fmt.Errorf("вещь %d в базе не проходит проверку: %v", item.ID, err)
	}
	return item, nil
}

// pgx пишет nil-срез как NULL, а NULL в колонке запрещён:
// запрос «вещи без чёрного» молча пропустил бы такую вещь.
func extraColors(item domain.Item) []domain.Color {
	if item.Colors.Extra == nil {
		return []domain.Color{}
	}
	return item.Colors.Extra
}
