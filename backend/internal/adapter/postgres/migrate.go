package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/djentelmanick/daily-stylist/backend/migrations"
)

var ErrPendingMigrations = errors.New("есть непримененные миграции")

func NewMigrationProvider(pool *pgxpool.Pool) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("блокировка миграций: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, stdlib.OpenDBFromPool(pool), migrations.FS,
		goose.WithSessionLocker(locker),
	)
	if err != nil {
		return nil, fmt.Errorf("загрузка миграций: %w", err)
	}
	return provider, nil
}

func CheckMigrations(ctx context.Context, pool *pgxpool.Pool) (err error) {
	provider, err := NewMigrationProvider(pool)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, provider.Close())
	}()

	pending, err := provider.HasPending(ctx)
	if err != nil {
		return fmt.Errorf("проверка миграций: %w", err)
	}
	if !pending {
		return nil
	}

	current, target, err := provider.GetVersions(ctx)
	if err != nil {
		return fmt.Errorf("проверка миграций: %w", err)
	}
	return fmt.Errorf("%w: версия базы %d, последняя миграция %d", ErrPendingMigrations, current, target)
}
