package service

import (
	"context"
	"errors"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

var (
	ErrItemNotFound     = errors.New("вещь не найдена")
	ErrLocationNotSet   = errors.New("город не выбран")
	ErrSettingsNotSet   = errors.New("настройки не менялись")
	ErrPhotoNotUploaded = errors.New("фотография не загружена")
)

type ItemRepository interface {
	Create(ctx context.Context, item domain.Item) (domain.Item, error)
	CountByUser(ctx context.Context, userID int64) (int, error)
	ListByUser(ctx context.Context, userID int64) ([]domain.Item, error)
	Get(ctx context.Context, userID, itemID int64) (domain.Item, error)
	Update(ctx context.Context, item domain.Item) error
	UpdateStatus(ctx context.Context, userID, itemID int64, status domain.ItemStatus) error
	Delete(ctx context.Context, userID int64, itemIDs []int64) ([]string, error)
}

type PhotoUploadRepository interface {
	Create(ctx context.Context, userID int64, key string) error
	Take(ctx context.Context, userID int64, key string) error
	TakeOlderThan(ctx context.Context, userID int64, before time.Time) ([]string, error)
}

type PhotoInfo struct {
	Size        int64
	ContentType string
}

type PhotoContent struct {
	Bytes       []byte
	ContentType string
}

type PhotoStorage interface {
	UploadLink(ctx context.Context, key, contentType string, size int64, ttl time.Duration) (string, error)
	DownloadLink(ctx context.Context, key string, ttl time.Duration) (string, error)
	Describe(ctx context.Context, key string) (PhotoInfo, error)
	Read(ctx context.Context, key string) (PhotoContent, error)
	Delete(ctx context.Context, keys []string) error
}

// Пустое поле - распознать не удалось.
type ItemSuggestion struct {
	Name        string
	Category    domain.Category
	Colors      domain.Colors
	Seasons     []domain.Season
	WarmthLevel domain.WarmthLevel
	Waterproof  bool
}

type PhotoRecognizer interface {
	Recognize(ctx context.Context, userID int64, photo PhotoContent) (ItemSuggestion, error)
}

type RecognitionCounter interface {
	// Окно отсчитывается от первого распознавания, а не от полуночи.
	Increment(ctx context.Context, userID int64, window time.Duration) (int, error)
}

type LocationRepository interface {
	Get(ctx context.Context, userID int64) (domain.Location, error)
	Save(ctx context.Context, userID int64, location domain.Location) error
}

type OutfitRepository interface {
	SaveWorn(ctx context.Context, userID int64, day time.Time, itemIDs []int64) error
	WornOn(ctx context.Context, userID int64, day time.Time) ([]domain.Item, error)
	LastWorn(ctx context.Context, userID int64, from, to time.Time) (map[int64]time.Time, error)
}

type Forecaster interface {
	Forecast(ctx context.Context, location domain.Location, from, to time.Time) (domain.Weather, error)
}

type CitySearch interface {
	SearchCities(ctx context.Context, query string) ([]domain.Location, error)
}

type SettingsRepository interface {
	Get(ctx context.Context, userID int64) (domain.Settings, error)
	Save(ctx context.Context, userID int64, settings domain.Settings) error
}

// Day - местный день пользователя, а не день сервера.
type MorningDelivery struct {
	UserID int64
	Day    time.Time
}

type DueParams struct {
	Now      time.Time
	Window   time.Duration
	Defaults domain.Settings
}

type DeliveryRepository interface {
	Due(ctx context.Context, params DueParams) ([]MorningDelivery, error)
	Claim(ctx context.Context, delivery MorningDelivery) (bool, error)
	Release(ctx context.Context, delivery MorningDelivery) error
}

type Notifier interface {
	SendRecommendation(ctx context.Context, userID int64, recommendation Recommendation) error
}
