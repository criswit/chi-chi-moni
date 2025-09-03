package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/criswit/chi-chi-moni/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions
func setupTestDB(t *testing.T) *DatabaseClient {
	t.Helper()
	
	// Create in-memory SQLite database
	db, err := sqlx.Connect("sqlite3", ":memory:")
	require.NoError(t, err, "Failed to create in-memory database")
	
	// Create schema
	schema := `
	CREATE TABLE IF NOT EXISTS BANK_ACCOUNT (
		ID TEXT PRIMARY KEY,
		NAME TEXT NOT NULL,
		INSTITUTION_NAME TEXT NOT NULL
	);
	
	CREATE TABLE IF NOT EXISTS BANK_ACCOUNT_BALANCE (
		ID TEXT,
		BANK_ACCOUNT_ID TEXT,
		RUN_ID TEXT NOT NULL,
		BALANCE TEXT NOT NULL,
		CREATED_AT TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(BANK_ACCOUNT_ID) REFERENCES BANK_ACCOUNT(ID)
	);
	
	CREATE INDEX IF NOT EXISTS idx_bank_account_balance_account_id ON BANK_ACCOUNT_BALANCE(BANK_ACCOUNT_ID);
	CREATE INDEX IF NOT EXISTS idx_bank_account_balance_run_id ON BANK_ACCOUNT_BALANCE(RUN_ID);
	`
	
	_, err = db.Exec(schema)
	require.NoError(t, err, "Failed to create schema")
	
	return &DatabaseClient{db: db}
}

func seedTestData(t *testing.T, client *DatabaseClient) {
	t.Helper()
	
	testAccounts := []model.Account{
		{
			ID:   "test_account_1",
			Name: "Checking Account",
			Org: model.Organization{
				Name:   "Test Bank",
				Domain: "testbank.com",
				ID:     "test_bank_1",
			},
		},
		{
			ID:   "test_account_2",
			Name: "Savings Account",
			Org: model.Organization{
				Name:   "Test Bank",
				Domain: "testbank.com",
				ID:     "test_bank_1",
			},
		},
	}
	
	for _, account := range testAccounts {
		err := client.PutBankAccount(account)
		require.NoError(t, err, "Failed to seed test account")
	}
}

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *DatabaseClient) {
	t.Helper()
	
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err, "Failed to create mock database")
	
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	client := &DatabaseClient{db: sqlxDB}
	
	return mockDB, mock, client
}

// TestNewDatabaseClient tests the constructor with various parameters
func TestNewDatabaseClient(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantErr     bool
		errContains string
		setup       func(t *testing.T) string
		cleanup     func(t *testing.T, path string)
	}{
		{
			name:    "valid_in_memory_database",
			path:    ":memory:",
			wantErr: false,
			setup: func(t *testing.T) string {
				return ":memory:"
			},
			cleanup: func(t *testing.T, path string) {},
		},
		{
			name:    "valid_file_database",
			path:    "",
			wantErr: false,
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "test.db")
			},
			cleanup: func(t *testing.T, path string) {
				os.Remove(path)
			},
		},
		{
			name:        "invalid_database_path",
			path:        "/invalid/path/to/database.db",
			wantErr:     true,
			errContains: "unable to open database file",
			setup: func(t *testing.T) string {
				return "/invalid/path/to/database.db"
			},
			cleanup: func(t *testing.T, path string) {},
		},
		{
			name:        "permission_denied",
			path:        "",
			wantErr:     true,
			errContains: "permission denied",
			setup: func(t *testing.T) string {
				if os.Getuid() == 0 {
					t.Skip("Cannot test permission denied as root")
				}
				tmpDir := t.TempDir()
				dbPath := filepath.Join(tmpDir, "test.db")
				os.Chmod(tmpDir, 0000)
				return dbPath
			},
			cleanup: func(t *testing.T, path string) {
				dir := filepath.Dir(path)
				os.Chmod(dir, 0755)
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.setup(t)
			defer tt.cleanup(t, dbPath)
			
			client, err := NewDatabaseClient(dbPath)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotNil(t, client.db)
				if client != nil {
					client.Close()
				}
			}
		})
	}
}

// TestDatabaseClientClose tests the Close method
func TestDatabaseClientClose(t *testing.T) {
	client := setupTestDB(t)
	require.NotNil(t, client)
	
	// Verify database is accessible before close
	err := client.db.Ping()
	assert.NoError(t, err, "Database should be accessible before close")
	
	// Close the database
	client.Close()
	
	// Verify database is not accessible after close
	err = client.db.Ping()
	assert.Error(t, err, "Database should not be accessible after close")
}

