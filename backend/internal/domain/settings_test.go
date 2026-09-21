package domain_test

import (
	"errors"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func TestParseDayTime(t *testing.T) {
	tests := map[string]struct {
		value string
		want  domain.DayTime
		valid bool
	}{
		"утро":            {value: "07:00", want: domain.DayTime{Hour: 7}, valid: true},
		"с минутами":      {value: "21:45", want: domain.DayTime{Hour: 21, Minute: 45}, valid: true},
		"полночь":         {value: "00:00", want: domain.DayTime{}, valid: true},
		"без двоеточия":   {value: "0700"},
		"не число":        {value: "семь:ноль"},
		"часов не бывает": {value: "24:00"},
		"минут не бывает": {value: "07:60"},
		"пусто":           {value: ""},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := domain.ParseDayTime(test.value)

			if !test.valid {
				if !errors.Is(err, domain.ErrInvalidSettings) {
					t.Fatalf("ошибка = %v, ожидалась ErrInvalidSettings", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDayTime(%q): %v", test.value, err)
			}
			if got != test.want {
				t.Errorf("ParseDayTime(%q) = %+v, ожидалось %+v", test.value, got, test.want)
			}
			if got.String() != test.value {
				t.Errorf("String() = %q, ожидалось %q", got.String(), test.value)
			}
		})
	}
}

func TestDefaultSettings(t *testing.T) {
	settings := domain.DefaultSettings()

	if !settings.MorningEnabled || settings.SendAt.String() != "07:00" {
		t.Errorf("настройки по умолчанию = %+v, ожидалась включённая рассылка в 07:00", settings)
	}
	if err := settings.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
