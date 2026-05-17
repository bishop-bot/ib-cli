package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

Examples:
  # Get 1 day of 1-minute bars for AAPL
  ib-cli history AAPL --bar 1 --unit min --duration "1 D"

  # Get 1 week of 5-minute bars
  ib-cli history AAPL -b 5 -u min -d "1 W" --format json

  # Get daily bars for ES futures
  ib-cli history ES --exchange SMART --type FUT -d "1 M"

  # Query with time range
  ib-cli history AAPL --start "2024-01-01T00:00:00Z" --end "2024-01-31T23:59:59Z"

Supported bar types: TRADES, BID, ASK, MIDPOINT, SCHEDULE
Supported bar units: S (seconds), min (minutes), D (days), W (weeks), M (months), Y (years)
`,
	Args: cobra.ExactArgs(1),
	RunE: runHistory,
}

var (
	historyParams struct {
		ConID         string
		Exchange      string
		SecType       string
		ExchangeRoute string
		StartTime     string
		EndTime       string
		BarSize       string
		BarUnit       string
		BarType       string
		Duration      string
		OutsideRTH    bool
		FormatDate    int
		UseRTH        bool
		Limit         int
		OverrideSpace bool
		OutputFormat  string
		Pretty        bool
	}
)

func init() {
	rootCmd.AddCommand(historyCmd)

	historyCmd.Flags().StringVarP(&historyParams.ConID, "conid", "", "", "Contract ID")
	historyCmd.Flags().StringVarP(&historyParams.Exchange, "exchange", "e", "", "Exchange (e.g., SMART, NYSE)")
	historyCmd.Flags().StringVarP(&historyParams.SecType, "type", "t", "", "Security type (STOCK, OPT, FUT)")
	historyCmd.Flags().StringVar(&historyParams.ExchangeRoute, "exchange-route", "", "Exchange for routing")
	historyCmd.Flags().StringVarP(&historyParams.StartTime, "start", "S", "", "Start time (RFC3339 format)")
	historyCmd.Flags().StringVarP(&historyParams.EndTime, "end", "E", "", "End time (RFC3339 format)")
	historyCmd.Flags().StringVarP(&historyParams.BarSize, "bar", "b", "", "Bar size number (1, 5, 15, etc.)")
	historyCmd.Flags().StringVarP(&historyParams.BarUnit, "unit", "u", "", "Bar unit (S, min, D, W, M, Y)")
	historyCmd.Flags().StringVarP(&historyParams.BarType, "bar-type", "m", "", "Bar type (TRADES, BID, ASK, MIDPOINT)")
	historyCmd.Flags().StringVarP(&historyParams.Duration, "duration", "d", "", "Duration string (e.g., 1 D, 1 W, 1 M)")
	historyCmd.Flags().BoolVarP(&historyParams.OutsideRTH, "outside-rth", "o", false, "Include data outside regular trading hours")
	historyCmd.Flags().IntVarP(&historyParams.FormatDate, "format-date", "f", 2, "Date format (1=unix, 2=RFC3339)")
	historyCmd.Flags().BoolVarP(&historyParams.UseRTH, "use-rth", "r", false, "Use regular trading hours only")
	historyCmd.Flags().IntVarP(&historyParams.Limit, "limit", "l", 0, "Maximum number of bars")
	historyCmd.Flags().BoolVar(&historyParams.OverrideSpace, "override-spacing", false, "Override default spacing")

	// Output flags
	historyCmd.Flags().StringVarP(&historyParams.OutputFormat, "output", "O", "", "Output format: json, csv (default: from config)")
	historyCmd.Flags().BoolVar(&historyParams.Pretty, "pretty", false, "Pretty print JSON output")
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

	symbol := args[0]

	// Build params - CLI args override config
	params := &models.HistoricalDataParams{
		Symbol:         symbol,
		ConID:         historyParams.ConID,
		Exchange:      historyParams.Exchange,
		SecType:       historyParams.SecType,
		ExchangeRoute: historyParams.ExchangeRoute,
		StartTime:     historyParams.StartTime,
		EndTime:       historyParams.EndTime,
		OutsideRTH:    historyParams.OutsideRTH,
		FormatDate:    historyParams.FormatDate,
		UseRTH:        historyParams.UseRTH,
		Limit:         historyParams.Limit,
		OverrideSpacing: historyParams.OverrideSpace,
	}

	// Apply bar parameters (CLI > config defaults)
	if historyParams.BarSize != "" {
		params.BarSize = historyParams.BarSize
	} else {
		params.BarSize = cfg.Request.BarSize
	}
	if historyParams.BarUnit != "" {
		params.BarUnit = historyParams.BarUnit
	} else {
		params.BarUnit = cfg.Request.BarUnit
	}
	if historyParams.BarType != "" {
		params.BarType = historyParams.BarType
	} else {
		params.BarType = cfg.Request.BarType
	}
	if historyParams.Duration != "" {
		params.Duration = historyParams.Duration
	}

	// Check auth before requesting data
	auth, err := client.AuthStatus(ctx)
	if err != nil {
		return fmt.Errorf("auth status: %w", err)
	}

	if !auth.IsAuthenticated() {
		return fmt.Errorf("not authenticated. Run 'ib-cli auth login' first")
	}

	// Fetch historical data
	data, err := client.MarketHistory(ctx, params)
	if err != nil {
		return fmt.Errorf("market history: %w", err)
	}

	// Determine output format
	outputFormat := historyParams.OutputFormat
	if outputFormat == "" {
		outputFormat = cfg.Output.DefaultFormat
	}
	pretty := historyParams.Pretty || cfg.Output.Pretty

	// Output
	var output []byte
	switch outputFormat {
	case "json", "":
		if pretty {
			output, err = json.MarshalIndent(data, "", "  ")
		} else {
			output, err = json.Marshal(data)
		}
	case "csv":
		output, err = toCSV(data)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	if err != nil {
		return fmt.Errorf("encoding output: %w", err)
	}

	os.Stdout.Write(output)
	os.Stdout.Write([]byte("\n"))
	return nil
}

func toCSV(data *models.HistoricalDataResponse) ([]byte, error) {
	var b strings.Builder
	b.WriteString("time,open,high,low,close,volume,wap,count\n")
	for _, bar := range data.Bars {
		fmt.Fprintf(&b, "%s,%.4f,%.4f,%.4f,%.4f,%d,%.4f,%d\n",
			bar.Time, bar.Open, bar.High, bar.Low, bar.Close,
			bar.Volume, bar.WAP, bar.Count)
	}
	return []byte(b.String()), nil
}