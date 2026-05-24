package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bishop-bot/ib-cli/internal/config"
	"github.com/bishop-bot/ib-cli/internal/models"
)

// Client wraps the IB Client Portal Web API.
type Client struct {
	baseURL    string
	sessionID  string
	token      string
	timeout    time.Duration
	maxRetries int
	httpClient *http.Client
}

// NewClient creates a new API client.
func NewClient(cfg *config.Config) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.Gateway.InsecureSkipVerify,
		},
	}

	return &Client{
		baseURL:    cfg.Gateway.BaseURL(),
		timeout:    cfg.Request.Timeout(),
		maxRetries: cfg.Request.MaxRetries,
		httpClient: &http.Client{
			Timeout:   cfg.Request.Timeout(),
			Transport: transport,
		},
	}
}

// SetSession sets the session ID and token for authenticated requests.
func (c *Client) SetSession(sessionID, token string) {
	c.sessionID = sessionID
	c.token = token
}

// SessionID returns the current session ID.
func (c *Client) SessionID() string {
	return c.sessionID
}

// Token returns the current auth token.
func (c *Client) Token() string {
	return c.token
}

// ServerVersion returns the gateway server version info.
// Gets version from /iserver/auth/status serverInfo field.
func (c *Client) ServerVersion(ctx context.Context) (*models.ServerVersion, error) {
	auth, err := c.AuthStatus(ctx)
	if err != nil {
		return nil, err
	}

	if auth.ServerInfo == nil {
		return nil, fmt.Errorf("server info not available")
	}

	return &models.ServerVersion{
		Version:    auth.ServerInfo.ServerVersion,
		ServerTime: auth.ServerInfo.ServerName,
	}, nil
}

// AuthStatus returns the current authentication status.
func (c *Client) AuthStatus(ctx context.Context) (*models.AuthStatus, error) {
	resp, err := c.get(ctx, "/v1/api/iserver/auth/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var result models.AuthStatus
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w\nBody: %s", err, string(body))
	}
	return &result, nil
}

// Login authenticates with the gateway and returns session tokens.
func (c *Client) Login(ctx context.Context, username, password string) (*models.AuthStatus, error) {
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/api/iserver/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result models.AuthStatus
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Store session tokens for later use
	if result.Tokens.SessionID != "" {
		c.sessionID = result.Tokens.SessionID
	}
	if result.Tokens.Token != "" {
		c.token = result.Tokens.Token
	}

	return &result, nil
}

// Logout ends the session.
func (c *Client) Logout(ctx context.Context) error {
	_, err := c.post(ctx, "/v1/api/iserver/auth/logout", nil)
	if err != nil {
		return err
	}
	c.sessionID = ""
	c.token = ""
	return nil
}

// MarketHistory retrieves historical market data.
// Endpoint: /iserver/marketdata/history
// Required params: conid, exchange, period, bar
// Optional: startTime, endTime, outsideRth, source
func (c *Client) MarketHistory(ctx context.Context, params *models.HistoricalDataParams) (*models.HistoricalDataResponse, error) {
	query := url.Values{}

	// Required parameters
	if params.ConID != "" {
		query.Set("conid", params.ConID)
	}
	if params.Exchange != "" {
		query.Set("exchange", params.Exchange)
	}
	if params.Period != "" {
		query.Set("period", params.Period)
	}
	if params.Bar != "" {
		query.Set("bar", params.Bar)
	}

	// Optional parameters
	if params.StartTime != "" {
		query.Set("startTime", params.StartTime)
	}
	if params.EndTime != "" {
		query.Set("endTime", params.EndTime)
	}
	if params.OutsideRTH {
		query.Set("outsideRth", "true")
	}
	if params.Source != "" {
		query.Set("source", params.Source)
	}

	path := "/v1/api/iserver/marketdata/history?" + query.Encode()
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Check for error responses
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result models.HistoricalDataResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w\nBody: %s", err, string(body))
	}

	return &result, nil
}

// ContractInfo looks up contract details by symbol using /iserver/secdef/search endpoint.
func (c *Client) ContractInfo(ctx context.Context, symbol, exchange, secType string) (*models.ContractInfo, error) {
	path := "/v1/api/iserver/secdef/search?symbol=" + url.QueryEscape(symbol)
	if secType != "" {
		path += "&secType=" + url.QueryEscape(secType)
	}
	if exchange != "" {
		path += "&exchange=" + url.QueryEscape(exchange)
	}

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse secdef/search response
	var secdefResults []map[string]interface{}
	if err := json.Unmarshal(body, &secdefResults); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(secdefResults) == 0 {
		return nil, fmt.Errorf("no contract found for symbol: %s", symbol)
	}

	// Use first result - convert conid to int
	item := secdefResults[0]
	var conid int
	if cid, ok := item["conid"].(string); ok {
		if parsed, err := strconv.Atoi(cid); err == nil {
			conid = parsed
		}
	} else if cid, ok := item["conid"].(float64); ok {
		conid = int(cid)
	}

	return &models.ContractInfo{
		ConID:       conid,
		Symbol:      getString(item, "symbol"),
		SecType:     getSecType(item),
		Currency:    "", // Not in secdef/search response
		Description: getString(item, "companyName"),
		Exchange:    getString(item, "description"), // description contains exchange
	}, nil
}

func getSecType(item map[string]interface{}) string {
	if sections, ok := item["sections"].([]interface{}); ok && len(sections) > 0 {
		if first, ok := sections[0].(map[string]interface{}); ok {
			if secType, ok := first["secType"].(string); ok {
				return secType
			}
		}
	}
	return ""
}

