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
			params := welcome(&models.Message{Chat: models.Chat{ID: 42, Type: models.ChatTypePrivate}, From: test.from})

			if params.ChatID != int64(42) {
				t.Errorf("чат = %v, ожидался 42", params.ChatID)
			}
			if !strings.HasPrefix(params.Text, test.wantGreeting+"\n") {
				t.Errorf("текст начинается не с %q:\n%s", test.wantGreeting, params.Text)
			}
		})
	}
}
