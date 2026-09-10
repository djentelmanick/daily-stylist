package telegram

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-telegram/bot"
)

const shutdownTimeout = 5 * time.Second

type Options struct {
	Token         string
	WebhookURL    string
	WebhookSecret string
	ListenAddr    string
}

type Bot struct {
	api  *bot.Bot
	srv  *http.Server
	opts Options
}

func New(opts Options, handler bot.HandlerFunc) (*Bot, error) {
	api, err := bot.New(
		opts.Token,
		bot.WithDefaultHandler(handler),
		bot.WithWebhookSecretToken(opts.WebhookSecret),
	)
	if err != nil {
		return nil, fmt.Errorf("создание клиента telegram: %w", err)
	}

	return &Bot{
		api:  api,
		srv:  &http.Server{Addr: opts.ListenAddr, Handler: api.WebhookHandler()},
		opts: opts,
	}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	if _, err := b.api.SetWebhook(ctx, &bot.SetWebhookParams{
		URL:         b.opts.WebhookURL,
		SecretToken: b.opts.WebhookSecret,
	}); err != nil {
		return fmt.Errorf("регистрация вебхука: %w", err)
	}

	go b.api.StartWebhook(ctx)

	serverErr := make(chan error, 1)
	go func() {
		err := b.srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErr <- err
	}()

	select {
	case err := <-serverErr:
		return errors.Join(err, b.shutdown())
	case <-ctx.Done():
		return b.shutdown()
	}
}

func (b *Bot) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	var errs []error
	if _, err := b.api.DeleteWebhook(ctx, &bot.DeleteWebhookParams{}); err != nil {
		errs = append(errs, fmt.Errorf("снятие вебхука: %w", err))
	}
	if err := b.srv.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("остановка http-сервера: %w", err))
	}
	return errors.Join(errs...)
}
