package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

var ErrRecognitionUnavailable = errors.New("распознавание недоступно")

const maxSuggestedExtraColors = 3

type photoContents interface {
	Content(ctx context.Context, userID int64, key string) (PhotoContent, error)
}

type Recognition struct {
	photos     photoContents
	recognizer PhotoRecognizer
}

func NewRecognition(photos photoContents, recognizer PhotoRecognizer) *Recognition {
	return &Recognition{photos: photos, recognizer: recognizer}
}

func (recognition *Recognition) FromPhoto(ctx context.Context, userID int64, key string) (ItemSuggestion, error) {
	content, err := recognition.photos.Content(ctx, userID, key)
	if err != nil {
		return ItemSuggestion{}, fmt.Errorf("распознавание: %w", err)
	}

	suggestion, err := recognition.recognizer.Recognize(ctx, content)
	if err != nil {
		return ItemSuggestion{}, fmt.Errorf("распознавание: %w", err)
	}
	return consistent(suggestion), nil
}

// Только после ошибки: пустой ответ второй распознаватель просто повторит.
type Fallback struct {
	primary   PhotoRecognizer
	secondary PhotoRecognizer
}

func NewFallback(primary, secondary PhotoRecognizer) *Fallback {
	return &Fallback{primary: primary, secondary: secondary}
}

func (fallback *Fallback) Recognize(ctx context.Context, photo PhotoContent) (ItemSuggestion, error) {
	suggestion, err := fallback.primary.Recognize(ctx, photo)
	if err == nil {
		return suggestion, nil
	}

	log.Printf("основной распознаватель не смог, спрашиваю запасного: %v", err)
	return fallback.secondary.Recognize(ctx, photo)
}

// Невозможное сочетание пользователь увидел бы как ошибку сохранения, не понимая,
// что сделал не так.
func consistent(suggestion ItemSuggestion) ItemSuggestion {
	if utf8.RuneCountInString(strings.TrimSpace(suggestion.Name)) > domain.MaxNameLength {
		suggestion.Name = ""
	}
	suggestion.Name = strings.TrimSpace(suggestion.Name)

	if !suggestion.Category.Valid() {
		suggestion.Category = ""
	}
	if !suggestion.Category.HasWarmth() || !suggestion.WarmthLevel.Valid() {
		suggestion.WarmthLevel = 0
	}

	suggestion.Colors = consistentColors(suggestion.Colors)
	suggestion.Seasons = known(suggestion.Seasons, domain.Season.Valid)
	return suggestion
}

func consistentColors(colors domain.Colors) domain.Colors {
	if !colors.Main.Valid() {
		return domain.Colors{}
	}

	extra := known(colors.Extra, domain.Color.Valid)
	extra = slices.DeleteFunc(extra, func(color domain.Color) bool { return color == colors.Main })
	if len(extra) > maxSuggestedExtraColors {
		extra = extra[:maxSuggestedExtraColors]
	}
	return domain.Colors{Main: colors.Main, Extra: extra}
}

func known[T comparable](values []T, valid func(T) bool) []T {
	result := make([]T, 0, len(values))
	for _, value := range values {
		if valid(value) && !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
