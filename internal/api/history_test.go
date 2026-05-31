package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/bishop-bot/ibcli-go/internal/config"
	"github.com/bishop-bot/ibcli-go/internal/models"
)

// TestMarketHistoryRequestParams verifies the correct params are sent to the API.
// Per IB documentation: conid, exchange, period, bar, startTime, outsideRth, source
func TestMarketHistoryRequestParams(t *testing.T) {
	var receivedQuery string

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the path and query params
		if r.URL.Path != "/v1/api/iserver/marketdata/history" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		receivedQuery = r.URL.RawQuery

		// Return sample response
		resp := models.HistoricalDataResponse{
			Symbol: "AAPL",
			Data:   []models.Bar{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			Host:               "localhost",
			Port:               5000,
			InsecureSkipVerify: true,
		},
		Request: config.RequestConfig{
			TimeoutSecs: 30,
			MaxRetries:  3,
		},
	}

	client := NewClient(cfg)
	client.baseURL = server.URL

	params := &models.HistoricalDataParams{
		ConID:      "265598",
		Exchange:   "SMART",
		Period:     "1d",
		Bar:        "1min",
		StartTime:  "20230821-13:30:00",
		OutsideRTH: true,
		Source:     "Midpoint",
	}

	_, err := client.MarketHistory(context.Background(), params)
	if err != nil {
		t.Fatalf("MarketHistory failed: %v", err)
	}

	// Verify query params are correctly formatted per IB API
	if receivedQuery == "" {
		t.Fatal("no query params received")
	}

	// Check conid
	if !containsParam(receivedQuery, "conid=265598") {
		t.Errorf("missing or incorrect conid param. Got: %s", receivedQuery)
	}

	// Check exchange
	if !containsParam(receivedQuery, "exchange=SMART") {
		t.Errorf("missing or incorrect exchange param. Got: %s", receivedQuery)
	}

	// Check period (duration)
	if !containsParam(receivedQuery, "period=1d") {
		t.Errorf("missing or incorrect period param. Got: %s", receivedQuery)
	}

	// Check bar (bar size)
	if !containsParam(receivedQuery, "bar=1min") {
		t.Errorf("missing or incorrect bar param. Got: %s", receivedQuery)
	}

	// Check startTime
	if !containsParam(receivedQuery, "startTime=20230821-13:30:00") {
		t.Errorf("missing or incorrect startTime param. Got: %s", receivedQuery)
	}

	// Check outsideRTH
	if !containsParam(receivedQuery, "outsideRth=true") {
		t.Errorf("missing or incorrect outsideRth param. Got: %s", receivedQuery)
	}

	// Check source
	if !containsParam(receivedQuery, "source=Midpoint") {
		t.Errorf("missing or incorrect source param. Got: %s", receivedQuery)
	}
}

