package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/jsonfile"
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
	loadDotEnv()

	cfg, err := config.LoadBot()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	wardrobe := service.NewWardrobe(jsonfile.NewItemRepository(cfg.ItemsFile))

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

func loadDotEnv() {
	err := godotenv.Load()
	switch {
	case err == nil:
	case errors.Is(err, fs.ErrNotExist):
		log.Println(".env не найден, читаю переменные окружения")
	default:
		log.Printf("не удалось прочитать .env: %v", err)
	}
}
