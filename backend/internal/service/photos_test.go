package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

var testNow = func() time.Time { return time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC) }

func TestRequestUpload_RemembersUploadAndSignsLink(t *testing.T) {
	storage := &fakePhotoStorage{}
	uploads := &memoryPhotoUploads{}
	photos := NewPhotos(storage, uploads, testNow)

	upload, err := photos.RequestUpload(t.Context(), 42, "image/jpeg", 1024)
	if err != nil {
		t.Fatalf("RequestUpload: %v", err)
	}

	if !strings.HasPrefix(upload.Key, "users/42/") || !strings.HasSuffix(upload.Key, ".jpg") {
		t.Errorf("ключ = %q, ожидался вида users/42/<имя>.jpg", upload.Key)
	}
	if upload.URL != "upload:"+upload.Key {
		t.Errorf("ссылка на загрузку = %q, ожидалась подписанная ссылка на этот ключ", upload.URL)
	}
	if upload.ViewURL != "download:"+upload.Key {
		t.Errorf("ссылка на просмотр = %q, ожидалась подписанная ссылка на этот ключ", upload.ViewURL)
	}
	if owner, ok := uploads.owners[upload.Key]; !ok || owner != 42 {
		t.Errorf("загрузка не записана за пользователем 42: %v", uploads.owners)
	}
	if storage.uploadSize != 1024 || storage.uploadType != "image/jpeg" {
		t.Errorf("подпись выдана на %d байт типа %q, ожидались 1024 и image/jpeg", storage.uploadSize, storage.uploadType)
	}
}

func TestRequestUpload_RejectsUnsupportedTypeAndSize(t *testing.T) {
	uploads := &memoryPhotoUploads{}
	photos := NewPhotos(&fakePhotoStorage{}, uploads, testNow)

	if _, err := photos.RequestUpload(t.Context(), 42, "application/pdf", 1024); !errors.Is(err, ErrPhotoTypeUnsupported) {
		t.Errorf("для pdf ошибка = %v, ожидалась ErrPhotoTypeUnsupported", err)
	}
	if _, err := photos.RequestUpload(t.Context(), 42, "image/jpeg", MaxPhotoBytes+1); !errors.Is(err, ErrPhotoTooLarge) {
		t.Errorf("для файла больше лимита ошибка = %v, ожидалась ErrPhotoTooLarge", err)
	}
	if len(uploads.owners) != 0 {
		t.Errorf("отклонённые загрузки не должны попадать в базу: %v", uploads.owners)
	}
}

func TestConfirm_TakesUploadOnlyOnce(t *testing.T) {
	storage := &fakePhotoStorage{}
	photos := NewPhotos(storage, &memoryPhotoUploads{}, testNow)
	upload := requestUpload(t, photos, 42)
	storage.put(upload.Key, PhotoInfo{Size: 2048, ContentType: "image/jpeg"})

	if err := photos.Confirm(t.Context(), 42, upload.Key); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	// Повторное подтверждение - признак того, что ключ подставили руками:
	// у настоящей загрузки запись уже забрана первой вещью.
	if err := photos.Confirm(t.Context(), 42, upload.Key); !errors.Is(err, ErrPhotoNotUploaded) {
		t.Errorf("повторный Confirm: ошибка = %v, ожидалась ErrPhotoNotUploaded", err)
	}
}

func TestConfirm_RejectsSomeoneElsesUpload(t *testing.T) {
	storage := &fakePhotoStorage{}
	photos := NewPhotos(storage, &memoryPhotoUploads{}, testNow)
	upload := requestUpload(t, photos, 42)
	storage.put(upload.Key, PhotoInfo{Size: 2048, ContentType: "image/jpeg"})

	if err := photos.Confirm(t.Context(), 7, upload.Key); !errors.Is(err, ErrPhotoNotUploaded) {
		t.Errorf("чужая загрузка: ошибка = %v, ожидалась ErrPhotoNotUploaded", err)
	}
}

func TestConfirm_DeletesFileThatBrokeTheLimits(t *testing.T) {
	storage := &fakePhotoStorage{}
	photos := NewPhotos(storage, &memoryPhotoUploads{}, testNow)
	upload := requestUpload(t, photos, 42)
	storage.put(upload.Key, PhotoInfo{Size: MaxPhotoBytes + 1, ContentType: "image/jpeg"})

	if err := photos.Confirm(t.Context(), 42, upload.Key); !errors.Is(err, ErrPhotoTooLarge) {
		t.Fatalf("ошибка = %v, ожидалась ErrPhotoTooLarge", err)
	}
	if _, ok := storage.files[upload.Key]; ok {
		t.Errorf("файл %q остался в хранилище", upload.Key)
	}
}

func TestConfirm_FailsWhenNothingWasUploaded(t *testing.T) {
	storage := &fakePhotoStorage{}
	photos := NewPhotos(storage, &memoryPhotoUploads{}, testNow)
	upload := requestUpload(t, photos, 42)

	// Ссылку выдали, но браузер файл так и не положил.
	if err := photos.Confirm(t.Context(), 42, upload.Key); !errors.Is(err, ErrPhotoNotUploaded) {
		t.Errorf("ошибка = %v, ожидалась ErrPhotoNotUploaded", err)
	}
}

