package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

type fakeRecognizer struct {
	suggestion ItemSuggestion
	err        error
	calls      int
	seen       PhotoContent
}

func (recognizer *fakeRecognizer) Recognize(_ context.Context, photo PhotoContent) (ItemSuggestion, error) {
	recognizer.calls++
	recognizer.seen = photo
	if recognizer.err != nil {
		return ItemSuggestion{}, recognizer.err
	}
	return recognizer.suggestion, nil
}

func recognitionWith(t *testing.T, suggestion ItemSuggestion) (*Recognition, *fakeRecognizer, string) {
	t.Helper()

	storage := &fakePhotoStorage{}
	photos := NewPhotos(storage, &memoryPhotoUploads{}, testNow)
	upload := requestUpload(t, photos, 42)
	storage.put(upload.Key, PhotoInfo{Size: 2048, ContentType: "image/jpeg"})

	recognizer := &fakeRecognizer{suggestion: suggestion}
	return NewRecognition(photos, recognizer), recognizer, upload.Key
}

func TestFromPhoto_GivesRecognizerTheFile(t *testing.T) {
	recognition, recognizer, key := recognitionWith(t, ItemSuggestion{
		Name:        "Пуховик",
		Category:    domain.CategoryOuterwear,
		Colors:      domain.Colors{Main: domain.ColorNavy, Extra: []domain.Color{domain.ColorGray}},
		Seasons:     []domain.Season{domain.SeasonWinter},
		WarmthLevel: domain.WarmthLevelExtreme,
	})

	suggestion, err := recognition.FromPhoto(t.Context(), 42, key)
	if err != nil {
		t.Fatalf("FromPhoto: %v", err)
	}

	if string(recognizer.seen.Bytes) != "байты "+key || recognizer.seen.ContentType != "image/jpeg" {
		t.Errorf("распознавателю ушло %+v, ожидались байты и тип фотографии", recognizer.seen)
	}
	if suggestion.Name != "Пуховик" || suggestion.Category != domain.CategoryOuterwear {
		t.Errorf("подсказка = %+v", suggestion)
	}
	if suggestion.WarmthLevel != domain.WarmthLevelExtreme {
		t.Errorf("теплота = %d, ожидалась %d", suggestion.WarmthLevel, domain.WarmthLevelExtreme)
	}
}

func TestFromPhoto_RefusesForeignPhoto(t *testing.T) {
	recognition, recognizer, key := recognitionWith(t, ItemSuggestion{})

	_, err := recognition.FromPhoto(t.Context(), 7, key)

	if !errors.Is(err, ErrPhotoNotUploaded) {
		t.Errorf("ошибка = %v, ожидалась ErrPhotoNotUploaded", err)
	}
	if recognizer.calls != 0 {
		t.Errorf("чужая фотография ушла на распознавание %d раз", recognizer.calls)
	}
}

func TestFromPhoto_DropsUnknownValues(t *testing.T) {
	recognition, _, key := recognitionWith(t, ItemSuggestion{
		Category:    "пиджачок",
		Colors:      domain.Colors{Main: "бирюзовый"},
		Seasons:     []domain.Season{domain.SeasonWinter, "межсезонье"},
		WarmthLevel: 9,
	})

	suggestion, err := recognition.FromPhoto(t.Context(), 42, key)
	if err != nil {
		t.Fatalf("FromPhoto: %v", err)
	}

	if suggestion.Category != "" {
		t.Errorf("категория = %q, незнакомую ожидалось отбросить", suggestion.Category)
	}
	if suggestion.Colors.Main != "" {
		t.Errorf("главный цвет = %q, незнакомый ожидалось отбросить", suggestion.Colors.Main)
	}
	if !slices.Equal(suggestion.Seasons, []domain.Season{domain.SeasonWinter}) {
		t.Errorf("сезоны = %v, ожидалась только зима", suggestion.Seasons)
	}
	if suggestion.WarmthLevel != 0 {
		t.Errorf("теплота = %d, вне диапазона ожидался ноль", suggestion.WarmthLevel)
	}
}

