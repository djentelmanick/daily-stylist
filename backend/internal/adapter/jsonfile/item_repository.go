package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var _ service.ItemRepository = (*ItemRepository)(nil)

type ItemRepository struct {
	path  string
	mutex sync.Mutex
}

func NewItemRepository(path string) *ItemRepository {
	return &ItemRepository{path: path}
}

type fileContent struct {
	LastID int64        `json:"last_id"`
	Items  []itemRecord `json:"items"`
}

type itemRecord struct {
	ID          int64    `json:"id"`
	UserID      int64    `json:"user_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	MainColor   string   `json:"main_color"`
	ExtraColors []string `json:"extra_colors"`
	Seasons     []string `json:"seasons"`
	WarmthLevel int      `json:"warmth_level"`
	Waterproof  bool     `json:"waterproof"`
	Status      string   `json:"status"`
}

func (repository *ItemRepository) Create(ctx context.Context, item domain.Item) (domain.Item, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	content, err := repository.load()
	if err != nil {
		return domain.Item{}, err
	}

	content.LastID++
	item.ID = content.LastID
	content.Items = append(content.Items, toRecord(item))

	if err := repository.save(content); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func (repository *ItemRepository) CountByUser(ctx context.Context, userID int64) (int, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()

	content, err := repository.load()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, record := range content.Items {
		if record.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (repository *ItemRepository) load() (fileContent, error) {
	data, err := os.ReadFile(repository.path)
	if errors.Is(err, fs.ErrNotExist) {
		return fileContent{}, nil
	}
	if err != nil {
		return fileContent{}, fmt.Errorf("чтение %s: %w", repository.path, err)
	}

	var content fileContent
	if err := json.Unmarshal(data, &content); err != nil {
		return fileContent{}, fmt.Errorf("разбор %s: %w", repository.path, err)
	}
	return content, nil
}

func (repository *ItemRepository) save(content fileContent) error {
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("кодирование вещей: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(repository.path), 0o755); err != nil {
		return fmt.Errorf("создание папки для %s: %w", repository.path, err)
	}

	temporaryPath := repository.path + ".tmp"
	if err := os.WriteFile(temporaryPath, data, 0o644); err != nil {
		return fmt.Errorf("запись %s: %w", temporaryPath, err)
	}
	if err := os.Rename(temporaryPath, repository.path); err != nil {
		return fmt.Errorf("замена %s: %w", repository.path, err)
	}
	return nil
}

func toRecord(item domain.Item) itemRecord {
	return itemRecord{
		ID:          item.ID,
		UserID:      item.UserID,
		Name:        item.Name,
		Description: item.Description,
		Category:    string(item.Category),
		MainColor:   string(item.Colors.Main),
		ExtraColors: toStrings(item.Colors.Extra),
		Seasons:     toStrings(item.Seasons),
		WarmthLevel: int(item.WarmthLevel),
		Waterproof:  item.Waterproof,
		Status:      string(item.Status),
	}
}

func toStrings[T ~string](values []T) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = string(value)
	}
	return result
}
