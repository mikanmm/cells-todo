package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type discordTestTransport func(*http.Request) (*http.Response, error)

func (f discordTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDiscordWebhookURL(t *testing.T) {
	for _, raw := range []string{"https://discord.com/api/webhooks/123/abc_DEF-123", "https://discordapp.com/api/webhooks/123/token"} {
		if err := validateDiscordURL(raw); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{"http://discord.com/api/webhooks/123/token", "https://example.com/api/webhooks/123/token", "https://discord.com.evil.com/api/webhooks/123/token", "https://discord.com:443/api/webhooks/123/token", "https://user@discord.com/api/webhooks/123/token", "https://discord.com/api/webhooks/123/token?redirect=bad", "https://discord.com/api/webhooks/123/token/extra", "https://127.0.0.1/api/webhooks/123/token"} {
		if validateDiscordURL(raw) == nil {
			t.Fatalf("unsafe webhook accepted: %s", raw)
		}
	}
}
func TestDiscordDelivery(t *testing.T) {
	original := discordClient
	defer func() { discordClient = original }()
	discordClient = &http.Client{Transport: discordTestTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("wait") != "true" || r.Method != "POST" {
			t.Fatal("missing Discord confirmation request")
		}
		var payload struct {
			Content  string `json:"content"`
			Mentions struct {
				Parse []string `json:"parse"`
			} `json:"allowed_mentions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Content != "test" || payload.Mentions.Parse == nil || len(payload.Mentions.Parse) != 0 {
			t.Fatal("unexpected payload or mention policy")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"123"}`)), Header: http.Header{}}, nil
	})}
	if err := sendDiscord(context.Background(), "https://discord.com/api/webhooks/123/token", "test"); err != nil {
		t.Fatal(err)
	}
	discordClient.Transport = discordTestTransport(func(r *http.Request) (*http.Response, error) { return nil, errors.New("secret token here") })
	err := sendDiscord(context.Background(), "https://discord.com/api/webhooks/123/token", "test")
	if err == nil || strings.Contains(err.Error(), "token") {
		t.Fatal("transport secret leaked")
	}
}
