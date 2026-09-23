package telegram

import (
	"slices"
	"strings"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func TestWearDataRoundTrip(t *testing.T) {
	itemIDs := []int64{7, 1, 42}

	data := WearData(itemIDs)

	got, ok := WearItemIDs(data)
	if !ok || !slices.Equal(got, itemIDs) {
		t.Errorf("WearItemIDs(%q) = %v, %v, ожидались %v", data, got, ok, itemIDs)
	}
}

func TestWearDataTooLong(t *testing.T) {
	itemIDs := make([]int64, 20)
	for index := range itemIDs {
		itemIDs[index] = int64(1_000_000 + index)
	}

	if data := WearData(itemIDs); data != "" {
		t.Errorf("данные кнопки = %q (%d байт), ожидалась пустая строка: в кнопку влезает 64 байта", data, len(data))
	}
}

func TestWearItemIDsRejectsJunk(t *testing.T) {
	for _, data := range []string{"", "wear:", "wear:0", "wear:1,,2", "wear:1,два", "другое:1"} {
		if _, ok := WearItemIDs(data); ok {
			t.Errorf("WearItemIDs(%q) приняты, ожидался отказ", data)
		}
	}
}

func TestMorningKeyboard(t *testing.T) {
	outfit := domain.Outfit{Items: []domain.Item{{ID: 3, Name: "Куртка"}, {ID: 5, Name: "Кеды"}}}

	keyboard := morningKeyboard("https://xxx.ngrok-free.app", outfit)

	rows := keyboard.InlineKeyboard
	if len(rows) != 2 || len(rows[0]) != 1 || len(rows[1]) != 1 {
		t.Fatalf("клавиатура = %+v, ожидались «надеваю» и «ещё варианты», по кнопке в ряд", rows)
	}
	if itemIDs, ok := WearItemIDs(rows[0][0].CallbackData); !ok || !slices.Equal(itemIDs, []int64{3, 5}) {
		t.Errorf("первая кнопка = %+v, ожидалась запись образа", rows[0][0])
	}
	if rows[1][0].WebApp == nil || !strings.Contains(rows[1][0].WebApp.URL, "screen=recommendation") {
		t.Errorf("вторая кнопка = %+v, ожидалось открытие подбора в приложении", rows[1][0])
	}
}

func TestMorningKeyboardWithoutWearButton(t *testing.T) {
	outfit := domain.Outfit{Items: make([]domain.Item, 20)}
	for index := range outfit.Items {
		outfit.Items[index].ID = int64(1_000_000 + index)
	}

	keyboard := morningKeyboard("https://xxx.ngrok-free.app", outfit)

	if len(keyboard.InlineKeyboard) != 1 || len(keyboard.InlineKeyboard[0]) != 1 {
		t.Errorf("клавиатура = %+v, ожидались только «ещё варианты»: образ в кнопку не влез", keyboard.InlineKeyboard)
	}
}
