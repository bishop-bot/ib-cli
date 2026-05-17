package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bishop-bot/ib-cli/internal/api"
	"github.com/bishop-bot/ib-cli/internal/config"
	"github.com/spf13/cobra"
)

var contractCmd = &cobra.Command{
	Use:   "contract",
	Short: "Contract/security lookup",
	Long:  `Look up contract details by symbol.`,
}

var lookupCmd = &cobra.Command{
	Use:   "lookup [symbol]",
	Short: "Look up contract details",
	Long: `Look up contract details by symbol.

Examples:
  ib-cli contract lookup AAPL
  ib-cli contract lookup AAPL --exchange NASDAQ --type STOCK
`,
	Args: cobra.ExactArgs(1),
	RunE: runLookup,
}

var (
	lookupParams struct {
		Exchange string
		SecType  string
	}
)

func init() {
	rootCmd.AddCommand(contractCmd)
	contractCmd.AddCommand(lookupCmd)

	lookupCmd.Flags().StringVarP(&lookupParams.Exchange, "exchange", "e", "", "Exchange")
	lookupCmd.Flags().StringVarP(&lookupParams.SecType, "type", "t", "", "Security type")
}

func runLookup(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	resp, err := client.ContractInfo(ctx, args[0], lookupParams.Exchange, lookupParams.SecType)
	if err != nil {
		return fmt.Errorf("contract lookup: %w", err)
	}

	data, _ := json.MarshalIndent(resp, "", "  ")
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))

	return nil
}