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
	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/rabbitmq"
	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/redis"
	"github.com/djentelmanick/daily-stylist/backend/internal/config"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config.LoadDotEnv()

	cfg, err := config.LoadSender()
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

	connection, err := rabbitmq.Connect(cfg.RabbitURL)
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close() }()

	consumer, err := rabbitmq.NewConsumer(connection, rabbitmq.MorningTopology(), cfg.Prefetch)
	if err != nil {
		return err
	}
	defer func() { _ = consumer.Close() }()

	recommender := service.NewRecommender(
		postgres.NewItemRepository(pool),
		postgres.NewOutfitRepository(pool),
		postgres.NewLocationRepository(pool),
		openmeteo.NewClient(redis.NewCache(redisClient)),
		time.Now,
	)
	sender := service.NewMorningSender(postgres.NewDeliveryRepository(pool), recommender, notifier)

	log.Print("отправщик запущен, жду задачи")
	err = consumer.Consume(ctx, func(ctx context.Context, delivery service.MorningDelivery) error {
		if err := sender.Deliver(ctx, delivery); err != nil {
			log.Printf("утренняя рассылка пользователю %d: %v", delivery.UserID, err)
			return err
		}
		return nil
	})
	if ctx.Err() != nil {
		return nil
	}
	return err
}
