package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bishop-bot/ibcli-go/internal/api"
	"github.com/bishop-bot/ibcli-go/internal/config"
	"github.com/bishop-bot/ibcli-go/internal/models"
	"github.com/spf13/cobra"
)

var watchlistCmd = &cobra.Command{
	Use:   "watchlist",
	Short: "Watchlist commands",
	Long:  `Manage and query watchlists from the IB Client Portal Gateway.`,
}

var listWatchlistsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all watchlists",
	Long: `Retrieve all watchlists for the authenticated user.

Examples:
  ib-cli watchlist list
  ib-cli watchlist list --format table
`,
	RunE: runListWatchlists,
}

var getWatchlistCmd = &cobra.Command{
	Use:   "get [watchlist-id]",
	Short: "Get a specific watchlist",
	Long: `Retrieve a specific watchlist by its ID.

Examples:
  ib-cli watchlist get abc123-def456
`,
	Args: cobra.ExactArgs(1),
	RunE: runGetWatchlist,
}

var (
	watchlistParams struct {
		Format string
		Pretty bool
	}
)

func init() {
	rootCmd.AddCommand(watchlistCmd)
	watchlistCmd.AddCommand(listWatchlistsCmd)
	watchlistCmd.AddCommand(getWatchlistCmd)

	listWatchlistsCmd.Flags().StringVarP(&watchlistParams.Format, "format", "f", "json", "Output format: json, table")
	listWatchlistsCmd.Flags().BoolVar(&watchlistParams.Pretty, "pretty", false, "Pretty print JSON output")

	getWatchlistCmd.Flags().StringVarP(&watchlistParams.Format, "format", "f", "json", "Output format: json, table")
	getWatchlistCmd.Flags().BoolVar(&watchlistParams.Pretty, "pretty", false, "Pretty print JSON output")
}

func runListWatchlists(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	// Check auth
	auth, err := client.AuthStatus(ctx)
	if err != nil {
		return fmt.Errorf("auth status: %w", err)
	}
	if !auth.IsAuthenticated() {
		return fmt.Errorf("not authenticated. Run 'ib-cli auth login' first")
	}

	// Fetch watchlists
	resp, err := client.Watchlists(ctx)
	if err != nil {
		return fmt.Errorf("fetching watchlists: %w", err)
	}

	// Output
	switch watchlistParams.Format {
	case "table":
		printWatchlistsTable(resp)
	case "json", "":
		printWatchlistsJSON(resp)
	default:
		return fmt.Errorf("unsupported format: %s (use json or table)", watchlistParams.Format)
	}

	return nil
}

func runGetWatchlist(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client := api.NewClient(cfg)

	// Check auth
	auth, err := client.AuthStatus(ctx)
	if err != nil {
		return fmt.Errorf("auth status: %w", err)
	}
	if !auth.IsAuthenticated() {
		return fmt.Errorf("not authenticated. Run 'ib-cli auth login' first")
	}

	// Fetch watchlist
	resp, err := client.WatchlistByID(ctx, args[0])
	if err != nil {
		return fmt.Errorf("fetching watchlist: %w", err)
	}

	// Output
	switch watchlistParams.Format {
	case "table":
		printWatchlistTable(resp)
	case "json", "":
		printWatchlistJSON(resp)
	default:
		return fmt.Errorf("unsupported format: %s (use json or table)", watchlistParams.Format)
	}

	return nil
}

func printWatchlistsJSON(resp *models.WatchlistsResponse) {
	var output []byte
	var err error

	if watchlistParams.Pretty {
		output, err = json.MarshalIndent(resp, "", "  ")
	} else {
		output, err = json.Marshal(resp)
	}

	if err != nil {
		fatal("encoding json: %v", err)
	}

	os.Stdout.Write(output)
	os.Stdout.Write([]byte("\n"))
}

func printWatchlistJSON(resp *models.Watchlist) {
	var output []byte
	var err error

	if watchlistParams.Pretty {
		output, err = json.MarshalIndent(resp, "", "  ")
	} else {
		output, err = json.Marshal(resp)
	}

	if err != nil {
		fatal("encoding json: %v", err)
	}

	os.Stdout.Write(output)
	os.Stdout.Write([]byte("\n"))
}

func printWatchlistsTable(resp *models.WatchlistsResponse) {
	fmt.Printf("%-10s %-30s %-10s\n", "ID", "NAME", "DEFAULT")
	fmt.Println("---------- ------------------------------ ----------")

	for _, wl := range resp.Watchlists {
		defaultMark := ""
		if wl.DefaultList {
			defaultMark = "✓"
		}
		fmt.Printf("%-10s %-30s %-10s\n", wl.ID, wl.Name, defaultMark)
	}
	fmt.Printf("\nTotal: %d watchlist(s)\n", len(resp.Watchlists))
}

func printWatchlistTable(resp *models.Watchlist) {
	fmt.Printf("Watchlist: %s (ID: %s)\n", resp.Name, resp.ID)
	if resp.DefaultList {
		fmt.Println("Default watchlist")
	}
	fmt.Println()

	if len(resp.Symbols) > 0 {
		fmt.Printf("%-10s %-15s %-10s\n", "CONID", "SYMBOL", "TYPE")
		fmt.Println("---------- --------------- ----------")

		for _, sym := range resp.Symbols {
			fmt.Printf("%-10d %-15s %-10s\n", sym.ConID, sym.Symbol, sym.SecType)
		}
		fmt.Printf("\nTotal: %d symbol(s)\n", len(resp.Symbols))
	} else {
		fmt.Println("No symbols in this watchlist")
	}
}