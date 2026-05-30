package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	exchange := flag.String("exchange", "", "Exchange value for output (e.g., NASDAQ, NYSE)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <input.csv>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Maps a CSV file to secdef instrument format.\n\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	if *exchange == "" {
		fmt.Fprintf(os.Stderr, "Error: --exchange is required\n")
		flag.Usage()
		os.Exit(1)
	}

	inputFile := flag.Arg(0)

	// Generate output filename: {inputFilename}_mapped.csv
	baseName := filepath.Base(inputFile)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	outputFile := filepath.Join(filepath.Dir(inputFile), nameWithoutExt+"_mapped"+ext)

	if err := mapCSV(inputFile, outputFile, *exchange); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created %s\n", outputFile)
}

func mapCSV(inputFile, outputFile, exchange string) error {
	// Open input file
	in, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("opening input file: %w", err)
	}
	defer in.Close()

	// Create output file
	out, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	reader := csv.NewReader(in)
	writer := csv.NewWriter(out)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("reading header: %w", err)
	}

	// Find column indices
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[col] = i
	}

	// Define new header
	newHeader := []string{
		"id", "symbol", "name", "publisher", "instrument_class", "currency", "exchange",
		"asset", "security_type", "min_lot_size", "expiration", "max_price_variation",
		"unit_of_measure_qty", "min_price_increment", "display_factor",
		"price_display_format", "price_ratio REAL", "underlying_symbol",
		"maturity_year", "maturity_month", "maturity_day", "group",
		"tick_rule", "strike_price", "strike_price_currency",
	}

	// Write new header
	if err := writer.Write(newHeader); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}

	// Asset class translation map
	assetClassMap := map[string]string{
		"BND":  "B",
		"CASH": "X",
		"IND":  "I",
		"STK":  "K",
		"FUT":  "F",
	}

	// Read and transform rows
	rowNum := 1
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		rowNum++

		getVal := func(col string) string {
			if idx, ok := colIndex[col]; ok && idx < len(record) {
				return record[idx]
			}
			return ""
		}

		// Get and translate assetClass
		assetClass := getVal("assetClass")
		instrumentClass := assetClassMap[assetClass]
		if instrumentClass == "" {
			instrumentClass = assetClass // use original if no mapping
		}

		// Build new row
		newRow := []string{
			getVal("conid"),                     // id
			getVal("ticker"),                    // symbol
			getVal("name"),                      // name
			"",                                  // publisher
			instrumentClass,                     // instrument_class
			getVal("currency"),                  // currency
			exchange,                            // exchange (from --exchange flag)
			"",                                  // asset
			"",                                  // security_type
			"",                                  // min_lot_size
			"",                                  // expiration
			"",                                  // max_price_variation
			"",                                  // unit_of_measure_qty
			"",                                  // min_price_increment
			"",                                  // display_factor
			"",                                  // price_display_format
			"",                                  // price_ratio REAL
			"",                                  // underlying_symbol
			"",                                  // maturity_year
			"",                                  // maturity_month
			"",                                  // maturity_day
			strings.Trim(getVal("group"), "\""), // group (remove quotes)
			"",                                  // tick_rule
			"",                                  // strike_price
			"",                                  // strike_price_currency
		}

		if err := writer.Write(newRow); err != nil {
			return fmt.Errorf("writing row %d: %w", rowNum, err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flushing writer: %w", err)
	}

	return nil
}
