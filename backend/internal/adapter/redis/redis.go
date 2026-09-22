package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func Connect(ctx context.Context, redisURL string) (*goredis.Client, error) {
	options, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("настройка redis: %w", err)
	}

	client := goredis.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("подключение к redis: %w", err)
	}
	return client, nil
}

type Cache struct {
	client *goredis.Client
}

func NewCache(client *goredis.Client) *Cache {
	return &Cache{client: client}
}

func (cache *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := cache.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("чтение кэша: %w", err)
	}
	return value, true, nil
}

func (cache *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := cache.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("запись в кэш: %w", err)
	}
	return nil
}

var _ service.RecognitionCounter = (*RecognitionCounter)(nil)

type RecognitionCounter struct {
	client *goredis.Client
}

func NewRecognitionCounter(client *goredis.Client) *RecognitionCounter {
	return &RecognitionCounter{client: client}
}

func (counter *RecognitionCounter) Increment(ctx context.Context, userID int64, window time.Duration) (int, error) {
	key := fmt.Sprintf("recognitions:%d", userID)

	pipeline := counter.client.TxPipeline()
	count := pipeline.Incr(ctx, key)
	pipeline.ExpireNX(ctx, key, window)
	if _, err := pipeline.Exec(ctx); err != nil {
		return 0, fmt.Errorf("счётчик распознаваний: %w", err)
	}
	return int(count.Val()), nil
}
