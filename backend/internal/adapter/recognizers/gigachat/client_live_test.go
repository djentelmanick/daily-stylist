//go:build integration

package gigachat_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/recognizers/gigachat"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

// Что ответит модель, не проверяем: это её свойство, а не свойство кода.
func TestRecognize_TalksToRealGigaChat(t *testing.T) {
	credentials := os.Getenv("GIGACHAT_AUTH_KEY")
	if credentials == "" {
		t.Skip("нет GIGACHAT_AUTH_KEY: тест GigaChat пропущен")
	}

	client, err := gigachat.New(gigachat.Config{
		Credentials: credentials,
		AuthURL:     envOr("GIGACHAT_AUTH_URL", "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"),
		BaseURL:     envOr("GIGACHAT_BASE_URL", "https://api.giga.chat/v1"),
		Scope:       envOr("GIGACHAT_SCOPE", "GIGACHAT_API_PERS"),
		Model:       envOr("GIGACHAT_MODEL", "GigaChat-3-Ultra"),
		Timeout:     time.Minute,
	})
	if err != nil {
		t.Fatalf("клиент: %v", err)
	}

	suggestion, err := client.Recognize(t.Context(), service.PhotoContent{Bytes: jpegPhoto(t), ContentType: "image/jpeg"})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}
	t.Logf("модель ответила: %+v", suggestion)
}

// Настоящий снимок подставляется через GIGACHAT_TEST_PHOTO=путь.
func jpegPhoto(t *testing.T) []byte {
	t.Helper()

	if path := os.Getenv("GIGACHAT_TEST_PHOTO"); path != "" {
		photo, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("фотография: %v", err)
		}
		return photo
	}

	photo := image.NewRGBA(image.Rect(0, 0, 320, 480))
	for y := range 480 {
		for x := range 320 {
			shade := color.RGBA{R: 40, G: 70, B: 160, A: 255}
			if x < 40 || x > 280 || y < 60 || y > 420 {
				shade = color.RGBA{R: 240, G: 240, B: 240, A: 255}
			}
			photo.Set(x, y, shade)
		}
	}

	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, photo, nil); err != nil {
		t.Fatalf("картинка: %v", err)
	}
	return encoded.Bytes()
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
