# Chi-Chi-Moni

An automated financial data synchronization service that fetches account information from the SimpleFIN API and stores it in a local SQLite database. The service uses AWS SSO for authentication and AWS Secrets Manager for secure credential storage.

## Architecture Overview

Chi-Chi-Moni is designed as an automated service that:
- **Authenticates** via AWS SSO to access AWS services
- **Retrieves** SimpleFIN API credentials from AWS Secrets Manager
- **Fetches** financial account data from SimpleFIN API
- **Persists** account information and balance history in SQLite database

### System Components

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────┐
│   AWS SSO       │────▶│ AWS Secrets  │────▶│  SimpleFIN   │
│ Authentication  │     │   Manager    │     │     API      │
└─────────────────┘     └──────────────┘     └──────────────┘
         │                      │                     │
         └──────────────────────┼─────────────────────┘
                                │
                         ┌──────▼──────┐
                         │   Service   │
                         │    Core     │
                         └──────┬──────┘
                                │
                         ┌──────▼──────┐
                         │   SQLite    │
                         │  Database   │
                         └─────────────┘
```

## Features

- 🔐 **AWS SSO Integration**: Secure authentication for AWS services access
- 🔑 **Secrets Manager**: Encrypted credential storage and retrieval
- 💰 **Financial Data Sync**: Automated account balance and transaction fetching
- 🗄️ **SQLite Persistence**: Local database for historical data tracking
- 📊 **Balance History**: Track account balance changes over time with job UUIDs
- 🧪 **Test Coverage**: Comprehensive test suites for all components
- 🚀 **Go Performance**: Fast, concurrent processing with minimal resource usage

## Installation

### Prerequisites

- Go 1.22.0 or later
- AWS CLI configured with SSO profile
- AWS account with appropriate permissions
- SimpleFIN API access token
- SQLite3

### AWS Prerequisites

1. **AWS SSO Configuration**:
   ```bash
   aws configure sso
   # Follow prompts to set up SSO profile named "monkstorage"
   ```

2. **AWS IAM Permissions**:
   - `secretsmanager:GetSecretValue` for accessing stored credentials
   - `sso:GetRoleCredentials` for SSO authentication
   - `sso-oidc:CreateToken` for device authorization flow

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/chi-chi-moni.git
cd chi-chi-moni

# Install dependencies
go mod download

# Build the service
go build -o bin/chi-chi-moni .

# Or use Make
make build
```

## Configuration

### Environment Variables

The service uses hardcoded configuration constants that can be modified in `main.go`:

```go
const ssoProfile = "monkstorage"           // AWS SSO profile name
const accessTokenSecretName = "monk-monies" // Secrets Manager secret name
const dbFilePath = "data/monk.db"          // SQLite database path
```

### AWS Secrets Manager Setup

1. Store your SimpleFIN access token in AWS Secrets Manager:
   ```bash
   aws secretsmanager create-secret \
     --name monk-monies \
     --secret-string '{"scheme":"https","host":"bridge.simplefin.org","accessToken":"YOUR_TOKEN"}'
   ```

2. The secret should contain a JSON object with:
   - `scheme`: API scheme (https)
   - `host`: SimpleFIN API host
   - `accessToken`: Your SimpleFIN access token

### Database Configuration

The SQLite database is automatically created at `~/data/monk.db` with the following schema:

```sql
-- Bank accounts table
CREATE TABLE IF NOT EXISTS bank_accounts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    currency TEXT,
    balance REAL,
    available_balance REAL,
    balance_date INTEGER
);

-- Account balances history table
CREATE TABLE IF NOT EXISTS account_balances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id TEXT NOT NULL,
    job_id TEXT NOT NULL,
    balance REAL NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (account_id) REFERENCES bank_accounts(id)
);
```

## Usage

### Running the Service

```bash
# Direct execution
./bin/chi-chi-moni

# Or if installed globally
chi-chi-moni
```

The service will:
1. Authenticate with AWS SSO
2. Retrieve SimpleFIN credentials from Secrets Manager
3. Fetch all accounts from SimpleFIN API
4. Store/update account information in SQLite
5. Record balance history with a unique job UUID

### Scheduling Automated Runs

For continuous monitoring, schedule the service using cron:

```bash
# Run every hour
0 * * * * /path/to/chi-chi-moni >> /var/log/chi-chi-moni.log 2>&1

# Run every day at 9 AM
0 9 * * * /path/to/chi-chi-moni >> /var/log/chi-chi-moni.log 2>&1
```

Or use systemd timer for more control:

