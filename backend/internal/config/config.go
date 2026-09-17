package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultBotListenAddr = ":2000"
	defaultWebhookPath   = "/telegram/webhook"
	defaultS3Region      = "us-east-1"
)

type Bot struct {
	Token          string
	WebhookBaseURL string
	WebhookPath    string
	WebhookSecret  string
	ListenAddr     string
	DatabaseURL    string
	Photos         PhotoStorage
}

type PhotoStorage struct {
	Endpoint  string
	PublicURL string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

func LoadBot() (Bot, error) {
	var env envReader

	cfg := Bot{
		Token:          env.required("TELEGRAM_BOT_TOKEN"),
		WebhookBaseURL: env.required("TELEGRAM_WEBHOOK_BASE_URL"),
		WebhookPath:    env.optional("TELEGRAM_WEBHOOK_PATH", defaultWebhookPath),
		WebhookSecret:  env.required("TELEGRAM_WEBHOOK_SECRET"),
		ListenAddr:     env.optional("BOT_LISTEN_ADDR", defaultBotListenAddr),
		DatabaseURL:    env.required("DATABASE_URL"),
		Photos: PhotoStorage{
			Endpoint:  env.required("S3_ENDPOINT"),
			PublicURL: env.required("S3_PUBLIC_URL"),
			Region:    env.optional("S3_REGION", defaultS3Region),
			Bucket:    env.required("S3_BUCKET"),
			AccessKey: env.required("S3_ACCESS_KEY"),
			SecretKey: env.required("S3_SECRET_KEY"),
		},
	}
	if err := env.err(); err != nil {
		return Bot{}, err
	}

	return cfg, nil
}

type Migrate struct {
	DatabaseURL string
}

func LoadMigrate() (Migrate, error) {
	var env envReader

	cfg := Migrate{
		DatabaseURL: env.required("DATABASE_URL"),
	}
	if err := env.err(); err != nil {
		return Migrate{}, err
	}

	return cfg, nil
}

func LoadDotEnv() {
	err := godotenv.Load()
	switch {
	case err == nil:
	case errors.Is(err, fs.ErrNotExist):
		log.Println(".env не найден, читаю переменные окружения")
	default:
		log.Printf("не удалось прочитать .env: %v", err)
	}
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
