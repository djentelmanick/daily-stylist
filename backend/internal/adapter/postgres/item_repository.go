package postgres

import (
	"context"
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
	extraColors := item.Colors.Extra
	if extraColors == nil {
		extraColors = []domain.Color{}
	}

	err := repository.pool.QueryRow(ctx, createItemQuery, pgx.NamedArgs{
		"user_id":      item.UserID,
		"name":         item.Name,
		"description":  item.Description,
		"category":     item.Category,
		"main_color":   item.Colors.Main,
		"extra_colors": extraColors,
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
