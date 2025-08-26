# Chi-Chi-Moni

A command-line tool for fetching financial account information using the SimpleFIN API. The tool resolves access tokens and retrieves account data including balances, transactions, and organization details.

## Features

- 🔐 Secure token-based authentication via SimpleFIN API
- 💰 Fetch account balances and transaction data
- 🏦 Display organization and account details
- 🚀 Fast, lightweight Go binary
- 🧪 Comprehensive test coverage (94.6%)
- 🔧 Cross-platform builds (Linux, macOS, Windows)

## Installation

### Prerequisites

- Go 1.25.0 or later
- Make (optional, for using Makefile targets)

### Building from Source

#### Using Make (Recommended)

```bash
# Build for current platform
make build

# Build for all platforms (Linux, macOS, Windows)
make build-all

# Development workflow (format, test, build)
make dev

# Show all available targets
make help
```

#### Manual Build

```bash
# Create bin directory and build
mkdir -p bin
go build -o bin/monies .
```

## Usage

### Command Line

```bash
./bin/monies <setup-token>
```

### Using Make

```bash
make run TOKEN="your-base64-setup-token"
```

### Examples

```bash
# Direct execution
./bin/monies "aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3JnL3NpbXBsZWZpbi9jbGFpbS8uLi4="

# Using make
make run TOKEN="aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3JnL3NpbXBsZWZpbi9jbGFpbS8uLi4="
```

## Output Format

The tool displays account information in a structured format:

```
Found 2 account(s):
1. Account: Checking Account
   ID: acc_123456
   Balance: 1,250.75 USD
   Organization: Example Bank
   Recent transactions: 15

2. Account: Savings Account
   ID: acc_789012
   Balance: 5,000.00 USD
   Organization: Example Bank
   Recent transactions: 3
```

## Error Handling

The application provides clear error messages for common issues:

- **Missing setup token**: Usage instructions are displayed
- **Invalid base64 token**: Base64 decoding error message
- **Network issues**: Connection error details
- **Authentication failures**: API authentication error
- **Invalid JSON responses**: JSON parsing error details

## Exit Codes

- `0`: Success - accounts retrieved and displayed
- `1`: Error - invalid arguments, API errors, or other failures

## Development

### Project Structure

```
├── main.go                   # Main application entry point
├── bin/                      # Built binaries
│   └── monies               # Main executable
├── api/                      # API client package
│   ├── access_token.go      # Token resolution logic
│   ├── client.go            # SimpleFIN HTTP client
│   ├── roundTripper.go      # HTTP transport with auth
│   └── *_test.go           # Comprehensive test suites
├── model/                    # Data models
│   └── account.go           # Financial data structures
├── Makefile                 # Build automation
└── go.mod                   # Go module definition
```

### Make Targets

The project includes a comprehensive Makefile with the following targets:

#### Build Targets
- `make build` - Build for current platform (creates `bin/monies`)
- `make build-all` - Cross-compile for multiple platforms
- `make clean` - Remove all build artifacts

#### Testing Targets
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report
- `make test-coverage-html` - Generate HTML coverage report
- `make test-race` - Run tests with race detection
- `make bench` - Run benchmark tests

#### Development Targets
- `make dev` - Development workflow (fmt + test + build)
- `make fmt` - Format code with `go fmt`
- `make lint` - Run linter (requires golangci-lint)
- `make security` - Run security analysis (requires gosec)

#### Dependency Management
- `make deps` - Install and tidy dependencies
- `make deps-verify` - Verify dependencies
- `make deps-update` - Update dependencies

#### Workflow Targets
- `make ci` - CI workflow (comprehensive checks)
- `make release` - Release workflow (build for all platforms)
- `make install` - Install binary to GOPATH/bin
- `make run TOKEN=<token>` - Run with setup token
- `make help` - Show all available targets

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Generate HTML coverage report
make test-coverage-html

# Run with race detection
make test-race
```

### Test Coverage

The project maintains high test coverage:
- **API package**: 94.6% statement coverage
- **29 test cases** covering success and error scenarios
- **Comprehensive mocking** for HTTP interactions
- **Race condition testing** for concurrent safety

### Code Quality

The project includes several code quality tools:

```bash
# Format code
make fmt

# Run linter (requires golangci-lint installation)
make lint

# Security analysis (requires gosec installation)
make security
```

## API Documentation

### Core Components

#### AccessTokenResolver
Resolves base64-encoded setup tokens into access credentials:
- Decodes base64 setup token
- Makes HTTP POST request to claim URL
- Parses response URL to extract credentials

#### SimpleFinClient
HTTP client for SimpleFIN API interactions:
- Handles authenticated requests
- Fetches account data
- Parses JSON responses into Go structs

#### SimpleFinRoundTripper
Custom HTTP transport for authentication:
- Injects Basic Auth headers
- Preserves request context
- Supports custom base transports

### Data Models

#### Account
Represents a financial account with:
- Basic info (ID, name, currency)
- Balance information
- Associated organization
- Transaction history
- Holdings data

#### Organization
Financial institution details:
- Institution name and domain
- SimpleFIN URLs
- Institution identifiers

#### Transaction
Individual transaction records:
- Transaction ID and timestamps
- Amount and description
- Payee and memo information

## Dependencies

- **Go 1.25.0+** - Core language runtime
- **github.com/stretchr/testify** - Testing framework for assertions

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/new-feature`)
3. Run tests (`make test`)
4. Format code (`make fmt`)
5. Run linter (`make lint`)
6. Commit changes (`git commit -am 'Add new feature'`)
7. Push to branch (`git push origin feature/new-feature`)
8. Create Pull Request

## License

This project is open source. Please check the repository for license details.

## Support

For issues and questions:
1. Check existing issues in the repository
2. Create a new issue with detailed information
3. Include error messages and steps to reproduce

---

**Note**: This tool requires valid SimpleFIN setup tokens. Contact your financial institution or SimpleFIN provider for access tokens.
