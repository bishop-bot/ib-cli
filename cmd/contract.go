package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/bishop-bot/ibcli-go/internal/api"
	"github.com/bishop-bot/ibcli-go/internal/config"
)

var contractCmd = &cobra.Command{
	Use:   "contract",
	Short: "Contract/security lookup and information",
	Long:  `Look up contract details, search securities, and retrieve contract information.`,
}

var lookupCmd = &cobra.Command{
	Use:   "lookup [symbol]",
	Short: "Look up contract details by symbol",
	Long: `Look up contract details by symbol.

Examples:
  ib-cli contract lookup AAPL
  ib-cli contract lookup AAPL --exchange NASDAQ --type STOCK
`,
	Args: cobra.ExactArgs(1),
	RunE: runLookup,
}

var secdefSearchCmd = &cobra.Command{
	Use:   "secdef [conid]",
	Short: "Search security definition by contract ID",
	Long: `Search for security definitions using the /trsrv/secdef endpoint.
Requires authentication.

Examples:
  ib-cli contract secdef 265598
  ib-cli contract secdef --symbol AAPL
`,
	RunE: runSecDefSearch,
}

var allConidsCmd = &cobra.Command{
	Use:   "all-conids",
	Short: "Get all contract IDs by exchange",
	Long: `Get all contract IDs for a given exchange using the /trsrv/all-conids endpoint.
Requires authentication.

Examples:
  ib-cli contract all-conids --exchange NASDAQ
  ib-cli contract all-conids --exchange SMART --type STOCK
`,
	RunE: runAllConids,
}

var conidInfoCmd = &cobra.Command{
	Use:   "info [conid]",
	Short: "Get detailed contract information by contract ID",
	Long: `Get detailed contract information by contract ID using the /iserver/contract/{conid}/info endpoint.
Requires authentication.

Examples:
  ib-cli contract info 265598
  ib-cli contract info 265598 --exchange NASDAQ
`,
	Args: cobra.ExactArgs(1),
	RunE: runConidInfo,
}

var scheduleCmd = &cobra.Command{
	Use:   "schedule [symbol]",
	Short: "Get trading schedule by symbol",
	Long: `Get trading schedule for a symbol using the /trsrv/secdef/schedule endpoint.
Requires authentication.

Examples:
  ib-cli contract schedule ES --exchange SMART --type FUT
  ib-cli contract schedule AAPL
`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runSchedule,
}

var tradingScheduleCmd = &cobra.Command{
	Use:   "trading-schedule",
	Short: "Get trading schedule for an exchange",
	Long: `Get trading schedule using the /contract/trading-schedule endpoint.
Requires authentication.

Examples:
  ib-cli contract trading-schedule --exchange NASDAQ
`,
	RunE: runTradingSchedule,
}

var (
	lookupParams struct {
		Exchange string
		SecType  string
	}

	secdefParams struct {
		Symbol  string
		SecType string
	}

	allConidsParams struct {
		Exchange string
		SecType  string
	}

	conidInfoParams struct {
		Exchange string
	}

	scheduleParams struct {
		Exchange string
		SecType  string
		Expiry   string
	}

	tradingScheduleParams struct {
		Exchange string
	}
)

func init() {
	rootCmd.AddCommand(contractCmd)
	contractCmd.AddCommand(lookupCmd)
	contractCmd.AddCommand(secdefSearchCmd)
	contractCmd.AddCommand(allConidsCmd)
	contractCmd.AddCommand(conidInfoCmd)
	contractCmd.AddCommand(scheduleCmd)
	contractCmd.AddCommand(tradingScheduleCmd)

	// lookup flags
	lookupCmd.Flags().StringVarP(&lookupParams.Exchange, "exchange", "e", "", "Exchange")
	lookupCmd.Flags().StringVarP(&lookupParams.SecType, "type", "t", "", "Security type")

	// secdef flags
	secdefSearchCmd.Flags().StringVar(&secdefParams.Symbol, "symbol", "", "Symbol to search")
	secdefSearchCmd.Flags().StringVarP(&secdefParams.SecType, "type", "t", "", "Security type")

	// all-conids flags
	allConidsCmd.Flags().StringVarP(&allConidsParams.Exchange, "exchange", "e", "", "Exchange (required)")
	allConidsCmd.Flags().StringVarP(&allConidsParams.SecType, "type", "t", "", "Security type")

	// conid-info flags
	conidInfoCmd.Flags().StringVarP(&conidInfoParams.Exchange, "exchange", "e", "", "Exchange")

	// schedule flags
	scheduleCmd.Flags().StringVarP(&scheduleParams.Exchange, "exchange", "e", "", "Exchange")
	scheduleCmd.Flags().StringVarP(&scheduleParams.SecType, "type", "t", "", "Security type")
	scheduleCmd.Flags().StringVar(&scheduleParams.Expiry, "expiry", "", "Contract expiry (YYYYMM)")

	// trading-schedule flags
	tradingScheduleCmd.Flags().StringVarP(&tradingScheduleParams.Exchange, "exchange", "e", "", "Exchange (required)")
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

func runSecDefSearch(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	var result interface{}
	var errOut error

	// If we have a positional arg (conid), use it; otherwise use --symbol
	if len(args) > 0 {
		result, errOut = client.SecDefSearch(ctx, args[0])
	} else if secdefParams.Symbol != "" {
		// Fall back to secdef/search endpoint with symbol
		result, errOut = client.SearchContracts(ctx, secdefParams.Symbol, secdefParams.SecType)
	} else {
		return fmt.Errorf("either a conid argument or --symbol flag is required")
	}

	if errOut != nil {
		return fmt.Errorf("secdef search: %w", errOut)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))

	return nil
}

func runAllConids(cmd *cobra.Command, args []string) error {
	if allConidsParams.Exchange == "" {
		return fmt.Errorf("--exchange flag is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	result, err := client.AllConidsByExchange(ctx, allConidsParams.Exchange, allConidsParams.SecType)
	if err != nil {
		return fmt.Errorf("all-conids: %w", err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))

	return nil
}

func runConidInfo(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	result, err := client.ContractInfoByConid(ctx, args[0])
	if err != nil {
		return fmt.Errorf("conid info: %w", err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))

	return nil
}

func runSchedule(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	var result interface{}
	var errOut error

	if len(args) > 0 {
		result, errOut = client.TradingScheduleBySymbol(ctx, args[0], scheduleParams.Exchange, scheduleParams.SecType, scheduleParams.Expiry)
	} else if scheduleParams.Exchange != "" {
		result, errOut = client.TradingSchedule(ctx, scheduleParams.Exchange)
	} else {
		return fmt.Errorf("either a symbol argument or --exchange flag is required")
	}

	if errOut != nil {
		return fmt.Errorf("schedule: %w", errOut)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))

	return nil
}

func runTradingSchedule(cmd *cobra.Command, args []string) error {
	if tradingScheduleParams.Exchange == "" {
		return fmt.Errorf("--exchange flag is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	result, err := client.TradingSchedule(ctx, tradingScheduleParams.Exchange)
	if err != nil {
		return fmt.Errorf("trading schedule: %w", err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))

	return nil
}

