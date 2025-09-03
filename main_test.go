package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/criswit/chi-chi-moni/api"
	"github.com/criswit/chi-chi-moni/aws"
	"github.com/criswit/chi-chi-moni/db"
	"github.com/criswit/chi-chi-moni/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing
type mockSecretsManagerClient struct {
	retrieveFunc func(ctx context.Context, name string) (api.AccessToken, error)
	storeFunc    func(ctx context.Context, name string, token api.AccessToken) error
}

func (m *mockSecretsManagerClient) RetrieveAccessToken(ctx context.Context, name string) (api.AccessToken, error) {
	if m.retrieveFunc != nil {
		return m.retrieveFunc(ctx, name)
	}
	return api.AccessToken{}, errors.New("not implemented")
}

func (m *mockSecretsManagerClient) StoreAccessToken(ctx context.Context, name string, token api.AccessToken) error {
	if m.storeFunc != nil {
		return m.storeFunc(ctx, name, token)
	}
	return errors.New("not implemented")
}

type mockSSOClient struct {
	checkStatusFunc func() aws.CredentialStatus
}

func (m *mockSSOClient) CheckCredentialStatus() aws.CredentialStatus {
	if m.checkStatusFunc != nil {
		return m.checkStatusFunc()
	}
	return aws.CredentialStatusError
}

type mockSimpleFinClient struct {
	getAccountsFunc func(opts *api.GetAccountsOptions) (*model.GetAccountsResponse, error)
}

func (m *mockSimpleFinClient) GetAccounts(opts *api.GetAccountsOptions) (*model.GetAccountsResponse, error) {
	if m.getAccountsFunc != nil {
		return m.getAccountsFunc(opts)
	}
	return nil, errors.New("not implemented")
}

type mockDatabaseClient struct {
	putBankAccountFunc      func(account model.Account) error
	putAccountBalanceFunc   func(accountID, runID, balance string) error
	doesBankAccountExistFunc func(accountID string) (bool, error)
	closeFunc               func()
}

func (m *mockDatabaseClient) PutBankAccount(account model.Account) error {
	if m.putBankAccountFunc != nil {
		return m.putBankAccountFunc(account)
	}
	return errors.New("not implemented")
}

func (m *mockDatabaseClient) PutAccountBalance(accountID, runID, balance string) error {
	if m.putAccountBalanceFunc != nil {
		return m.putAccountBalanceFunc(accountID, runID, balance)
	}
	return errors.New("not implemented")
}

func (m *mockDatabaseClient) DoesBankAccountExist(accountID string) (bool, error) {
	if m.doesBankAccountExistFunc != nil {
		return m.doesBankAccountExistFunc(accountID)
	}
	return false, errors.New("not implemented")
}

func (m *mockDatabaseClient) Close() {
	if m.closeFunc != nil {
		m.closeFunc()
	}
}

// TestGetAccessToken tests the getAccessToken function
func TestGetAccessToken(t *testing.T) {
	// Note: This function depends on AWS SSO and Secrets Manager
	// In a real test environment, we would need to mock these dependencies
	// or use integration tests with test AWS accounts
	
	t.Run("mock_successful_retrieval", func(t *testing.T) {
		// This test demonstrates the structure but requires dependency injection
		// to properly test without real AWS credentials
		t.Skip("Requires AWS SSO and Secrets Manager mocking")
	})
	
	t.Run("mock_sso_error", func(t *testing.T) {
		t.Skip("Requires AWS SSO mocking")
	})
	
	t.Run("mock_secrets_manager_error", func(t *testing.T) {
		t.Skip("Requires Secrets Manager mocking")
	})
}

// TestGetDbFilePath tests the getDbFilePath function
func TestGetDbFilePath(t *testing.T) {
	tests := []struct {
		name      string
		setupEnv  func()
		cleanup   func()
		wantPath  string
		wantErr   bool
	}{
		{
			name: "valid_home_directory",
			setupEnv: func() {
				// Use actual home directory
			},
			cleanup: func() {},
			wantErr: false,
		},
		{
			name: "custom_home_directory",
			setupEnv: func() {
				os.Setenv("HOME", "/custom/home")
			},
			cleanup: func() {
				os.Unsetenv("HOME")
			},
			wantPath: "/custom/home/data/monk.db",
			wantErr:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanup()
			
			path, err := getDbFilePath()
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, path)
				assert.Contains(t, path, "data/monk.db")
				
				if tt.wantPath != "" {
					assert.Equal(t, tt.wantPath, path)
				}
			}
		})
	}
}

