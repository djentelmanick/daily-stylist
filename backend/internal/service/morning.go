package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

// Столько времени утренняя рекомендация ещё имеет смысл: перезапуск планировщика
// или недолгая авария не съедают день, а позже человек уже оделся сам.
const MorningWindow = time.Hour

// Бота заблокировали или чат удалён: повторять отправку бессмысленно.
var ErrChatUnavailable = errors.New("чат недоступен")

type morningRecommender interface {
	Recommend(ctx context.Context, userID int64) (Recommendation, error)
}

type Morning struct {
	deliveries  DeliveryRepository
	recommender morningRecommender
	notifier    Notifier
	now         func() time.Time
}

func NewMorning(
	deliveries DeliveryRepository,
	recommender morningRecommender,
	notifier Notifier,
	now func() time.Time,
) *Morning {
	return &Morning{deliveries: deliveries, recommender: recommender, notifier: notifier, now: now}
}

func (morning *Morning) SendDue(ctx context.Context) error {
	due, err := morning.deliveries.Due(ctx, DueParams{
		Now:      morning.now(),
		Window:   MorningWindow,
		Defaults: domain.DefaultSettings(),
	})
	if err != nil {
		return fmt.Errorf("утренняя рассылка: %w", err)
	}

	for _, delivery := range due {
		if err := morning.send(ctx, delivery); err != nil {
			log.Printf("утренняя рассылка пользователю %d: %v", delivery.UserID, err)
		}
	}
	return nil
}

func (morning *Morning) send(ctx context.Context, delivery MorningDelivery) error {
	claimed, err := morning.deliveries.Claim(ctx, delivery)
	if err != nil || !claimed {
		return err
	}

	recommendation, err := morning.recommender.Recommend(ctx, delivery.UserID)
	if err != nil {
		return morning.release(ctx, delivery, err)
	}
	// Подобрать нечего: гардероб пуст или не по погоде. Молчим, но день считаем
	// разобранным - за оставшийся час гардероб вряд ли изменится.
	if len(recommendation.Outfits) == 0 {
		return nil
	}

	if err := morning.notifier.SendRecommendation(ctx, delivery.UserID, recommendation); err != nil {
		if errors.Is(err, ErrChatUnavailable) {
			return err
		}
		return morning.release(ctx, delivery, err)
	}
	return nil
}

func (morning *Morning) release(ctx context.Context, delivery MorningDelivery, cause error) error {
	if err := morning.deliveries.Release(ctx, delivery); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}
