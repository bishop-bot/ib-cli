package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bishop-bot/ibcli/internal/api"
	"github.com/bishop-bot/ibcli/internal/config"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Gateway server information",
	Long:  `Query the Client Portal Gateway for server status and version info.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show gateway version",
	RunE:  runVersion,
}

var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "List available services",
	RunE:  runServices,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(versionCmd)
	serverCmd.AddCommand(servicesCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	resp, err := client.ServerVersion(ctx)
	if err != nil {
		return fmt.Errorf("server version: %w", err)
	}

	if verbose {
		data, _ := json.MarshalIndent(resp, "", "  ")
		os.Stdout.Write(data)
		os.Stdout.Write([]byte("\n"))
	} else {
		fmt.Printf("Version: %s\nServer Time: %s\n", resp.Version, resp.ServerTime)
	}

	return nil
}

func runServices(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	resp, err := client.ServiceStatus(ctx)
	if err != nil {
		return fmt.Errorf("services: %w", err)
	}

	data, _ := json.MarshalIndent(resp, "", "  ")
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))

	return nil
}