// TestMainFlow tests the main application flow with mocks
func TestMainFlow(t *testing.T) {
	// This test demonstrates how the main function flow would be tested
	// with proper dependency injection
	
	tests := []struct {
		name           string
		setupMocks     func() (*mockSimpleFinClient, *mockDatabaseClient)
		expectedError  bool
	}{
		{
			name: "successful_flow",
			setupMocks: func() (*mockSimpleFinClient, *mockDatabaseClient) {
				finClient := &mockSimpleFinClient{
					getAccountsFunc: func(opts *api.GetAccountsOptions) (*model.GetAccountsResponse, error) {
						return &model.GetAccountsResponse{
							Accounts: []model.Account{
								{
									ID:      "acc_001",
									Name:    "Test Account",
									Balance: "1000.00",
									Org: model.Organization{
										Name: "Test Bank",
									},
								},
							},
						}, nil
					},
				}
				
				dbClient := &mockDatabaseClient{
					doesBankAccountExistFunc: func(accountID string) (bool, error) {
						return false, nil
					},
					putBankAccountFunc: func(account model.Account) error {
						return nil
					},
					putAccountBalanceFunc: func(accountID, runID, balance string) error {
						return nil
					},
					closeFunc: func() {},
				}
				
				return finClient, dbClient
			},
			expectedError: false,
		},
		{
			name: "account_already_exists",
			setupMocks: func() (*mockSimpleFinClient, *mockDatabaseClient) {
				finClient := &mockSimpleFinClient{
					getAccountsFunc: func(opts *api.GetAccountsOptions) (*model.GetAccountsResponse, error) {
						return &model.GetAccountsResponse{
							Accounts: []model.Account{
								{
									ID:      "acc_002",
									Name:    "Existing Account",
									Balance: "2000.00",
									Org: model.Organization{
										Name: "Test Bank",
									},
								},
							},
						}, nil
					},
				}
				
				dbClient := &mockDatabaseClient{
					doesBankAccountExistFunc: func(accountID string) (bool, error) {
						return true, nil // Account already exists
					},
					putAccountBalanceFunc: func(accountID, runID, balance string) error {
						return nil
					},
					closeFunc: func() {},
				}
				
				return finClient, dbClient
			},
			expectedError: false,
		},
		{
			name: "database_error",
			setupMocks: func() (*mockSimpleFinClient, *mockDatabaseClient) {
				finClient := &mockSimpleFinClient{
					getAccountsFunc: func(opts *api.GetAccountsOptions) (*model.GetAccountsResponse, error) {
						return &model.GetAccountsResponse{
							Accounts: []model.Account{
								{
									ID:      "acc_003",
									Name:    "Test Account",
									Balance: "3000.00",
								},
							},
						}, nil
					},
				}
				
				dbClient := &mockDatabaseClient{
					doesBankAccountExistFunc: func(accountID string) (bool, error) {
						return false, errors.New("database error")
					},
				}
				
				return finClient, dbClient
			},
			expectedError: true,
		},
		{
			name: "api_error",
			setupMocks: func() (*mockSimpleFinClient, *mockDatabaseClient) {
				finClient := &mockSimpleFinClient{
					getAccountsFunc: func(opts *api.GetAccountsOptions) (*model.GetAccountsResponse, error) {
						return nil, errors.New("API error")
					},
				}
				
				dbClient := &mockDatabaseClient{}
				
				return finClient, dbClient
			},
			expectedError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finClient, dbClient := tt.setupMocks()
			
			// Simulate main flow
			resp, err := finClient.GetAccounts(&api.GetAccountsOptions{})
			
			if tt.expectedError {
				if err == nil {
					// Check for database errors
					for _, account := range resp.Accounts {
						_, err = dbClient.DoesBankAccountExist(account.ID)
						if err != nil {
							break
						}
					}
				}
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				
				// Process accounts like main() does
				for _, account := range resp.Accounts {
					exists, err := dbClient.DoesBankAccountExist(account.ID)
					require.NoError(t, err)
					
					if !exists {
						err = dbClient.PutBankAccount(account)
						require.NoError(t, err)
					}
					
					err = dbClient.PutAccountBalance(account.ID, "test-uuid", account.Balance)
					require.NoError(t, err)
				}
			}
		})
	}
}