```ini
# /etc/systemd/system/chi-chi-moni.service
[Unit]
Description=Chi-Chi-Moni Financial Sync Service
After=network.target

[Service]
Type=oneshot
User=your-user
ExecStart=/path/to/chi-chi-moni
StandardOutput=journal
StandardError=journal

# /etc/systemd/system/chi-chi-moni.timer
[Unit]
Description=Run Chi-Chi-Moni hourly
Requires=chi-chi-moni.service

[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
```

## Project Structure

```
chi-chi-moni/
├── main.go                   # Service entry point and orchestration
├── api/                      # SimpleFIN API client package
│   ├── client.go            # HTTP client implementation
│   ├── client_test.go       # Client unit tests
│   ├── access_token.go      # Token resolution logic
│   ├── access_token_test.go # Token tests
│   ├── roundTripper.go      # HTTP transport with auth
│   └── roundTripper_test.go # Transport tests
├── aws/                      # AWS service integrations
│   ├── sso_client.go        # SSO authentication client
│   ├── sso_client_test.go   # SSO client tests
│   ├── secrets_manager.go   # Secrets Manager client
│   └── secrets_manager_test.go # Secrets tests
├── db/                       # Database package
│   └── client.go            # SQLite client and operations
├── model/                    # Data models
│   └── account.go           # Account, transaction, and balance structs
├── go.mod                   # Go module definition
├── go.sum                   # Dependency checksums
└── Makefile                 # Build automation

Generated files:
├── bin/                     # Compiled binaries
│   └── chi-chi-moni        # Main executable
└── ~/data/                  # User data directory
    └── monk.db             # SQLite database
```

## Package Documentation

### `api` Package
Handles all SimpleFIN API interactions:
- **AccessTokenResolver**: Resolves setup tokens to access credentials
- **SimpleFINClient**: HTTP client for API requests
- **SimpleFINRoundTripper**: Custom transport for authentication

### `aws` Package
Manages AWS service integrations:
- **SSOClient**: Handles AWS SSO authentication and token refresh
- **SecretsManagerClient**: Retrieves and manages secrets

### `db` Package
Database operations and management:
- **DatabaseClient**: SQLite connection and query execution
- **Schema migrations**: Automatic database setup
- **Transaction management**: Atomic operations for data consistency

### `model` Package
Data structures for financial information:
- **Account**: Bank account representation
- **Organization**: Financial institution details
- **Transaction**: Individual transaction records
- **Balance**: Point-in-time balance information

## Authentication Flow

### AWS SSO Authentication

1. **Device Authorization**: Service initiates device authorization flow
2. **Token Exchange**: Exchanges device code for access tokens
3. **Role Credentials**: Retrieves temporary AWS credentials
4. **Auto-refresh**: Handles token expiration and refresh

### SimpleFIN Authentication

1. **Secret Retrieval**: Fetches access token from Secrets Manager
2. **Header Injection**: Adds Basic Auth to API requests
3. **Request Execution**: Makes authenticated API calls

## Database Operations

### Account Management

```go
// Check if account exists
exists, err := dbClient.DoesBankAccountExist(accountID)

// Create or update account
err := dbClient.PutBankAccount(account)

// Record balance history
err := dbClient.PutAccountBalance(accountID, jobID, balance)
```

### Data Integrity

- Foreign key constraints ensure referential integrity
- Transactions for atomic operations
- Unique job IDs for tracking sync runs
- Timestamps for historical analysis

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./api
go test ./aws
go test ./db

# Run with race detection
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Coverage Goals

- **api** package: >90% coverage
- **aws** package: >90% coverage  
- **db** package: >90% coverage
- **model** package: >90% coverage
- **main** package: >90% coverage

### Test Structure

Each package includes comprehensive test suites:
- Unit tests for individual functions
- Integration tests for component interactions
- Mock implementations for external dependencies
- Error scenario coverage
- Concurrent access testing

## Dependencies

### Core Dependencies

```go
// Production dependencies
github.com/aws/aws-sdk-go-v2/config
github.com/aws/aws-sdk-go-v2/service/secretsmanager
github.com/aws/aws-sdk-go-v2/service/sso
github.com/aws/aws-sdk-go-v2/service/ssooidc
github.com/google/uuid
github.com/mattn/go-sqlite3

// Test dependencies
github.com/stretchr/testify
```

### Dependency Management

```bash
# Install dependencies
go mod download

# Update dependencies
go get -u ./...

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify
```

## Error Handling

The service implements comprehensive error handling:

### Fatal Errors (Service Termination)
- AWS SSO authentication failure
- Secrets Manager access denied
- Database initialization failure
- SimpleFIN API authentication failure