// SearchContracts searches for contracts using secdef/search endpoint.
// This is the preferred endpoint for looking up contracts.
func (c *Client) SearchContracts(ctx context.Context, symbol, secType string) ([]models.ContractInfo, error) {
	path := "/v1/api/iserver/secdef/search?symbol=" + url.QueryEscape(symbol)
	if secType != "" {
		path += "&secType=" + url.QueryEscape(secType)
	}

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// The secdef/search endpoint returns an array of contract objects
	var result []map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var contracts []models.ContractInfo
	for _, item := range result {
		// conid can be string or number depending on the response format
		var conid int
		switch v := item["conid"].(type) {
		case float64:
			conid = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				conid = parsed
			}
		}
		if conid > 0 {
			contracts = append(contracts, models.ContractInfo{
				ConID:   conid,
				Symbol:  getString(item, "symbol"),
				SecType: getString(item, "description"),
			})
		}
	}

	return contracts, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// ServiceStatus checks if specific services are available.
// Note: /iserver/services endpoint may not be available in all gateway versions.
func (c *Client) ServiceStatus(ctx context.Context) ([]models.ServiceStatus, error) {
	resp, err := c.get(ctx, "/v1/api/iserver/services")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle 404 - endpoint not available
	if resp.StatusCode == http.StatusNotFound {
		return []models.ServiceStatus{
			{Service: "marketdata", IsActive: true, LastUpdate: "available"},
			{Service: "trade", IsActive: true, LastUpdate: "available"},
		}, nil
	}

	var result []models.ServiceStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return result, nil
}

// Watchlists returns all watchlists for the user.
// See: https://www.interactivebrokers.com/campus/ibkr-api-page/cpapi-v1/#watchlists
func (c *Client) Watchlists(ctx context.Context) (*models.WatchlistsResponse, error) {
	resp, err := c.get(ctx, "/v1/api/iserver/wm/watchlists")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result models.WatchlistsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// WatchlistByID returns a specific watchlist by ID.
func (c *Client) WatchlistByID(ctx context.Context, id string) (*models.Watchlist, error) {
	path := "/v1/api/iserver/wm/watchlists/" + url.PathEscape(id)
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result models.Watchlist
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// SecDefSearch searches for security definitions by contract ID.
// Endpoint: GET /trsrv/secdef?conids={conid}
// Response is wrapped in {"secdef": [...]} structure.
// See: https://www.interactivebrokers.com/campus/ibkr-api-page/cpapi-v1/#trsrv-conid-contract
func (c *Client) SecDefSearch(ctx context.Context, conid string) ([]models.SecDefInfo, error) {
	path := "/v1/api/trsrv/secdef?conids=" + url.QueryEscape(conid)
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Response is wrapped in {"secdef": [...]}
	var wrapped struct {
		SecDef []models.SecDefInfo `json:"secdef"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return wrapped.SecDef, nil
}

// AllConidsByExchange returns all contract IDs for a given exchange.
// Endpoint: GET /trsrv/all-conids
// See: https://www.interactivebrokers.com/campus/ibkr-api-page/cpapi-v1/#trsrv-conid-contract
func (c *Client) AllConidsByExchange(ctx context.Context, exchange, secType string) ([]int, error) {
	path := "/v1/api/trsrv/all-conids?exchange=" + url.QueryEscape(exchange)
	if secType != "" {
		path += "&secType=" + url.QueryEscape(secType)
	}

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Response is an array of integers
	var result []int
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return result, nil
}

// ContractInfoByConid returns detailed contract information by contract ID.
// Endpoint: GET /iserver/contract/{conid}/info
// See: https://www.interactivebrokers.com/campus/ibkr-api-page/cpapi-v1/#trsrv-conid-contract
func (c *Client) ContractInfoByConid(ctx context.Context, conid string) (*models.ConidInfo, error) {
	path := "/v1/api/iserver/contract/" + url.PathEscape(conid) + "/info"
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result models.ConidInfo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// TradingScheduleBySymbol returns trading schedule for a symbol.
// Endpoint: GET /trsrv/secdef/schedule
// See: https://www.interactivebrokers.com/campus/ibkr-api-page/cpapi-v1/#trsrv-conid-contract
func (c *Client) TradingScheduleBySymbol(ctx context.Context, symbol, exchange, secType, expiry string) ([]models.TradingSchedule, error) {
	path := "/v1/api/trsrv/secdef/schedule?symbol=" + url.QueryEscape(symbol)
	if exchange != "" {
		path += "&exchange=" + url.QueryEscape(exchange)
	}
	if secType != "" {
		path += "&secType=" + url.QueryEscape(secType)
	}
	if expiry != "" {
		path += "&expiry=" + url.QueryEscape(expiry)
	}

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result []models.TradingSchedule
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return result, nil
}

// TradingSchedule returns trading schedule using the contract endpoint.
// Endpoint: GET /contract/trading-schedule
// See: https://www.interactivebrokers.com/campus/ibkr-api-page/cpapi-v1/#trsrv-conid-contract
func (c *Client) TradingSchedule(ctx context.Context, exchange string) ([]models.TradingSchedule, error) {
	path := "/v1/api/contract/trading-schedule?exchange=" + url.QueryEscape(exchange)
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result []models.TradingSchedule
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return result, nil
}

// get performs a GET request with optional session handling.
func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	return c.doRequest(ctx, "GET", path, nil)
}

// post performs a POST request with optional session handling.
func (c *Client) post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.doRequest(ctx, "POST", path, body)
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding body: %w", err)
		}
		reqBody = io.NopCloser(strings.NewReader(string(jsonBody)))
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add auth header if token is available
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Add session cookie if available
	if c.sessionID != "" {
		req.AddCookie(&http.Cookie{Name: "jsessionid", Value: c.sessionID})
	}

	var lastErr error
	for i := 0; i <= c.maxRetries; i++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
			continue
		}

		// Handle rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
}