package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/sethvargo/go-retry"
)

const (
	shutdownTimeout   = 5 * time.Second
	readHeaderTimeout = 10 * time.Second
	startupAttempts   = 3
	startupPause      = time.Second
)

type Options struct {
	Token          string
	WebhookBaseURL string
	WebhookPath    string
	WebhookSecret  string
	ListenAddr     string
	MiniApp        http.Handler
	Docs           http.Handler
}

const DocsPath = "/api/docs/"

type Bot struct {
	api        *bot.Bot
	srv        *http.Server
	webhookURL string
	opts       Options
}

func New(ctx context.Context, opts Options, handler bot.HandlerFunc) (*Bot, error) {
	webhookURL, pattern, err := webhookRoute(opts.WebhookBaseURL, opts.WebhookPath)
	if err != nil {
		return nil, err
	}

	var api *bot.Bot
	err = retryStartup(ctx, startupPause, func() error {
		var err error
		api, err = bot.New(
			opts.Token,
			bot.WithDefaultHandler(handler),
			bot.WithWebhookSecretToken(opts.WebhookSecret),
		)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("создание клиента telegram: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(pattern, api.WebhookHandler())
	if opts.MiniApp != nil {
		mux.Handle("/api/", opts.MiniApp)
	}
	if opts.Docs != nil {
		mux.Handle(DocsPath, http.StripPrefix(strings.TrimSuffix(DocsPath, "/"), opts.Docs))
	}

	return &Bot{
		api:        api,
		srv:        &http.Server{Addr: opts.ListenAddr, Handler: mux, ReadHeaderTimeout: readHeaderTimeout},
		webhookURL: webhookURL,
		opts:       opts,
	}, nil
}

func webhookRoute(baseURL, path string) (webhookURL, pattern string, err error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	webhookURL, err = url.JoinPath(baseURL, path)
	if err != nil {
		return "", "", fmt.Errorf("адрес вебхука: %w", err)
	}

	if path == "/" {
		return webhookURL, "POST /{$}", nil
	}
	return webhookURL, "POST " + path, nil
}

func (b *Bot) Run(ctx context.Context) error {
	err := retryStartup(ctx, startupPause, func() error {
		_, err := b.api.SetWebhook(ctx, &bot.SetWebhookParams{
			URL:         b.webhookURL,
			SecretToken: b.opts.WebhookSecret,
		})
		return err
	})
	if err != nil {
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

func retryStartup(ctx context.Context, pause time.Duration, action func() error) error {
	backoff := retry.WithMaxRetries(startupAttempts-1, retry.NewExponential(pause))
	attempt := 0
	return retry.Do(ctx, backoff, func(context.Context) error {
		attempt++
		err := action()
		if err == nil || !transient(err) {
			return err
		}
		log.Printf("telegram: попытка %d из %d не удалась: %v", attempt, startupAttempts, err)
		return retry.RetryableError(err)
	})
}

func transient(err error) bool {
	for _, final := range []error{
		bot.ErrorUnauthorized, bot.ErrorForbidden, bot.ErrorBadRequest, bot.ErrorNotFound, bot.ErrorConflict,
	} {
		if errors.Is(err, final) {
			return false
		}
	}
	return true
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
