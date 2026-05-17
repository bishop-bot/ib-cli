package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the CLI.
type Config struct {
	Gateway GatewayConfig `toml:"gateway"`
	Auth    AuthConfig    `toml:"auth"`
	Output  OutputConfig  `toml:"output"`
	Request RequestConfig `toml:"request"`
}

type GatewayConfig struct {
	Host   string `toml:"host"`
	Port   int    `toml:"port"`
	UseTLS bool   `toml:"use_tls"`
}

type AuthConfig struct {
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type OutputConfig struct {
	DefaultFormat string `toml:"default_format"`
	Pretty        bool   `toml:"pretty"`
}

type RequestConfig struct {
	BarType       string `toml:"bar_type"`
	BarSize       string `toml:"bar_size"`
	BarUnit       string `toml:"bar_unit"`
	TimeoutSecs   int    `toml:"timeout_seconds"`
	MaxRetries    int    `toml:"max_retries"`
}

// BaseURL returns the full gateway URL.
func (g *GatewayConfig) BaseURL() string {
	scheme := "http"
	if g.UseTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, g.Host, g.Port)
}

// Timeout returns the request timeout as a duration.
func (r *RequestConfig) Timeout() time.Duration {
	if r.TimeoutSecs <= 0 {
		return 30 * time.Second
	}
	return time.Duration(r.TimeoutSecs) * time.Second
}

// Default is a global config instance.
var Default *Config

// Load reads configuration from file and environment variables.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Search for config in standard locations
		v.SetConfigName("config")
		v.SetConfigType("toml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.ib-cli")
		v.AddConfigPath("/etc/ib-cli")
	}

	// Environment variable overrides
	v.SetEnvPrefix("IB")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	var cfg Config
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		// Config file not found, use defaults
		cfg = *DefaultConfig()
	} else {
		if err := v.Unmarshal(&cfg); err != nil {
			return nil, fmt.Errorf("unmarshaling config: %w", err)
		}
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	Default = &cfg
	return &cfg, nil
}

func (c *Config) applyEnvOverrides() {
	if host := os.Getenv("IB_GATEWAY_HOST"); host != "" {
		c.Gateway.Host = host
	}
	if port := os.Getenv("IB_GATEWAY_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Gateway.Port = p
		}
	}
	if username := os.Getenv("IB_AUTH_USERNAME"); username != "" {
		c.Auth.Username = username
	}
	if password := os.Getenv("IB_AUTH_PASSWORD"); password != "" {
		c.Auth.Password = password
	}
}

func (c *Config) Validate() error {
	if c.Gateway.Host == "" {
		c.Gateway.Host = "127.0.0.1"
	}
	if c.Gateway.Port == 0 {
		c.Gateway.Port = 5000
	}
	if c.Request.BarType == "" {
		c.Request.BarType = "TRADES"
	}
	if c.Request.BarSize == "" {
		c.Request.BarSize = "1"
	}
	if c.Request.BarUnit == "" {
		c.Request.BarUnit = "min"
	}
	if c.Request.TimeoutSecs == 0 {
		c.Request.TimeoutSecs = 30
	}
	if c.Request.MaxRetries == 0 {
		c.Request.MaxRetries = 3
	}
	if c.Output.DefaultFormat == "" {
		c.Output.DefaultFormat = "json"
	}
	return nil
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Gateway: GatewayConfig{
			Host:   "127.0.0.1",
			Port:   5000,
			UseTLS: false,
		},
		Output: OutputConfig{
			DefaultFormat: "json",
			Pretty:        true,
		},
		Request: RequestConfig{
			BarType:     "TRADES",
			BarSize:     "1",
			BarUnit:     "min",
			TimeoutSecs: 30,
			MaxRetries:  3,
		},
	}
}

// ParseDuration converts bar_size and bar_unit to IB duration string.
// e.g., "1", "min" -> "1 min", "5", "D" -> "5 D"
func ParseDuration(size, unit string) string {
	return fmt.Sprintf("%s %s", size, unit)
}