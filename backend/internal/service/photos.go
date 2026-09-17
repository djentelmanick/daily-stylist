package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"
)

const MaxPhotoBytes = 20 << 20

const (
	uploadLinkTTL      = 10 * time.Minute
	downloadLinkTTL    = 2 * time.Hour
	abandonedUploadAge = 24 * time.Hour
)

var (
	ErrPhotoTooLarge        = errors.New("недопустимый размер фотографии")
	ErrPhotoTypeUnsupported = errors.New("неподдерживаемый формат фотографии")
)

var photoExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type Photos struct {
	storage PhotoStorage
	uploads PhotoUploadRepository
	now     func() time.Time
}

func NewPhotos(storage PhotoStorage, uploads PhotoUploadRepository, now func() time.Time) *Photos {
	return &Photos{storage: storage, uploads: uploads, now: now}
}

type PhotoUpload struct {
	Key     string
	URL     string
	ViewURL string
}

func (photos *Photos) RequestUpload(ctx context.Context, userID int64, contentType string, size int64) (PhotoUpload, error) {
	extension, ok := photoExtensions[contentType]
	if !ok {
		return PhotoUpload{}, fmt.Errorf("загрузка фотографии: %w: %q", ErrPhotoTypeUnsupported, contentType)
	}
	if size <= 0 || size > MaxPhotoBytes {
		return PhotoUpload{}, fmt.Errorf("загрузка фотографии: %w: %d байт", ErrPhotoTooLarge, size)
	}

	photos.cleanAbandoned(ctx, userID)

	key := fmt.Sprintf("users/%d/%s%s", userID, randomName(), extension)
	if err := photos.uploads.Create(ctx, userID, key); err != nil {
		return PhotoUpload{}, fmt.Errorf("загрузка фотографии: %w", err)
	}

	url, err := photos.storage.UploadLink(ctx, key, contentType, size, uploadLinkTTL)
	if err != nil {
		return PhotoUpload{}, fmt.Errorf("загрузка фотографии: %w", err)
	}
	viewURL, err := photos.Link(ctx, key)
	if err != nil {
		return PhotoUpload{}, fmt.Errorf("загрузка фотографии: %w", err)
	}
	return PhotoUpload{Key: key, URL: url, ViewURL: viewURL}, nil
}

func (photos *Photos) Confirm(ctx context.Context, userID int64, key string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("фотография %q: %w", key, err)
		}
	}()

	if err := photos.uploads.Take(ctx, userID, key); err != nil {
		return err
	}

	info, err := photos.storage.Describe(ctx, key)
	if err != nil {
		return err
	}
	if info.Size <= 0 || info.Size > MaxPhotoBytes {
		photos.Discard(ctx, key)
		return fmt.Errorf("%w: %d байт", ErrPhotoTooLarge, info.Size)
	}
	if _, ok := photoExtensions[info.ContentType]; !ok {
		photos.Discard(ctx, key)
		return fmt.Errorf("%w: %q", ErrPhotoTypeUnsupported, info.ContentType)
	}
	return nil
}

func (photos *Photos) Link(ctx context.Context, key string) (string, error) {
	url, err := photos.storage.DownloadLink(ctx, key, downloadLinkTTL)
	if err != nil {
		return "", fmt.Errorf("ссылка на фотографию %q: %w", key, err)
	}
	return url, nil
}

func (photos *Photos) Discard(ctx context.Context, keys ...string) {
	keys = notEmpty(keys)
	if len(keys) == 0 {
		return
	}
	if err := photos.storage.Delete(ctx, keys); err != nil {
		log.Printf("фотографии %v остались в хранилище: %v", keys, err)
	}
}

// Уборка при запросе новой ссылки, а не по расписанию: мусор копит тот же, кто грузит.
func (photos *Photos) cleanAbandoned(ctx context.Context, userID int64) {
	keys, err := photos.uploads.TakeOlderThan(ctx, userID, photos.now().Add(-abandonedUploadAge))
	if err != nil {
		log.Printf("брошенные загрузки пользователя %d: %v", userID, err)
		return
	}
	photos.Discard(ctx, keys...)
}

func randomName() string {
	name := make([]byte, 16)
	_, _ = rand.Read(name)
	return hex.EncodeToString(name)
}

func notEmpty(keys []string) []string {
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		if key != "" {
			result = append(result, key)
		}
	}
	return result
}
