package handlers

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/djentelmanick/daily-stylist/backend/internal/telegram"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/texts"
)

type outfits interface {
	WearToday(ctx context.Context, userID int64, itemIDs []int64) error
}

func New(outfits outfits) bot.HandlerFunc {
	handler := &handler{outfits: outfits}
	return handler.handle
}

type handler struct {
	outfits outfits
}

func (h *handler) handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery != nil {
		h.wear(ctx, b, update.CallbackQuery)
		return
	}

	message := update.Message
	// В группе ответ на каждое сообщение был бы спамом.
	if message == nil || message.Chat.Type != models.ChatTypePrivate {
		return
	}

	if _, err := b.SendMessage(ctx, welcome(message)); err != nil {
		log.Printf("telegram: отправка сообщения в чат %d: %v", message.Chat.ID, err)
	}
}

func (h *handler) wear(ctx context.Context, b *bot.Bot, query *models.CallbackQuery) {
	itemIDs, ok := telegram.WearItemIDs(query.Data)
	if !ok {
		answer(ctx, b, query.ID, "")
		return
	}

	if err := h.outfits.WearToday(ctx, query.From.ID, itemIDs); err != nil {
		log.Printf("telegram: запись образа пользователя %d: %v", query.From.ID, err)
		answer(ctx, b, query.ID, texts.WearFailed)
		return
	}

	answer(ctx, b, query.ID, texts.Worn)

	message := query.Message.Message
	if message == nil {
		return
	}
	_, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      message.Chat.ID,
		MessageID:   message.ID,
		ReplyMarkup: withoutPressed(message.ReplyMarkup),
	})
	if err != nil {
		log.Printf("telegram: обновление кнопок сообщения %d: %v", message.ID, err)
	}
}

func withoutPressed(markup *models.InlineKeyboardMarkup) *models.InlineKeyboardMarkup {
	kept := &models.InlineKeyboardMarkup{}
	if markup == nil {
		return kept
	}

	for _, row := range markup.InlineKeyboard {
		var buttons []models.InlineKeyboardButton
		for _, button := range row {
			if button.CallbackData == "" {
				buttons = append(buttons, button)
			}
		}
		if len(buttons) > 0 {
			kept.InlineKeyboard = append(kept.InlineKeyboard, buttons)
		}
	}
	return kept
}

func answer(ctx context.Context, b *bot.Bot, queryID, text string) {
	params := &bot.AnswerCallbackQueryParams{CallbackQueryID: queryID, Text: text}
	if _, err := b.AnswerCallbackQuery(ctx, params); err != nil {
		log.Printf("telegram: ответ на нажатие: %v", err)
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
