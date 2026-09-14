package telegram

import "testing"

func TestWebhookRoute(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		path        string
		wantURL     string
		wantPattern string
	}{
		{"путь по умолчанию", "https://example.ngrok.app", "/telegram/webhook", "https://example.ngrok.app/telegram/webhook", "POST /telegram/webhook"},
		{"слеш в конце адреса", "https://example.ngrok.app/", "/telegram/webhook", "https://example.ngrok.app/telegram/webhook", "POST /telegram/webhook"},
		{"корень", "https://example.ngrok.app", "/", "https://example.ngrok.app/", "POST /{$}"},
		{"путь без слеша", "https://example.ngrok.app", "telegram/webhook", "https://example.ngrok.app/telegram/webhook", "POST /telegram/webhook"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			webhookURL, pattern, err := webhookRoute(test.baseURL, test.path)
			if err != nil {
				t.Fatalf("webhookRoute: %v", err)
			}
			if webhookURL != test.wantURL {
				t.Errorf("адрес = %q, ожидался %q", webhookURL, test.wantURL)
			}
			if pattern != test.wantPattern {
				t.Errorf("шаблон = %q, ожидался %q", pattern, test.wantPattern)
			}
		})
	}
}
