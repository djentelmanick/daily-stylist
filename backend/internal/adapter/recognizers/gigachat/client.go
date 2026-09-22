package gigachat

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sethvargo/go-retry"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

var _ service.PhotoRecognizer = (*Client)(nil)

// Сертификаты GigaChat выпущены НУЦ Минцифры, которого нет в системном списке доверия.
//
//go:embed russian_trusted_ca.pem
var russianTrustedCA []byte

const (
	maxSuggestedColors = 3
	tokenEarlyRefresh  = time.Minute
	maxAnswerBytes     = 1 << 20

	maxAttempts = 3
	retryPause  = 300 * time.Millisecond
)

type Config struct {
	Credentials string
	AuthURL     string
	BaseURL     string
	Scope       string
	Model       string
	Timeout     time.Duration
}

type Client struct {
	cfg  Config
	http *http.Client

	mu      sync.Mutex
	token   string
	expires time.Time
	now     func() time.Time
}

func New(cfg Config) (*Client, error) {
	pool, err := certPool()
	if err != nil {
		return nil, err
	}

	return &Client{
		cfg: cfg,
		http: &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}},
		},
		now: time.Now,
	}, nil
}

func certPool() (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(russianTrustedCA) {
		return nil, fmt.Errorf("сертификаты НУЦ Минцифры не читаются")
	}
	return pool, nil
}

func (client *Client) Recognize(ctx context.Context, _ int64, photo service.PhotoContent) (service.ItemSuggestion, error) {
	ctx, cancel := context.WithTimeout(ctx, client.cfg.Timeout)
	defer cancel()

	suggestion, err := client.recognize(ctx, photo)
	if err != nil {
		return service.ItemSuggestion{}, fmt.Errorf("%w: %w", service.ErrRecognitionUnavailable, err)
	}
	return suggestion, nil
}