func TestFromPhoto_ClearsWarmthWhereCategoryHasNone(t *testing.T) {
	recognition, _, key := recognitionWith(t, ItemSuggestion{
		Category:    domain.CategoryUmbrella,
		WarmthLevel: domain.WarmthLevelWarm,
	})

	suggestion, err := recognition.FromPhoto(t.Context(), 42, key)
	if err != nil {
		t.Fatalf("FromPhoto: %v", err)
	}

	if suggestion.WarmthLevel != 0 {
		t.Errorf("теплота зонта = %d, ожидался ноль", suggestion.WarmthLevel)
	}
}

func TestFromPhoto_KeepsColorsUsableByTheForm(t *testing.T) {
	recognition, _, key := recognitionWith(t, ItemSuggestion{
		Category: domain.CategoryTop,
		Colors: domain.Colors{
			Main: domain.ColorBlue,
			Extra: []domain.Color{
				domain.ColorBlue, domain.ColorWhite, domain.ColorWhite,
				domain.ColorGray, domain.ColorBlack, domain.ColorNavy,
			},
		},
	})

	suggestion, err := recognition.FromPhoto(t.Context(), 42, key)
	if err != nil {
		t.Fatalf("FromPhoto: %v", err)
	}

	want := []domain.Color{domain.ColorWhite, domain.ColorGray, domain.ColorBlack}
	if !slices.Equal(suggestion.Colors.Extra, want) {
		t.Errorf("дополнительные цвета = %v, ожидались %v без главного, повторов и сверх трёх", suggestion.Colors.Extra, want)
	}
}

func TestFromPhoto_DropsNameThatFormCannotAccept(t *testing.T) {
	recognition, _, key := recognitionWith(t, ItemSuggestion{
		Name:     strings.Repeat("я", domain.MaxNameLength+1),
		Category: domain.CategoryTop,
	})

	suggestion, err := recognition.FromPhoto(t.Context(), 42, key)
	if err != nil {
		t.Fatalf("FromPhoto: %v", err)
	}

	if suggestion.Name != "" {
		t.Errorf("название = %q, слишком длинное ожидалось отбросить", suggestion.Name)
	}
}

func TestFromPhoto_PassesRecognizerFailureThrough(t *testing.T) {
	recognition, recognizer, key := recognitionWith(t, ItemSuggestion{})
	recognizer.err = ErrRecognitionUnavailable

	_, err := recognition.FromPhoto(t.Context(), 42, key)

	if !errors.Is(err, ErrRecognitionUnavailable) {
		t.Errorf("ошибка = %v, ожидалась ErrRecognitionUnavailable", err)
	}
}

func TestFallback_AsksSecondOnlyWhenFirstFailed(t *testing.T) {
	primary := &fakeRecognizer{suggestion: ItemSuggestion{Name: "от основного"}}
	secondary := &fakeRecognizer{suggestion: ItemSuggestion{Name: "от запасного"}}
	fallback := NewFallback(primary, secondary)

	suggestion, err := fallback.Recognize(t.Context(), PhotoContent{Bytes: []byte("снимок")})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	if suggestion.Name != "от основного" {
		t.Errorf("подсказка = %q, ожидалась от основного", suggestion.Name)
	}
	if secondary.calls != 0 {
		t.Errorf("запасного спросили %d раз, хотя основной справился", secondary.calls)
	}
}

func TestFallback_AsksSecondWhenFirstFailed(t *testing.T) {
	primary := &fakeRecognizer{err: ErrRecognitionUnavailable}
	secondary := &fakeRecognizer{suggestion: ItemSuggestion{Name: "от запасного"}}
	fallback := NewFallback(primary, secondary)

	suggestion, err := fallback.Recognize(t.Context(), PhotoContent{Bytes: []byte("снимок")})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	if suggestion.Name != "от запасного" {
		t.Errorf("подсказка = %q, ожидалась от запасного", suggestion.Name)
	}
	if string(secondary.seen.Bytes) != "снимок" {
		t.Errorf("запасному ушло %+v", secondary.seen)
	}
}

func TestFallback_BothFailedMeansNoSuggestion(t *testing.T) {
	primary := &fakeRecognizer{err: ErrRecognitionUnavailable}
	secondary := &fakeRecognizer{err: ErrRecognitionUnavailable}

	_, err := NewFallback(primary, secondary).Recognize(t.Context(), PhotoContent{})

	if !errors.Is(err, ErrRecognitionUnavailable) {
		t.Errorf("ошибка = %v, ожидалась ErrRecognitionUnavailable", err)
	}
}
