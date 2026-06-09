package main

import (
	"context"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bishop-bot/ibcli-go/internal/config"
	"github.com/schollz/progressbar/v3"
)

// SecDefResponse is the response structure from /trsrv/secdef endpoint.
type SecDefResponse struct {
	SecDef []SecDefItem `json:"secdef"`
}

// SecDefItem represents a single security definition.
type SecDefItem struct {
	ConID           int     `json:"conid"`
	Currency        string  `json:"currency"`
	ListingExchange string  `json:"listingExchange"`
	CountryCode     string  `json:"countryCode"`
	Name            string  `json:"name"`
	AssetClass      string  `json:"assetClass"`
	Group           string  `json:"group"`
	Sector          string  `json:"sector"`
	SectorGroup     string  `json:"sectorGroup"`
	Type            string  `json:"type"`
	HasOptions      bool    `json:"hasOptions"`
	FullName        string  `json:"fullName"`
	Ticker          string  `json:"ticker"`
}

// CSVColumns defines the fields to extract for CSV output.
var CSVColumns = []string{
	"conid",
	"ticker",
	"currency",
	"listingExchange",
	"countryCode",
	"name",
	"assetClass",
	"group",
	"sector",
	"sectorGroup",
	"type",
	"hasOptions",
	"fullName",
}

type secdefConfig struct {
	exchange   string
	conidDir   string
	outputDir  string
	errorLog   string
	workers    int
	delay      time.Duration
	limit      int
	configPath string
	verbose    bool
}

func main() {
	cfg := parseFlags()

	// Setup logging
	errorWriter := os.Stderr
	if cfg.errorLog != "" {
		file, err := os.Create(cfg.errorLog)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating error log file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		errorWriter = file
		fmt.Fprintf(errorWriter, "Starting secdef fetch at %s\n", time.Now().Format(time.RFC3339))
	}

	// Load configuration
	if cfg.verbose {
		fmt.Printf("Loading config from: %s\n", cfg.configPath)
	}
	configLoader, err := config.Load(cfg.configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	if cfg.verbose {
		fmt.Printf("Gateway URL: %s\n", configLoader.Gateway.BaseURL())
	}

	// Load conids from JSON file
	if cfg.verbose {
		fmt.Printf("Loading conids from: %s/%s.json\n", cfg.conidDir, cfg.exchange)
	}
	conids, err := loadConids(cfg.conidDir, cfg.exchange)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading conids: %v\n", err)
		os.Exit(1)
	}

	if cfg.limit > 0 && cfg.limit < len(conids) {
		conids = conids[:cfg.limit]
	}

	fmt.Printf("Found %d conids for exchange: %s\n", len(conids), cfg.exchange)

	// Create HTTP client with TLS skip
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: tr,
	}

	baseURL := configLoader.Gateway.BaseURL()

	// Process conids
	fmt.Printf("Fetching security definitions (workers=%d, delay=%v)...\n", cfg.workers, cfg.delay)

	results := processConids(client, baseURL, conids, cfg.workers, cfg.delay, errorWriter, cfg.verbose)

	// Generate output filename with date
	dateStr := time.Now().Format("20060102")
	outputFile := filepath.Join(cfg.outputDir, fmt.Sprintf("%s_%s.csv", cfg.exchange, dateStr))

	// Write CSV
	if cfg.verbose {
		fmt.Printf("Writing CSV to: %s\n", outputFile)
	}
	if err := writeCSV(outputFile, results); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nComplete! Processed %d/%d records.\n", len(results), len(conids))
	fmt.Printf("Output: %s\n", outputFile)

	// Count errors
	errorCount := 0
	for _, r := range results {
		if r.Error != "" {
			errorCount++
		}
	}
	if errorCount > 0 {
		fmt.Printf("Errors: %d\n", errorCount)
		if cfg.errorLog != "" {
			fmt.Printf("Errors logged to: %s\n", cfg.errorLog)
		}
	}
}

// ConidRecord represents a conid entry from the JSON file.
type ConidRecord struct {
	Ticker   string `json:"ticker"`
	ConID    int    `json:"conid"`
	Exchange string `json:"exchange"`
}

// Result holds the result of a secdef lookup.
type Result struct {
	ConID           int
	Ticker          string
	Currency        string
	ListingExchange string
	CountryCode     string
	Name            string
	AssetClass      string
	Group           string
	Sector          string
	SectorGroup     string
	Type            string
	HasOptions      bool
	FullName        string
	Error           string
}

