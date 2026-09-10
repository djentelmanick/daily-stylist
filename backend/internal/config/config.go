package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const defaultBotListenAddr = ":2000"

type Bot struct {
	Token         string
	WebhookURL    string
	WebhookSecret string
	ListenAddr    string
}

func LoadBot() (Bot, error) {
	var env envReader

	cfg := Bot{
		Token:         env.required("TELEGRAM_BOT_TOKEN"),
		WebhookURL:    env.required("TELEGRAM_WEBHOOK_URL"),
		WebhookSecret: env.required("TELEGRAM_WEBHOOK_SECRET"),
		ListenAddr:    env.optional("BOT_LISTEN_ADDR", defaultBotListenAddr),
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
