package service_test

import (
	"context"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func TestSettings_DefaultsUntilChanged(t *testing.T) {
	settings := service.NewSettings(missingSettings{})

	got, err := settings.Get(t.Context(), 42)

	if err != nil || got != domain.DefaultSettings() {
		t.Errorf("Get = %+v (%v), ожидались настройки по умолчанию", got, err)
	}
}

type missingSettings struct{}

func (missingSettings) Get(context.Context, int64) (domain.Settings, error) {
	return domain.Settings{}, service.ErrSettingsNotSet
}

func (missingSettings) Save(context.Context, int64, domain.Settings) error {
	return nil
}