func parseFlags() secdefConfig {
	exchange := flag.String("exchange", "", "Exchange name (e.g., ARCA, NYSE, NASDAQ)")
	conidDir := flag.String("conid-dir", "assets/conid", "Directory containing conid JSON files")
	outputDir := flag.String("output-dir", ".", "Output directory for CSV")
	errorLog := flag.String("error-log", "", "File path to log failed lookups")
	workers := flag.Int("workers", 3, "Number of concurrent workers")
	delay := flag.Duration("delay", 300*time.Millisecond, "Delay between requests")
	limit := flag.Int("limit", 0, "Limit number of conids to process (0 = all)")
	configPath := flag.String("config", "config.toml", "Path to config file")
	verbose := flag.Bool("verbose", false, "Enable verbose output")

	flag.Parse()

	if *exchange == "" {
		fmt.Fprintf(os.Stderr, "Error: --exchange is required\n")
		flag.Usage()
		os.Exit(1)
	}

	return secdefConfig{
		exchange:   *exchange,
		conidDir:   *conidDir,
		outputDir:  *outputDir,
		errorLog:   *errorLog,
		workers:    *workers,
		delay:      *delay,
		limit:      *limit,
		configPath: *configPath,
		verbose:    *verbose,
	}
}

func loadConids(dir, exchange string) ([]ConidRecord, error) {
	filePath := filepath.Join(dir, exchange+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	var records []ConidRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return records, nil
}

func fetchSecDef(client *http.Client, baseURL string, conid int, verbose bool) (*SecDefItem, string, error) {
	path := fmt.Sprintf("/v1/api/trsrv/secdef?conids=%d", conid)
	url := baseURL + path

	if verbose {
		fmt.Printf("Fetching: %s\n", url)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, url, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, url, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, url, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, url, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result SecDefResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, url, fmt.Errorf("JSON decode failed: %w (body: %s)", err, string(body)[:min(200, len(body))])
	}

	// Find the matching conid in the secdef array
	for _, item := range result.SecDef {
		if item.ConID == conid {
			return &item, url, nil
		}
	}

	return nil, url, fmt.Errorf("conid %d not found in response (found %d items)", conid, len(result.SecDef))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func processConids(client *http.Client, baseURL string, conids []ConidRecord, workers int, delay time.Duration, errorWriter *os.File, verbose bool) []Result {
	results := make([]Result, len(conids))
	var wg sync.WaitGroup

	total := len(conids)

	// Create progress bar with ETA
	bar := progressbar.NewOptions(total,
		progressbar.OptionSetDescription(fmt.Sprintf("Fetching (%d workers)", workers)),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionShowCount(),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionClearOnFinish(),
	)

	semaphore := make(chan struct{}, workers)

	for i, c := range conids {
		wg.Add(1)
		go func(index int, conid int, ticker string, pb *progressbar.ProgressBar) {
			defer wg.Done()
			defer func() { <-semaphore }()

			semaphore <- struct{}{}

			// Small delay before processing
			time.Sleep(delay)

			item, url, err := fetchSecDef(client, baseURL, conid, verbose)
			if err != nil {
				errMsg := fmt.Sprintf("Failed: conid=%d, ticker=%s, url=%s, error=%v", conid, ticker, url, err)
				fmt.Fprintln(errorWriter, errMsg)
				results[index] = Result{
					ConID:  conid,
					Ticker: ticker,
					Error:  err.Error(),
				}
				pb.Add(1)
				return
			}

			results[index] = Result{
				ConID:           item.ConID,
				Ticker:          ticker,
				Currency:        item.Currency,
				ListingExchange: item.ListingExchange,
				CountryCode:     item.CountryCode,
				Name:            item.Name,
				AssetClass:      item.AssetClass,
				Group:           item.Group,
				Sector:          item.Sector,
				SectorGroup:     item.SectorGroup,
				Type:            item.Type,
				HasOptions:      item.HasOptions,
				FullName:        item.FullName,
			}
			pb.Add(1)
		}(i, c.ConID, c.Ticker, bar)
	}

	wg.Wait()
	bar.Close()

	return results
}

func writeCSV(path string, results []Result) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write(CSVColumns); err != nil {
		return err
	}

	// Write data rows
	for _, r := range results {
		row := []string{
			fmt.Sprintf("%d", r.ConID),
			r.Ticker,
			r.Currency,
			r.ListingExchange,
			r.CountryCode,
			r.Name,
			r.AssetClass,
			r.Group,
			r.Sector,
			r.SectorGroup,
			r.Type,
			fmt.Sprintf("%t", r.HasOptions),
			r.FullName,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}