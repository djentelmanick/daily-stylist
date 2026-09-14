package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	defaultBotListenAddr = ":2000"
	defaultWebhookPath   = "/telegram/webhook"
	defaultItemsFile     = "data/items.json"
)

type Bot struct {
	Token          string
	WebhookBaseURL string
	WebhookPath    string
	WebhookSecret  string
	ListenAddr     string
	ItemsFile      string
}

func LoadBot() (Bot, error) {
	var env envReader

	cfg := Bot{
		Token:          env.required("TELEGRAM_BOT_TOKEN"),
		WebhookBaseURL: env.required("TELEGRAM_WEBHOOK_BASE_URL"),
		WebhookPath:    env.optional("TELEGRAM_WEBHOOK_PATH", defaultWebhookPath),
		WebhookSecret:  env.required("TELEGRAM_WEBHOOK_SECRET"),
		ListenAddr:     env.optional("BOT_LISTEN_ADDR", defaultBotListenAddr),
		ItemsFile:      env.optional("ITEMS_FILE", defaultItemsFile),
	}
	if err := env.err(); err != nil {
		return Bot{}, err
	}

	return cfg, nil
}

var ErrMissingEnv = errors.New("не заданы обязательные переменные окружения")

type envReader struct {
	missing []string
}

func (r *envReader) required(name string) string {
	value := os.Getenv(name)
	if value == "" {
		r.missing = append(r.missing, name)
	}
	return value
}

func (r *envReader) err() error {
	if len(r.missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrMissingEnv, strings.Join(r.missing, ", "))
}

func (*envReader) optional(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
