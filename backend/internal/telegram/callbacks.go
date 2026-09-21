package telegram

import (
	"strconv"
	"strings"
)

// Телеграм разрешает 64 байта на кнопку, поэтому длинный образ в неё не влезает.
const (
	wearPrefix           = "wear:"
	maxCallbackDataBytes = 64
)

func WearData(itemIDs []int64) string {
	if len(itemIDs) == 0 {
		return ""
	}

	ids := make([]string, len(itemIDs))
	for index, itemID := range itemIDs {
		ids[index] = strconv.FormatInt(itemID, 10)
	}

	data := wearPrefix + strings.Join(ids, ",")
	if len(data) > maxCallbackDataBytes {
		return ""
	}
	return data
}

func WearItemIDs(data string) ([]int64, bool) {
	list, found := strings.CutPrefix(data, wearPrefix)
	if !found || list == "" {
		return nil, false
	}

	ids := strings.Split(list, ",")
	itemIDs := make([]int64, 0, len(ids))
	for _, text := range ids {
		itemID, err := strconv.ParseInt(text, 10, 64)
		if err != nil || itemID <= 0 {
			return nil, false
		}
		itemIDs = append(itemIDs, itemID)
	}
	return itemIDs, true
}
