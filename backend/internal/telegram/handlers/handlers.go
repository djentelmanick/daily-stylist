package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/texts"
)

const namePrefix = "Меня зовут "

func Default(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	answer := update.Message.Text
	if name, ok := strings.CutPrefix(update.Message.Text, namePrefix); ok {
		answer = fmt.Sprintf(texts.Greeting, name)
	}

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   answer,
	}); err != nil {
		log.Printf("telegram: отправка сообщения в чат %d: %v", update.Message.Chat.ID, err)
	}
}
