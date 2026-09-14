package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/postgres"
	"github.com/djentelmanick/daily-stylist/backend/internal/config"
)

const usage = `использование: go run ./cmd/migrate <команда>

команды:
  status            какие миграции применены, какие нет
  up                накатить все новые
  up-to ВЕРСИЯ      накатить до версии включительно
  down              откатить последнюю
  down-to ВЕРСИЯ    откатить до версии, down-to 0 - откатить все`

func main() {
	command, version, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n\n%s\n", err, usage)
		os.Exit(2)
	}

	if err := run(command, version); err != nil {
		log.Fatal(err)
	}
}

func parseArgs(args []string) (command string, version int64, err error) {
	if len(args) == 0 {
		return "", 0, errors.New("не указана команда")
	}

	command = args[0]
	switch command {
	case "status", "up", "down":
		if len(args) != 1 {
			return "", 0, fmt.Errorf("у команды %s нет аргументов", command)
		}
	case "up-to", "down-to":
		if len(args) != 2 {
			return "", 0, fmt.Errorf("команде %s нужна версия", command)
		}
		version, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil || version < 0 {
			return "", 0, fmt.Errorf("версия %q - не номер миграции", args[1])
		}
	default:
		return "", 0, fmt.Errorf("неизвестная команда %q", command)
	}
	return command, version, nil
}

func run(command string, version int64) (err error) {
	config.LoadDotEnv()

	cfg, err := config.LoadMigrate()
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

	provider, err := postgres.NewMigrationProvider(pool)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, provider.Close())
	}()

	switch command {
	case "status":
		return printStatus(ctx, provider)
	case "up":
		return report(provider.Up(ctx))
	case "up-to":
		return report(provider.UpTo(ctx, version))
	case "down":
		result, err := provider.Down(ctx)
		if errors.Is(err, goose.ErrNoNextVersion) {
			fmt.Println("откатывать нечего: ни одна миграция не применена")
			return nil
		}
		if err != nil {
			return report(nil, err)
		}
		return report([]*goose.MigrationResult{result}, nil)
	case "down-to":
		return report(provider.DownTo(ctx, version))
	}
	return fmt.Errorf("неизвестная команда %q", command)
}

func report(results []*goose.MigrationResult, err error) error {
	if partial, ok := errors.AsType[*goose.PartialError](err); ok {
		results = partial.Applied
	}
	for _, result := range results {
		fmt.Println(result)
	}
	if err == nil && len(results) == 0 {
		fmt.Println("изменений нет: база уже на нужной версии")
	}
	return err
}

func printStatus(ctx context.Context, provider *goose.Provider) error {
	statuses, err := provider.Status(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("%-19s  %s\n", "Применена", "Миграция")
	for _, status := range statuses {
		appliedAt := "нет"
		if status.State == goose.StateApplied {
			appliedAt = status.AppliedAt.Local().Format(time.DateTime)
		}
		fmt.Printf("%-19s  %s\n", appliedAt, path.Base(status.Source.Path))
	}
	return nil
}
