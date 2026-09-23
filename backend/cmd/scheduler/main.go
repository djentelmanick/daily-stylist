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

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/postgres"
	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/rabbitmq"
	"github.com/djentelmanick/daily-stylist/backend/internal/config"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
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

	connection, err := rabbitmq.Connect(cfg.RabbitURL)
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close() }()

	tasks, err := rabbitmq.NewTasks(connection, rabbitmq.MorningTopology())
	if err != nil {
		return err
	}
	defer func() { _ = tasks.Close() }()

	planner := service.NewMorningPlanner(postgres.NewDeliveryRepository(pool), tasks, time.Now)

	log.Printf("планировщик запущен, проверяю раз в %s", tick)
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		if err := planner.PublishDue(ctx); err != nil && ctx.Err() == nil {
			log.Print(err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