// TestPutBankAccount tests bank account creation
func TestPutBankAccount(t *testing.T) {
	tests := []struct {
		name    string
		account model.Account
		wantErr bool
		setup   func(t *testing.T, client *DatabaseClient)
	}{
		{
			name: "create_new_account",
			account: model.Account{
				ID:   "new_account_1",
				Name: "New Test Account",
				Org: model.Organization{
					Name:   "New Bank",
					Domain: "newbank.com",
					ID:     "new_bank_1",
				},
			},
			wantErr: false,
			setup:   func(t *testing.T, client *DatabaseClient) {},
		},
		{
			name: "create_account_with_special_characters",
			account: model.Account{
				ID:   "special_account",
				Name: "Account's \"Special\" Name & Co.",
				Org: model.Organization{
					Name:   "Bank & Trust Co.",
					Domain: "bank-trust.com",
					ID:     "special_bank",
				},
			},
			wantErr: false,
			setup:   func(t *testing.T, client *DatabaseClient) {},
		},
		{
			name: "create_duplicate_account",
			account: model.Account{
				ID:   "duplicate_account",
				Name: "Duplicate Account",
				Org: model.Organization{
					Name:   "Test Bank",
					Domain: "testbank.com",
					ID:     "test_bank",
				},
			},
			wantErr: true,
			setup: func(t *testing.T, client *DatabaseClient) {
				// Pre-create the account to cause duplicate
				err := client.PutBankAccount(model.Account{
					ID:   "duplicate_account",
					Name: "Original Account",
					Org: model.Organization{
						Name: "Original Bank",
					},
				})
				require.NoError(t, err)
			},
		},
		{
			name: "create_account_with_empty_id",
			account: model.Account{
				ID:   "",
				Name: "Account without ID",
				Org: model.Organization{
					Name: "Test Bank",
				},
			},
			wantErr: false, // SQLite allows empty string as primary key
			setup:   func(t *testing.T, client *DatabaseClient) {},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := setupTestDB(t)
			defer client.Close()
			
			tt.setup(t, client)
			
			err := client.PutBankAccount(tt.account)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Verify account was created
				exists, err := client.DoesBankAccountExist(tt.account.ID)
				assert.NoError(t, err)
				assert.True(t, exists)
			}
		})
	}
}

// TestDoesBankAccountExist tests account existence checking
func TestDoesBankAccountExist(t *testing.T) {
	tests := []struct {
		name      string
		accountID string
		want      bool
		wantErr   bool
		setup     func(t *testing.T, client *DatabaseClient)
	}{
		{
			name:      "existing_account",
			accountID: "existing_account",
			want:      true,
			wantErr:   false,
			setup: func(t *testing.T, client *DatabaseClient) {
				err := client.PutBankAccount(model.Account{
					ID:   "existing_account",
					Name: "Existing Account",
					Org:  model.Organization{Name: "Test Bank"},
				})
				require.NoError(t, err)
			},
		},
		{
			name:      "non_existing_account",
			accountID: "non_existing_account",
			want:      false,
			wantErr:   false,
			setup:     func(t *testing.T, client *DatabaseClient) {},
		},
		{
			name:      "empty_account_id",
			accountID: "",
			want:      false,
			wantErr:   false,
			setup:     func(t *testing.T, client *DatabaseClient) {},
		},
		{
			name:      "account_with_special_characters",
			accountID: "account'with\"special&chars",
			want:      false,
			wantErr:   false,
			setup:     func(t *testing.T, client *DatabaseClient) {},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := setupTestDB(t)
			defer client.Close()
			
			tt.setup(t, client)
			
			exists, err := client.DoesBankAccountExist(tt.accountID)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, exists)
			}
		})
	}
}

