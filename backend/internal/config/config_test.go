package config

import (
	"errors"
	"testing"
)

func recognition(primary, fallback, key string) Recognition {
	return Recognition{
		Primary:  primary,
		Fallback: fallback,
		GigaChat: GigaChat{Credentials: key},
	}
}

func TestRecognition_AcceptsUsableSettings(t *testing.T) {
	tests := map[string]Recognition{
		"только облако":             recognition(RecognizerGigaChat, "", "ключ"),
		"только свой сервис":        recognition(RecognizerVision, "", ""),
		"свой сервис под облаком":   recognition(RecognizerGigaChat, RecognizerVision, "ключ"),
		"облако под своим сервисом": recognition(RecognizerVision, RecognizerGigaChat, "ключ"),
	}
	for name, settings := range tests {
		t.Run(name, func(t *testing.T) {
			if err := settings.validate(); err != nil {
				t.Errorf("validate: %v", err)
			}
		})
	}
}

func TestRecognition_RefusesBrokenSettings(t *testing.T) {
	tests := map[string]Recognition{
		"опечатка в основном":     recognition("гигачат", "", "ключ"),
		"опечатка в запасном":     recognition(RecognizerGigaChat, "виженн", "ключ"),
		"запасной такой же":       recognition(RecognizerVision, RecognizerVision, ""),
		"облако без ключа":        recognition(RecognizerGigaChat, "", ""),
		"облако без ключа вторым": recognition(RecognizerVision, RecognizerGigaChat, ""),
	}
	for name, settings := range tests {
		t.Run(name, func(t *testing.T) {
			if err := settings.validate(); !errors.Is(err, ErrInvalidRecognizer) {
				t.Errorf("ошибка = %v, ожидалась ErrInvalidRecognizer", err)
			}
		})
	}
}

func TestBot_RefusesUnknownEnv(t *testing.T) {
	if err := (Bot{Env: "staging"}).validateEnv(); !errors.Is(err, ErrInvalidEnv) {
		t.Errorf("ошибка = %v, ожидалась ErrInvalidEnv: описание API не должно включаться по опечатке", err)
	}
}

func TestBot_DocsOnlyInDev(t *testing.T) {
	if (Bot{Env: EnvProd}).DocsEnabled() {
		t.Error("в проде описание API включено")
	}
	if !(Bot{Env: EnvDev}).DocsEnabled() {
		t.Error("в разработке описание API выключено")
	}
}