### Recoverable Errors (Logged)
- Transient network issues
- Rate limiting responses
- Partial data updates

### Error Messages

Common error scenarios and solutions:

| Error | Cause | Solution |
|-------|-------|----------|
| `SSO session expired` | AWS SSO token expired | Run `aws sso login` |
| `Secret not found` | Missing Secrets Manager entry | Create secret in AWS |
| `Database locked` | Concurrent access | Implement retry logic |
| `API rate limited` | Too many requests | Add backoff strategy |

## Troubleshooting

### AWS SSO Issues

```bash
# Verify SSO configuration
aws sso login --profile monkstorage
aws sts get-caller-identity --profile monkstorage

# Check SSO cache
ls ~/.aws/sso/cache/

# Clear SSO cache if needed
rm -rf ~/.aws/sso/cache/*
```

### Secrets Manager Access

```bash
# Test secret access
aws secretsmanager get-secret-value \
  --secret-id monk-monies \
  --profile monkstorage

# Verify IAM permissions
aws iam get-role-policy --role-name YourRole
```

### Database Issues

```bash
# Check database integrity
sqlite3 ~/data/monk.db "PRAGMA integrity_check;"

# View database schema
sqlite3 ~/data/monk.db ".schema"

# Query recent balances
sqlite3 ~/data/monk.db "SELECT * FROM account_balances ORDER BY created_at DESC LIMIT 10;"
```

### SimpleFIN API Issues

```bash
# Test API connectivity
curl -u "YOUR_ACCESS_TOKEN:" https://bridge.simplefin.org/simplefin/accounts

# Check rate limits in response headers
curl -I -u "YOUR_ACCESS_TOKEN:" https://bridge.simplefin.org/simplefin/accounts
```

## Security Considerations

### Credential Security
- ✅ No hardcoded credentials in source code
- ✅ AWS Secrets Manager for sensitive data
- ✅ SSO for temporary AWS credentials
- ✅ Basic Auth only over HTTPS

### Database Security
- ✅ Local SQLite file with filesystem permissions
- ✅ No network exposure
- ✅ Parameterized queries prevent SQL injection

### Best Practices
- Regular rotation of SimpleFIN access tokens
- Audit AWS access logs
- Restrict IAM permissions to minimum required
- Keep dependencies updated
- Use read-only SimpleFIN tokens when possible

## Performance Optimization

### Concurrency
- Goroutines for parallel account processing
- Connection pooling for database operations
- HTTP client reuse for API calls

### Caching
- AWS credentials cached until expiration
- Database connections persisted
- SimpleFIN client instance reused

### Resource Usage
- Minimal memory footprint (~10MB)
- Single SQLite file for all data
- Efficient batch processing

## Monitoring and Logging

### Logging Strategy

```go
// Current implementation uses log.Fatal for errors
// Consider implementing structured logging:

import "github.com/sirupsen/logrus"

log.WithFields(logrus.Fields{
    "account_id": account.ID,
    "job_id": jobUuid.String(),
}).Info("Processing account")
```

### Metrics to Monitor
- Sync duration
- Number of accounts processed
- Balance change alerts
- Error rates
- API response times

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/improvement`)
3. Make your changes
4. Add/update tests
5. Run tests (`go test ./...`)
6. Commit changes (`git commit -am 'Add feature'`)
7. Push branch (`git push origin feature/improvement`)
8. Create Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Maintain test coverage above 90%
- Document exported functions
- Use meaningful commit messages
- Update README for new features

## Roadmap

### Planned Features
- [ ] Transaction history synchronization
- [ ] Multiple account provider support
- [ ] Web dashboard for visualization
- [ ] Alerting for balance changes
- [ ] Export to common formats (CSV, JSON)
- [ ] Kubernetes deployment manifests
- [ ] Prometheus metrics endpoint
- [ ] GraphQL API for data access

### Future Improvements
- [ ] Plugin architecture for providers
- [ ] Multi-currency support
- [ ] Budget tracking features
- [ ] Category management
- [ ] Recurring transaction detection

## License

This project is open source. Please check the repository for license details.

## Support

For issues and questions:
1. Check existing GitHub issues
2. Review troubleshooting section
3. Create detailed issue with:
   - Error messages
   - Log outputs
   - Environment details
   - Steps to reproduce

## Acknowledgments

- SimpleFIN for providing the financial data API
- AWS SDK Go v2 contributors
- Go SQLite3 driver maintainers
- Open source community

---

**Note**: This service requires valid AWS credentials and SimpleFIN API access. Ensure proper security measures are in place when handling financial data.