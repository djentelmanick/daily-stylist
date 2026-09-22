package vision

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/recognizers/vision/visionpb"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var _ service.PhotoRecognizer = (*Client)(nil)

const messageMarginBytes = 1 << 20

type Client struct {
	connection *grpc.ClientConn
	recognizer visionpb.RecognizerClient
	timeout    time.Duration
}

func New(address string, timeout time.Duration) (*Client, error) {
	connection, err := grpc.NewClient(
		address,
		// Без TLS: сервис распознавания стоит рядом с ботом и наружу не смотрит.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// Фотография едет одним сообщением, а по умолчанию gRPC разрешает 4 МБ.
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(service.MaxPhotoBytes+messageMarginBytes)),
	)
	if err != nil {
		return nil, fmt.Errorf("клиент распознавания %q: %w", address, err)
	}
	return &Client{
		connection: connection,
		recognizer: visionpb.NewRecognizerClient(connection),
		timeout:    timeout,
	}, nil
}

func (client *Client) Close() error {
	return client.connection.Close()
}

func (client *Client) Recognize(ctx context.Context, _ int64, photo service.PhotoContent) (service.ItemSuggestion, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	response, err := client.recognizer.RecognizeItem(ctx, &visionpb.RecognizeItemRequest{
		Photo:       photo.Bytes,
		ContentType: photo.ContentType,
	})
	if err != nil {
		return service.ItemSuggestion{}, fmt.Errorf("%w: %w", service.ErrRecognitionUnavailable, err)
	}
	return suggestion(response), nil
}

func suggestion(response *visionpb.RecognizeItemResponse) service.ItemSuggestion {
	return service.ItemSuggestion{
		Name:     response.GetName(),
		Category: domain.Category(response.GetCategory()),
		Colors: domain.Colors{
			Main:  domain.Color(response.GetMainColor()),
			Extra: values[domain.Color](response.GetExtraColors()),
		},
		Seasons:     values[domain.Season](response.GetSeasons()),
		WarmthLevel: domain.WarmthLevel(response.GetWarmthLevel()),
		Waterproof:  response.GetWaterproof(),
	}
}

func values[T ~string](raw []string) []T {
	if len(raw) == 0 {
		return nil
	}
	result := make([]T, len(raw))
	for index, value := range raw {
		result[index] = T(value)
	}
	return result
}
