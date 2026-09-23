package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

type Topology struct {
	Exchange   string
	RoutingKey string
	Queue      string
}

func MorningTopology() Topology {
	return Topology{Exchange: "morning", RoutingKey: "send", Queue: "morning.send"}
}

func Connect(rabbitURL string) (*amqp.Connection, error) {
	connection, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("подключение к rabbitmq: %w", err)
	}
	return connection, nil
}

func (topology Topology) declare(channel *amqp.Channel) error {
	if err := channel.ExchangeDeclare(topology.Exchange, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return fmt.Errorf("обменник %s: %w", topology.Exchange, err)
	}
	if _, err := channel.QueueDeclare(topology.Queue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("очередь %s: %w", topology.Queue, err)
	}
	if err := channel.QueueBind(topology.Queue, topology.RoutingKey, topology.Exchange, false, nil); err != nil {
		return fmt.Errorf("привязка очереди %s: %w", topology.Queue, err)
	}
	return nil
}

// Задача едет строкой даты: местный день пользователя - это дата, а не момент времени.
type morningTask struct {
	UserID int64  `json:"user_id"`
	Day    string `json:"day"`
}

var _ service.MorningTasks = (*Tasks)(nil)

type Tasks struct {
	channel  *amqp.Channel
	topology Topology
}

func NewTasks(connection *amqp.Connection, topology Topology) (*Tasks, error) {
	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("канал для задач: %w", err)
	}
	if err := topology.declare(channel); err != nil {
		_ = channel.Close()
		return nil, err
	}
	// Без подтверждений публикация считалась бы удачной, даже если брокер задачу потерял.
	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("подтверждения публикации: %w", err)
	}
	return &Tasks{channel: channel, topology: topology}, nil
}

func (tasks *Tasks) Close() error {
	return tasks.channel.Close()
}

func (tasks *Tasks) Publish(ctx context.Context, delivery service.MorningDelivery) error {
	day := delivery.Day.Format(time.DateOnly)
	body, err := json.Marshal(morningTask{UserID: delivery.UserID, Day: day})
	if err != nil {
		return fmt.Errorf("задача на рассылку: %w", err)
	}

	confirmation, err := tasks.channel.PublishWithDeferredConfirmWithContext(
		ctx, tasks.topology.Exchange, tasks.topology.RoutingKey, true, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    fmt.Sprintf("%d:%s", delivery.UserID, day),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("постановка задачи на рассылку: %w", err)
	}
	accepted, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("подтверждение задачи на рассылку: %w", err)
	}
	if !accepted {
		return fmt.Errorf("постановка задачи на рассылку: брокер не принял задачу пользователя %d", delivery.UserID)
	}
	return nil
}

type Consumer struct {
	channel  *amqp.Channel
	topology Topology
}

// prefetch - сколько задач отправщик берёт вперёд, не подтвердив предыдущие.
func NewConsumer(connection *amqp.Connection, topology Topology, prefetch int) (*Consumer, error) {
	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("канал отправщика: %w", err)
	}
	if err := topology.declare(channel); err != nil {
		_ = channel.Close()
		return nil, err
	}
	if err := channel.Qos(prefetch, 0, false); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("настройка канала отправщика: %w", err)
	}
	return &Consumer{channel: channel, topology: topology}, nil
}

func (consumer *Consumer) Close() error {
	return consumer.channel.Close()
}

// Задача подтверждается, когда обработчик вернул nil. Неудачную задачу заново поставит
// планировщик: пока день не занят, следующая минута найдёт того же человека.
func (consumer *Consumer) Consume(ctx context.Context, handle func(context.Context, service.MorningDelivery) error) error {
	messages, err := consumer.channel.ConsumeWithContext(ctx, consumer.topology.Queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("чтение очереди %s: %w", consumer.topology.Queue, err)
	}

	for message := range messages {
		delivery, err := decodeMorningTask(message.Body)
		if err != nil {
			// Разобрать такую задачу не выйдет и со второй попытки.
			_ = message.Reject(false)
			return err
		}
		if err := handle(ctx, delivery); err != nil {
			_ = message.Nack(false, false)
			continue
		}
		if err := message.Ack(false); err != nil {
			return fmt.Errorf("подтверждение задачи: %w", err)
		}
	}
	return ctx.Err()
}

func decodeMorningTask(body []byte) (service.MorningDelivery, error) {
	var task morningTask
	if err := json.Unmarshal(body, &task); err != nil {
		return service.MorningDelivery{}, fmt.Errorf("задача на рассылку: %w", err)
	}
	day, err := time.Parse(time.DateOnly, task.Day)
	if err != nil {
		return service.MorningDelivery{}, fmt.Errorf("задача на рассылку: день %q: %w", task.Day, err)
	}
	return service.MorningDelivery{UserID: task.UserID, Day: day}, nil
}
