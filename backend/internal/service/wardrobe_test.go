package service

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
)

func TestAddItem_KeepsConfirmedPhoto(t *testing.T) {
	storage, wardrobe, photos := newWardrobeWithPhotos()
	key := uploadedPhoto(t, photos, storage, 42)

	item, err := wardrobe.AddItem(t.Context(), newItemParams(42, key))
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if item.PhotoKey != key {
		t.Errorf("ключ фотографии = %q, ожидался %q", item.PhotoKey, key)
	}
	if _, ok := storage.files[key]; !ok {
		t.Errorf("файл %q пропал из хранилища", key)
	}
}

func TestAddItem_RejectsPhotoNobodyUploaded(t *testing.T) {
	_, wardrobe, _ := newWardrobeWithPhotos()

	_, err := wardrobe.AddItem(t.Context(), newItemParams(42, "users/42/чужая.jpg"))

	if !errors.Is(err, ErrPhotoNotUploaded) {
		t.Fatalf("ошибка = %v, ожидалась ErrPhotoNotUploaded", err)
	}
}

func TestEditItem_RemovesReplacedPhoto(t *testing.T) {
	storage, wardrobe, photos := newWardrobeWithPhotos()
	oldKey := uploadedPhoto(t, photos, storage, 42)
	item, err := wardrobe.AddItem(t.Context(), newItemParams(42, oldKey))
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	newKey := uploadedPhoto(t, photos, storage, 42)

	edited, err := wardrobe.EditItem(t.Context(), 42, item.ID, editItemParams(newKey))
	if err != nil {
		t.Fatalf("EditItem: %v", err)
	}

	if edited.PhotoKey != newKey {
		t.Errorf("ключ фотографии = %q, ожидался %q", edited.PhotoKey, newKey)
	}
	if _, ok := storage.files[oldKey]; ok {
		t.Errorf("заменённый файл %q остался в хранилище", oldKey)
	}
	if _, ok := storage.files[newKey]; !ok {
		t.Errorf("новый файл %q пропал из хранилища", newKey)
	}
}

func TestEditItem_KeepsPhotoWhenItDidNotChange(t *testing.T) {
	storage, wardrobe, photos := newWardrobeWithPhotos()
	key := uploadedPhoto(t, photos, storage, 42)
	item, err := wardrobe.AddItem(t.Context(), newItemParams(42, key))
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	// Тот же ключ второй раз подтверждать нечего: запись о загрузке уже забрана.
	if _, err := wardrobe.EditItem(t.Context(), 42, item.ID, editItemParams(key)); err != nil {
		t.Fatalf("EditItem: %v", err)
	}
	if _, ok := storage.files[key]; !ok {
		t.Errorf("файл %q удалён, хотя фотографию не меняли", key)
	}
}

func TestDeleteItems_RemovesPhotos(t *testing.T) {
	storage, wardrobe, photos := newWardrobeWithPhotos()
	key := uploadedPhoto(t, photos, storage, 42)
	item, err := wardrobe.AddItem(t.Context(), newItemParams(42, key))
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	if err := wardrobe.DeleteItems(t.Context(), 42, []int64{item.ID}); err != nil {
		t.Fatalf("DeleteItems: %v", err)
	}
	if _, ok := storage.files[key]; ok {
		t.Errorf("файл %q остался в хранилище после удаления вещи", key)
	}
}

func newWardrobeWithPhotos() (*fakePhotoStorage, *Wardrobe, *Photos) {
	storage := &fakePhotoStorage{}
	photos := NewPhotos(storage, &memoryPhotoUploads{}, testNow)
	return storage, NewWardrobe(&memoryItems{}, photos), photos
}

func uploadedPhoto(t *testing.T, photos *Photos, storage *fakePhotoStorage, userID int64) string {
	t.Helper()
	upload := requestUpload(t, photos, userID)
	storage.put(upload.Key, PhotoInfo{Size: 2048, ContentType: "image/jpeg"})
	return upload.Key
}

func newItemParams(userID int64, photoKey string) domain.NewItemParams {
	return domain.NewItemParams{
		UserID:      userID,
		Name:        "Синее худи",
		Category:    domain.CategoryTop,
		Colors:      domain.Colors{Main: domain.ColorBlue},
		Seasons:     []domain.Season{domain.SeasonAutumn},
		WarmthLevel: domain.WarmthLevelMedium,
		PhotoKey:    photoKey,
	}
}

func editItemParams(photoKey string) domain.EditItemParams {
	return domain.EditItemParams{
		Name:        "Синее худи",
		Category:    domain.CategoryTop,
		Colors:      domain.Colors{Main: domain.ColorBlue},
		Seasons:     []domain.Season{domain.SeasonAutumn},
		WarmthLevel: domain.WarmthLevelMedium,
		PhotoKey:    photoKey,
	}
}

type memoryItems struct {
	items  []domain.Item
	lastID int64
}

func (repository *memoryItems) Create(_ context.Context, item domain.Item) (domain.Item, error) {
	repository.lastID++
	item.ID = repository.lastID
	repository.items = append(repository.items, item)
	return item, nil
}

func (repository *memoryItems) CountByUser(_ context.Context, userID int64) (int, error) {
	count := 0
	for _, item := range repository.items {
		if item.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (repository *memoryItems) ListByUser(_ context.Context, userID int64) ([]domain.Item, error) {
	var items []domain.Item
	for _, item := range repository.items {
		if item.UserID == userID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (repository *memoryItems) Get(_ context.Context, userID, itemID int64) (domain.Item, error) {
	index := repository.find(userID, itemID)
	if index == -1 {
		return domain.Item{}, ErrItemNotFound
	}
	return repository.items[index], nil
}

func (repository *memoryItems) Update(_ context.Context, item domain.Item) error {
	index := repository.find(item.UserID, item.ID)
	if index == -1 {
		return ErrItemNotFound
	}
	repository.items[index] = item
	return nil
}

func (repository *memoryItems) UpdateStatus(_ context.Context, userID, itemID int64, status domain.ItemStatus) error {
	index := repository.find(userID, itemID)
	if index == -1 {
		return ErrItemNotFound
	}
	repository.items[index].Status = status
	return nil
}

func (repository *memoryItems) Delete(_ context.Context, userID int64, itemIDs []int64) ([]string, error) {
	var photoKeys []string
	repository.items = slices.DeleteFunc(repository.items, func(item domain.Item) bool {
		if item.UserID != userID || !slices.Contains(itemIDs, item.ID) {
			return false
		}
		photoKeys = append(photoKeys, item.PhotoKey)
		return true
	})
	return photoKeys, nil
}

func (repository *memoryItems) find(userID, itemID int64) int {
	return slices.IndexFunc(repository.items, func(item domain.Item) bool {
		return item.UserID == userID && item.ID == itemID
	})
}
