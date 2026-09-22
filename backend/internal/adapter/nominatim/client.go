package nominatim

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/service"
)

const (
	reverseURL       = "https://nominatim.openstreetmap.org/reverse"
	requestTimeout   = 5 * time.Second
	maxResponseBytes = 1 << 16
	// Правила Nominatim требуют представиться: без User-Agent запросы блокируют.
	userAgent = "daily-stylist (+https://github.com/djentelmanick/daily-stylist)"
	// Уровень города: ближе был бы район или улица.
	cityZoom = "10"
)

var _ service.PlaceNames = (*Client)(nil)

type Client struct {
	http       *http.Client
	reverseURL string
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: requestTimeout}, reverseURL: reverseURL}
}

type reverseResponse struct {
	Error   string `json:"error"`
	Name    string `json:"name"`
	Address struct {
		City         string `json:"city"`
		Town         string `json:"town"`
		Village      string `json:"village"`
		Municipality string `json:"municipality"`
		State        string `json:"state"`
		Country      string `json:"country"`
	} `json:"address"`
}

func (client *Client) PlaceAt(ctx context.Context, latitude, longitude float64) (string, string, error) {
	query := url.Values{
		"lat":             {strconv.FormatFloat(latitude, 'f', -1, 64)},
		"lon":             {strconv.FormatFloat(longitude, 'f', -1, 64)},
		"format":          {"jsonv2"},
		"zoom":            {cityZoom},
		"accept-language": {"ru"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.reverseURL+"?"+query.Encode(), nil)
	if err != nil {
		return "", "", err
	}
	request.Header.Set("User-Agent", userAgent)

	response, err := client.http.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("место по координатам: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("место по координатам: ответ %s", response.Status)
	}

	var body reverseResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&body); err != nil {
		return "", "", fmt.Errorf("место по координатам: %w", err)
	}

	address := body.Address
	name := firstNonEmpty(address.City, address.Town, address.Village, address.Municipality, body.Name)
	if body.Error != "" || name == "" {
		return "", "", service.ErrPlaceNotFound
	}
	return name, region(name, address.State, address.Country), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func region(name string, parts ...string) string {
	var kept []string
	for _, part := range parts {
		if part != "" && part != name {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, ", ")
}
