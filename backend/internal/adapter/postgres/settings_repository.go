package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var _ service.SettingsRepository = (*SettingsRepository)(nil)

type SettingsRepository struct {
	pool *pgxpool.Pool
}

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

const getSettingsQuery = `SELECT morning_enabled, send_at FROM settings WHERE user_id = $1`

func (repository *SettingsRepository) Get(ctx context.Context, userID int64) (domain.Settings, error) {
	var settings domain.Settings
	var sendAt pgtype.Time

	err := repository.pool.QueryRow(ctx, getSettingsQuery, userID).Scan(&settings.MorningEnabled, &sendAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Settings{}, service.ErrSettingsNotSet
	}
	if err != nil {
		return domain.Settings{}, fmt.Errorf("чтение настроек: %w", err)
	}

	settings.SendAt = toDayTime(sendAt)
	if err := settings.Validate(); err != nil {
		return domain.Settings{}, fmt.Errorf("настройки пользователя %d в базе не проходят проверку: %v", userID, err)
	}
	return settings, nil
}

const saveSettingsQuery = `
INSERT INTO settings (user_id, morning_enabled, send_at)
VALUES (@user_id, @morning_enabled, @send_at)
ON CONFLICT (user_id) DO UPDATE
SET morning_enabled = excluded.morning_enabled, send_at = excluded.send_at`

func (repository *SettingsRepository) Save(ctx context.Context, userID int64, settings domain.Settings) error {
	_, err := repository.pool.Exec(ctx, saveSettingsQuery, pgx.NamedArgs{
		"user_id":         userID,
		"morning_enabled": settings.MorningEnabled,
		"send_at":         fromDayTime(settings.SendAt),
	})
	if err != nil {
		return fmt.Errorf("сохранение настроек: %w", err)
	}
	return nil
}

func fromDayTime(dayTime domain.DayTime) pgtype.Time {
	minutes := int64(dayTime.Hour*60 + dayTime.Minute)
	return pgtype.Time{Microseconds: minutes * int64(time.Minute/time.Microsecond), Valid: true}
}

func toDayTime(value pgtype.Time) domain.DayTime {
	minutes := int(value.Microseconds / int64(time.Minute/time.Microsecond))
	return domain.DayTime{Hour: minutes / 60, Minute: minutes % 60}
}
