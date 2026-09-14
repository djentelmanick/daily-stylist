//go:build integration

package postgres_test

import (
	"errors"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/postgres"
)

func TestCheckMigrations(t *testing.T) {
	pool := newEmptyTestPool(t)

	if err := postgres.CheckMigrations(t.Context(), pool); !errors.Is(err, postgres.ErrPendingMigrations) {
		t.Fatalf("на пустой базе CheckMigrations = %v, ожидалась ErrPendingMigrations", err)
	}

	migrate(t, pool)

	if err := postgres.CheckMigrations(t.Context(), pool); err != nil {
		t.Errorf("после миграций CheckMigrations = %v, ожидался nil", err)
	}
}
