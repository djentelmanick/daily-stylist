//go:build integration

package s3_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/s3"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

const linkTTL = time.Minute

// Тесты ходят в живое хранилище: важно как раз то, что нельзя проверить фейком, -
// принимает ли оно подписанные ссылки такими, какими их выдаёт адаптер.
func newTestStorage(t *testing.T) *s3.PhotoStorage {
	t.Helper()

	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("нет TEST_S3_ENDPOINT: тесты хранилища пропущены")
	}
	bucket := os.Getenv("TEST_S3_BUCKET")
	if bucket == "" {
		t.Fatal("нет TEST_S3_BUCKET: не указывайте здесь рабочий бакет")
	}

	return s3.NewPhotoStorage(s3.Config{
		Endpoint:  endpoint,
		PublicURL: endpoint,
		Region:    envOr("S3_REGION", "us-east-1"),
		Bucket:    bucket,
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
	})
}

func TestPhotoStorage_UploadDescribeDownloadDelete(t *testing.T) {
	storage := newTestStorage(t)
	key := testKey(t)
	photo := []byte("не настоящая картинка, но байты настоящие")

	upload, err := storage.UploadLink(t.Context(), key, "image/jpeg", int64(len(photo)), linkTTL)
	if err != nil {
		t.Fatalf("UploadLink: %v", err)
	}
	if code := put(t, upload, "image/jpeg", photo); code != http.StatusOK {
		t.Fatalf("загрузка вернула %d, ожидался 200", code)
	}

	info, err := storage.Describe(t.Context(), key)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if info.Size != int64(len(photo)) || info.ContentType != "image/jpeg" {
		t.Errorf("в хранилище %+v, ожидались %d байт типа image/jpeg", info, len(photo))
	}

	download, err := storage.DownloadLink(t.Context(), key, linkTTL)
	if err != nil {
		t.Fatalf("DownloadLink: %v", err)
	}
	if got := get(t, download); !bytes.Equal(got, photo) {
		t.Errorf("скачалось %q, ожидалось %q", got, photo)
	}

	if err := storage.Delete(t.Context(), []string{key}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := storage.Describe(t.Context(), key); !errors.Is(err, service.ErrPhotoNotUploaded) {
		t.Errorf("после удаления ошибка = %v, ожидалась ErrPhotoNotUploaded", err)
	}
}

func TestPhotoStorage_LinkBindsSizeAndType(t *testing.T) {
	storage := newTestStorage(t)
	key := testKey(t)

	link, err := storage.UploadLink(t.Context(), key, "image/jpeg", 10, linkTTL)
	if err != nil {
		t.Fatalf("UploadLink: %v", err)
	}

	// Ссылка выдана на 10 байт: файл побольше по ней не загрузить.
	if code := put(t, link, "image/jpeg", bytes.Repeat([]byte("ф"), 100)); code == http.StatusOK {
		t.Errorf("хранилище приняло файл, который больше подписанного размера")
	}
	// И тип подменить нельзя: он тоже входит в подпись.
	if code := put(t, link, "application/pdf", []byte("0123456789")); code == http.StatusOK {
		t.Errorf("хранилище приняло файл с подменённым типом")
	}
	if _, err := storage.Describe(t.Context(), key); !errors.Is(err, service.ErrPhotoNotUploaded) {
		t.Errorf("после отклонённых загрузок ошибка = %v, ожидалась ErrPhotoNotUploaded", err)
	}
}

func TestPhotoStorage_LinkIsRequired(t *testing.T) {
	storage := newTestStorage(t)
	key := testKey(t)

	link, err := storage.DownloadLink(t.Context(), key, linkTTL)
	if err != nil {
		t.Fatalf("DownloadLink: %v", err)
	}
	unsigned, _, _ := bytes.Cut([]byte(link), []byte("?"))

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, string(unsigned), nil)
	if err != nil {
		t.Fatalf("запрос: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("запрос без подписи: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusForbidden {
		t.Errorf("запрос без подписи вернул %d, ожидался 403", response.StatusCode)
	}
}

func put(t *testing.T, link, contentType string, body []byte) int {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPut, link, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("запрос: %v", err)
	}
	request.Header.Set("Content-Type", contentType)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("загрузка: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode
}

func get(t *testing.T, link string) []byte {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, link, nil)
	if err != nil {
		t.Fatalf("запрос: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("скачивание: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("скачивание вернуло %d, ожидался 200", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("чтение ответа: %v", err)
	}
	return body
}

// Ключ по имени теста: тесты не мешают друг другу и видно, чей файл остался.
func testKey(t *testing.T) string {
	t.Helper()
	return "tests/" + t.Name() + ".jpg"
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
