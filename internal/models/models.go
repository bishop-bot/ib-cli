package models

import "time"

// HistoricalDataParams holds parameters for historical data requests.
// See: https://www.interactivebrokers.com/campus/ibkr-api-page/cpapi-v1/#hist-md
type HistoricalDataParams struct {
	// ConID is the contract ID (required by IB API)
	ConID string `json:"-"`
	// Symbol is the security symbol (e.g., AAPL, ES) - used for lookup only
	Symbol string `json:"-"`
	// Exchange where the security is listed
	Exchange string `json:"-"`
	// Period is the overall duration for which data should be returned.
	// Format: {1-30}min, {1-8}h, {1-1000}d, {1-792}w, {1-182}m, {1-15}y
	Period string `json:"-"`
	// Bar is the individual bar size/interval.
	// Possible values: 1min, 2min, 3min, 5min, 10min, 15min, 30min, 1h, 2h, 3h, 4h, 8h, 1d, 1w, 1m
	Bar string `json:"-"`
	// StartTime is the starting date of the request duration (YYYYMMDD-HH:mm:ss format)
	StartTime string `json:"-"`
	// EndTime is the ending date of the request duration
	EndTime string `json:"-"`
	// OutsideRTH determines if you want data after regular trading hours
	OutsideRTH bool `json:"-"`
	// Source is the type of data to be returned: Trades, Midpoint, Bid_Ask
	Source string `json:"-"`
}

// SetConIDFromSymbol resolves the conid from the symbol (called before API request)
func (p *HistoricalDataParams) SetConIDFromSymbol(conid string) {
	p.ConID = conid
}

// Bar represents a single OHLCV bar from historical data.
// Per IB API: data array contains objects with o, c, h, l, v, t fields.
type Bar struct {
	Open  float64 `json:"o"`
	Close float64 `json:"c"`
	High  float64 `json:"h"`
	Low   float64 `json:"l"`
	Volume float64 `json:"v"`  // Volume factor: volume = actual / 100
	Timestamp int64  `json:"t"`  // Epoch time in milliseconds
}

// HistoricalDataResponse is the API response for historical data.
// See: https://www.interactivebrokers.com/campus/ibkr-api-page/cpapi-v1/#hist-md
type HistoricalDataResponse struct {
	ServerID         string `json:"serverId"`
	Symbol           string `json:"symbol"`
	Text            string `json:"text"`
	PriceFactor     int    `json:"priceFactor"`
	StartTime       string `json:"startTime"`
	TimePeriod      string `json:"timePeriod"`
	BarLength       int    `json:"barLength"`
	MDAvailability  string `json:"mdAvailability"`
	MktDataDelay    int    `json:"mktDataDelay"`
	OutsideRTH      bool   `json:"outsideRth"`
	VolumeFactor    int    `json:"volumeFactor"`
	PriceDisplayRule int    `json:"priceDisplayRule"`
	PriceDisplayValue string `json:"priceDisplayValue"`
	NegativeCapable bool   `json:"negativeCapable"`
	MessageVersion  int    `json:"messageVersion"`
	Data            []Bar `json:"data"`
	Points          int   `json:"points"`
	TravelTime      int   `json:"travelTime"`
	Error           string `json:"error,omitempty"`
}

// ServerVersion represents the gateway server version info.
type ServerVersion struct {
	Version    string `json:"version"`
	ServerTime string `json:"serverTime"`
}

// AuthStatus represents the current authentication state.
// Note: IB API uses 'authenticated', 'connected', 'fail' (string) not isAuthenticated.
type AuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	Connected     bool   `json:"connected"`
	Fail          string `json:"fail"`           // Empty string = not failed, non-empty = error message
	Message       string `json:"message,omitempty"`
	Tokens        Tokens `json:"tokens,omitempty"`
}

// IsAuthenticated returns true if authenticated.
func (a *AuthStatus) IsAuthenticated() bool {
	return a.Authenticated
}

// IsConnected returns true if connected.
func (a *AuthStatus) IsConnected() bool {
	return a.Connected
}

// IsFailed returns true if authentication failed (fail field is non-empty).
func (a *AuthStatus) IsFailed() bool {
	return a.Fail != ""
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

// SecDefSearchParams holds parameters for security definition search.
// See: https://www.interactivebrokers.com/campus/ibkr-api-page/cpapi-v1/#trsrv-conid-contract
type SecDefSearchParams struct {
	ConID   string `json:"-"` // Contract ID to search
	Symbol  string `json:"-"`
	SecType string `json:"-"`
	Exchange string `json:"-"`
}

// SecDefInfo represents security definition info from secdef endpoint.
type SecDefInfo struct {
	ConID       int    `json:"conid"`
	Symbol      string `json:"symbol"`
	SecType     string `json:"secType"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	Exchange    string `json:"exchange"`
	Category    string `json:"category,omitempty"`
}

// AllConidsParams holds parameters for all-conids request.
type AllConidsParams struct {
	Exchange string `json:"-"`
	SecType   string `json:"-"`
}

// ConidInfo represents contract info from /iserver/contract/{conid}/info endpoint.
type ConidInfo struct {
	ConID          int    `json:"conid"`
	Symbol         string `json:"symbol"`
	SecType        string `json:"secType"`
	Currency       string `json:"currency"`
	Description    string `json:"description"`
	Exchange       string `json:"exchange"`
	PrimaryExchange string `json:"primaryExchange,omitempty"`
	ContractType   string `json:"contractType,omitempty"`
	Category       string `json:"category,omitempty"`
	SubCategory    string `json:"subCategory,omitempty"`
	TickSize       float64 `json:"tickSize,omitempty"`
	MinSize        float64 `json:"minSize,omitempty"`
	MaxSize        float64 `json:"maxSize,omitempty"`
	SizeIncrement  float64 `json:"sizeIncrement,omitempty"`
	MarketDataAvailable bool `json:"marketDataAvailable,omitempty"`
}

// TradingScheduleParams holds parameters for trading schedule requests.
type TradingScheduleParams struct {
	Symbol   string `json:"-"`
	Exchange string `json:"-"`
	SecType  string `json:"-"`
	Expiry   string `json:"-"`
}

// TradingScheduleDay represents a single day's trading schedule.
type TradingScheduleDay struct {
	Date      string `json:"date"`
	Open      string `json:"open"`
	Close     string `json:"close"`
	IsHoliday bool   `json:"isHoliday,omitempty"`
	IsEarlyClose bool `json:"isEarlyClose,omitempty"`
}

// TradingSchedule represents trading schedule response.
type TradingSchedule struct {
	Exchange string `json:"exchange"`
	Days     []TradingScheduleDay `json:"days"`
}

// SavedSession holds encrypted session data for persistence.
type SavedSession struct {
	SessionID   string    `json:"sessionId"`
	Token       string    `json:"token"`
	Username    string    `json:"username"`
	SavedAt     time.Time `json:"savedAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
}