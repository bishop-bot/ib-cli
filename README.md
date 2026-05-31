# IB CLI

A CLI for Interactive Broker's Client Portal Web API written in Go.

## Features

- **Historical Market Data** - Query OHLCV bars with configurable intervals
- **Authentication** - Secure session management with encrypted persistence
- **Session Persistence** - Save/restore sessions with master password encryption
- **Watchlists** - List and retrieve user watchlists
- **Contract Lookup** - Find security contract details by symbol
- **Contract API** - Search by conid, all conids by exchange, contract info, trading schedules
- **Utility Scripts** - Batch fetch security definitions to CSV
- **Server Info** - Check gateway version and service status

## Prerequisites

- Go 1.21+
- [IB Client Portal Gateway](https://www.interactivebrokers.com/campus/trading-concepts/what-is-the-client-portal-gateway/) running locally (default: `https://127.0.0.1:5001`)

## Quick Start

```bash
# Build
go build -o ibcli .

# Copy config example
cp config.toml.example config.toml

# Authenticate (interactive)
./ibcli auth login

# Or with credentials
./ibcli auth login --username YOUR_USER --password YOUR_PASS

# Save session for later use
./ibcli auth login --username USER --password PASS --save

# Query historical data by symbol (auto-lookup contract ID)
./ibcli history AAPL --period 1d --bar 1min

# Query by contract ID directly
./ibcli history --conid 265598 --exchange SMART --period 1d --bar 1min

# Check status
./ibcli auth status
```

## Configuration

Copy `config.toml.example` to `config.toml` and customize:

```toml
[gateway]
host = "127.0.0.1"
port = 5001
use_tls = true
insecure_skip_verify = true

[output]
default_format = "json"
pretty = true
```

Environment variables override config values:
- `IB_GATEWAY_HOST`
- `IB_GATEWAY_PORT`
- `IB_GATEWAY_USE_TLS`
- `IB_AUTH_USERNAME`
- `IB_AUTH_PASSWORD`

## Commands

### `auth`
Manage authentication.

```bash
# Interactive login
./ibcli auth login

# Login with credentials
./ibcli auth login -u USER -p PASS

# Login and save encrypted session
./ibcli auth login -u USER -p PASS --save

# Restore saved session
./ibcli auth login --restore

# Check status
./ibcli auth status

# Logout
./ibcli auth logout
```

**Session Persistence**: Sessions are encrypted with AES-256-GCM using PBKDF2 key derivation from your master password.

### `history [symbol]`
Query historical market data.

```bash
# 1 day of 1-minute bars (lookup by symbol)
./ibcli history AAPL --period 1d --bar 1min

# By contract ID directly (faster, no lookup)
./ibcli history --conid 265598 --exchange SMART --period 1d --bar 1min

# 1 week of 5-minute bars
./ibcli history AAPL --period 1w --bar 5min

# Include data outside regular trading hours
./ibcli history AAPL --period 1w --bar 15min --outside-rth

# Output as CSV (header + data rows only)
./ibcli history AAPL --period 1d --bar 1min --format csv > data.csv

# Use with saved session
./ibcli auth login --restore
./ibcli history AAPL --period 1w --bar 1min
```

**Flags:**
- `-i, --conid` Contract ID (optional if symbol provided)
- `-e, --exchange` Exchange (default: SMART)
- `-p, --period` Duration (e.g., 1d, 1w, 1m)
- `-b, --bar` Bar size (e.g., 1min, 5min, 1h, 1d)
- `-o, --outside-rth` Include data outside regular trading hours
- `-S, --source` Data source (Trades, Midpoint, Bid_Ask)
- `-f, --format` Output format: json (default) or csv

### `watchlist`
Manage watchlists.

```bash
# List all watchlists
./ibcli watchlist list

# Get specific watchlist
./ibcli watchlist get <watchlist-id>
```

### `server`
Gateway server commands.

```bash
./ibcli server version
./ibcli server services
```

### `contract`
Contract and security information lookup commands.

```bash
# Look up contract by symbol
./ibcli contract lookup AAPL
./ibcli contract lookup ES --exchange SMART

# Search security definition by contract ID (GET /trsrv/secdef)
./ibcli contract secdef 265598
./ibcli contract secdef --symbol AAPL --type STOCK

# Get all contract IDs by exchange (GET /trsrv/all-conids)
./ibcli contract all-conids --exchange NASDAQ
./ibcli contract all-conids --exchange SMART --type STOCK

# Get detailed contract information by conid (GET /iserver/contract/{conid}/info)
./ibcli contract info 265598
./ibcli contract info 265598 --exchange NASDAQ

# Get trading schedule by symbol (GET /trsrv/secdef/schedule)
./ibcli contract schedule ES --exchange SMART --type FUT
./ibcli contract schedule AAPL

# Get trading schedule for exchange (GET /contract/trading-schedule)
./ibcli contract trading-schedule --exchange NASDAQ
```

## Utility Scripts

Standalone utility programs that consume the IB Gateway API.

### `secdef`
Batch fetch security definitions from conid JSON files and export to CSV.

```bash
# Build
go build -o secdef ./scripts/secdef.go

# Fetch all conids for an exchange
./secdef --exchange NYSE

# Limit to first 100 conids
./secdef --exchange ARCA --limit 100

# Custom output directory and workers
./secdef --exchange NASDAQ --output-dir ./data --workers 5

# Log errors to file
./secdef --exchange NYSE --error-log errors.csv
```

**Output:** `{exchange}_YYYYMMDD.csv` with columns:
`conid`, `ticker`, `currency`, `listingExchange`, `countryCode`, `name`, `assetClass`, `group`, `sector`, `sectorGroup`, `type`, `hasOptions`, `fullName`

**Flags:**
- `-e, --exchange` Exchange name (required)
- `-l, --limit` Limit number of conids (0 = all)
- `-o, --output-dir` Output directory (default: .)
- `--conid-dir` Directory with conid JSON files (default: assets/conid)
- `--error-log` File to log failed lookups
- `-w, --workers` Concurrent workers (default: 3)
- `--delay` Delay between requests (default: 300ms)
- `-c, --config` Config file path (default: config.toml)
- `-v, --verbose` Enable verbose output

### `secdef_instrument`
Map CSV output to secdef instrument format for trading systems.

```bash
# Build
go build -o secdef_instrument ./scripts/secdef_instrument.go

# Map NYSE CSV to instrument format
./secdef_instrument --exchange NYSE NYSE_20260524.csv

# Output: NYSE_20260524_mapped.csv
```

**Flags:**
- `-e, --exchange` Exchange value (required)

**Output:** `{input}_mapped.csv` with columns:
`id`, `symbol`, `name`, `publisher`, `instrument_class`, `currency`, `exchange`, `asset`, `security_type`, etc.

## Bar Sizes

| Size | Description |
|------|-------------|
| `1min` | 1 minute |
| `2min` | 2 minutes |
| `3min` | 3 minutes |
| `5min` | 5 minutes |
| `10min` | 10 minutes |
| `15min` | 15 minutes |
| `30min` | 30 minutes |
| `1h` | 1 hour |
| `2h`, `3h`, `4h`, `8h` | Hour bars |
| `1d` | Daily bar |
| `1w` | Weekly bar |
| `1m` | Monthly bar |

## Periods

| Period | Description |
|--------|-------------|
| `1-30min` | Minutes (e.g., 30min) |
| `1-8h` | Hours (e.g., 2h) |
| `1-1000d` | Days (e.g., 1d, 100d) |
| `1-792w` | Weeks (e.g., 1w, 4w) |
| `1-182m` | Months (e.g., 1m, 6m) |
| `1-15y` | Years (e.g., 1y) |

## Architecture

```
ibcli/
├── cmd/           # CLI tool commands (cobra)
│   ├── root.go    # Root command
│   ├── auth.go    # Authentication commands
│   ├── history.go # Historical data command
│   ├── watchlist.go # Watchlist commands
│   ├── contract.go # Contract lookup
│   └── server.go  # Server info commands
├── internal/
│   ├── api/       # HTTP client for IB Gateway API
│   ├── auth/      # Session encryption/decryption
│   ├── config/    # Configuration management
│   └── models/    # Data models
├── scripts/       # Utility programs (consume IB API)
│   ├── secdef.go     # Batch fetch security definitions to CSV
│   └── secdef_instrument.go # CSV column mapping utility
└── main.go
```

## Security

- **Session Encryption**: Sessions are encrypted with AES-256-GCM
- **Key Derivation**: Master passwords are processed with PBKDF2 (100,000 iterations)
- **No Plaintext Storage**: Credentials are never stored; only encrypted session tokens
- **Environment Variables**: Sensitive config via env vars, not hardcoded

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/auth -v
```

## License

MIT