func TestRequestUpload_CleansOnlyAbandonedUploads(t *testing.T) {
	storage := &fakePhotoStorage{}
	now := testNow()
	clock := func() time.Time { return now }
	uploads := &memoryPhotoUploads{now: clock}
	photos := NewPhotos(storage, uploads, clock)

	abandoned := requestUpload(t, photos, 42)
	storage.put(abandoned.Key, PhotoInfo{Size: 2048, ContentType: "image/jpeg"})
	now = now.Add(abandonedUploadAge - time.Minute)
	recent := requestUpload(t, photos, 42)
	storage.put(recent.Key, PhotoInfo{Size: 2048, ContentType: "image/jpeg"})

	now = now.Add(2 * time.Minute)
	fresh := requestUpload(t, photos, 42)

	if _, ok := storage.files[abandoned.Key]; ok {
		t.Errorf("брошенный файл %q остался в хранилище", abandoned.Key)
	}
	if _, ok := uploads.owners[abandoned.Key]; ok {
		t.Errorf("запись о брошенной загрузке осталась в базе")
	}
	if _, ok := storage.files[recent.Key]; !ok {
		t.Errorf("недавняя загрузка %q удалена вместе с брошенной", recent.Key)
	}
	if _, ok := uploads.owners[fresh.Key]; !ok {
		t.Errorf("свежая загрузка потерялась")
	}
}

func TestRequestUpload_KeepsOtherUsersUploads(t *testing.T) {
	storage := &fakePhotoStorage{}
	now := testNow()
	clock := func() time.Time { return now }
	uploads := &memoryPhotoUploads{now: clock}
	photos := NewPhotos(storage, uploads, clock)

	stranger := requestUpload(t, photos, 7)
	storage.put(stranger.Key, PhotoInfo{Size: 2048, ContentType: "image/jpeg"})

	now = now.Add(abandonedUploadAge + time.Minute)
	requestUpload(t, photos, 42)

	if _, ok := storage.files[stranger.Key]; !ok {
		t.Errorf("уборка тронула файл чужого пользователя")
	}
}

func requestUpload(t *testing.T, photos *Photos, userID int64) PhotoUpload {
	t.Helper()
	upload, err := photos.RequestUpload(t.Context(), userID, "image/jpeg", 2048)
	if err != nil {
		t.Fatalf("RequestUpload: %v", err)
	}
	return upload
}

type fakePhotoStorage struct {
	files      map[string]PhotoInfo
	uploadSize int64
	uploadType string
	readErr    error
}

func (storage *fakePhotoStorage) put(key string, info PhotoInfo) {
	if storage.files == nil {
		storage.files = map[string]PhotoInfo{}
	}
	storage.files[key] = info
}

func (storage *fakePhotoStorage) UploadLink(_ context.Context, key, contentType string, size int64, _ time.Duration) (string, error) {
	storage.uploadSize, storage.uploadType = size, contentType
	return "upload:" + key, nil
}

func (storage *fakePhotoStorage) DownloadLink(_ context.Context, key string, _ time.Duration) (string, error) {
	return "download:" + key, nil
}

func (storage *fakePhotoStorage) Describe(_ context.Context, key string) (PhotoInfo, error) {
	info, ok := storage.files[key]
	if !ok {
		return PhotoInfo{}, ErrPhotoNotUploaded
	}
	return info, nil
}

func (storage *fakePhotoStorage) Read(_ context.Context, key string) (PhotoContent, error) {
	if storage.readErr != nil {
		return PhotoContent{}, storage.readErr
	}
	info, ok := storage.files[key]
	if !ok {
		return PhotoContent{}, ErrPhotoNotUploaded
	}
	return PhotoContent{Bytes: []byte("байты " + key), ContentType: info.ContentType}, nil
}

func (storage *fakePhotoStorage) Delete(_ context.Context, keys []string) error {
	for _, key := range keys {
		delete(storage.files, key)
	}
	return nil
}

type memoryPhotoUploads struct {
	owners  map[string]int64
	created map[string]time.Time
	now     func() time.Time
}

func (uploads *memoryPhotoUploads) Create(_ context.Context, userID int64, key string) error {
	if uploads.owners == nil {
		uploads.owners, uploads.created = map[string]int64{}, map[string]time.Time{}
	}
	if uploads.now == nil {
		uploads.now = testNow
	}
	uploads.owners[key] = userID
	uploads.created[key] = uploads.now()
	return nil
}

func (uploads *memoryPhotoUploads) Take(_ context.Context, userID int64, key string) error {
	if owner, ok := uploads.owners[key]; !ok || owner != userID {
		return ErrPhotoNotUploaded
	}
	delete(uploads.owners, key)
	delete(uploads.created, key)
	return nil
}

func (uploads *memoryPhotoUploads) TakeOlderThan(_ context.Context, userID int64, before time.Time) ([]string, error) {
	var keys []string
	for key, owner := range uploads.owners {
		if owner == userID && uploads.created[key].Before(before) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	for _, key := range keys {
		delete(uploads.owners, key)
		delete(uploads.created, key)
	}
	return keys, nil
}