// TestPutBankAccountBalance tests the PutBankAccountBalance method
func TestPutBankAccountBalance(t *testing.T) {
	tests := []struct {
		name          string
		bankAccountID string
		runID         string
		balance       string
		wantErr       bool
		setup         func(t *testing.T, client *DatabaseClient)
	}{
		{
			name:          "valid_balance_entry",
			bankAccountID: "account_1",
			runID:         "run_1",
			balance:       "1000.50",
			wantErr:       false,
			setup: func(t *testing.T, client *DatabaseClient) {
				err := client.PutBankAccount(model.Account{
					ID:   "account_1",
					Name: "Test Account",
					Org:  model.Organization{Name: "Test Bank"},
				})
				require.NoError(t, err)
			},
		},
		{
			name:          "negative_balance",
			bankAccountID: "account_2",
			runID:         "run_2",
			balance:       "-500.25",
			wantErr:       false,
			setup: func(t *testing.T, client *DatabaseClient) {
				err := client.PutBankAccount(model.Account{
					ID:   "account_2",
					Name: "Test Account",
					Org:  model.Organization{Name: "Test Bank"},
				})
				require.NoError(t, err)
			},
		},
		{
			name:          "zero_balance",
			bankAccountID: "account_3",
			runID:         "run_3",
			balance:       "0.00",
			wantErr:       false,
			setup: func(t *testing.T, client *DatabaseClient) {
				err := client.PutBankAccount(model.Account{
					ID:   "account_3",
					Name: "Test Account",
					Org:  model.Organization{Name: "Test Bank"},
				})
				require.NoError(t, err)
			},
		},
		{
			name:          "balance_with_special_characters",
			bankAccountID: "account_4",
			runID:         "run_4",
			balance:       "$1,234.56",
			wantErr:       false,
			setup: func(t *testing.T, client *DatabaseClient) {
				err := client.PutBankAccount(model.Account{
					ID:   "account_4",
					Name: "Test Account",
					Org:  model.Organization{Name: "Test Bank"},
				})
				require.NoError(t, err)
			},
		},
		{
			name:          "empty_balance",
			bankAccountID: "account_5",
			runID:         "run_5",
			balance:       "",
			wantErr:       false,
			setup: func(t *testing.T, client *DatabaseClient) {
				err := client.PutBankAccount(model.Account{
					ID:   "account_5",
					Name: "Test Account",
					Org:  model.Organization{Name: "Test Bank"},
				})
				require.NoError(t, err)
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := setupTestDB(t)
			defer client.Close()
			
			tt.setup(t, client)
			
			err := client.PutBankAccountBalance(tt.bankAccountID, tt.runID, tt.balance)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Verify balance was inserted
				var count int
				query := "SELECT COUNT(*) FROM BANK_ACCOUNT_BALANCE WHERE ID = ? AND RUN_ID = ? AND BALANCE = ?"
				err = client.db.Get(&count, query, tt.bankAccountID, tt.runID, tt.balance)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
			}
		})
	}
}

// TestPutAccountBalance tests the PutAccountBalance method
func TestPutAccountBalance(t *testing.T) {
	tests := []struct {
		name          string
		bankAccountID string
		runID         string
		balance       string
		wantErr       bool
		setup         func(t *testing.T, client *DatabaseClient)
	}{
		{
			name:          "valid_account_balance",
			bankAccountID: "account_1",
			runID:         "run_1",
			balance:       "2500.75",
			wantErr:       false,
			setup: func(t *testing.T, client *DatabaseClient) {
				err := client.PutBankAccount(model.Account{
					ID:   "account_1",
					Name: "Test Account",
					Org:  model.Organization{Name: "Test Bank"},
				})
				require.NoError(t, err)
			},
		},
		{
			name:          "multiple_balances_same_account",
			bankAccountID: "account_2",
			runID:         "run_2",
			balance:       "3000.00",
			wantErr:       false,
			setup: func(t *testing.T, client *DatabaseClient) {
				err := client.PutBankAccount(model.Account{
					ID:   "account_2",
					Name: "Test Account",
					Org:  model.Organization{Name: "Test Bank"},
				})
				require.NoError(t, err)
				
				// Add first balance
				err = client.PutAccountBalance("account_2", "run_1", "2000.00")
				require.NoError(t, err)
			},
		},
		{
			name:          "balance_for_non_existing_account",
			bankAccountID: "non_existing",
			runID:         "run_3",
			balance:       "1000.00",
			wantErr:       false, // SQLite doesn't enforce foreign key by default
			setup:         func(t *testing.T, client *DatabaseClient) {},
		},
		{
			name:          "large_balance_value",
			bankAccountID: "account_3",
			runID:         "run_4",
			balance:       "999999999999.99",
			wantErr:       false,
			setup: func(t *testing.T, client *DatabaseClient) {
				err := client.PutBankAccount(model.Account{
					ID:   "account_3",
					Name: "Test Account",
					Org:  model.Organization{Name: "Test Bank"},
				})
				require.NoError(t, err)
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := setupTestDB(t)
			defer client.Close()
			
			tt.setup(t, client)
			
			err := client.PutAccountBalance(tt.bankAccountID, tt.runID, tt.balance)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Verify balance was inserted
				var count int
				query := "SELECT COUNT(*) FROM BANK_ACCOUNT_BALANCE WHERE BANK_ACCOUNT_ID = ? AND RUN_ID = ? AND BALANCE = ?"
				err = client.db.Get(&count, query, tt.bankAccountID, tt.runID, tt.balance)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
			}
		})
	}
}

