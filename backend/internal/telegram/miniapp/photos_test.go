package miniapp

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func TestRequestPhotoUpload_ReturnsKeyAndLink(t *testing.T) {
	handler := newHandler(service.NewWardrobe(&memoryRepository{}, &stubPhotos{}))

	response := serve(handler, newRequest(http.MethodPost, "/api/photos",
		`{"content_type":"image/jpeg","size":2048}`, SignInitData(testBotToken, 42, time.Now())))

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusCreated, response.Body)
	}
	var upload photoUploadResponse
	if err := json.NewDecoder(response.Body).Decode(&upload); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if upload.Key == "" || upload.UploadURL == "" {
		t.Errorf("ответ = %+v, ожидались ключ и ссылка на загрузку", upload)
	}
}

func TestRequestPhotoUpload_TellsWhyFileWasRejected(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{"файл больше лимита", service.ErrPhotoTooLarge, "photo_too_large"},
		{"неподдерживаемый формат", service.ErrPhotoTypeUnsupported, "photo_type_unsupported"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			photos := &stubPhotos{uploadErr: test.err}
			handler := NewHandler(testBotToken, failingWardrobe{}, &stubRecommender{}, nil, nil, photos, &stubRecognition{})

			response := serve(handler, newRequest(http.MethodPost, "/api/photos",
				`{"content_type":"image/jpeg","size":2048}`, SignInitData(testBotToken, 42, time.Now())))

			checkError(t, response, http.StatusUnprocessableEntity, test.code)
		})
	}
}

func TestCreateItem_AttachesPhotoAndReturnsItsLink(t *testing.T) {
	photos := &stubPhotos{}
	handler := newHandlerWithPhotos(service.NewWardrobe(&memoryRepository{}, photos), photos)

	body := `{"name":"Синее худи","category":"top","main_color":"blue","seasons":["autumn"],"warmth_level":2,"photo_key":"users/42/photo.jpg"}`
	response := serve(handler, newRequest(http.MethodPost, "/api/items", body, SignInitData(testBotToken, 42, time.Now())))

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, ожидался %d; тело: %s", response.Code, http.StatusCreated, response.Body)
	}
	var item itemResponse
	if err := json.NewDecoder(response.Body).Decode(&item); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if item.PhotoKey != "users/42/photo.jpg" {
		t.Errorf("ключ фотографии = %q", item.PhotoKey)
	}
	if item.PhotoURL == "" {
		t.Errorf("в ответе нет ссылки на фотографию: %+v", item)
	}
	if len(photos.confirmed) != 1 || photos.confirmed[0] != "users/42/photo.jpg" {
		t.Errorf("подтверждены загрузки %v, ожидалась одна", photos.confirmed)
	}
}

func TestCreateItem_RefusesPhotoNobodyUploaded(t *testing.T) {
	photos := &stubPhotos{confirmErr: service.ErrPhotoNotUploaded}
	handler := newHandlerWithPhotos(service.NewWardrobe(&memoryRepository{}, photos), photos)

	body := `{"name":"Синее худи","category":"top","main_color":"blue","seasons":["autumn"],"warmth_level":2,"photo_key":"users/7/чужая.jpg"}`
	response := serve(handler, newRequest(http.MethodPost, "/api/items", body, SignInitData(testBotToken, 42, time.Now())))

	checkError(t, response, http.StatusUnprocessableEntity, "photo_not_uploaded")
}

func TestListItems_ItemWithoutPhotoHasNoLink(t *testing.T) {
	handler := newHandler(service.NewWardrobe(&memoryRepository{}, &stubPhotos{}))
	createAs(t, handler, 42)

	response := serve(handler, newRequest(http.MethodGet, "/api/items", "", SignInitData(testBotToken, 42, time.Now())))

	var body itemsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("разбор ответа: %v", err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("вещей в ответе %d, ожидалась одна", len(body.Items))
	}
	if body.Items[0].PhotoURL != "" {
		t.Errorf("ссылка на фотографию = %q, ожидалась пустая", body.Items[0].PhotoURL)
	}
}

func newHandlerWithPhotos(wardrobe wardrobe, photos photos) http.Handler {
	return NewHandler(testBotToken, wardrobe, &stubRecommender{err: errors.New("подбор не должен вызываться")}, nil, nil, photos, &stubRecognition{})
}
