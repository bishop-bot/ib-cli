package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pelletier/go-toml/v2"
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
	cfg := DefaultConfig()

	// Determine config file path
	var configFile string
	if configPath != "" {
		configFile = configPath
	} else {
		// Search for config in standard locations
		paths := []string{
			"config.toml",
		}
		if home, err := os.UserHomeDir(); err == nil {
			paths = append(paths, home+"/.ib-cli/config.toml")
		}
		paths = append(paths, "/etc/ib-cli/config.toml")

		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				configFile = p
				break
			}
		}
	}

	// Read config file if found
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}

		// Parse TOML directly
		var fileCfg Config
		if err := toml.Unmarshal(data, &fileCfg); err != nil {
			return nil, fmt.Errorf("parsing config: %w", err)
		}

		// Merge file config into defaults
		cfg.mergeFrom(&fileCfg)
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	Default = cfg
	return cfg, nil
}

// mergeFrom merges non-zero values from another config.
func (c *Config) mergeFrom(other *Config) {
	if other.Gateway.Host != "" {
		c.Gateway.Host = other.Gateway.Host
	}
	if other.Gateway.Port != 0 {
		c.Gateway.Port = other.Gateway.Port
	}
	// Bool fields - only override if explicitly set to true
	if other.Gateway.UseTLS {
		c.Gateway.UseTLS = other.Gateway.UseTLS
	}

	if other.Auth.Username != "" {
		c.Auth.Username = other.Auth.Username
	}
	if other.Auth.Password != "" {
		c.Auth.Password = other.Auth.Password
	}

	if other.Output.DefaultFormat != "" {
		c.Output.DefaultFormat = other.Output.DefaultFormat
	}
	// Handle bool explicitly
	if other.Output.Pretty {
		c.Output.Pretty = other.Output.Pretty
	}

	if other.Request.BarType != "" {
		c.Request.BarType = other.Request.BarType
	}
	if other.Request.BarSize != "" {
		c.Request.BarSize = other.Request.BarSize
	}
	if other.Request.BarUnit != "" {
		c.Request.BarUnit = other.Request.BarUnit
	}
	if other.Request.TimeoutSecs != 0 {
		c.Request.TimeoutSecs = other.Request.TimeoutSecs
	}
	if other.Request.MaxRetries != 0 {
		c.Request.MaxRetries = other.Request.MaxRetries
	}
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
	// Handle IB_GATEWAY_USE_TLS
	if useTLS := os.Getenv("IB_GATEWAY_USE_TLS"); useTLS != "" {
		c.Gateway.UseTLS = useTLS == "true" || useTLS == "1" || useTLS == "yes"
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