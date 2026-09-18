package telegram

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-telegram/bot"
)

const testPause = time.Millisecond

var errTimeout = errors.New("context deadline exceeded")

func TestRetry_NetworkBlinkIsRetried(t *testing.T) {
	calls := 0
	err := retryStartup(t.Context(), testPause, func() error {
		calls++
		if calls < 2 {
			return errTimeout
		}
		return nil
	})

	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if calls != 2 {
		t.Errorf("попыток %d, ожидалось 2", calls)
	}
}

func TestRetry_WrongTokenIsNotRetried(t *testing.T) {
	calls := 0
	err := retryStartup(t.Context(), testPause, func() error {
		calls++
		return fmt.Errorf("error call getMe, %w, Unauthorized", bot.ErrorUnauthorized)
	})

	if !errors.Is(err, bot.ErrorUnauthorized) {
		t.Errorf("ошибка = %v, ожидалась ErrorUnauthorized", err)
	}
	if calls != 1 {
		t.Errorf("попыток %d, ожидалась одна", calls)
	}
}

func TestRetry_GivesUpAfterAllAttempts(t *testing.T) {
	calls := 0
	err := retryStartup(t.Context(), testPause, func() error {
		calls++
		return errTimeout
	})

	if !errors.Is(err, errTimeout) {
		t.Errorf("ошибка = %v, ожидалась последняя ошибка попытки", err)
	}
	if calls != startupAttempts {
		t.Errorf("попыток %d, ожидалось %d", calls, startupAttempts)
	}
}

func TestRetry_StopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	calls := 0
	err := retryStartup(ctx, time.Hour, func() error {
		calls++
		cancel()
		return errTimeout
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("ошибка = %v, ожидалась отмена", err)
	}
	if calls != 1 {
		t.Errorf("попыток %d: после Ctrl+C ждать следующую незачем", calls)
	}
}
