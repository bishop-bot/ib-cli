package api

import (
	"testing"
	"time"

	"github.com/bishop-bot/ib-cli/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			Host:   "localhost",
			Port:   5000,
			UseTLS: false,
		},
		Request: config.RequestConfig{
			TimeoutSecs: 45,
			MaxRetries:  5,
		},
	}

	client := NewClient(cfg)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.baseURL != "http://localhost:5000" {
		t.Errorf("baseURL = %v, want http://localhost:5000", client.baseURL)
	}

	if client.timeout != 45*time.Second {
		t.Errorf("timeout = %v, want 45s", client.timeout)
	}

	if client.maxRetries != 5 {
		t.Errorf("maxRetries = %v, want 5", client.maxRetries)
	}
}

func TestClient_SetSession(t *testing.T) {
	cfg := config.DefaultConfig()
	client := NewClient(cfg)

	client.SetSession("session123", "token456")

	if client.SessionID() != "session123" {
		t.Errorf("SessionID() = %v, want session123", client.SessionID())
	}

	if client.Token() != "token456" {
		t.Errorf("Token() = %v, want token456", client.Token())
	}
}

func TestClient_SessionID_Empty(t *testing.T) {
	cfg := config.DefaultConfig()
	client := NewClient(cfg)

	if client.SessionID() != "" {
		t.Errorf("SessionID() = %v, want empty string", client.SessionID())
	}

	if client.Token() != "" {
		t.Errorf("Token() = %v, want empty string", client.Token())
	}
}