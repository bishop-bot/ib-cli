# IB-CLI

A production-grade CLI for Interactive Broker's Client Portal Web API.

## Features

- **Historical Market Data** - Query OHLCV bars with configurable intervals
- **Authentication** - Secure session management with encrypted persistence
- **Session Persistence** - Save/restore sessions with master password encryption
- **Watchlists** - List and retrieve user watchlists
- **Contract Lookup** - Find security contract details by symbol
- **Server Info** - Check gateway version and service status

## Prerequisites

- Go 1.21+
- [IB Client Portal Gateway](https://www.interactivebrokers.com/campus/trading-concepts/what-is-the-client-portal-gateway/) running locally (default: `http://127.0.0.1:5000`)

## Quick Start

```bash
# Build
go build -o ib-cli .

# Copy config example
cp config.toml.example config.toml

# Authenticate (interactive)
./ib-cli auth login

# Or with credentials
./ib-cli auth login --username YOUR_USER --password YOUR_PASS

# Save session for later use
./ib-cli auth login --username USER --password PASS --save

# Query historical data
./ib-cli history AAPL --bar 1 --unit min --duration "1 D"

# Check status
./ib-cli auth status
./ib-cli server version
```

## Configuration

Copy `config.toml.example` to `config.toml` and customize:

```toml
[gateway]
host = "127.0.0.1"
port = 5000
use_tls = false

[output]
default_format = "json"
pretty = true
```

Environment variables override config values:
- `IB_GATEWAY_HOST`
- `IB_GATEWAY_PORT`
- `IB_AUTH_USERNAME`
- `IB_AUTH_PASSWORD`

## Commands

### `auth`
Manage authentication.

```bash
# Interactive login
./ib-cli auth login

# Login with credentials
./ib-cli auth login -u USER -p PASS

# Login and save encrypted session
./ib-cli auth login -u USER -p PASS --save

# Restore saved session
./ib-cli auth login --restore

# Check status
./ib-cli auth status

# Logout
./ib-cli auth logout
```

**Session Persistence**: Sessions are encrypted with AES-256-GCM using PBKDF2 key derivation from your master password.

### `history [symbol]`
Query historical market data.

```bash
# 1 day of 1-minute bars
./ib-cli history AAPL --bar 1 --unit min --duration "1 D"

# Weekly bars for 1 month
./ib-cli history SPY --bar 1 --unit W --duration "1 M" -e SMART -t STOCK

# Specific time range
./ib-cli history AAPL --start 2024-01-01T00:00:00Z --end 2024-01-31T23:59:59Z

# Output as CSV
./ib-cli history AAPL --duration "5 D" --output csv > data.csv

# Use with saved session
./ib-cli auth login --restore
./ib-cli history AAPL -d "1 W"
```

### `watchlist`
Manage watchlists.

```bash
# List all watchlists
./ib-cli watchlist list

# List as table
./ib-cli watchlist list --format table

# Get specific watchlist
./ib-cli watchlist get <watchlist-id>
```

### `server`
Gateway server commands.

```bash
./ib-cli server version
./ib-cli server services
```

### `contract`
Contract lookup.

```bash
./ib-cli contract lookup AAPL
./ib-cli contract lookup ES --exchange SMART --type FUT
```

## Bar Types

| Type | Description |
|------|-------------|
| `TRADES` | Standard price/volume bars |
| `BID` | Bid prices |
| `ASK` | Ask prices |
| `MIDPOINT` | Midpoint of bid/ask |
| `SCHEDULE` | Trading schedule info |

## Bar Units

| Unit | Description |
|------|-------------|
| `S` | Seconds |
| `min` | Minutes |
| `D` | Days |
| `W` | Weeks |
| `M` | Months |
| `Y` | Years |

## Architecture

```
ib-cli/
├── cmd/           # CLI commands (cobra)
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