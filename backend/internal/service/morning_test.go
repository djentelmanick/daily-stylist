package service_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var morningDay = date(2026, 9, 16)

func TestMorningPlanner_PublishesTaskForEveryoneDue(t *testing.T) {
	deliveries := &fakeDeliveries{due: []service.MorningDelivery{{UserID: 42, Day: morningDay}, {UserID: 7, Day: morningDay}}}
	tasks := &fakeTasks{}
	planner := service.NewMorningPlanner(deliveries, tasks, fixedNow)

	if err := planner.PublishDue(t.Context()); err != nil {
		t.Fatalf("PublishDue: %v", err)
	}

	if !slices.Equal(tasks.published, []int64{42, 7}) {
		t.Errorf("поставлены задачи %v, ожидались оба пользователя", tasks.published)
	}
	if len(deliveries.claimed) != 0 {
		t.Error("планировщик занял день, хотя это дело отправщика")
	}
	if deliveries.params.Window != service.MorningWindow || deliveries.params.Defaults != domain.DefaultSettings() {
		t.Errorf("параметры выборки = %+v", deliveries.params)
	}
}

func TestMorningPlanner_ReportsQueueFailure(t *testing.T) {
	deliveries := &fakeDeliveries{due: []service.MorningDelivery{{UserID: 42, Day: morningDay}}}
	planner := service.NewMorningPlanner(deliveries, &fakeTasks{err: errors.New("очередь недоступна")}, fixedNow)

	if err := planner.PublishDue(t.Context()); err == nil {
		t.Error("ошибка очереди потерялась, о ней некому узнать")
	}
}

func TestMorningSender_SendsRecommendation(t *testing.T) {
	deliveries := &fakeDeliveries{}
	notifier := &fakeNotifier{}
	sender := service.NewMorningSender(deliveries, &fakeMorningRecommender{recommendation: outfitFor("Футболка")}, notifier)

	if err := sender.Deliver(t.Context(), service.MorningDelivery{UserID: 42, Day: morningDay}); err != nil {
		t.Fatalf("Deliver: %v", err)
	}

	if !slices.Equal(notifier.sent, []int64{42}) {
		t.Errorf("рекомендация ушла %v, ожидался пользователь 42", notifier.sent)
	}
	if len(deliveries.claimed) != 1 || len(deliveries.released) != 0 {
		t.Errorf("занято дней %d, возвращено %d, ожидалось 1 и 0", len(deliveries.claimed), len(deliveries.released))
	}
}

func TestMorningSender_SkipsDayTakenByAnother(t *testing.T) {
	deliveries := &fakeDeliveries{taken: true}
	notifier := &fakeNotifier{}
	sender := service.NewMorningSender(deliveries, &fakeMorningRecommender{recommendation: outfitFor("Футболка")}, notifier)

	if err := sender.Deliver(t.Context(), service.MorningDelivery{UserID: 42, Day: morningDay}); err != nil {
		t.Fatalf("Deliver: %v", err)
	}

	if len(notifier.sent) != 0 {
		t.Errorf("отправлено %v, ожидалось молчание: день уже занят", notifier.sent)
	}
}

func TestMorningSender_ReturnsDayAfterFailure(t *testing.T) {
	tests := map[string]struct {
		recommender *fakeMorningRecommender
		notifier    *fakeNotifier
	}{
		"подбор не удался": {
			recommender: &fakeMorningRecommender{err: service.ErrWeatherUnavailable},
			notifier:    &fakeNotifier{},
		},
		"отправка не удалась": {
			recommender: &fakeMorningRecommender{recommendation: outfitFor("Футболка")},
			notifier:    &fakeNotifier{err: errors.New("таймаут")},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			deliveries := &fakeDeliveries{}
			sender := service.NewMorningSender(deliveries, test.recommender, test.notifier)

			if err := sender.Deliver(t.Context(), service.MorningDelivery{UserID: 42, Day: morningDay}); err == nil {
				t.Error("ошибка потерялась, задача считается выполненной")
			}
			if len(deliveries.released) != 1 {
				t.Error("день не возвращён в работу, следующая минута не попробует снова")
			}
		})
	}
}

func TestMorningSender_KeepsDayWhenChatUnavailable(t *testing.T) {
	deliveries := &fakeDeliveries{}
	notifier := &fakeNotifier{err: service.ErrChatUnavailable}
	sender := service.NewMorningSender(deliveries, &fakeMorningRecommender{recommendation: outfitFor("Футболка")}, notifier)

	if err := sender.Deliver(t.Context(), service.MorningDelivery{UserID: 42, Day: morningDay}); err == nil {
		t.Error("ошибка потерялась, задача считается выполненной")
	}
	if len(deliveries.released) != 0 {
		t.Error("день возвращён в работу, хотя бота заблокировали: повтор ничего не изменит")
	}
}

func TestMorningSender_SilentWhenNothingToRecommend(t *testing.T) {
	deliveries := &fakeDeliveries{}
	notifier := &fakeNotifier{}
	sender := service.NewMorningSender(deliveries, &fakeMorningRecommender{}, notifier)

	if err := sender.Deliver(t.Context(), service.MorningDelivery{UserID: 42, Day: morningDay}); err != nil {
		t.Fatalf("Deliver: %v", err)
	}

	if len(notifier.sent) != 0 {
		t.Errorf("отправлено %v, ожидалось молчание: подбирать не из чего", notifier.sent)
	}
	if len(deliveries.released) != 0 {
		t.Error("день возвращён в работу, хотя гардероб за этот час вряд ли изменится")
	}
}

func outfitFor(name string) service.Recommendation {
	return service.Recommendation{
		Location: vladivostok,
		Outfits:  []domain.Outfit{{Items: []domain.Item{testItem(1, name, domain.CategoryTop)}}},
	}
}

type fakeTasks struct {
	published []int64
	err       error
}

func (tasks *fakeTasks) Publish(_ context.Context, delivery service.MorningDelivery) error {
	if tasks.err != nil {
		return tasks.err
	}
	tasks.published = append(tasks.published, delivery.UserID)
	return nil
}

type fakeDeliveries struct {
	due      []service.MorningDelivery
	params   service.DueParams
	taken    bool
	claimed  []service.MorningDelivery
	released []service.MorningDelivery
}

func (deliveries *fakeDeliveries) Due(_ context.Context, params service.DueParams) ([]service.MorningDelivery, error) {
	deliveries.params = params
	return deliveries.due, nil
}

func (deliveries *fakeDeliveries) Claim(_ context.Context, delivery service.MorningDelivery) (bool, error) {
	if deliveries.taken {
		return false, nil
	}
	deliveries.claimed = append(deliveries.claimed, delivery)
	return true, nil
}

func (deliveries *fakeDeliveries) Release(_ context.Context, delivery service.MorningDelivery) error {
	deliveries.released = append(deliveries.released, delivery)
	return nil
}

type fakeMorningRecommender struct {
	recommendation service.Recommendation
	err            error
}

func (recommender *fakeMorningRecommender) Recommend(context.Context, int64) (service.Recommendation, error) {
	return recommender.recommendation, recommender.err
}

type fakeNotifier struct {
	sent []int64
	err  error
}

func (notifier *fakeNotifier) SendRecommendation(_ context.Context, userID int64, _ service.Recommendation) error {
	if notifier.err != nil {
		return notifier.err
	}
	notifier.sent = append(notifier.sent, userID)
	return nil
}
