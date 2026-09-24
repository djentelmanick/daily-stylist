package miniapp

import (
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testBotToken = "123456:TEST-TOKEN"

const (
	compatibilityInitData = "query_id=AAHdF6IQAAAAAN0XohDhrOrc&user=%7B%22id%22%3A279058397%2C%22first_name%22%3A%22Vladislav%22%2C%22last_name%22%3A%22Kibenko%22%2C%22username%22%3A%22vdkfrost%22%2C%22language_code%22%3A%22ru%22%2C%22is_premium%22%3Atrue%7D&auth_date=1662771648&hash=c501b71e775f74ce10e377dea85a7ea24ecd640b223ea86dfe453e0eaed2e2b2"
	compatibilityToken    = "5768337691:AAH5YkoiEuPk8-FZa32hStHTqXiLPtAEhx8"
	compatibilityUserID   = 279058397
	compatibilityAuthDate = 1662771648
)

func TestValidateInitData_AcceptsIndependentSignature(t *testing.T) {
	now := time.Unix(compatibilityAuthDate, 0).Add(time.Minute)

	userID, err := validateInitData(compatibilityInitData, compatibilityToken, initDataMaxAge, now)
	if err != nil {
		t.Fatalf("validateInitData: %v", err)
	}
	if userID != compatibilityUserID {
		t.Errorf("userID = %d, ожидался %d", userID, compatibilityUserID)
	}
}

func TestValidateInitData_AcceptsOwnSignature(t *testing.T) {
	now := time.Now()

	userID, err := validateInitData(SignInitData(testBotToken, 42, now), testBotToken, initDataMaxAge, now)
	if err != nil {
		t.Fatalf("validateInitData: %v", err)
	}
	if userID != 42 {
		t.Errorf("userID = %d, ожидался 42", userID)
	}
}

func TestValidateInitData_Rejects(t *testing.T) {
	now := time.Now()

	withoutUser := url.Values{}
	withoutUser.Set("auth_date", strconv.FormatInt(now.Unix(), 10))

	tests := []struct {
		name     string
		initData string
	}{
		{"подпись другим токеном", SignInitData("999:OTHER-TOKEN", 42, now)},
		{"изменённые данные", withUser(SignInitData(testBotToken, 42, now), `{"id":43}`)},
		{"устаревшие данные", SignInitData(testBotToken, 42, now.Add(-initDataMaxAge-time.Minute))},
		{"без подписи", "auth_date=1&user=%7B%22id%22%3A42%7D"},
		{"не query-строка", "here comes something wrong;"},
		{"без пользователя", withHash(withoutUser, testBotToken)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateInitData(test.initData, testBotToken, initDataMaxAge, now)
			if !errors.Is(err, errInvalidInitData) {
				t.Errorf("ошибка = %v, ожидалась errInvalidInitData", err)
			}
		})
	}
}

func withHash(values url.Values, botToken string) string {
	pairs := make([]string, 0, len(values))
	for key := range values {
		pairs = append(pairs, key+"="+values.Get(key))
	}
	slices.Sort(pairs)

	values.Set("hash", sign(strings.Join(pairs, "\n"), botToken))
	return values.Encode()
}

func withUser(initData, user string) string {
	values, err := url.ParseQuery(initData)
	if err != nil {
		panic(err)
	}
	values.Set("user", user)
	return values.Encode()
}
