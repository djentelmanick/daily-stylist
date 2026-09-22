package vision

import (
	"context"
	"errors"
	"net"
	"slices"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/recognizers/vision/visionpb"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

type fakeRecognizer struct {
	visionpb.UnimplementedRecognizerServer
	response *visionpb.RecognizeItemResponse
	err      error
	seen     *visionpb.RecognizeItemRequest
}

func (recognizer *fakeRecognizer) RecognizeItem(
	_ context.Context,
	request *visionpb.RecognizeItemRequest,
) (*visionpb.RecognizeItemResponse, error) {
	recognizer.seen = request
	if recognizer.err != nil {
		return nil, recognizer.err
	}
	return recognizer.response, nil
}

func startService(t *testing.T, recognizer *fakeRecognizer) *Client {
	t.Helper()

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("слушатель: %v", err)
	}
	server := grpc.NewServer()
	visionpb.RegisterRecognizerServer(server, recognizer)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	client, err := New(listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("клиент: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestRecognize_SendsPhotoAndReadsSuggestion(t *testing.T) {
	recognizer := &fakeRecognizer{response: &visionpb.RecognizeItemResponse{
		Name:        "Пуховик",
		Category:    "outerwear",
		MainColor:   "navy",
		ExtraColors: []string{"gray", "white"},
		Seasons:     []string{"winter"},
		WarmthLevel: 5,
		Waterproof:  true,
	}}
	client := startService(t, recognizer)

	suggestion, err := client.Recognize(t.Context(), 42, service.PhotoContent{Bytes: []byte("jpeg"), ContentType: "image/jpeg"})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	if string(recognizer.seen.GetPhoto()) != "jpeg" || recognizer.seen.GetContentType() != "image/jpeg" {
		t.Errorf("сервису ушло %+v, ожидались байты и тип фотографии", recognizer.seen)
	}
	if suggestion.Name != "Пуховик" || suggestion.Category != domain.CategoryOuterwear {
		t.Errorf("подсказка = %+v", suggestion)
	}
	if suggestion.Colors.Main != domain.ColorNavy {
		t.Errorf("главный цвет = %q", suggestion.Colors.Main)
	}
	if !slices.Equal(suggestion.Colors.Extra, []domain.Color{domain.ColorGray, domain.ColorWhite}) {
		t.Errorf("дополнительные цвета = %v", suggestion.Colors.Extra)
	}
	if !slices.Equal(suggestion.Seasons, []domain.Season{domain.SeasonWinter}) {
		t.Errorf("сезоны = %v", suggestion.Seasons)
	}
	if suggestion.WarmthLevel != domain.WarmthLevelExtreme || !suggestion.Waterproof {
		t.Errorf("теплота = %d, непромокаемость = %v", suggestion.WarmthLevel, suggestion.Waterproof)
	}
}

// Отбраковкой занимается service.
func TestRecognize_PassesUnknownValuesAsIs(t *testing.T) {
	client := startService(t, &fakeRecognizer{response: &visionpb.RecognizeItemResponse{
		Category:  "пиджачок",
		MainColor: "бирюзовый",
	}})

	suggestion, err := client.Recognize(t.Context(), 42, service.PhotoContent{Bytes: []byte("jpeg")})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	if suggestion.Category != "пиджачок" || suggestion.Colors.Main != "бирюзовый" {
		t.Errorf("подсказка = %+v, ожидались значения без изменений", suggestion)
	}
}

func TestRecognize_EmptyAnswerHasNoColorsOrSeasons(t *testing.T) {
	client := startService(t, &fakeRecognizer{response: &visionpb.RecognizeItemResponse{}})

	suggestion, err := client.Recognize(t.Context(), 42, service.PhotoContent{Bytes: []byte("jpeg")})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	if suggestion.Colors.Extra != nil || suggestion.Seasons != nil {
		t.Errorf("подсказка = %+v, ожидалась пустая", suggestion)
	}
}

func TestRecognize_FailureMeansNoSuggestion(t *testing.T) {
	client := startService(t, &fakeRecognizer{err: status.Error(codes.Internal, "модель не загрузилась")})

	_, err := client.Recognize(t.Context(), 42, service.PhotoContent{Bytes: []byte("jpeg")})

	if !errors.Is(err, service.ErrRecognitionUnavailable) {
		t.Errorf("ошибка = %v, ожидалась ErrRecognitionUnavailable", err)
	}
}

func TestRecognize_UnreachableServiceMeansNoSuggestion(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("слушатель: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("закрытие слушателя: %v", err)
	}

	client, err := New(address, time.Second)
	if err != nil {
		t.Fatalf("клиент: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	if _, err := client.Recognize(t.Context(), 42, service.PhotoContent{Bytes: []byte("jpeg")}); !errors.Is(err, service.ErrRecognitionUnavailable) {
		t.Errorf("ошибка = %v, ожидалась ErrRecognitionUnavailable", err)
	}
}
