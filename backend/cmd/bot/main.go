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

	"github.com/djentelmanick/daily-stylist/backend/internal/config"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/handlers"
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

	b, err := telegram.New(telegram.Options{
		Token:         cfg.Token,
		WebhookURL:    cfg.WebhookURL,
		WebhookSecret: cfg.WebhookSecret,
		ListenAddr:    cfg.ListenAddr,
	}, handlers.Default)
	if err != nil {
		return err
	}

	log.Printf("бот слушает вебхук на %s", cfg.ListenAddr)
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
