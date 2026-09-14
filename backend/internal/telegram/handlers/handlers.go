package handlers

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/texts"
)

func Default(ctx context.Context, b *bot.Bot, update *models.Update) {
	message := update.Message
	// В группе ответ на каждое сообщение был бы спамом.
	if message == nil || message.Chat.Type != models.ChatTypePrivate {
		return
	}

	if _, err := b.SendMessage(ctx, welcome(message)); err != nil {
		log.Printf("telegram: отправка сообщения в чат %d: %v", message.Chat.ID, err)
	}
}

func welcome(message *models.Message) *bot.SendMessageParams {
	var firstName string
	if message.From != nil {
		firstName = message.From.FirstName
	}

	return &bot.SendMessageParams{
		ChatID: message.Chat.ID,
		Text:   texts.Welcome(firstName),
	}
}
