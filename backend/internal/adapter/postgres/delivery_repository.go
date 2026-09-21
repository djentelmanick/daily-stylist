package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var _ service.DeliveryRepository = (*DeliveryRepository)(nil)

type DeliveryRepository struct {
	pool *pgxpool.Pool
}

func NewDeliveryRepository(pool *pgxpool.Pool) *DeliveryRepository {
	return &DeliveryRepository{pool: pool}
}

// Время рассылки у каждого своё и считается в его поясе, поэтому местный момент
// вычисляет база: в Go пришлось бы вычитать всех пользователей.
const dueQuery = `
WITH moments AS (
    SELECT locations.user_id,
           (@now AT TIME ZONE locations.timezone) AS moment,
           coalesce(settings.morning_enabled, @default_enabled) AS enabled,
           coalesce(settings.send_at, @default_send_at) AS send_at
    FROM locations LEFT JOIN settings ON settings.user_id = locations.user_id
)
SELECT user_id, moment::date
FROM moments
WHERE enabled
  AND moment >= date_trunc('day', moment) + send_at
  AND moment < date_trunc('day', moment) + send_at + make_interval(mins => @window_minutes)
  AND NOT EXISTS (
      SELECT 1 FROM morning_deliveries
      WHERE morning_deliveries.user_id = moments.user_id AND morning_deliveries.day = moment::date
  )
ORDER BY user_id`

func (repository *DeliveryRepository) Due(ctx context.Context, params service.DueParams) ([]service.MorningDelivery, error) {
	rows, err := repository.pool.Query(ctx, dueQuery, pgx.NamedArgs{
		"now":             params.Now,
		"default_enabled": params.Defaults.MorningEnabled,
		"default_send_at": fromDayTime(params.Defaults.SendAt),
		"window_minutes":  int(params.Window / time.Minute),
	})
	if err != nil {
		return nil, fmt.Errorf("кому пора слать рекомендацию: %w", err)
	}

	due, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (service.MorningDelivery, error) {
		var delivery service.MorningDelivery
		err := row.Scan(&delivery.UserID, &delivery.Day)
		return delivery, err
	})
	if err != nil {
		return nil, fmt.Errorf("кому пора слать рекомендацию: %w", err)
	}
	return due, nil
}

// Старые отметки убирает тот же, кто их создаёт: отдельная задача ради этого не нужна.
// Пара дней в запасе - на случай переезда в другой пояс, когда местный день сдвигается назад.
const keepDeliveryDays = 2

const claimDeliveryQuery = `
WITH cleanup AS (
    DELETE FROM morning_deliveries WHERE user_id = $1 AND day < $2::date - $3::int
)
INSERT INTO morning_deliveries (user_id, day) VALUES ($1, $2) ON CONFLICT DO NOTHING`

func (repository *DeliveryRepository) Claim(ctx context.Context, delivery service.MorningDelivery) (bool, error) {
	tag, err := repository.pool.Exec(ctx, claimDeliveryQuery, delivery.UserID, delivery.Day, keepDeliveryDays)
	if err != nil {
		return false, fmt.Errorf("отметка о рассылке: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

const releaseDeliveryQuery = `DELETE FROM morning_deliveries WHERE user_id = $1 AND day = $2`

func (repository *DeliveryRepository) Release(ctx context.Context, delivery service.MorningDelivery) error {
	if _, err := repository.pool.Exec(ctx, releaseDeliveryQuery, delivery.UserID, delivery.Day); err != nil {
		return fmt.Errorf("снятие отметки о рассылке: %w", err)
	}
	return nil
}
