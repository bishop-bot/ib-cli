package config

import (
	"os"
	"testing"
	"time"
)

func TestGatewayConfig_BaseURL(t *testing.T) {
	tests := []struct {
		name   string
		gw     GatewayConfig
		expect string
	}{
		{
			name:   "default http",
			gw:     GatewayConfig{Host: "127.0.0.1", Port: 5000},
			expect: "http://127.0.0.1:5000",
		},
		{
			name:   "https",
			gw:     GatewayConfig{Host: "localhost", Port: 5001, UseTLS: true},
			expect: "https://localhost:5001",
		},
		{
			name:   "custom host",
			gw:     GatewayConfig{Host: "192.168.1.100", Port: 8080},
			expect: "http://192.168.1.100:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.gw.BaseURL()
			if got != tt.expect {
				t.Errorf("BaseURL() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestRequestConfig_Timeout(t *testing.T) {
	tests := []struct {
		name   string
		req    RequestConfig
		expect time.Duration
	}{
		{
			name:   "default 30 seconds",
			req:    RequestConfig{},
			expect: 30 * time.Second,
		},
		{
			name:   "custom timeout",
			req:    RequestConfig{TimeoutSecs: 60},
			expect: 60 * time.Second,
		},
		{
			name:   "zero defaults to 30",
			req:    RequestConfig{TimeoutSecs: 0},
			expect: 30 * time.Second,
		},
		{
			name:   "negative defaults to 30",
			req:    RequestConfig{TimeoutSecs: -1},
			expect: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.Timeout()
			if got != tt.expect {
				t.Errorf("Timeout() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	cfg := &Config{}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	// Check defaults are set
	if cfg.Gateway.Host != "127.0.0.1" {
		t.Errorf("Gateway.Host = %v, want 127.0.0.1", cfg.Gateway.Host)
	}
	if cfg.Gateway.Port != 5000 {
		t.Errorf("Gateway.Port = %v, want 5000", cfg.Gateway.Port)
	}
	if cfg.Request.BarType != "TRADES" {
		t.Errorf("Request.BarType = %v, want TRADES", cfg.Request.BarType)
	}
	if cfg.Request.TimeoutSecs != 30 {
		t.Errorf("Request.TimeoutSecs = %v, want 30", cfg.Request.TimeoutSecs)
	}
}

func TestConfig_ApplyEnvOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("IB_GATEWAY_HOST", "10.0.0.1")
	os.Setenv("IB_GATEWAY_PORT", "9000")
	os.Setenv("IB_AUTH_USERNAME", "testuser")
	defer func() {
		os.Unsetenv("IB_GATEWAY_HOST")
		os.Unsetenv("IB_GATEWAY_PORT")
		os.Unsetenv("IB_AUTH_USERNAME")
	}()

	cfg := &Config{
		Gateway: GatewayConfig{
			Host: "127.0.0.1",
			Port: 5000,
		},
		Auth: AuthConfig{
			Username: "",
		},
	}

	cfg.applyEnvOverrides()

	if cfg.Gateway.Host != "10.0.0.1" {
		t.Errorf("Gateway.Host = %v, want 10.0.0.1", cfg.Gateway.Host)
	}
	if cfg.Gateway.Port != 9000 {
		t.Errorf("Gateway.Port = %v, want 9000", cfg.Gateway.Port)
	}
	if cfg.Auth.Username != "testuser" {
		t.Errorf("Auth.Username = %v, want testuser", cfg.Auth.Username)
	}
}

func TestConfig_Load_FileNotFound(t *testing.T) {
	// When no config file exists, Load should succeed with defaults
	cfg, err := Load("") // Empty path triggers default search behavior
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil")
	}

	// Check defaults
	if cfg.Gateway.Host != "127.0.0.1" {
		t.Errorf("Gateway.Host = %v, want 127.0.0.1", cfg.Gateway.Host)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		size   string
		unit   string
		expect string
	}{
		{"1", "min", "1 min"},
		{"5", "D", "5 D"},
		{"15", "S", "15 S"},
		{"1", "W", "1 W"},
		{"3", "M", "3 M"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			got := ParseDuration(tt.size, tt.unit)
			if got != tt.expect {
				t.Errorf("ParseDuration(%s, %s) = %v, want %v", tt.size, tt.unit, got, tt.expect)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Gateway.Host != "127.0.0.1" {
		t.Errorf("Gateway.Host = %v, want 127.0.0.1", cfg.Gateway.Host)
	}
	if cfg.Gateway.Port != 5000 {
		t.Errorf("Gateway.Port = %v, want 5000", cfg.Gateway.Port)
	}
	if cfg.Output.DefaultFormat != "json" {
		t.Errorf("Output.DefaultFormat = %v, want json", cfg.Output.DefaultFormat)
	}
	if cfg.Request.MaxRetries != 3 {
		t.Errorf("Request.MaxRetries = %v, want 3", cfg.Request.MaxRetries)
	}
}