// TestConcurrentDatabaseAccess tests concurrent database operations
func TestConcurrentDatabaseAccess(t *testing.T) {
	// Skip this test for now as in-memory SQLite has issues with WAL mode and concurrency
	t.Skip("Skipping concurrent test - SQLite in-memory doesn't properly support WAL mode")
	
	client := setupTestDB(t)
	defer client.Close()
	
	const numGoroutines = 10
	const numOperations = 5
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	errors := make(chan error, numGoroutines*numOperations*3)
	
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				accountID := fmt.Sprintf("account_%d_%d", goroutineID, j)
				account := model.Account{
					ID:   accountID,
					Name: fmt.Sprintf("Account %d-%d", goroutineID, j),
					Org: model.Organization{
						Name: "Test Bank",
					},
				}
				
				// Create account
				if err := client.PutBankAccount(account); err != nil {
					errors <- fmt.Errorf("failed to create account %s: %v", accountID, err)
					continue
				}
				
				// Check existence
				exists, err := client.DoesBankAccountExist(accountID)
				if err != nil {
					errors <- fmt.Errorf("failed to check existence of %s: %v", accountID, err)
					continue
				}
				if !exists {
					errors <- fmt.Errorf("account %s should exist but doesn't", accountID)
					continue
				}
				
				// Add balance
				runID := fmt.Sprintf("run_%d_%d", goroutineID, j)
				balance := fmt.Sprintf("%d.%02d", goroutineID*100+j, j)
				if err := client.PutAccountBalance(accountID, runID, balance); err != nil {
					errors <- fmt.Errorf("failed to add balance for %s: %v", accountID, err)
				}
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	var errCount int
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
		errCount++
	}
	
	assert.Equal(t, 0, errCount, "Should have no errors during concurrent operations")
	
	// Verify all accounts were created
	var count int
	err := client.db.Get(&count, "SELECT COUNT(*) FROM BANK_ACCOUNT")
	assert.NoError(t, err)
	assert.Equal(t, numGoroutines*numOperations, count, "All accounts should be created")
	
	// Verify all balances were created
	err = client.db.Get(&count, "SELECT COUNT(*) FROM BANK_ACCOUNT_BALANCE")
	assert.NoError(t, err)
	assert.Equal(t, numGoroutines*numOperations, count, "All balances should be created")
}

