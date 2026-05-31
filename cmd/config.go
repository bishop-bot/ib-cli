package cmd

import (
	"fmt"
	"os"

	"github.com/bishop-bot/ibcli-go/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show configuration",
	Long:  `Display the current configuration being used by the CLI.`,
}

var showConfigCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current config",
	Long:  `Display the current configuration values and which file they're loaded from.`,
	RunE:  runShowConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(showConfigCmd)
}

func runShowConfig(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Println("=== IB-CLI Configuration ===")
	fmt.Println()

	// Show which file was loaded
	fmt.Println("Config file search order:")
	fmt.Println("  1. --config flag (if provided)")
	fmt.Println("  2. ./config.toml (current directory)")
	fmt.Println("  3. ~/.ib-cli/config.toml")
	fmt.Println("  4. /etc/ib-cli/config.toml")
	fmt.Println("  5. Defaults (if no file found)")
	fmt.Println()

	// Check where the config actually comes from
	configFilePath := getConfigFilePath()
	if configFilePath != "<none>" {
		fmt.Printf("✓ Loaded from: %s\n\n", configFilePath)
	} else {
		fmt.Println("✓ Using defaults (no config file found)")
		fmt.Println("  Create a config.toml or use --config to specify one")
		fmt.Println()
	}

	// Show effective config
	fmt.Println("Effective configuration:")
	fmt.Println()
	fmt.Printf("  Gateway:\n")
	fmt.Printf("    URL:              %s\n", cfg.Gateway.BaseURL())
	fmt.Printf("    Host:             %s\n", cfg.Gateway.Host)
	fmt.Printf("    Port:             %d\n", cfg.Gateway.Port)
	fmt.Printf("    Use TLS:          %v\n", cfg.Gateway.UseTLS)
	fmt.Printf("    Insecure Skip Verify: %v\n", cfg.Gateway.InsecureSkipVerify)
	fmt.Println()

	if cfg.Auth.Username != "" {
		fmt.Printf("  Auth:\n")
		fmt.Printf("    Username: %s\n", cfg.Auth.Username)
		fmt.Printf("    Password: [set via config or IB_AUTH_PASSWORD env]\n")
	} else {
		fmt.Printf("  Auth:\n")
		fmt.Printf("    Username: [not set]\n")
		fmt.Printf("    Password: [not set]\n")
		fmt.Println("    Hint: Set IB_AUTH_USERNAME and IB_AUTH_PASSWORD env vars")
	}
	fmt.Println()

	fmt.Printf("  Output:\n")
	fmt.Printf("    Format:   %s\n", cfg.Output.DefaultFormat)
	fmt.Printf("    Pretty:   %v\n", cfg.Output.Pretty)
	fmt.Println()

	fmt.Printf("  Request:\n")
	fmt.Printf("    Bar Type:   %s\n", cfg.Request.BarType)
	fmt.Printf("    Bar Size:   %s\n", cfg.Request.BarSize)
	fmt.Printf("    Bar Unit:   %s\n", cfg.Request.BarUnit)
	fmt.Printf("    Timeout:    %d seconds\n", cfg.Request.TimeoutSecs)
	fmt.Printf("    Max Retries: %d\n", cfg.Request.MaxRetries)
	fmt.Println()

	fmt.Println("Environment variables that override config:")
	fmt.Println("  IB_GATEWAY_HOST")
	fmt.Println("  IB_GATEWAY_PORT")
	fmt.Println("  IB_GATEWAY_USE_TLS")
	fmt.Println("  IB_AUTH_USERNAME")
	fmt.Println("  IB_AUTH_PASSWORD")
	fmt.Println()

	return nil
}

// getConfigFilePath returns the path to the config file that will be used.
func getConfigFilePath() string {
	if configPath != "" {
		return configPath
	}

	// Check standard locations
	paths := []string{"config.toml"}

	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, home+"/.ib-cli/config.toml")
	}

	paths = append(paths, "/etc/ib-cli/config.toml")

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return "<none>"
}