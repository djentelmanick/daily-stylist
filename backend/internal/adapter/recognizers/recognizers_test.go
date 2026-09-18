package recognizers

import (
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/recognizers/gigachat"
	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/recognizers/vision"
	"github.com/djentelmanick/daily-stylist/backend/internal/config"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func settings(primary, fallback string) config.Recognition {
	return config.Recognition{
		Primary:  primary,
		Fallback: fallback,
		Timeout:  time.Second,
		GigaChat: config.GigaChat{Credentials: "ключ"},
		Vision:   config.Vision{Address: "localhost:1"},
	}
}

func TestNew_OnlyPrimary(t *testing.T) {
	recognizer, closeRecognizer, err := New(settings(config.RecognizerGigaChat, ""))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer closeRecognizer()

	if _, ok := recognizer.(*gigachat.Client); !ok {
		t.Errorf("распознаватель = %T, ожидался клиент GigaChat", recognizer)
	}
}

func TestNew_PrimaryWithFallback(t *testing.T) {
	recognizer, closeRecognizer, err := New(settings(config.RecognizerVision, config.RecognizerGigaChat))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer closeRecognizer()

	if _, ok := recognizer.(*service.Fallback); !ok {
		t.Errorf("распознаватель = %T, ожидалась связка основного и запасного", recognizer)
	}
}

func TestNew_VisionAlone(t *testing.T) {
	recognizer, closeRecognizer, err := New(settings(config.RecognizerVision, ""))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer closeRecognizer()

	if _, ok := recognizer.(*vision.Client); !ok {
		t.Errorf("распознаватель = %T, ожидался клиент своего сервиса", recognizer)
	}
}
