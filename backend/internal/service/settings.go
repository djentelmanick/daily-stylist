package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

type Settings struct {
	repository SettingsRepository
}

func NewSettings(repository SettingsRepository) *Settings {
	return &Settings{repository: repository}
}

func (settings *Settings) Get(ctx context.Context, userID int64) (domain.Settings, error) {
	current, err := settings.repository.Get(ctx, userID)
	if errors.Is(err, ErrSettingsNotSet) {
		return domain.DefaultSettings(), nil
	}
	if err != nil {
		return domain.Settings{}, fmt.Errorf("чтение настроек: %w", err)
	}
	return current, nil
}

func (settings *Settings) Save(ctx context.Context, userID int64, updated domain.Settings) error {
	if err := updated.Validate(); err != nil {
		return fmt.Errorf("сохранение настроек: %w", err)
	}
	if err := settings.repository.Save(ctx, userID, updated); err != nil {
		return fmt.Errorf("сохранение настроек: %w", err)
	}
	return nil
}
