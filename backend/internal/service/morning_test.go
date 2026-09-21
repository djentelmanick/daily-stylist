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

func TestMorning_SendsToEveryoneDue(t *testing.T) {
	deliveries := &fakeDeliveries{due: []service.MorningDelivery{{UserID: 42, Day: morningDay}, {UserID: 7, Day: morningDay}}}
	notifier := &fakeNotifier{}
	morning := service.NewMorning(deliveries, &fakeMorningRecommender{recommendation: outfitFor("Футболка")}, notifier, fixedNow)

	if err := morning.SendDue(t.Context()); err != nil {
		t.Fatalf("SendDue: %v", err)
	}

	if !slices.Equal(notifier.sent, []int64{42, 7}) {
		t.Errorf("рекомендация ушла %v, ожидались оба пользователя", notifier.sent)
	}
	if len(deliveries.claimed) != 2 || len(deliveries.released) != 0 {
		t.Errorf("занято дней %d, возвращено %d, ожидалось 2 и 0", len(deliveries.claimed), len(deliveries.released))
	}
	if deliveries.params.Window != service.MorningWindow || deliveries.params.Defaults != domain.DefaultSettings() {
		t.Errorf("параметры выборки = %+v", deliveries.params)
	}
}

func TestMorning_SkipsDayTakenByAnother(t *testing.T) {
	deliveries := &fakeDeliveries{due: []service.MorningDelivery{{UserID: 42, Day: morningDay}}, taken: true}
	notifier := &fakeNotifier{}
	morning := service.NewMorning(deliveries, &fakeMorningRecommender{recommendation: outfitFor("Футболка")}, notifier, fixedNow)

	if err := morning.SendDue(t.Context()); err != nil {
		t.Fatalf("SendDue: %v", err)
	}

	if len(notifier.sent) != 0 {
		t.Errorf("отправлено %v, ожидалось молчание: день уже занят", notifier.sent)
	}
}

func TestMorning_ReturnsDayAfterFailure(t *testing.T) {
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
			deliveries := &fakeDeliveries{due: []service.MorningDelivery{{UserID: 42, Day: morningDay}}}
			morning := service.NewMorning(deliveries, test.recommender, test.notifier, fixedNow)

			if err := morning.SendDue(t.Context()); err != nil {
				t.Fatalf("SendDue: %v", err)
			}

			if len(deliveries.released) != 1 {
				t.Errorf("день не возвращён в работу, следующая минута не попробует снова")
			}
		})
	}
}

func TestMorning_KeepsDayWhenChatUnavailable(t *testing.T) {
	deliveries := &fakeDeliveries{due: []service.MorningDelivery{{UserID: 42, Day: morningDay}}}
	notifier := &fakeNotifier{err: service.ErrChatUnavailable}
	morning := service.NewMorning(deliveries, &fakeMorningRecommender{recommendation: outfitFor("Футболка")}, notifier, fixedNow)

	if err := morning.SendDue(t.Context()); err != nil {
		t.Fatalf("SendDue: %v", err)
	}

	if len(deliveries.released) != 0 {
		t.Error("день возвращён в работу, хотя бота заблокировали: повтор ничего не изменит")
	}
}

func TestMorning_SilentWhenNothingToRecommend(t *testing.T) {
	deliveries := &fakeDeliveries{due: []service.MorningDelivery{{UserID: 42, Day: morningDay}}}
	notifier := &fakeNotifier{}
	morning := service.NewMorning(deliveries, &fakeMorningRecommender{}, notifier, fixedNow)

	if err := morning.SendDue(t.Context()); err != nil {
		t.Fatalf("SendDue: %v", err)
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
