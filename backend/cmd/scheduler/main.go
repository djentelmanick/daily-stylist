package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/openmeteo"
	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/postgres"
	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/redis"
	"github.com/djentelmanick/daily-stylist/backend/internal/config"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram"
)

const tick = time.Minute

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config.LoadDotEnv()

	cfg, err := config.LoadScheduler()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := postgres.CheckMigrations(ctx, pool); err != nil {
		if errors.Is(err, postgres.ErrPendingMigrations) {
			return fmt.Errorf("%w. Накатите их: go run ./cmd/migrate up", err)
		}
		return err
	}

	redisClient, err := redis.Connect(ctx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer func() { _ = redisClient.Close() }()

	notifier, err := telegram.NewNotifier(ctx, cfg.Token, cfg.MiniAppURL)
	if err != nil {
		return err
	}

	recommender := service.NewRecommender(
		postgres.NewItemRepository(pool),
		postgres.NewOutfitRepository(pool),
		postgres.NewLocationRepository(pool),
		openmeteo.NewClient(redis.NewCache(redisClient)),
		time.Now,
	)
	morning := service.NewMorning(postgres.NewDeliveryRepository(pool), recommender, notifier, time.Now)

	log.Printf("планировщик запущен, проверяю раз в %s", tick)
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		if err := morning.SendDue(ctx); err != nil && ctx.Err() == nil {
			log.Print(err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
