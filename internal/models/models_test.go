package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHistoricalDataParams_Defaults(t *testing.T) {
	params := HistoricalDataParams{}

	if params.Symbol != "" {
		t.Errorf("Symbol = %v, want empty", params.Symbol)
	}
	if params.BarType != "" {
		t.Errorf("BarType = %v, want empty", params.BarType)
	}
}

func TestBar_JSONSerialization(t *testing.T) {
	bar := Bar{
		Time:   "2024-01-15T09:30:00-05:00",
		Open:   150.25,
		High:   151.00,
		Low:    149.50,
		Close:  150.75,
		Volume: 1000000,
		WAP:    150.50,
		Count:  5000,
	}

	data, err := json.Marshal(bar)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Bar
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Time != bar.Time {
		t.Errorf("Time = %v, want %v", decoded.Time, bar.Time)
	}
	if decoded.Open != bar.Open {
		t.Errorf("Open = %v, want %v", decoded.Open, bar.Open)
	}
	if decoded.Volume != bar.Volume {
		t.Errorf("Volume = %v, want %v", decoded.Volume, bar.Volume)
	}
}

func TestHistoricalDataResponse_JSONSerialization(t *testing.T) {
	resp := HistoricalDataResponse{
		Symbol:   "AAPL",
		ConID:    123456,
		Status:   "ok",
		Bars: []Bar{
			{Time: "2024-01-15T09:30:00", Open: 150.0, High: 151.0, Low: 149.0, Close: 150.5, Volume: 1000},
			{Time: "2024-01-15T09:31:00", Open: 150.5, High: 151.5, Low: 150.0, Close: 151.0, Volume: 1200},
		},
		Completed: true,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded HistoricalDataResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Symbol != resp.Symbol {
		t.Errorf("Symbol = %v, want %v", decoded.Symbol, resp.Symbol)
	}
	if len(decoded.Bars) != 2 {
		t.Errorf("len(Bars) = %v, want 2", len(decoded.Bars))
	}
	if decoded.Completed != true {
		t.Errorf("Completed = %v, want true", decoded.Completed)
	}
}

func TestAuthStatus_JSONParsing(t *testing.T) {
	// Simulate IB API response - fail is empty string when not failed
	jsonData := `{"authenticated":true,"connected":true,"fail":"","message":""}`

	var status AuthStatus
	if err := json.Unmarshal([]byte(jsonData), &status); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !status.IsAuthenticated() {
		t.Error("IsAuthenticated() = false, want true")
	}
	if !status.IsConnected() {
		t.Error("IsConnected() = false, want true")
	}
	if status.IsFailed() {
		t.Error("IsFailed() = true, want false")
	}
}

func TestAuthStatus_Methods(t *testing.T) {
	tests := []struct {
		name    string
		auth    AuthStatus
		wantAuth bool
		wantConn bool
		wantFail bool
	}{
		{
			name:    "authenticated",
			auth:    AuthStatus{Authenticated: true, Connected: true},
			wantAuth: true,
			wantConn: true,
			wantFail: false,
		},
		{
			name:    "not authenticated",
			auth:    AuthStatus{Authenticated: false, Connected: true},
			wantAuth: false,
			wantConn: true,
			wantFail: false,
		},
		{
			name:    "failed",
			auth:    AuthStatus{Authenticated: false, Connected: false, Fail: "auth error"},
			wantAuth: false,
			wantConn: false,
			wantFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.auth.IsAuthenticated() != tt.wantAuth {
				t.Errorf("IsAuthenticated() = %v, want %v", tt.auth.IsAuthenticated(), tt.wantAuth)
			}
			if tt.auth.IsConnected() != tt.wantConn {
				t.Errorf("IsConnected() = %v, want %v", tt.auth.IsConnected(), tt.wantConn)
			}
			if tt.auth.IsFailed() != tt.wantFail {
				t.Errorf("IsFailed() = %v, want %v", tt.auth.IsFailed(), tt.wantFail)
			}
		})
	}
}

