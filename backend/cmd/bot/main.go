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
	// Часовые пояса городов: в контейнере без системной базы поясов time.LoadLocation их не найдёт.
	_ "time/tzdata"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/openmeteo"
	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/postgres"
	"github.com/djentelmanick/daily-stylist/backend/internal/config"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/handlers"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/miniapp"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config.LoadDotEnv()

	cfg, err := config.LoadBot()
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

	items := postgres.NewItemRepository(pool)
	locations := postgres.NewLocationRepository(pool)
	weather := openmeteo.NewClient()

	wardrobe := service.NewWardrobe(items)
	recommender := service.NewRecommender(items, postgres.NewOutfitRepository(pool), locations, weather, time.Now)

	b, err := telegram.New(telegram.Options{
		Token:          cfg.Token,
		WebhookBaseURL: cfg.WebhookBaseURL,
		WebhookPath:    cfg.WebhookPath,
		WebhookSecret:  cfg.WebhookSecret,
		ListenAddr:     cfg.ListenAddr,
		MiniApp:        miniapp.NewHandler(cfg.Token, wardrobe, recommender, service.NewLocations(locations, weather)),
	}, handlers.Default)
	if err != nil {
		return err
	}

	log.Printf("HTTP-сервер слушает %s: вебхук Telegram и API Mini App на %s", cfg.ListenAddr, cfg.WebhookPath)
	return b.Run(ctx)
}
