package gigachat

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

const jeansAnswer = `{"name":"Джинсы","category":"bottom","main_color":"blue","extra_colors":["black"],` +
	`"seasons":["spring","summer","autumn"],"warmth_level":2,"waterproof":false}`

type fakeGigaChat struct {
	answer        string
	chatStatus    int
	tokenFailures int
	tokenStatus   int
	uploadDrops   int

	tokens   int
	uploads  int
	uploaded []byte
	fileType string
	purpose  string
	asked    chatRequest
	deleted  []string
}

func (fake *fakeGigaChat) start(t *testing.T) *Client {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth", func(writer http.ResponseWriter, request *http.Request) {
		fake.tokens++
		if fake.tokens <= fake.tokenFailures {
			status := fake.tokenStatus
			if status == 0 {
				status = http.StatusInternalServerError
			}
			http.Error(writer, "не сейчас", status)
			return
		}
		if request.Header.Get("Authorization") != "Basic ключ" || request.Header.Get("RqUID") == "" {
			http.Error(writer, "нет авторизации", http.StatusUnauthorized)
			return
		}
		expires := time.Now().Add(30 * time.Minute).UnixMilli()
		_, _ = fmt.Fprintf(writer, `{"access_token":"токен-%d","expires_at":%d}`, fake.tokens, expires)
	})
	mux.HandleFunc("POST /v1/files", func(writer http.ResponseWriter, request *http.Request) {
		if !fake.authorized(writer, request) {
			return
		}
		fake.uploads++
		if fake.uploads <= fake.uploadDrops {
			drop(writer)
			return
		}
		file, header, err := request.FormFile("file")
		if err != nil {
			http.Error(writer, "нет файла", http.StatusBadRequest)
			return
		}
		fake.uploaded, _ = io.ReadAll(file)
		fake.fileType = header.Header.Get("Content-Type")
		fake.purpose = request.FormValue("purpose")
		_, _ = fmt.Fprint(writer, `{"id":"файл-1"}`)
	})
	mux.HandleFunc("POST /v1/chat/completions", func(writer http.ResponseWriter, request *http.Request) {
		if !fake.authorized(writer, request) {
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&fake.asked); err != nil {
			http.Error(writer, "не разобрать", http.StatusBadRequest)
			return
		}
		if fake.chatStatus != 0 {
			http.Error(writer, "кончились токены", fake.chatStatus)
			return
		}
		answer, _ := json.Marshal(chatResponse{Choices: []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Role: "assistant", Content: fake.answer}}}})
		_, _ = writer.Write(answer)
	})
	mux.HandleFunc("POST /v1/files/{id}/delete", func(writer http.ResponseWriter, request *http.Request) {
		if !fake.authorized(writer, request) {
			return
		}
		fake.deleted = append(fake.deleted, request.PathValue("id"))
		_, _ = fmt.Fprint(writer, `{"deleted":true}`)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client, err := New(Config{
		Credentials: "ключ",
		AuthURL:     server.URL + "/oauth",
		BaseURL:     server.URL + "/v1",
		Scope:       "GIGACHAT_API_PERS",
		Model:       "GigaChat-3-Ultra",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("клиент: %v", err)
	}
	return client
}

func (fake *fakeGigaChat) authorized(writer http.ResponseWriter, request *http.Request) bool {
	if !strings.HasPrefix(request.Header.Get("Authorization"), "Bearer токен-") {
		http.Error(writer, "нет токена", http.StatusUnauthorized)
		return false
	}
	return true
}

func photo() service.PhotoContent {
	return service.PhotoContent{Bytes: []byte("байты снимка"), ContentType: "image/jpeg"}
}

func TestRecognize_SendsPhotoAndReadsAnswer(t *testing.T) {
	fake := &fakeGigaChat{answer: jeansAnswer}
	client := fake.start(t)

	suggestion, err := client.Recognize(t.Context(), photo())
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	if string(fake.uploaded) != "байты снимка" || fake.fileType != "image/jpeg" {
		t.Errorf("загрузили %q типа %q", fake.uploaded, fake.fileType)
	}
	if fake.purpose != "general" {
		t.Errorf("purpose = %q, ожидался general", fake.purpose)
	}
	if len(fake.asked.Messages) != 1 || len(fake.asked.Messages[0].Attachments) != 1 {
		t.Fatalf("модели ушло %+v, ожидалось одно сообщение с фотографией", fake.asked)
	}
	if fake.asked.Messages[0].Attachments[0] != "файл-1" {
		t.Errorf("к сообщению приложили %q", fake.asked.Messages[0].Attachments[0])
	}
	if fake.asked.Model != "GigaChat-3-Ultra" {
		t.Errorf("модель = %q", fake.asked.Model)
	}

	if suggestion.Name != "Джинсы" || suggestion.Category != domain.CategoryBottom {
		t.Errorf("подсказка = %+v", suggestion)
	}
	if suggestion.Colors.Main != domain.ColorBlue || len(suggestion.Colors.Extra) != 1 {
		t.Errorf("цвета = %+v", suggestion.Colors)
	}
	if len(suggestion.Seasons) != 3 || suggestion.WarmthLevel != domain.WarmthLevelMedium {
		t.Errorf("сезоны = %v, теплота = %d", suggestion.Seasons, suggestion.WarmthLevel)
	}
}

func TestRecognize_ForgetsPhotoAfterAnswer(t *testing.T) {
	fake := &fakeGigaChat{answer: jeansAnswer}
	client := fake.start(t)

	if _, err := client.Recognize(t.Context(), photo()); err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	if len(fake.deleted) != 1 || fake.deleted[0] != "файл-1" {
		t.Errorf("удалено %v, ожидался файл-1: снимки пользователей не должны копиться в облаке", fake.deleted)
	}
}

func TestRecognize_ForgetsPhotoEvenWhenModelFailed(t *testing.T) {
	fake := &fakeGigaChat{answer: jeansAnswer, chatStatus: http.StatusPaymentRequired}
	client := fake.start(t)

	if _, err := client.Recognize(t.Context(), photo()); !errors.Is(err, service.ErrRecognitionUnavailable) {
		t.Errorf("ошибка = %v, ожидалась ErrRecognitionUnavailable", err)
	}
	if len(fake.deleted) != 1 {
		t.Errorf("после отказа модели удалено %v, фотография осталась в облаке", fake.deleted)
	}
}

func TestRecognize_AsksForTokenOnce(t *testing.T) {
	fake := &fakeGigaChat{answer: jeansAnswer}
	client := fake.start(t)

	for range 3 {
		if _, err := client.Recognize(t.Context(), photo()); err != nil {
			t.Fatalf("Recognize: %v", err)
		}
	}

	if fake.tokens != 1 {
		t.Errorf("токен запрашивали %d раза, ожидался один: он живёт полчаса", fake.tokens)
	}
}

func TestRecognize_RenewsTokenBeforeItExpires(t *testing.T) {
	fake := &fakeGigaChat{answer: jeansAnswer}
	client := fake.start(t)

	if _, err := client.Recognize(t.Context(), photo()); err != nil {
		t.Fatalf("Recognize: %v", err)
	}
	client.now = func() time.Time { return time.Now().Add(30 * time.Minute) }
	if _, err := client.Recognize(t.Context(), photo()); err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	if fake.tokens != 2 {
		t.Errorf("токен запрашивали %d раз, ожидалось два", fake.tokens)
	}
}

func TestRecognize_UnderstandsAnswerWrappedInMarkdown(t *testing.T) {
	fake := &fakeGigaChat{answer: "```json\n" + jeansAnswer + "\n```"}
	client := fake.start(t)

	suggestion, err := client.Recognize(t.Context(), photo())
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	if suggestion.Category != domain.CategoryBottom {
		t.Errorf("подсказка = %+v", suggestion)
	}
}

func TestRecognize_UnreadableAnswerLeavesFormEmpty(t *testing.T) {
	tests := map[string]string{
		"не одежда":         "{}",
		"не json":           "извините, я не понял",
		"ответ не объектом": "[1, 2]",
	}
	for name, answer := range tests {
		t.Run(name, func(t *testing.T) {
			fake := &fakeGigaChat{answer: answer}
			client := fake.start(t)

			suggestion, err := client.Recognize(t.Context(), photo())
			if err != nil {
				t.Fatalf("Recognize: %v", err)
			}
			if suggestion.Category != "" || suggestion.Name != "" {
				t.Errorf("подсказка = %+v, ожидалась пустая", suggestion)
			}
		})
	}
}

func TestRecognize_PassesUnknownValuesAsIs(t *testing.T) {
	fake := &fakeGigaChat{answer: `{"category":"пиджачок","main_color":"бирюзовый"}`}
	client := fake.start(t)

	suggestion, err := client.Recognize(t.Context(), photo())
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	// Отбраковкой занимается service.
	if suggestion.Category != "пиджачок" || suggestion.Colors.Main != "бирюзовый" {
		t.Errorf("подсказка = %+v, ожидались значения без изменений", suggestion)
	}
}

func TestRecognize_ServiceGoneMeansNoSuggestion(t *testing.T) {
	client, err := New(Config{
		Credentials: "ключ",
		AuthURL:     "http://127.0.0.1:1/oauth",
		BaseURL:     "http://127.0.0.1:1/v1",
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("клиент: %v", err)
	}

	if _, err := client.Recognize(t.Context(), photo()); !errors.Is(err, service.ErrRecognitionUnavailable) {
		t.Errorf("ошибка = %v, ожидалась ErrRecognitionUnavailable", err)
	}
}

func drop(writer http.ResponseWriter) {
	connection, _, err := writer.(http.Hijacker).Hijack()
	if err == nil {
		_ = connection.Close()
	}
}

func TestRecognize_RetriesAfterServerFailure(t *testing.T) {
	fake := &fakeGigaChat{answer: jeansAnswer, tokenFailures: 2}
	client := fake.start(t)

	suggestion, err := client.Recognize(t.Context(), photo())
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	if fake.tokens != 3 {
		t.Errorf("запросов токена %d, ожидалось 3: две неудачи и удачная попытка", fake.tokens)
	}
	if suggestion.Category != domain.CategoryBottom {
		t.Errorf("подсказка = %+v", suggestion)
	}
}

func TestRecognize_RetriesWithWholePhotoAfterBrokenConnection(t *testing.T) {
	fake := &fakeGigaChat{answer: jeansAnswer, uploadDrops: 1}
	client := fake.start(t)

	if _, err := client.Recognize(t.Context(), photo()); err != nil {
		t.Fatalf("Recognize: %v", err)
	}

	if fake.uploads != 2 {
		t.Errorf("загрузок %d, ожидалось 2", fake.uploads)
	}
	if string(fake.uploaded) != "байты снимка" {
		t.Errorf("после повтора загрузилось %q, ожидался снимок целиком", fake.uploaded)
	}
}

func TestRecognize_DoesNotRetryWhenKeyIsWrong(t *testing.T) {
	fake := &fakeGigaChat{answer: jeansAnswer, tokenFailures: maxAttempts, tokenStatus: http.StatusUnauthorized}
	client := fake.start(t)

	if _, err := client.Recognize(t.Context(), photo()); !errors.Is(err, service.ErrRecognitionUnavailable) {
		t.Errorf("ошибка = %v, ожидалась ErrRecognitionUnavailable", err)
	}

	if fake.tokens != 1 {
		t.Errorf("запросов токена %d, ожидался один", fake.tokens)
	}
}

func TestRecognize_GivesUpAfterThreeAttempts(t *testing.T) {
	fake := &fakeGigaChat{answer: jeansAnswer, tokenFailures: maxAttempts}
	client := fake.start(t)

	if _, err := client.Recognize(t.Context(), photo()); !errors.Is(err, service.ErrRecognitionUnavailable) {
		t.Errorf("ошибка = %v, ожидалась ErrRecognitionUnavailable", err)
	}

	if fake.tokens != maxAttempts {
		t.Errorf("запросов токена %d, ожидалось %d", fake.tokens, maxAttempts)
	}
}