func TestTokens_JSONSerialization(t *testing.T) {
	tokens := Tokens{
		SessionID: "abc123",
		Token:     "xyz789",
	}

	data, err := json.Marshal(tokens)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Tokens
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.SessionID != tokens.SessionID {
		t.Errorf("SessionID = %v, want %v", decoded.SessionID, tokens.SessionID)
	}
	if decoded.Token != tokens.Token {
		t.Errorf("Token = %v, want %v", decoded.Token, tokens.Token)
	}
}

func TestServerVersion_JSONSerialization(t *testing.T) {
	sv := ServerVersion{
		Version:    "v1.2.3",
		ServerTime: "2024-01-15T12:00:00Z",
	}

	data, err := json.Marshal(sv)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ServerVersion
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Version != sv.Version {
		t.Errorf("Version = %v, want %v", decoded.Version, sv.Version)
	}
}

func TestServiceStatus_JSONSerialization(t *testing.T) {
	ss := ServiceStatus{
		Service:    "marketdata",
		IsActive:   true,
		LastUpdate: "2024-01-15T12:00:00Z",
	}

	data, err := json.Marshal(ss)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ServiceStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Service != ss.Service {
		t.Errorf("Service = %v, want %v", decoded.Service, ss.Service)
	}
	if decoded.IsActive != ss.IsActive {
		t.Errorf("IsActive = %v, want %v", decoded.IsActive, ss.IsActive)
	}
}

func TestWatchlist_JSONSerialization(t *testing.T) {
	wl := Watchlist{
		ID:          "wl-123",
		Name:        "My Watchlist",
		DefaultList: true,
		Symbols: []WatchlistSymbol{
			{ConID: 123, Symbol: "AAPL", SecType: "STOCK"},
			{ConID: 456, Symbol: "GOOGL", SecType: "STOCK"},
		},
	}

	data, err := json.Marshal(wl)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Watchlist
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.ID != wl.ID {
		t.Errorf("ID = %v, want %v", decoded.ID, wl.ID)
	}
	if decoded.Name != wl.Name {
		t.Errorf("Name = %v, want %v", decoded.Name, wl.Name)
	}
	if decoded.DefaultList != wl.DefaultList {
		t.Errorf("DefaultList = %v, want %v", decoded.DefaultList, wl.DefaultList)
	}
	if len(decoded.Symbols) != 2 {
		t.Errorf("len(Symbols) = %v, want 2", len(decoded.Symbols))
	}
}

func TestWatchlistsResponse_JSONSerialization(t *testing.T) {
	resp := WatchlistsResponse{
		Watchlists: []Watchlist{
			{ID: "wl-1", Name: "Favorites"},
			{ID: "wl-2", Name: "Tech Stocks"},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded WatchlistsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(decoded.Watchlists) != 2 {
		t.Errorf("len(Watchlists) = %v, want 2", len(decoded.Watchlists))
	}
}

func TestSavedSession_JSONSerialization(t *testing.T) {
	now := time.Now()
	expires := now.Add(24 * time.Hour)

	ss := SavedSession{
		SessionID: "session-abc",
		Token:     "token-xyz",
		Username:  "testuser",
		SavedAt:   now,
		ExpiresAt: expires,
	}

	data, err := json.Marshal(ss)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded SavedSession
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.SessionID != ss.SessionID {
		t.Errorf("SessionID = %v, want %v", decoded.SessionID, ss.SessionID)
	}
	if decoded.Username != ss.Username {
		t.Errorf("Username = %v, want %v", decoded.Username, ss.Username)
	}
}

func TestContractInfo_JSONSerialization(t *testing.T) {
	ci := ContractInfo{
		ConID:       123456,
		Symbol:      "AAPL",
		SecType:     "STOCK",
		Currency:    "USD",
		Description: "Apple Inc.",
		Exchange:    "NASDAQ",
	}

	data, err := json.Marshal(ci)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ContractInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.ConID != ci.ConID {
		t.Errorf("ConID = %v, want %v", decoded.ConID, ci.ConID)
	}
	if decoded.Symbol != ci.Symbol {
		t.Errorf("Symbol = %v, want %v", decoded.Symbol, ci.Symbol)
	}
}