func (client *Client) recognize(ctx context.Context, photo service.PhotoContent) (service.ItemSuggestion, error) {
	token, err := client.accessToken(ctx)
	if err != nil {
		return service.ItemSuggestion{}, err
	}

	fileID, err := client.upload(ctx, token, photo)
	if err != nil {
		return service.ItemSuggestion{}, err
	}
	// Фотография нужна только на время запроса: в хранилище GigaChat она копилась бы вечно.
	defer client.forget(ctx, token, fileID)

	answer, err := client.ask(ctx, token, fileID)
	if err != nil {
		return service.ItemSuggestion{}, err
	}
	return suggestion(answer), nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

func (client *Client) accessToken(ctx context.Context) (string, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.token != "" && client.now().Add(tokenEarlyRefresh).Before(client.expires) {
		return client.token, nil
	}

	body := strings.NewReader(url.Values{"scope": {client.cfg.Scope}}.Encode())
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.cfg.AuthURL, body)
	if err != nil {
		return "", fmt.Errorf("запрос токена: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Basic "+client.cfg.Credentials)
	request.Header.Set("RqUID", requestID())

	var answer tokenResponse
	if err := client.send(request, &answer); err != nil {
		return "", fmt.Errorf("запрос токена: %w", err)
	}

	client.token = answer.AccessToken
	client.expires = time.UnixMilli(answer.ExpiresAt)
	return client.token, nil
}

type uploadedFile struct {
	ID string `json:"id"`
}

func (client *Client) upload(ctx context.Context, token string, photo service.PhotoContent) (string, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)

	// Заголовок части пишется руками: стандартный CreateFormFile всегда ставит
	// application/octet-stream, а GigaChat по типу понимает, что это картинка.
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="photo"`)
	header.Set("Content-Type", photo.ContentType)
	part, err := form.CreatePart(header)
	if err != nil {
		return "", fmt.Errorf("загрузка фотографии: %w", err)
	}
	if _, err := part.Write(photo.Bytes); err != nil {
		return "", fmt.Errorf("загрузка фотографии: %w", err)
	}
	if err := form.WriteField("purpose", "general"); err != nil {
		return "", fmt.Errorf("загрузка фотографии: %w", err)
	}
	if err := form.Close(); err != nil {
		return "", fmt.Errorf("загрузка фотографии: %w", err)
	}

	request, err := client.authorized(ctx, token, "/files", &body)
	if err != nil {
		return "", fmt.Errorf("загрузка фотографии: %w", err)
	}
	request.Header.Set("Content-Type", form.FormDataContentType())

	var answer uploadedFile
	if err := client.send(request, &answer); err != nil {
		return "", fmt.Errorf("загрузка фотографии: %w", err)
	}
	if answer.ID == "" {
		return "", fmt.Errorf("загрузка фотографии: хранилище не вернуло идентификатор")
	}
	return answer.ID, nil
}

type chatRequest struct {
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature"`
	Messages    []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role        string   `json:"role"`
	Content     string   `json:"content"`
	Attachments []string `json:"attachments,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (client *Client) ask(ctx context.Context, token, fileID string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:       client.cfg.Model,
		Temperature: 0.1,
		Messages: []chatMessage{{
			Role:        "user",
			Content:     prompt,
			Attachments: []string{fileID},
		}},
	})
	if err != nil {
		return "", fmt.Errorf("запрос к модели: %w", err)
	}

	request, err := client.authorized(ctx, token, "/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("запрос к модели: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	var answer chatResponse
	if err := client.send(request, &answer); err != nil {
		return "", fmt.Errorf("запрос к модели: %w", err)
	}
	if len(answer.Choices) == 0 {
		return "", fmt.Errorf("запрос к модели: пустой ответ")
	}

	log.Printf("gigachat: токенов %d + %d", answer.Usage.PromptTokens, answer.Usage.CompletionTokens)
	return answer.Choices[0].Message.Content, nil
}

func (client *Client) forget(ctx context.Context, token, fileID string) {
	// Свой контекст: удалять надо и тогда, когда запрос к модели отменили по таймауту.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), client.cfg.Timeout)
	defer cancel()

	request, err := client.authorized(ctx, token, "/files/"+fileID+"/delete", nil)
	if err == nil {
		err = client.send(request, nil)
	}
	if err != nil {
		log.Printf("gigachat: фотография %s осталась в хранилище: %v", fileID, err)
	}
}

func (client *Client) authorized(ctx context.Context, token, path string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.cfg.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	return request, nil
}

func (client *Client) send(request *http.Request, answer any) error {
	backoff := retry.WithMaxRetries(maxAttempts-1, retry.NewExponential(retryPause))
	attempt := 0
	return retry.Do(request.Context(), backoff, func(context.Context) error {
		attempt++
		if attempt > 1 {
			if err := rewind(request); err != nil {
				return err
			}
		}

		err := client.attempt(request, answer)
		if err == nil || !worthRetry(err) {
			return err
		}
		log.Printf("gigachat: %s, попытка %d из %d: %v", request.URL.Path, attempt, maxAttempts, err)
		return retry.RetryableError(err)
	})
}

func worthRetry(err error) bool {
	var refused statusError
	if errors.As(err, &refused) {
		return refused.status >= http.StatusInternalServerError
	}
	return true
}

func rewind(request *http.Request) error {
	if request.GetBody == nil {
		return nil
	}
	body, err := request.GetBody()
	if err != nil {
		return err
	}
	request.Body = body
	return nil
}

type statusError struct {
	status int
	path   string
	body   string
}

func (err statusError) Error() string {
	return fmt.Sprintf("%s ответил %d: %s", err.path, err.status, err.body)
}

func (client *Client) attempt(request *http.Request, answer any) error {
	response, err := client.http.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxAnswerBytes))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return statusError{status: response.StatusCode, path: request.URL.Path, body: firstLine(body)}
	}
	if answer == nil {
		return nil
	}
	if err := json.Unmarshal(body, answer); err != nil {
		return fmt.Errorf("ответ %s не разобрать: %w", request.URL.Path, err)
	}
	return nil
}

type recognized struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	MainColor   string   `json:"main_color"`
	ExtraColors []string `json:"extra_colors"`
	Seasons     []string `json:"seasons"`
	WarmthLevel int      `json:"warmth_level"`
	Waterproof  bool     `json:"waterproof"`
}

func suggestion(answer string) service.ItemSuggestion {
	var fields recognized
	if err := json.Unmarshal([]byte(withoutFence(answer)), &fields); err != nil {
		log.Printf("gigachat: ответ модели не разобрать: %.200s", answer)
		return service.ItemSuggestion{}
	}

	return service.ItemSuggestion{
		Name:     fields.Name,
		Category: domain.Category(fields.Category),
		Colors: domain.Colors{
			Main:  domain.Color(fields.MainColor),
			Extra: items[domain.Color](fields.ExtraColors),
		},
		Seasons:     items[domain.Season](fields.Seasons),
		WarmthLevel: domain.WarmthLevel(fields.WarmthLevel),
		Waterproof:  fields.Waterproof,
	}
}

// Модель просили не оборачивать ответ, но иногда она всё равно это делает.
func withoutFence(answer string) string {
	answer = strings.TrimSpace(answer)
	answer = strings.TrimPrefix(answer, "```json")
	answer = strings.TrimPrefix(answer, "```")
	answer = strings.TrimSuffix(answer, "```")
	return strings.TrimSpace(answer)
}

func items[T ~string](raw []string) []T {
	if len(raw) == 0 {
		return nil
	}
	result := make([]T, len(raw))
	for index, value := range raw {
		result[index] = T(value)
	}
	return result
}

func firstLine(body []byte) string {
	line, _, _ := strings.Cut(string(body), "\n")
	if len(line) > 200 {
		return line[:200]
	}
	return line
}

func requestID() string {
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(id[0:4]), hex.EncodeToString(id[4:6]), hex.EncodeToString(id[6:8]),
		hex.EncodeToString(id[8:10]), hex.EncodeToString(id[10:16]))
}
