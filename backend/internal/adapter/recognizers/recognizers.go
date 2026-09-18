package recognizers

import (
	"fmt"
	"log"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/recognizers/gigachat"
	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/recognizers/vision"
	"github.com/djentelmanick/daily-stylist/backend/internal/config"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func New(cfg config.Recognition) (service.PhotoRecognizer, func(), error) {
	primary, closePrimary, err := byName(cfg.Primary, cfg)
	if err != nil {
		return nil, nil, err
	}
	if cfg.Fallback == "" {
		log.Printf("одежду на фотографиях распознаёт %s", cfg.Primary)
		return primary, closePrimary, nil
	}

	secondary, closeSecondary, err := byName(cfg.Fallback, cfg)
	if err != nil {
		closePrimary()
		return nil, nil, err
	}

	log.Printf("одежду на фотографиях распознаёт %s, на подстраховке %s", cfg.Primary, cfg.Fallback)
	return service.NewFallback(primary, secondary), func() {
		closePrimary()
		closeSecondary()
	}, nil
}

func byName(name string, cfg config.Recognition) (service.PhotoRecognizer, func(), error) {
	switch name {
	case config.RecognizerGigaChat:
		client, err := gigachat.New(gigachat.Config{
			Credentials: cfg.GigaChat.Credentials,
			AuthURL:     cfg.GigaChat.AuthURL,
			BaseURL:     cfg.GigaChat.BaseURL,
			Scope:       cfg.GigaChat.Scope,
			Model:       cfg.GigaChat.Model,
			Timeout:     cfg.Timeout,
		})
		if err != nil {
			return nil, nil, err
		}
		return client, func() {}, nil

	case config.RecognizerVision:
		client, err := vision.New(cfg.Vision.Address, cfg.Timeout)
		if err != nil {
			return nil, nil, err
		}
		return client, func() { _ = client.Close() }, nil
	}

	return nil, nil, fmt.Errorf("неизвестный распознаватель %q", name)
}
