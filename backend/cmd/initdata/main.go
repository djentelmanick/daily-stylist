package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/config"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/miniapp"
)

func main() {
	userID := flag.Int64("user", 0, "идентификатор пользователя Telegram")
	flag.Parse()

	if err := run(*userID); err != nil {
		log.Fatal(err)
	}
}

func run(userID int64) error {
	config.LoadDotEnv()

	if userID <= 0 {
		return fmt.Errorf("укажите пользователя: go run ./cmd/initdata -user 12345")
	}
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return fmt.Errorf("%w: TELEGRAM_BOT_TOKEN", config.ErrMissingEnv)
	}

	fmt.Println("tma " + miniapp.SignInitData(token, userID, time.Now()))
	return nil
}
