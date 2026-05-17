package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bishop-bot/ib-cli/internal/api"
	"github.com/bishop-bot/ib-cli/internal/config"
	"github.com/bishop-bot/ib-cli/internal/models"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history [symbol]",
	Short: "Query historical market data",
	Long: `Retrieve historical market data for a security.

The historical market data endpoint requires:
- conid (contract ID) - can be looked up by symbol or provided directly
- exchange - where the security is listed (e.g., SMART, NYSE)
- period - the duration (e.g., 1d, 1w, 1m)
- bar - the bar size (e.g., 1min, 5min, 1h, 1d)

If no exchange is specified, "SMART" is used as the default.

Examples:
  # Get 1 day of 1-minute bars for AAPL (symbol lookup)
  ib-cli history AAPL --period 1d --bar 1min

  # Get 1 week of 5-minute bars with exchange
  ib-cli history AAPL -e SMART --period 1w --bar 5min --format json

  # Use conid directly (faster, no lookup needed)
  ib-cli history --conid 265598 --exchange SMART --period 1d --bar 1min

  # Get data outside regular trading hours
  ib-cli history AAPL --period 1w --bar 15min --outside-rth

Bar sizes: 1min, 2min, 3min, 5min, 10min, 15min, 30min, 1h, 2h, 3h, 4h, 8h, 1d, 1w, 1m
Periods: 1-30min, 1-8h, 1-1000d, 1-792w, 1-182m, 1-15y (e.g., 1d, 1w, 1m, 1y)
`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Symbol is optional if conid is provided
		if historyParams.ConID == "" && len(args) == 0 {
			return fmt.Errorf("requires either [symbol] argument or --conid flag")
		}
		if len(args) > 1 {
			return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
		}
		return nil
	},
	RunE: runHistory,
}

var (
	historyParams struct {
		ConID      string
		Exchange   string
		Period     string
		Bar        string
		StartTime  string
		EndTime    string
		OutsideRTH bool
		Source     string
		Format     string
	}
)

func init() {
	rootCmd.AddCommand(historyCmd)

	historyCmd.Flags().StringVarP(&historyParams.ConID, "conid", "i", "", "Contract ID (conId)")
	historyCmd.Flags().StringVarP(&historyParams.Exchange, "exchange", "e", "", "Exchange (e.g., SMART, NYSE)")
	historyCmd.Flags().StringVarP(&historyParams.Period, "period", "p", "", "Duration (e.g., 1d, 1w, 1m)")
	historyCmd.Flags().StringVarP(&historyParams.Bar, "bar", "b", "", "Bar size (e.g., 1min, 5min, 1h, 1d)")
	historyCmd.Flags().StringVarP(&historyParams.StartTime, "start", "s", "", "Start time (YYYYMMDD-HH:mm:ss)")
	historyCmd.Flags().StringVarP(&historyParams.EndTime, "end", "t", "", "End time")
	historyCmd.Flags().BoolVarP(&historyParams.OutsideRTH, "outside-rth", "o", false, "Include data outside regular trading hours")
	historyCmd.Flags().StringVarP(&historyParams.Source, "source", "S", "", "Data source (Trades, Midpoint, Bid_Ask)")
	historyCmd.Flags().StringVarP(&historyParams.Format, "format", "f", "json", "Output format: json, csv")
}

func runHistory(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Setup client
	client := api.NewClient(cfg)

	symbol := ""
	if len(args) > 0 {
		symbol = args[0]
	}

	// Check auth before requesting data
	auth, err := client.AuthStatus(ctx)
	if err != nil {
		return fmt.Errorf("auth status: %w", err)
	}

	if !auth.IsAuthenticated() {
		return fmt.Errorf("not authenticated. Run 'ib-cli auth login' first")
	}

	// Build params
	params := &models.HistoricalDataParams{
		Symbol:     symbol,
		Exchange:   historyParams.Exchange,
		StartTime:  historyParams.StartTime,
		EndTime:    historyParams.EndTime,
		OutsideRTH: historyParams.OutsideRTH,
		Source:     historyParams.Source,
	}

	// Apply CLI args or defaults
	if historyParams.Period != "" {
		params.Period = historyParams.Period
	} else {
		params.Period = "1d" // Default to 1 day
	}

	if historyParams.Bar != "" {
		params.Bar = historyParams.Bar
	} else {
		params.Bar = "1min" // Default to 1 minute
	}

	// Resolve conid: use provided conid or look up by symbol
	if historyParams.ConID != "" {
		params.ConID = historyParams.ConID
	} else {
		// Look up conid from symbol using secdef/search endpoint
		contracts, err := client.SearchContracts(ctx, symbol, "STOCK")
		if err != nil {
			return fmt.Errorf("looking up contract %s: %w", symbol, err)
		}
		if len(contracts) == 0 {
			return fmt.Errorf("no contract found for symbol: %s", symbol)
		}
		params.ConID = strconv.Itoa(contracts[0].ConID)
	}

	// Use SMART as default exchange if not specified
	if params.Exchange == "" {
		params.Exchange = "SMART"
	}

	// Fetch historical data
	data, err := client.MarketHistory(ctx, params)
	if err != nil {
		return fmt.Errorf("market history: %w", err)
	}

	// Output
	switch historyParams.Format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.Encode(data)
	case "csv":
		return printCSV(data)
	default:
		return fmt.Errorf("unsupported format: %s", historyParams.Format)
	}

	return nil
}

func printCSV(data *models.HistoricalDataResponse) error {
	fmt.Println("timestamp,open,high,low,close,volume")
	for _, bar := range data.Data {
		ts := time.UnixMilli(bar.Timestamp).UTC().Format(time.RFC3339)
		fmt.Printf("%s,%.4f,%.4f,%.4f,%.4f,%.4f\n",
			ts, bar.Open, bar.High, bar.Low, bar.Close, bar.Volume)
	}
	return nil
}

// parsePeriod converts human-friendly duration to IB API format.
// e.g., "1 D" -> "1d", "1 W" -> "1w", "1 M" -> "1m"
func parsePeriod(duration string) string {
	duration = strings.ToUpper(strings.TrimSpace(duration))

	// Handle common formats
	switch duration {
	case "1 D", "1D", "D":
		return "1d"
	case "1 W", "1W", "W":
		return "1w"
	case "1 M", "1M", "M":
		return "1m"
	case "1 Y", "1Y", "Y":
		return "1y"
	default:
		// Try to parse number + unit
		parts := strings.Split(duration, " ")
		if len(parts) == 1 {
			parts = strings.Split(duration, "")
		}
		if len(parts) >= 2 {
			num := strings.TrimRight(parts[0], " \t")
			unit := strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
			switch unit {
			case "s", "sec", "second":
				return num + "min" // Not supported, fallback
			case "min", "minute":
				return num + "min"
			case "h", "hour":
				return num + "h"
			case "d", "day":
				return num + "d"
			case "w", "week":
				return num + "w"
			case "m", "month":
				return num + "m"
			case "y", "year":
				return num + "y"
			}
		}
		return duration
	}
}