// TestDatabaseMigration tests database schema migration
func TestDatabaseMigration(t *testing.T) {
	tests := []struct {
		name            string
		initialSchema   string
		migrationSchema string
		wantErr         bool
	}{
		{
			name: "add_new_column",
			initialSchema: `
				CREATE TABLE BANK_ACCOUNT (
					ID TEXT PRIMARY KEY,
					NAME TEXT NOT NULL,
					INSTITUTION_NAME TEXT NOT NULL
				);
			`,
			migrationSchema: `
				ALTER TABLE BANK_ACCOUNT ADD COLUMN ACCOUNT_TYPE TEXT DEFAULT 'CHECKING';
			`,
			wantErr: false,
		},
		{
			name: "add_new_index",
			initialSchema: `
				CREATE TABLE BANK_ACCOUNT (
					ID TEXT PRIMARY KEY,
					NAME TEXT NOT NULL,
					INSTITUTION_NAME TEXT NOT NULL
				);
			`,
			migrationSchema: `
				CREATE INDEX idx_bank_account_name ON BANK_ACCOUNT(NAME);
			`,
			wantErr: false,
		},
		{
			name: "add_new_table",
			initialSchema: `
				CREATE TABLE BANK_ACCOUNT (
					ID TEXT PRIMARY KEY,
					NAME TEXT NOT NULL,
					INSTITUTION_NAME TEXT NOT NULL
				);
			`,
			migrationSchema: `
				CREATE TABLE ACCOUNT_METADATA (
					ACCOUNT_ID TEXT PRIMARY KEY,
					METADATA TEXT,
					FOREIGN KEY(ACCOUNT_ID) REFERENCES BANK_ACCOUNT(ID)
				);
			`,
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create database with initial schema
			db, err := sqlx.Connect("sqlite3", ":memory:")
			require.NoError(t, err)
			defer db.Close()
			
			_, err = db.Exec(tt.initialSchema)
			require.NoError(t, err)
			
			// Apply migration
			_, err = db.Exec(tt.migrationSchema)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTransactionRollback tests transaction rollback behavior
func TestTransactionRollback(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()
	
	// Start transaction
	tx, err := client.db.Beginx()
	require.NoError(t, err)
	
	// Insert account in transaction
	query := "INSERT INTO BANK_ACCOUNT (ID, NAME, INSTITUTION_NAME) VALUES (?, ?, ?)"
	_, err = tx.Exec(query, "tx_account", "Transaction Account", "Test Bank")
	require.NoError(t, err)
	
	// Verify account exists in transaction
	var count int
	err = tx.Get(&count, "SELECT COUNT(*) FROM BANK_ACCOUNT WHERE ID = ?", "tx_account")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	
	// Rollback transaction
	err = tx.Rollback()
	require.NoError(t, err)
	
	// Verify account does not exist after rollback
	exists, err := client.DoesBankAccountExist("tx_account")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// Benchmark tests
func BenchmarkPutBankAccount(b *testing.B) {
	client := setupTestDB(&testing.T{})
	defer client.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		account := model.Account{
			ID:   fmt.Sprintf("bench_account_%d", i),
			Name: fmt.Sprintf("Benchmark Account %d", i),
			Org: model.Organization{
				Name: "Benchmark Bank",
			},
		}
		_ = client.PutBankAccount(account)
	}
}

func BenchmarkDoesBankAccountExist(b *testing.B) {
	client := setupTestDB(&testing.T{})
	defer client.Close()
	
	// Pre-create an account
	account := model.Account{
		ID:   "bench_account",
		Name: "Benchmark Account",
		Org: model.Organization{
			Name: "Benchmark Bank",
		},
	}
	_ = client.PutBankAccount(account)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.DoesBankAccountExist("bench_account")
	}
}

func BenchmarkPutAccountBalance(b *testing.B) {
	client := setupTestDB(&testing.T{})
	defer client.Close()
	
	// Pre-create an account
	account := model.Account{
		ID:   "bench_account",
		Name: "Benchmark Account",
		Org: model.Organization{
			Name: "Benchmark Bank",
		},
	}
	_ = client.PutBankAccount(account)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runID := fmt.Sprintf("run_%d", i)
		balance := fmt.Sprintf("%d.00", i)
		_ = client.PutAccountBalance("bench_account", runID, balance)
	}
}

// TestMockDatabaseOperations tests using sqlmock for complex scenarios
func TestMockDatabaseOperations(t *testing.T) {
	t.Run("mock_database_error", func(t *testing.T) {
		mockDB, mock, client := setupMockDB(t)
		defer mockDB.Close()
		
		// Expect a query and return an error
		mock.ExpectExec("INSERT INTO BANK_ACCOUNT").
			WithArgs("test_id", "test_name", "test_bank").
			WillReturnError(fmt.Errorf("database connection lost"))
		
		account := model.Account{
			ID:   "test_id",
			Name: "test_name",
			Org: model.Organization{
				Name: "test_bank",
			},
		}
		
		err := client.PutBankAccount(account)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection lost")
		
		// Verify all expectations were met
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
	
	t.Run("mock_successful_operation", func(t *testing.T) {
		mockDB, mock, client := setupMockDB(t)
		defer mockDB.Close()
		
		// Expect a successful insert
		mock.ExpectExec("INSERT INTO BANK_ACCOUNT").
			WithArgs("test_id", "test_name", "test_bank").
			WillReturnResult(sqlmock.NewResult(1, 1))
		
		account := model.Account{
			ID:   "test_id",
			Name: "test_name",
			Org: model.Organization{
				Name: "test_bank",
			},
		}
		
		err := client.PutBankAccount(account)
		assert.NoError(t, err)
		
		// Verify all expectations were met
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

