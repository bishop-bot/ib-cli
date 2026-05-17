package models

import "time"

// HistoricalDataParams holds parameters for historical data requests.
// See: https://www.interactivebrokers.com/campus/ibkr-api-page/cpapi-v1/#hist-md
type HistoricalDataParams struct {
	// ConID is the contract ID (leave empty to look up by symbol)
	ConID string `long:"conid" description:"Contract ID (conId)"`
	// Symbol is the security symbol (e.g., AAPL, ES)
	Symbol string `long:"symbol" short:"s" description:"Security symbol" required:"true"`
	// Exchange where the security is listed
	Exchange string `long:"exchange" short:"e" description:"Exchange (e.g., SMART, NYSE)"`
	// SecType is the security type
	SecType string `long:"type" short:"t" description:"Security type: STOCK, OPT, FUT, etc."`
	// Exchange for the exchange to route orders to
	ExchangeRoute string `long:"exchange-route" description:"Exchange for routing"`
	// StartTime is the query start time (RFC3339 format)
	StartTime string `long:"start" short:"S" description:"Start time (RFC3339)"`
	// EndTime is the query end time (RFC3339 format)
	EndTime string `long:"end" short:"E" description:"End time (RFC3339)"`
	// BarSize is the bar duration (e.g., 1, 5, 15, 1h, 1d)
	BarSize string `long:"bar-size" short:"b" description:"Bar size (e.g., 1, 5, 15, 1h, 1d)"`
	// BarUnit is the bar unit (S, D, W, M, Y)
	BarUnit string `long:"bar-unit" short:"u" description:"Bar unit (S, min, D, W, M, Y)"`
	// BarType is the type of data (TRADES, BID, ASK, MIDPOINT, SCHEDULE)
	BarType string `long:"bar-type" short:"m" description:"Bar type (TRADES, BID, ASK, MIDPOINT, SCHEDULE)"`
	// Duration is the total time to query (e.g., "1 D", "1 W", "1 M")
	Duration string `long:"duration" short:"d" description:"Duration (e.g., 1 D, 1 W, 1 M)"`
	// OutsideRTH includes data outside regular trading hours
	OutsideRTH bool `long:"outside-rth" description:"Include data outside regular trading hours"`
	// FormatDate controls date formatting in response
	FormatDate int `long:"format-date" description:"Date format (1=unix epoch, 2=RFC3339)"`
	// UseRTH restricts to regular trading hours only
	UseRTH bool `long:"use-rth" description:"Use regular trading hours only"`
	// Limit the number of bars returned
	Limit int `long:"limit" short:"l" description:"Maximum number of bars"`
	// OverrideSpacing allows non-standard bar spacing
	OverrideSpacing bool `long:"override-spacing" description:"Override default spacing"`
}

// Bar represents a single OHLCV bar from historical data.
type Bar struct {
	Time    string  `json:"time"`
	Open    float64 `json:"open"`
	High    float64 `json:"high"`
	Low     float64 `json:"low"`
	Close   float64 `json:"close"`
	Volume  int64   `json:"volume"`
	WAP     float64 `json:"wap,omitempty"`
	Count   int     `json:"count,omitempty"`
}

// HistoricalDataResponse is the API response for historical data.
type HistoricalDataResponse struct {
	Symbol        string    `json:"symbol"`
	ConID         int       `json:"conId,omitempty"`
	StartTime     time.Time `json:"startTime,omitempty"`
	EndTime       time.Time `json:"endTime,omitempty"`
	Status        string    `json:"status"`
	Bars          []Bar     `json:"bars"`
	Error         string    `json:"error,omitempty"`
	LastDuration  string    `json:"lastDuration,omitempty"`
	Completed     bool      `json:"completed"`
}

// ServerVersion represents the gateway server version info.
type ServerVersion struct {
	Version    string `json:"version"`
	ServerTime string `json:"serverTime"`
}

// AuthStatus represents the current authentication state.
type AuthStatus struct {
	IsAuthenticated bool   `json:"isAuthenticated"`
	IsConnected     bool   `json:"isConnected"`
	IsFailed        bool   `json:"isFailed"`
	ErrorMessage    string `json:"errorMsg,omitempty"`
	Tokens          Tokens `json:"tokens,omitempty"`
}

// Tokens holds authentication tokens from the gateway.
type Tokens struct {
	SessionID string `json:"sessionId"`
	Token     string `json:"token"`
}

// ContractInfo holds contract/security information.
type ContractInfo struct {
	ConID       int    `json:"conId"`
	Symbol      string `json:"symbol"`
	SecType     string `json:"secType"`
	Currency    string `json:"currency"`
	Description string `json:"description,omitempty"`
	Exchange    string `json:"exchange,omitempty"`
}

// ServiceStatus represents the health status of a service.
type ServiceStatus struct {
	Service    string `json:"service"`
	IsActive   bool   `json:"isActive"`
	LastUpdate string `json:"lastUpdate,omitempty"`
}

// Watchlist represents a user's watchlist.
type Watchlist struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	DefaultList bool        `json:"defaultList,omitempty"`
	Symbols     []WatchlistSymbol `json:"symbols,omitempty"`
}

// WatchlistSymbol represents a symbol in a watchlist.
type WatchlistSymbol struct {
	ConID   int    `json:"conId"`
	Symbol  string `json:"symbol"`
	SecType string `json:"secType"`
}

// WatchlistsResponse is the API response for watchlists.
type WatchlistsResponse struct {
	Watchlists []Watchlist `json:"watchlists"`
}

// SavedSession holds encrypted session data for persistence.
type SavedSession struct {
	SessionID   string    `json:"sessionId"`
	Token       string    `json:"token"`
	Username    string    `json:"username"`
	SavedAt     time.Time `json:"savedAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
}