package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultBotListenAddr = ":2000"
	defaultWebhookPath   = "/telegram/webhook"
	defaultS3Region      = "us-east-1"
	defaultVisionAddr    = "localhost:59090"

	// Телеграм не даёт слать в один чат чаще раза в секунду, брать задачи пачкой незачем.
	defaultSenderPrefetch = 1

	defaultGigaChatAuthURL = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	defaultGigaChatBaseURL = "https://api.giga.chat/v1"
	defaultGigaChatScope   = "GIGACHAT_API_PERS"
	defaultGigaChatModel   = "GigaChat-3-Ultra"

	recognitionTimeout = 45 * time.Second
)

const (
	RecognizerGigaChat = "gigachat"
	RecognizerVision   = "vision"
)

const (
	EnvDev  = "dev"
	EnvProd = "prod"
)

var ErrInvalidEnv = errors.New("неверно задан режим работы")

var allRecognizers = []string{RecognizerGigaChat, RecognizerVision}

var ErrInvalidRecognizer = errors.New("неверно настроен распознаватель")

type Bot struct {
	Token          string
	WebhookBaseURL string
	WebhookPath    string
	WebhookSecret  string
	ListenAddr     string
	Env            string
	DatabaseURL    string
	RedisURL       string
	Photos         PhotoStorage
	Recognition    Recognition
}

type Recognition struct {
	Primary  string
	Fallback string
	Timeout  time.Duration
	GigaChat GigaChat
	Vision   Vision
}

type GigaChat struct {
	Credentials string
	AuthURL     string
	BaseURL     string
	Scope       string
	Model       string
}

type Vision struct {
	Address string
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
		Env:            env.optional("APP_ENV", EnvDev),
		DatabaseURL:    env.required("DATABASE_URL"),
		RedisURL:       env.required("REDIS_URL"),
		Photos: PhotoStorage{
			Endpoint:  env.required("S3_ENDPOINT"),
			PublicURL: env.required("S3_PUBLIC_URL"),
			Region:    env.optional("S3_REGION", defaultS3Region),
			Bucket:    env.required("S3_BUCKET"),
			AccessKey: env.required("S3_ACCESS_KEY"),
			SecretKey: env.required("S3_SECRET_KEY"),
		},
		Recognition: Recognition{
			Primary:  env.optional("RECOGNIZER", RecognizerGigaChat),
			Fallback: env.optional("RECOGNIZER_FALLBACK", ""),
			Timeout:  recognitionTimeout,
			GigaChat: GigaChat{
				Credentials: env.optional("GIGACHAT_AUTH_KEY", ""),
				AuthURL:     env.optional("GIGACHAT_AUTH_URL", defaultGigaChatAuthURL),
				BaseURL:     env.optional("GIGACHAT_BASE_URL", defaultGigaChatBaseURL),
				Scope:       env.optional("GIGACHAT_SCOPE", defaultGigaChatScope),
				Model:       env.optional("GIGACHAT_MODEL", defaultGigaChatModel),
			},
			Vision: Vision{Address: env.optional("VISION_ADDR", defaultVisionAddr)},
		},
	}
	if err := env.err(); err != nil {
		return Bot{}, err
	}
	if err := cfg.Recognition.validate(); err != nil {
		return Bot{}, err
	}
	if err := cfg.validateEnv(); err != nil {
		return Bot{}, err
	}

	return cfg, nil
}

func (cfg Bot) validateEnv() error {
	if cfg.Env != EnvDev && cfg.Env != EnvProd {
		return fmt.Errorf("%w: APP_ENV=%q, ожидалось %s или %s", ErrInvalidEnv, cfg.Env, EnvDev, EnvProd)
	}
	return nil
}

func (cfg Bot) DocsEnabled() bool {
	return cfg.Env == EnvDev
}

func (recognition Recognition) validate() error {
	if !slices.Contains(allRecognizers, recognition.Primary) {
		return fmt.Errorf("%w: RECOGNIZER=%q, ожидалось одно из %s",
			ErrInvalidRecognizer, recognition.Primary, strings.Join(allRecognizers, ", "))
	}
	if recognition.Fallback != "" {
		if !slices.Contains(allRecognizers, recognition.Fallback) {
			return fmt.Errorf("%w: RECOGNIZER_FALLBACK=%q, ожидалось одно из %s или пусто",
				ErrInvalidRecognizer, recognition.Fallback, strings.Join(allRecognizers, ", "))
		}
		if recognition.Fallback == recognition.Primary {
			return fmt.Errorf("%w: запасной совпадает с основным (%s)", ErrInvalidRecognizer, recognition.Primary)
		}
	}
	if recognition.uses(RecognizerGigaChat) && recognition.GigaChat.Credentials == "" {
		return fmt.Errorf("%w: для %s нужен GIGACHAT_AUTH_KEY", ErrInvalidRecognizer, RecognizerGigaChat)
	}
	return nil
}

func (recognition Recognition) uses(name string) bool {
	return recognition.Primary == name || recognition.Fallback == name
}

type Scheduler struct {
	DatabaseURL string
	RabbitURL   string
}

func LoadScheduler() (Scheduler, error) {
	var env envReader

	cfg := Scheduler{
		DatabaseURL: env.required("DATABASE_URL"),
		RabbitURL:   env.required("RABBITMQ_URL"),
	}
	if err := env.err(); err != nil {
		return Scheduler{}, err
	}

	return cfg, nil
}

type Sender struct {
	Token       string
	MiniAppURL  string
	DatabaseURL string
	RedisURL    string
	RabbitURL   string
	Prefetch    int
}

func LoadSender() (Sender, error) {
	var env envReader

	cfg := Sender{
		Token:       env.required("TELEGRAM_BOT_TOKEN"),
		MiniAppURL:  env.required("TELEGRAM_WEBHOOK_BASE_URL"),
		DatabaseURL: env.required("DATABASE_URL"),
		RedisURL:    env.required("REDIS_URL"),
		RabbitURL:   env.required("RABBITMQ_URL"),
		Prefetch:    defaultSenderPrefetch,
	}
	if err := env.err(); err != nil {
		return Sender{}, err
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
