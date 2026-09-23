package handlers

import (
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestWelcome(t *testing.T) {
	tests := []struct {
		name         string
		from         *models.User
		wantGreeting string
	}{
		{"по имени", &models.User{FirstName: "Коля"}, "Привет, Коля!"},
		{"без имени", &models.User{}, "Привет!"},
		{"без отправителя", nil, "Привет!"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := welcome(&models.Message{Chat: models.Chat{ID: 42, Type: models.ChatTypePrivate}, From: test.from}, "https://app.example")

			if params.ChatID != int64(42) {
				t.Errorf("чат = %v, ожидался 42", params.ChatID)
			}
			if !strings.HasPrefix(params.Text, test.wantGreeting+"\n") {
				t.Errorf("текст начинается не с %q:\n%s", test.wantGreeting, params.Text)
			}
			keyboard, ok := params.ReplyMarkup.(*models.InlineKeyboardMarkup)
			if !ok || keyboard.InlineKeyboard[0][0].WebApp == nil || keyboard.InlineKeyboard[0][0].WebApp.URL != "https://app.example" {
				t.Errorf("нет кнопки приложения: %+v", params.ReplyMarkup)
			}
		})
	}
}
