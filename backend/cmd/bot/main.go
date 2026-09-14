package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	wardrobe := service.NewWardrobe(postgres.NewItemRepository(pool))

	b, err := telegram.New(telegram.Options{
		Token:          cfg.Token,
		WebhookBaseURL: cfg.WebhookBaseURL,
		WebhookPath:    cfg.WebhookPath,
		WebhookSecret:  cfg.WebhookSecret,
		ListenAddr:     cfg.ListenAddr,
		MiniApp:        miniapp.NewHandler(cfg.Token, wardrobe),
	}, handlers.Default)
	if err != nil {
		return err
	}

	log.Printf("HTTP-сервер слушает %s: вебхук Telegram и API Mini App на %s", cfg.ListenAddr, cfg.WebhookPath)
	return b.Run(ctx)
}
