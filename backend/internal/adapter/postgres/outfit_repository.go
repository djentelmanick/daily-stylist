package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var _ service.OutfitRepository = (*OutfitRepository)(nil)

type OutfitRepository struct {
	pool *pgxpool.Pool
}

func NewOutfitRepository(pool *pgxpool.Pool) *OutfitRepository {
	return &OutfitRepository{pool: pool}
}

// Один запрос вместо трёх, поэтому транзакция не нужна: образ дня либо заменится
// целиком, либо останется прежним. Вещи, которые остаются в образе, не удаляются
// и не вставляются заново, а чужие вещи отсекает JOIN.
const saveWornQuery = `
WITH outfit AS (
    INSERT INTO outfits (user_id, worn_on) VALUES (@user_id, @worn_on)
    ON CONFLICT (user_id, worn_on) DO UPDATE SET worn_on = excluded.worn_on
    RETURNING id
), removed AS (
    DELETE FROM outfit_items
    WHERE outfit_id = (SELECT id FROM outfit) AND item_id <> ALL(@item_ids)
)
INSERT INTO outfit_items (outfit_id, item_id)
SELECT outfit.id, items.id
FROM outfit JOIN items ON items.user_id = @user_id AND items.id = ANY(@item_ids)
ON CONFLICT DO NOTHING`

func (repository *OutfitRepository) SaveWorn(ctx context.Context, userID int64, day time.Time, itemIDs []int64) error {
	_, err := repository.pool.Exec(ctx, saveWornQuery, pgx.NamedArgs{
		"user_id":  userID,
		"worn_on":  day,
		"item_ids": itemIDs,
	})
	if err != nil {
		return fmt.Errorf("запись образа: %w", err)
	}
	return nil
}

const wornOnQuery = `
SELECT ` + itemColumns + ` FROM items
WHERE user_id = $1 AND id IN (
    SELECT outfit_items.item_id
    FROM outfits JOIN outfit_items ON outfit_items.outfit_id = outfits.id
    WHERE outfits.user_id = $1 AND outfits.worn_on = $2
)`

func (repository *OutfitRepository) WornOn(ctx context.Context, userID int64, day time.Time) ([]domain.Item, error) {
	rows, err := repository.pool.Query(ctx, wornOnQuery, userID, day)
	if err != nil {
		return nil, fmt.Errorf("чтение образа дня: %w", err)
	}
	items, err := pgx.CollectRows(rows, scanItem)
	if err != nil {
		return nil, fmt.Errorf("чтение образа дня: %w", err)
	}
	return items, nil
}

const lastWornQuery = `
SELECT outfit_items.item_id, max(outfits.worn_on)
FROM outfits JOIN outfit_items ON outfit_items.outfit_id = outfits.id
WHERE outfits.user_id = $1 AND outfits.worn_on >= $2 AND outfits.worn_on < $3
GROUP BY outfit_items.item_id`

func (repository *OutfitRepository) LastWorn(ctx context.Context, userID int64, from, to time.Time) (map[int64]time.Time, error) {
	rows, err := repository.pool.Query(ctx, lastWornQuery, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("чтение истории: %w", err)
	}
	defer rows.Close()

	lastWorn := map[int64]time.Time{}
	for rows.Next() {
		var itemID int64
		var day time.Time
		if err := rows.Scan(&itemID, &day); err != nil {
			return nil, fmt.Errorf("чтение истории: %w", err)
		}
		lastWorn[itemID] = day
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("чтение истории: %w", err)
	}
	return lastWorn, nil
}
