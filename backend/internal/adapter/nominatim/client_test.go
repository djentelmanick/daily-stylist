package nominatim

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

func TestPlaceAt(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		wantName   string
		wantRegion string
		wantErr    error
	}{
		{
			"город",
			`{"name":"Казань","address":{"city":"Казань","state":"Татарстан","country":"Россия"}}`,
			"Казань", "Татарстан, Россия", nil,
		},
		{
			"регион совпадает с городом",
			`{"name":"Москва","address":{"city":"Москва","state":"Москва","country":"Россия"}}`,
			"Москва", "Россия", nil,
		},
		{
			"посёлок",
			`{"name":"Какое-то поселение","address":{"village":"Кукмор","state":"Татарстан","country":"Россия"}}`,
			"Кукмор", "Татарстан, Россия", nil,
		},
		{"море", `{"error":"Unable to geocode"}`, "", "", service.ErrPlaceNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var query, agent string
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				query, agent = request.URL.RawQuery, request.UserAgent()
				_, _ = writer.Write([]byte(test.response))
			}))
			defer server.Close()
			client := NewClient()
			client.reverseURL = server.URL

			name, region, err := client.PlaceAt(t.Context(), 55.79, 49.11)

			if !errors.Is(err, test.wantErr) || name != test.wantName || region != test.wantRegion {
				t.Errorf("PlaceAt = %q, %q, %v; ожидалось %q, %q, %v", name, region, err, test.wantName, test.wantRegion, test.wantErr)
			}
			if agent != userAgent {
				t.Errorf("User-Agent = %q", agent)
			}
			if want := "accept-language=ru&format=jsonv2&lat=55.79&lon=49.11&zoom=10"; query != want {
				t.Errorf("запрос = %s, ожидался %s", query, want)
			}
		})
	}
}

func TestPlaceAt_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	client := NewClient()
	client.reverseURL = server.URL

	if _, _, err := client.PlaceAt(t.Context(), 55.79, 49.11); err == nil || errors.Is(err, service.ErrPlaceNotFound) {
		t.Errorf("ошибка = %v, ожидался отказ сервиса, а не «места нет»", err)
	}
}
