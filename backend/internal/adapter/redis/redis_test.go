//go:build integration

package redis_test

import (
	"crypto/rand"
	"math/big"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/redis"
)

func TestCache_ReturnsWhatWasStored(t *testing.T) {
	client := newTestClient(t)
	cache := redis.NewCache(client)
	key := "test:" + t.Name() + ":" + randomSuffix(t)

	if _, found, err := cache.Get(t.Context(), key); err != nil || found {
		t.Fatalf("Get до записи = найдено %v (%v), ожидался промах", found, err)
	}

	if err := cache.Set(t.Context(), key, []byte("прогноз"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	value, found, err := cache.Get(t.Context(), key)
	if err != nil || !found || string(value) != "прогноз" {
		t.Errorf("Get = %q, найдено %v (%v), ожидался «прогноз»", value, found, err)
	}
	if ttl := client.TTL(t.Context(), key).Val(); ttl <= 0 || ttl > time.Minute {
		t.Errorf("срок записи %s, ожидалось не больше минуты", ttl)
	}
}

func TestRecognitionCounter_WindowStartsWithFirstPhoto(t *testing.T) {
	client := newTestClient(t)
	counter := redis.NewRecognitionCounter(client)
	userID := randomUserID(t)
	key := "recognitions:" + big.NewInt(userID).String()
	t.Cleanup(func() { client.Del(t.Context(), key) })

	for want := 1; want <= 3; want++ {
		count, err := counter.Increment(t.Context(), userID, time.Hour)
		if err != nil {
			t.Fatalf("Increment: %v", err)
		}
		if count != want {
			t.Errorf("счётчик = %d, ожидалось %d", count, want)
		}
		if want == 1 {
			client.Expire(t.Context(), key, 10*time.Minute)
		}
	}

	if ttl := client.TTL(t.Context(), key).Val(); ttl <= 0 || ttl > 10*time.Minute {
		t.Errorf("срок счётчика %s: следующие распознавания не должны продлевать окно", ttl)
	}
}

func newTestClient(t *testing.T) *goredis.Client {
	t.Helper()

	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("TEST_REDIS_URL не задан")
	}
	client, err := redis.Connect(t.Context(), redisURL)
	if err != nil {
		t.Fatalf("redis недоступен, поднимите его: make db\n%v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func randomUserID(t *testing.T) int64 {
	t.Helper()

	value, err := rand.Int(rand.Reader, big.NewInt(1<<40))
	if err != nil {
		t.Fatalf("случайный пользователь: %v", err)
	}
	return value.Int64() + 1
}

func randomSuffix(t *testing.T) string {
	return big.NewInt(randomUserID(t)).String()
}