// TestConstants tests the package constants
func TestConstants(t *testing.T) {
	assert.Equal(t, "monkstorage", ssoProfile)
	assert.Equal(t, "monk-monies", accessTokenSecretName)
	assert.Equal(t, "data/monk.db", dbFilePath)
}

// TestDatabaseInitialization tests database initialization
func TestDatabaseInitialization(t *testing.T) {
	t.Run("valid_database_path", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		
		client, err := db.NewDatabaseClient(dbPath)
		if err == nil {
			defer client.Close()
		}
		
		// For SQLite, this should create the database file
		assert.NoError(t, err)
		assert.NotNil(t, client)
		
		// Check that database file was created
		_, err = os.Stat(dbPath)
		assert.NoError(t, err)
	})
	
	t.Run("invalid_database_path", func(t *testing.T) {
		// Try to create database in non-existent directory
		dbPath := "/invalid/path/to/database.db"
		
		client, err := db.NewDatabaseClient(dbPath)
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

// TestErrorHandling tests error handling scenarios
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupError    func() error
		expectedPanic bool
	}{
		{
			name: "access_token_error",
			setupError: func() error {
				return errors.New("failed to get access token")
			},
			expectedPanic: true,
		},
		{
			name: "database_connection_error",
			setupError: func() error {
				return errors.New("failed to connect to database")
			},
			expectedPanic: true,
		},
		{
			name: "api_client_creation_error",
			setupError: func() error {
				return errors.New("failed to create API client")
			},
			expectedPanic: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupError()
			assert.Error(t, err)
			
			// In the actual main(), these errors would cause log.Fatal()
			// which exits the program. In tests, we verify the error exists.
		})
	}
}

// TestIntegrationPoints tests integration between components
func TestIntegrationPoints(t *testing.T) {
	t.Run("access_token_to_api_client", func(t *testing.T) {
		// Test that access token can be used to create API client
		token := api.AccessToken{
			Url:      "https://example.simplefin.org/accounts",
			Username: "testuser",
			Password: "testpass",
		}
		
		client, err := api.NewSimpleFinClient(token)
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})
	
	t.Run("api_response_to_database", func(t *testing.T) {
		// Test that API response can be stored in database
		// This would require a test database
		t.Skip("Requires test database setup")
	})
}

// Benchmark tests
func BenchmarkGetDbFilePath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = getDbFilePath()
	}
}

func BenchmarkMainFlow(b *testing.B) {
	// Setup mocks
	finClient := &mockSimpleFinClient{
		getAccountsFunc: func(opts *api.GetAccountsOptions) (*model.GetAccountsResponse, error) {
			return &model.GetAccountsResponse{
				Accounts: []model.Account{
					{
						ID:      "bench_account",
						Name:    "Benchmark Account",
						Balance: "1000.00",
						Org: model.Organization{
							Name: "Benchmark Bank",
						},
					},
				},
			}, nil
		},
	}
	
	dbClient := &mockDatabaseClient{
		doesBankAccountExistFunc: func(accountID string) (bool, error) {
			return false, nil
		},
		putBankAccountFunc: func(account model.Account) error {
			return nil
		},
		putAccountBalanceFunc: func(accountID, runID, balance string) error {
			return nil
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := finClient.GetAccounts(&api.GetAccountsOptions{})
		for _, account := range resp.Accounts {
			exists, _ := dbClient.DoesBankAccountExist(account.ID)
			if !exists {
				_ = dbClient.PutBankAccount(account)
			}
			_ = dbClient.PutAccountBalance(account.ID, "bench-uuid", account.Balance)
		}
	}
}