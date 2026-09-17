package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var _ service.PhotoUploadRepository = (*PhotoUploadRepository)(nil)

type PhotoUploadRepository struct {
	pool *pgxpool.Pool
}

func NewPhotoUploadRepository(pool *pgxpool.Pool) *PhotoUploadRepository {
	return &PhotoUploadRepository{pool: pool}
}

const createPhotoUploadQuery = `INSERT INTO photo_uploads (key, user_id) VALUES ($1, $2)`

func (repository *PhotoUploadRepository) Create(ctx context.Context, userID int64, key string) error {
	if _, err := repository.pool.Exec(ctx, createPhotoUploadQuery, key, userID); err != nil {
		return fmt.Errorf("запись о загрузке фотографии: %w", err)
	}
	return nil
}

// Условие по user_id тут не лишнее: по чужому ключу загрузку не забрать.
const takePhotoUploadQuery = `DELETE FROM photo_uploads WHERE key = $1 AND user_id = $2`

func (repository *PhotoUploadRepository) Take(ctx context.Context, userID int64, key string) error {
	tag, err := repository.pool.Exec(ctx, takePhotoUploadQuery, key, userID)
	if err != nil {
		return fmt.Errorf("поиск загрузки фотографии: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return service.ErrPhotoNotUploaded
	}
	return nil
}

const takeOldPhotoUploadsQuery = `DELETE FROM photo_uploads WHERE user_id = $1 AND created_at < $2 RETURNING key`

func (repository *PhotoUploadRepository) TakeOlderThan(ctx context.Context, userID int64, before time.Time) ([]string, error) {
	rows, err := repository.pool.Query(ctx, takeOldPhotoUploadsQuery, userID, before)
	if err != nil {
		return nil, fmt.Errorf("брошенные загрузки фотографий: %w", err)
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("брошенные загрузки фотографий: %w", err)
	}
	return keys, nil
}
