//go:build integration

package postgres_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/postgres"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func TestPhotoUploadRepository_TakeReturnsUploadOnlyOnceAndOnlyToOwner(t *testing.T) {
	repository := postgres.NewPhotoUploadRepository(newTestPool(t))
	const key = "users/1/photo.jpg"

	if err := repository.Create(t.Context(), 1, key); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repository.Take(t.Context(), 2, key); !errors.Is(err, service.ErrPhotoNotUploaded) {
		t.Errorf("чужая загрузка: ошибка = %v, ожидалась ErrPhotoNotUploaded", err)
	}
	if err := repository.Take(t.Context(), 1, key); err != nil {
		t.Fatalf("Take: %v", err)
	}
	if err := repository.Take(t.Context(), 1, key); !errors.Is(err, service.ErrPhotoNotUploaded) {
		t.Errorf("повторный Take: ошибка = %v, ожидалась ErrPhotoNotUploaded", err)
	}
}

func TestPhotoUploadRepository_TakeOlderThanReturnsOwnAbandonedKeys(t *testing.T) {
	pool := newTestPool(t)
	repository := postgres.NewPhotoUploadRepository(pool)

	for _, upload := range []struct {
		key    string
		userID int64
	}{
		{"users/1/old.jpg", 1},
		{"users/1/fresh.jpg", 1},
		{"users/2/old.jpg", 2},
	} {
		if err := repository.Create(t.Context(), upload.userID, upload.key); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	// created_at ставит база, поэтому состариваем записи запросом.
	_, err := pool.Exec(t.Context(),
		`UPDATE photo_uploads SET created_at = now() - interval '2 days' WHERE key LIKE '%old.jpg'`)
	if err != nil {
		t.Fatalf("старение записей: %v", err)
	}

	keys, err := repository.TakeOlderThan(t.Context(), 1, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("TakeOlderThan: %v", err)
	}

	if !slices.Equal(keys, []string{"users/1/old.jpg"}) {
		t.Errorf("ключи = %v, ожидалась только брошенная загрузка пользователя 1", keys)
	}
	if err := repository.Take(t.Context(), 1, "users/1/fresh.jpg"); err != nil {
		t.Errorf("свежая загрузка пропала: %v", err)
	}
	if err := repository.Take(t.Context(), 2, "users/2/old.jpg"); err != nil {
		t.Errorf("тронута загрузка чужого пользователя: %v", err)
	}
}
