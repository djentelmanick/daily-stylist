//go:build integration

package postgres_test

import (
	"context"
	"crypto/rand"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/postgres"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool := newEmptyTestPool(t)
	migrate(t, pool)
	return pool
}

func newEmptyTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	admin, err := postgres.Connect(t.Context(), databaseURL)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	schema := "test_" + strings.ToLower(rand.Text())
	if _, err := admin.Exec(t.Context(), "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("создание схемы: %v", err)
	}
	t.Cleanup(func() {
		defer admin.Close()
		// t.Context() к этому моменту уже отменён.
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("удаление схемы: %v", err)
		}
	})

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func migrate(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	provider, err := postgres.NewMigrationProvider(pool)
	if err != nil {
		t.Fatalf("NewMigrationProvider: %v", err)
	}
	defer func() {
		if err := provider.Close(); err != nil {
			t.Errorf("закрытие провайдера миграций: %v", err)
		}
	}()

	if _, err := provider.Up(t.Context()); err != nil {
		t.Fatalf("Up: %v", err)
	}
}
