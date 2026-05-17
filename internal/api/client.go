package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bishop-bot/datajobs-go/ib-cli/internal/config"
	"github.com/bishop-bot/datajobs-go/ib-cli/internal/models"
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
func (c *Client) ServerVersion(ctx context.Context) (*models.ServerVersion, error) {
	resp, err := c.get(ctx, "/v1/api/iserver/contracts/version")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.ServerVersion
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// AuthStatus returns the current authentication status.
func (c *Client) AuthStatus(ctx context.Context) (*models.AuthStatus, error) {
	resp, err := c.get(ctx, "/v1/api/iserver/auth/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.AuthStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
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
func (c *Client) MarketHistory(ctx context.Context, params *models.HistoricalDataParams) (*models.HistoricalDataResponse, error) {
	query := url.Values{}

	if params.Symbol != "" {
		query.Set("symbol", params.Symbol)
	}
	if params.ConID != "" {
		query.Set("conid", params.ConID)
	}
	if params.Exchange != "" {
		query.Set("exchange", params.Exchange)
	}
	if params.SecType != "" {
		query.Set("type", params.SecType)
	}
	if params.ExchangeRoute != "" {
		query.Set("exchange-route", params.ExchangeRoute)
	}
	if params.StartTime != "" {
		query.Set("start", params.StartTime)
	}
	if params.EndTime != "" {
		query.Set("end", params.EndTime)
	}
	if params.BarSize != "" {
		query.Set("bar", params.BarSize)
	}
	if params.BarUnit != "" {
		query.Set("period", params.BarUnit)
	}
	if params.BarType != "" {
		query.Set("barType", params.BarType)
	}
	if params.Duration != "" {
		query.Set("duration", params.Duration)
	}
	if params.OutsideRTH {
		query.Set("outsideRTH", "true")
	}
	if params.FormatDate != 0 {
		query.Set("formatDate", fmt.Sprintf("%d", params.FormatDate))
	}
	if params.UseRTH {
		query.Set("useRTH", "true")
	}
	if params.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	if params.OverrideSpacing {
		query.Set("overrideSpacing", "true")
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
		// Try parsing as error object
		var errResp map[string]interface{}
		if json.Unmarshal(body, &errResp) == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				return nil, fmt.Errorf("API error: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}

// ContractInfo looks up contract details by symbol.
func (c *Client) ContractInfo(ctx context.Context, symbol, exchange, secType string) (*models.ContractInfo, error) {
	path := "/v1/api/iserver/contract/" + url.PathEscape(symbol)
	if exchange != "" {
		path += "?exchange=" + url.QueryEscape(exchange)
		if secType != "" {
			path += "&secType=" + url.QueryEscape(secType)
		}
	}

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []models.ContractInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no contract found for symbol: %s", symbol)
	}
	return &result[0], nil
}

// ServiceStatus checks if specific services are available.
func (c *Client) ServiceStatus(ctx context.Context) ([]models.ServiceStatus, error) {
	resp, err := c.get(ctx, "/v1/api/iserver/services")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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