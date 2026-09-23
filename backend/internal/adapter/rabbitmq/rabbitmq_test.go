//go:build integration

package rabbitmq_test

import (
	"context"
	"crypto/rand"
	"math/big"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/djentelmanick/daily-stylist/backend/internal/adapter/rabbitmq"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func TestTasks_TaskReachesConsumer(t *testing.T) {
	connection, topology := newTestQueue(t)
	tasks, consumer := newTestPair(t, connection, topology)
	sent := service.MorningDelivery{UserID: 42, Day: time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)}

	if err := tasks.Publish(t.Context(), sent); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	got := consumeOne(t, consumer, func(service.MorningDelivery) error { return nil })
	if got != sent {
		t.Errorf("отправщик получил %+v, ожидалось %+v", got, sent)
	}
	if left := queueLength(t, connection, topology); left != 0 {
		t.Errorf("в очереди осталось задач: %d, ожидалось 0 - задача подтверждена", left)
	}
}

func TestConsumer_FailedTaskLeavesQueue(t *testing.T) {
	connection, topology := newTestQueue(t)
	tasks, consumer := newTestPair(t, connection, topology)

	if err := tasks.Publish(t.Context(), service.MorningDelivery{UserID: 42, Day: time.Now()}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	consumeOne(t, consumer, func(service.MorningDelivery) error { return context.DeadlineExceeded })
	// Задачу снимаем с очереди: повтор - дело планировщика, он поставит её заново.
	if left := queueLength(t, connection, topology); left != 0 {
		t.Errorf("в очереди осталось задач: %d, ожидалось 0", left)
	}
}

func newTestQueue(t *testing.T) (*amqp.Connection, rabbitmq.Topology) {
	t.Helper()

	rabbitURL := os.Getenv("TEST_RABBITMQ_URL")
	if rabbitURL == "" {
		t.Skip("TEST_RABBITMQ_URL не задан")
	}
	connection, err := rabbitmq.Connect(rabbitURL)
	if err != nil {
		t.Fatalf("rabbitmq недоступен, поднимите его: make db\n%v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	// Рабочие обменник и очередь тесты не трогают: у каждого прогона свои.
	suffix := randomSuffix(t)
	topology := rabbitmq.Topology{
		Exchange:   "test.morning." + suffix,
		RoutingKey: "send",
		Queue:      "test.morning.send." + suffix,
	}
	t.Cleanup(func() {
		channel, err := connection.Channel()
		if err != nil {
			return
		}
		defer func() { _ = channel.Close() }()
		_, _ = channel.QueueDelete(topology.Queue, false, false, false)
		_ = channel.ExchangeDelete(topology.Exchange, false, false)
	})
	return connection, topology
}

func newTestPair(t *testing.T, connection *amqp.Connection, topology rabbitmq.Topology) (*rabbitmq.Tasks, *rabbitmq.Consumer) {
	t.Helper()

	tasks, err := rabbitmq.NewTasks(connection, topology)
	if err != nil {
		t.Fatalf("NewTasks: %v", err)
	}
	t.Cleanup(func() { _ = tasks.Close() })

	consumer, err := rabbitmq.NewConsumer(connection, topology, 1)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}
	t.Cleanup(func() { _ = consumer.Close() })
	return tasks, consumer
}

// Ждёт одну задачу и останавливает чтение: контекст отменяется прямо из обработчика.
func consumeOne(
	t *testing.T,
	consumer *rabbitmq.Consumer,
	handle func(service.MorningDelivery) error,
) service.MorningDelivery {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	var got service.MorningDelivery
	err := consumer.Consume(ctx, func(_ context.Context, delivery service.MorningDelivery) error {
		got = delivery
		cancel()
		return handle(delivery)
	})
	if err != nil && ctx.Err() == nil {
		t.Fatalf("Consume: %v", err)
	}
	if got == (service.MorningDelivery{}) {
		t.Fatal("задача не дошла до отправщика")
	}
	return got
}

func queueLength(t *testing.T, connection *amqp.Connection, topology rabbitmq.Topology) int {
	t.Helper()

	channel, err := connection.Channel()
	if err != nil {
		t.Fatalf("канал для проверки очереди: %v", err)
	}
	defer func() { _ = channel.Close() }()

	queue, err := channel.QueueDeclarePassive(topology.Queue, true, false, false, false, nil)
	if err != nil {
		t.Fatalf("состояние очереди: %v", err)
	}
	return queue.Messages
}

func randomSuffix(t *testing.T) string {
	t.Helper()

	value, err := rand.Int(rand.Reader, big.NewInt(1<<40))
	if err != nil {
		t.Fatalf("случайное имя очереди: %v", err)
	}
	return value.String()
}
