package handlers

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestWithoutPressed(t *testing.T) {
	markup := &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
		{Text: "Ещё варианты", WebApp: &models.WebAppInfo{URL: "https://example.test"}},
		{Text: "Надеваю", CallbackData: "wear:1,2"},
	}}}

	kept := withoutPressed(markup)

	if len(kept.InlineKeyboard) != 1 || len(kept.InlineKeyboard[0]) != 1 {
		t.Fatalf("кнопки = %+v, ожидалась одна", kept.InlineKeyboard)
	}
	if kept.InlineKeyboard[0][0].WebApp == nil {
		t.Errorf("осталась кнопка %+v, ожидалось открытие приложения", kept.InlineKeyboard[0][0])
	}
}

func TestWithoutPressedEmptyKeyboard(t *testing.T) {
	if kept := withoutPressed(nil); kept == nil || len(kept.InlineKeyboard) != 0 {
		t.Errorf("кнопки = %+v, ожидалась пустая клавиатура", kept)
	}
}
