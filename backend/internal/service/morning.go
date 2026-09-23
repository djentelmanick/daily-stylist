package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

const MorningWindow = time.Hour

var ErrChatUnavailable = errors.New("чат недоступен")

type morningRecommender interface {
	Recommend(ctx context.Context, userID int64) (Recommendation, error)
}

// Планировщик только ищет, кому пора: собирает образ и отправляет его MorningSender.
type MorningPlanner struct {
	deliveries DeliveryRepository
	tasks      MorningTasks
	now        func() time.Time
}

func NewMorningPlanner(deliveries DeliveryRepository, tasks MorningTasks, now func() time.Time) *MorningPlanner {
	return &MorningPlanner{deliveries: deliveries, tasks: tasks, now: now}
}

func (planner *MorningPlanner) PublishDue(ctx context.Context) error {
	due, err := planner.deliveries.Due(ctx, DueParams{
		Now:      planner.now(),
		Window:   MorningWindow,
		Defaults: domain.DefaultSettings(),
	})
	if err != nil {
		return fmt.Errorf("утренняя рассылка: %w", err)
	}

	// День занимает отправщик, поэтому неудачная постановка ничего не теряет:
	// следующая минута найдёт тех же людей снова.
	for _, delivery := range due {
		if err := planner.tasks.Publish(ctx, delivery); err != nil {
			return fmt.Errorf("утренняя рассылка: %w", err)
		}
	}
	return nil
}

type MorningSender struct {
	deliveries  DeliveryRepository
	recommender morningRecommender
	notifier    Notifier
}

func NewMorningSender(deliveries DeliveryRepository, recommender morningRecommender, notifier Notifier) *MorningSender {
	return &MorningSender{deliveries: deliveries, recommender: recommender, notifier: notifier}
}

func (sender *MorningSender) Deliver(ctx context.Context, delivery MorningDelivery) error {
	claimed, err := sender.deliveries.Claim(ctx, delivery)
	if err != nil || !claimed {
		return err
	}

	recommendation, err := sender.recommender.Recommend(ctx, delivery.UserID)
	if err != nil {
		return sender.release(ctx, delivery, err)
	}
	// День не возвращаем: за оставшийся час гардероб вряд ли изменится.
	if len(recommendation.Outfits) == 0 {
		return nil
	}

	if err := sender.notifier.SendRecommendation(ctx, delivery.UserID, recommendation); err != nil {
		if errors.Is(err, ErrChatUnavailable) {
			return err
		}
		return sender.release(ctx, delivery, err)
	}
	return nil
}

func (sender *MorningSender) release(ctx context.Context, delivery MorningDelivery, cause error) error {
	if err := sender.deliveries.Release(ctx, delivery); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}
