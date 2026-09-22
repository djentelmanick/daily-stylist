package telegram

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
	"github.com/djentelmanick/daily-stylist/backend/internal/telegram/texts"
)

type Notifier struct {
	api        *bot.Bot
	miniAppURL string
}

func NewNotifier(ctx context.Context, token, miniAppURL string) (*Notifier, error) {
	var api *bot.Bot
	err := retryStartup(ctx, startupPause, func() error {
		var err error
		api, err = bot.New(token)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("создание клиента telegram: %w", err)
	}
	return &Notifier{api: api, miniAppURL: miniAppURL}, nil
}

func (notifier *Notifier) SendRecommendation(
	ctx context.Context,
	userID int64,
	recommendation service.Recommendation,
) error {
	outfit := recommendation.Outfits[0]
	_, err := notifier.api.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userID,
		Text:        texts.Morning(recommendation.Location, recommendation.Weather, outfit, recommendation.Notes),
		ReplyMarkup: morningKeyboard(notifier.miniAppURL, outfit),
	})
	if err == nil {
		return nil
	}
	if !transient(err) {
		return fmt.Errorf("%w: %w", service.ErrChatUnavailable, err)
	}
	return fmt.Errorf("отправка рекомендации: %w", err)
}

func morningKeyboard(miniAppURL string, outfit domain.Outfit) *models.InlineKeyboardMarkup {
	itemIDs := make([]int64, len(outfit.Items))
	for index, item := range outfit.Items {
		itemIDs[index] = item.ID
	}

	var buttons []models.InlineKeyboardButton
	if data := WearData(itemIDs); data != "" {
		buttons = append(buttons, models.InlineKeyboardButton{Text: texts.Wear, CallbackData: data})
	}
	buttons = append(buttons, models.InlineKeyboardButton{
		Text:   texts.MoreOutfits,
		WebApp: &models.WebAppInfo{URL: recommendationURL(miniAppURL)},
	})

	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{buttons}}
}

func recommendationURL(miniAppURL string) string {
	parsed, err := url.Parse(miniAppURL)
	if err != nil {
		return miniAppURL
	}
	query := parsed.Query()
	query.Set("screen", "recommendation")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