func containsParam(query, param string) bool {
	// Handle URL encoding: %3A -> :, %20 -> space, etc.
	decodedQuery, _ := url.QueryUnescape(query)
	decodedParam, _ := url.QueryUnescape(param)
	
	return contains(decodedQuery, decodedParam+"&") || contains(decodedQuery, "&"+decodedParam) ||
		contains(decodedQuery, decodedParam+"?") || contains(decodedQuery, decodedParam)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestMarketHistoryResponseParsing verifies the API response is parsed correctly.
func TestMarketHistoryResponseParsing(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return sample response matching IB API format
		resp := `{
			"serverId": "20477",
			"symbol": "AAPL",
			"text": "APPLE INC",
			"priceFactor": 100,
			"startTime": "20230818-08:00:00",
			"high": "17510/472117.45/0",
			"low": "17170/472117.45/0",
			"timePeriod": "1d",
			"barLength": 86400,
			"mdAvailability": "S",
			"mktDataDelay": 0,
			"outsideRth": true,
			"volumeFactor": 1,
			"priceDisplayRule": 1,
			"priceDisplayValue": "2",
			"chartPanStartTime": "20230821-13:30:00",
			"direction": -1,
			"negativeCapable": false,
			"messageVersion": 2,
			"data": [
				{
					"o": 173.4,
					"c": 174.7,
					"h": 175.1,
					"l": 171.7,
					"v": 472117.45,
					"t": 16923456000
				},
				{
					"o": 174.5,
					"c": 175.2,
					"h": 176.0,
					"l": 174.0,
					"v": 523450.30,
					"t": 16924320000
				}
			],
			"points": 2,
			"travelTime": 48
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}))
	defer server.Close()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			Host:               "localhost",
			Port:               5000,
			InsecureSkipVerify: true,
		},
		Request: config.RequestConfig{
			TimeoutSecs: 30,
			MaxRetries:  3,
		},
	}

	client := NewClient(cfg)
	client.baseURL = server.URL

	params := &models.HistoricalDataParams{
		ConID:  "265598",
		Period: "1d",
		Bar:    "1d",
	}

	result, err := client.MarketHistory(context.Background(), params)
	if err != nil {
		t.Fatalf("MarketHistory failed: %v", err)
	}

	// Verify Symbol
	if result.Symbol != "AAPL" {
		t.Errorf("Symbol = %s, want AAPL", result.Symbol)
	}

	// Verify ServerID
	if result.ServerID != "20477" {
		t.Errorf("ServerID = %s, want 20477", result.ServerID)
	}

	// Verify Bar count
	if len(result.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(result.Data))
	}

	// Verify first bar data
	bar := result.Data[0]
	if bar.Open != 173.4 {
		t.Errorf("first bar Open = %f, want 173.4", bar.Open)
	}
	if bar.Close != 174.7 {
		t.Errorf("first bar Close = %f, want 174.7", bar.Close)
	}
	if bar.High != 175.1 {
		t.Errorf("first bar High = %f, want 175.1", bar.High)
	}
	if bar.Low != 171.7 {
		t.Errorf("first bar Low = %f, want 171.7", bar.Low)
	}
	if bar.Volume != 472117.45 {
		t.Errorf("first bar Volume = %f, want 472117.45", bar.Volume)
	}
	if bar.Timestamp != 16923456000 {
		t.Errorf("first bar Timestamp = %d, want 16923456000", bar.Timestamp)
	}
}

// TestMarketHistoryErrorResponse verifies error responses are handled.
func TestMarketHistoryErrorResponse(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid conid"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			Host:               "localhost",
			Port:               5000,
			InsecureSkipVerify: true,
		},
		Request: config.RequestConfig{
			TimeoutSecs: 30,
			MaxRetries:  0, // No retries for faster test
		},
	}

	client := NewClient(cfg)
	client.baseURL = server.URL

	params := &models.HistoricalDataParams{
		ConID: "invalid",
		Bar:   "1d",
	}

	_, err := client.MarketHistory(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for invalid conid")
	}
}

// TestMarketHistoryEmptyConid tests the case when conid is empty (should still send request)
func TestMarketHistoryWithoutOptionalConid(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request was made (even with empty conid)
		if r.URL.RawQuery == "" {
			t.Error("expected query params even with minimal params")
		}
		resp := models.HistoricalDataResponse{
			Symbol: "AAPL",
			Data:   []models.Bar{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			Host:               "localhost",
			Port:               5000,
			InsecureSkipVerify: true,
		},
		Request: config.RequestConfig{
			TimeoutSecs: 30,
			MaxRetries:  3,
		},
	}

	client := NewClient(cfg)
	client.baseURL = server.URL

	params := &models.HistoricalDataParams{
		ConID:  "", // Empty conid
		Bar:    "1d",
		Period: "1d",
	}

	_, err := client.MarketHistory(context.Background(), params)
	if err != nil {
		// Error expected for empty conid with this mock
		t.Logf("expected error with empty conid in mock: %v", err)
	}
}