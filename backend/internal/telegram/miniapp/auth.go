package miniapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const initDataMaxAge = 24 * time.Hour

var errInvalidInitData = errors.New("недействительные данные запуска Mini App")

type userIDKey struct{}

func requireUser(botToken string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		initData, ok := strings.CutPrefix(request.Header.Get("Authorization"), "tma ")
		if !ok {
			writeError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}

		userID, err := validateInitData(initData, botToken, initDataMaxAge, time.Now())
		if err != nil {
			writeError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(request.Context(), userIDKey{}, userID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func userIDFrom(ctx context.Context) int64 {
	userID, _ := ctx.Value(userIDKey{}).(int64)
	return userID
}

// validateInitData проверяет данные по алгоритму из документации Telegram:
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
func validateInitData(initData, botToken string, maxAge time.Duration, now time.Time) (int64, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return 0, fmt.Errorf("%w: не query-строка", errInvalidInitData)
	}

	hash := values.Get("hash")
	if hash == "" {
		return 0, fmt.Errorf("%w: нет подписи", errInvalidInitData)
	}

	// В строку для проверки входят все поля, кроме hash, — в том числе signature:
	// его исключают только при проверке для сторонних сервисов, без токена бота.
	pairs := make([]string, 0, len(values))
	for key := range values {
		if key != "hash" {
			pairs = append(pairs, key+"="+values.Get(key))
		}
	}
	slices.Sort(pairs)

	if !hmac.Equal([]byte(sign(strings.Join(pairs, "\n"), botToken)), []byte(hash)) {
		return 0, fmt.Errorf("%w: подпись не совпадает", errInvalidInitData)
	}

	authDate, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: неверный auth_date", errInvalidInitData)
	}
	if now.Sub(time.Unix(authDate, 0)) > maxAge {
		return 0, fmt.Errorf("%w: данные устарели", errInvalidInitData)
	}

	var user struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID == 0 {
		return 0, fmt.Errorf("%w: нет пользователя", errInvalidInitData)
	}
	return user.ID, nil
}

// Нужна для разработки: подпись для Swagger UI и curl, без открытия приложения.
func SignInitData(botToken string, userID int64, authDate time.Time) string {
	values := url.Values{}
	values.Set("auth_date", strconv.FormatInt(authDate.Unix(), 10))
	values.Set("user", fmt.Sprintf(`{"id":%d,"first_name":"Тест"}`, userID))

	pairs := make([]string, 0, len(values))
	for key := range values {
		pairs = append(pairs, key+"="+values.Get(key))
	}
	slices.Sort(pairs)

	values.Set("hash", sign(strings.Join(pairs, "\n"), botToken))
	return values.Encode()
}

func sign(dataCheckString, botToken string) string {
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))

	signature := hmac.New(sha256.New, secretKey.Sum(nil))
	signature.Write([]byte(dataCheckString))
	return hex.EncodeToString(signature.Sum(nil))
}
