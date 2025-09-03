package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures
var (
	validOrganization = Organization{
		Domain:  "example.bank.com",
		Name:    "Example Bank",
		SfinURL: "https://api.simplefin.org/example",
		URL:     "https://example.bank.com",
		ID:      "org_123",
	}

	validTransaction = Transaction{
		ID:           "txn_456",
		Posted:       1704067200, // 2024-01-01 00:00:00 UTC
		Amount:       "-150.50",
		Description:  "Purchase at Store",
		Payee:        "Store Name",
		Memo:         "Transaction memo",
		TransactedAt: 1704067200,
	}

	validAccount = Account{
		Org:              validOrganization,
		ID:               "acc_789",
		Name:             "Checking Account",
		Currency:         "USD",
		Balance:          "1234.56",
		AvailableBalance: "1200.00",
		BalanceDate:      1704067200,
		Transactions: []Transaction{
			validTransaction,
		},
		Holdings: []interface{}{},
	}
)

// TestOrganizationJSONMarshaling tests JSON marshaling and unmarshaling of Organization
func TestOrganizationJSONMarshaling(t *testing.T) {
	tests := []struct {
		name         string
		org          Organization
		wantErr      bool
		validateJSON func(t *testing.T, data []byte)
	}{
		{
			name: "complete_organization",
			org:  validOrganization,
			validateJSON: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, "example.bank.com", result["domain"])
				assert.Equal(t, "Example Bank", result["name"])
				assert.Equal(t, "org_123", result["id"])
			},
		},
		{
			name: "empty_organization",
			org:  Organization{},
			validateJSON: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, "", result["domain"])
				assert.Equal(t, "", result["name"])
			},
		},
		{
			name: "organization_with_special_characters",
			org: Organization{
				Domain:  "банк.рф",
				Name:    "Bank & Trust \"Co.\"",
				SfinURL: "https://api.simplefin.org/bank%20trust",
				URL:     "https://банк.рф",
				ID:      "org_!@#$%",
			},
			validateJSON: func(t *testing.T, data []byte) {
				var result Organization
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, "банк.рф", result.Domain)
				assert.Equal(t, "Bank & Trust \"Co.\"", result.Name)
				assert.Equal(t, "org_!@#$%", result.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.org)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Validate JSON structure
			if tt.validateJSON != nil {
				tt.validateJSON(t, data)
			}

			// Unmarshal back to struct
			var result Organization
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)
			assert.Equal(t, tt.org, result)
		})
	}
}

// TestTransactionJSONMarshaling tests JSON marshaling and unmarshaling of Transaction
func TestTransactionJSONMarshaling(t *testing.T) {
	tests := []struct {
		name         string
		transaction  Transaction
		wantErr      bool
		validateJSON func(t *testing.T, data []byte)
	}{
		{
			name:        "complete_transaction",
			transaction: validTransaction,
			validateJSON: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, "txn_456", result["id"])
				assert.Equal(t, float64(1704067200), result["posted"])
				assert.Equal(t, "-150.50", result["amount"])
			},
		},
		{
			name:        "empty_transaction",
			transaction: Transaction{},
			validateJSON: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, "", result["id"])
				assert.Equal(t, float64(0), result["posted"])
			},
		},
		{
			name: "transaction_with_large_amounts",
			transaction: Transaction{
				ID:           "txn_large",
				Posted:       1704067200,
				Amount:       "999999999999.99",
				Description:  "Large transaction",
				Payee:        "Payee",
				Memo:         "Memo",
				TransactedAt: 1704067200,
			},
		},
		{
			name: "transaction_with_negative_amount",
			transaction: Transaction{
				ID:           "txn_neg",
				Posted:       1704067200,
				Amount:       "-0.01",
				Description:  "Small debit",
				Payee:        "Payee",
				Memo:         "Memo",
				TransactedAt: 1704067200,
			},
		},
		{
			name: "transaction_with_unicode",
			transaction: Transaction{
				ID:           "txn_unicode",
				Posted:       1704067200,
				Amount:       "100.00",
				Description:  "購入 at 店舗",
				Payee:        "日本の店",
				Memo:         "メモ 📝",
				TransactedAt: 1704067200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.transaction)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Validate JSON structure
			if tt.validateJSON != nil {
				tt.validateJSON(t, data)
			}

			// Unmarshal back to struct
			var result Transaction
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)
			assert.Equal(t, tt.transaction, result)
		})
	}
}

// TestAccountJSONMarshaling tests JSON marshaling and unmarshaling of Account
func TestAccountJSONMarshaling(t *testing.T) {
	tests := []struct {
		name         string
		account      Account
		wantErr      bool
		validateJSON func(t *testing.T, data []byte)
	}{
		{
			name:    "complete_account",
			account: validAccount,
			validateJSON: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, "acc_789", result["id"])
				assert.Equal(t, "Checking Account", result["name"])
				assert.Equal(t, "USD", result["currency"])
				assert.Equal(t, "1234.56", result["balance"])
				assert.Equal(t, "1200.00", result["available-balance"])
				
				// Check nested organization
				org := result["org"].(map[string]interface{})
				assert.Equal(t, "Example Bank", org["name"])
				
				// Check transactions array
				txns := result["transactions"].([]interface{})
				assert.Len(t, txns, 1)
			},
		},
		{
			name:    "empty_account",
			account: Account{},
			validateJSON: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, "", result["id"])
				assert.Nil(t, result["transactions"])
				assert.Nil(t, result["holdings"])
			},
		},
		{
			name: "account_with_no_transactions",
			account: Account{
				Org:              validOrganization,
				ID:               "acc_no_txn",
				Name:             "Savings Account",
				Currency:         "USD",
				Balance:          "5000.00",
				AvailableBalance: "5000.00",
				BalanceDate:      1704067200,
				Transactions:     []Transaction{},
				Holdings:         []interface{}{},
			},
		},
		{
			name: "account_with_multiple_transactions",
			account: Account{
				Org:              validOrganization,
				ID:               "acc_multi",
				Name:             "Checking",
				Currency:         "USD",
				Balance:          "1000.00",
				AvailableBalance: "900.00",
				BalanceDate:      1704067200,
				Transactions: []Transaction{
					validTransaction,
					{
						ID:           "txn_002",
						Posted:       1704153600,
						Amount:       "500.00",
						Description:  "Deposit",
						Payee:        "Employer",
						Memo:         "Salary",
						TransactedAt: 1704153600,
					},
				},
				Holdings: []interface{}{},
			},
		},
		{
			name: "account_with_zero_balance",
			account: Account{
				Org:              validOrganization,
				ID:               "acc_zero",
				Name:             "Empty Account",
				Currency:         "USD",
				Balance:          "0.00",
				AvailableBalance: "0.00",
				BalanceDate:      1704067200,
				Transactions:     []Transaction{},
				Holdings:         []interface{}{},
			},
		},
		{
			name: "account_with_negative_balance",
			account: Account{
				Org:              validOrganization,
				ID:               "acc_negative",
				Name:             "Overdrawn Account",
				Currency:         "USD",
				Balance:          "-50.00",
				AvailableBalance: "-50.00",
				BalanceDate:      1704067200,
				Transactions:     []Transaction{},
				Holdings:         []interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.account)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Validate JSON structure
			if tt.validateJSON != nil {
				tt.validateJSON(t, data)
			}

			// Unmarshal back to struct
			var result Account
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)
			assert.Equal(t, tt.account, result)
		})
	}
}

// TestGetAccountsResponseJSONMarshaling tests JSON marshaling and unmarshaling of GetAccountsResponse
func TestGetAccountsResponseJSONMarshaling(t *testing.T) {
	tests := []struct {
		name         string
		response     GetAccountsResponse
		wantErr      bool
		validateJSON func(t *testing.T, data []byte)
	}{
		{
			name: "complete_response",
			response: GetAccountsResponse{
				Errors: []string{},
				Accounts: []Account{
					validAccount,
				},
				XAPIMessage: []string{"API message 1", "API message 2"},
			},
			validateJSON: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				
				errors := result["errors"].([]interface{})
				assert.Len(t, errors, 0)
				
				accounts := result["accounts"].([]interface{})
				assert.Len(t, accounts, 1)
				
				messages := result["x-api-message"].([]interface{})
				assert.Len(t, messages, 2)
			},
		},
		{
			name:     "empty_response",
			response: GetAccountsResponse{},
			validateJSON: func(t *testing.T, data []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Nil(t, result["errors"])
				assert.Nil(t, result["accounts"])
				assert.Nil(t, result["x-api-message"])
			},
		},
		{
			name: "response_with_errors",
			response: GetAccountsResponse{
				Errors: []string{
					"Authentication failed",
					"Invalid token",
				},
				Accounts:    []Account{},
				XAPIMessage: []string{},
			},
		},
		{
			name: "response_with_multiple_accounts",
			response: GetAccountsResponse{
				Errors: []string{},
				Accounts: []Account{
					validAccount,
					{
						Org:              validOrganization,
						ID:               "acc_002",
						Name:             "Savings",
						Currency:         "USD",
						Balance:          "5000.00",
						AvailableBalance: "5000.00",
						BalanceDate:      1704067200,
						Transactions:     []Transaction{},
						Holdings:         []interface{}{},
					},
				},
				XAPIMessage: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.response)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Validate JSON structure
			if tt.validateJSON != nil {
				tt.validateJSON(t, data)
			}

			// Unmarshal back to struct
			var result GetAccountsResponse
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)
			assert.Equal(t, tt.response, result)
		})
	}
}

// TestTransactionHelperMethods tests the helper methods for Transaction
func TestTransactionHelperMethods(t *testing.T) {
	tests := []struct {
		name        string
		transaction Transaction
		wantPosted  time.Time
		wantTxnTime time.Time
	}{
		{
			name:        "valid_timestamps",
			transaction: validTransaction,
			wantPosted:  time.Unix(1704067200, 0),
			wantTxnTime: time.Unix(1704067200, 0),
		},
		{
			name:        "zero_timestamps",
			transaction: Transaction{},
			wantPosted:  time.Unix(0, 0),
			wantTxnTime: time.Unix(0, 0),
		},
		{
			name: "different_timestamps",
			transaction: Transaction{
				Posted:       1704067200,
				TransactedAt: 1704153600,
			},
			wantPosted:  time.Unix(1704067200, 0),
			wantTxnTime: time.Unix(1704153600, 0),
		},
		{
			name: "negative_timestamp",
			transaction: Transaction{
				Posted:       -1,
				TransactedAt: -1,
			},
			wantPosted:  time.Unix(-1, 0),
			wantTxnTime: time.Unix(-1, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantPosted, tt.transaction.PostedTime())
			assert.Equal(t, tt.wantTxnTime, tt.transaction.TransactedTime())
		})
	}
}

// TestAccountHelperMethods tests the helper methods for Account
func TestAccountHelperMethods(t *testing.T) {
	tests := []struct {
		name            string
		account         Account
		wantBalanceTime time.Time
	}{
		{
			name:            "valid_balance_date",
			account:         validAccount,
			wantBalanceTime: time.Unix(1704067200, 0),
		},
		{
			name:            "zero_balance_date",
			account:         Account{},
			wantBalanceTime: time.Unix(0, 0),
		},
		{
			name: "negative_balance_date",
			account: Account{
				BalanceDate: -1,
			},
			wantBalanceTime: time.Unix(-1, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantBalanceTime, tt.account.BalanceTime())
		})
	}
}

// TestJSONUnmarshalingErrors tests error handling during JSON unmarshaling
func TestJSONUnmarshalingErrors(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		target  interface{}
		wantErr bool
	}{
		{
			name:    "invalid_json_organization",
			json:    `{"domain": "test", "name": 123}`, // name should be string
			target:  &Organization{},
			wantErr: true,
		},
		{
			name:    "invalid_json_transaction",
			json:    `{"id": "test", "posted": "not_a_number"}`, // posted should be number
			target:  &Transaction{},
			wantErr: true,
		},
		{
			name:    "invalid_json_account",
			json:    `{"id": "test", "transactions": "not_an_array"}`, // transactions should be array
			target:  &Account{},
			wantErr: true,
		},
		{
			name:    "malformed_json",
			json:    `{"id": "test"`,
			target:  &Account{},
			wantErr: true,
		},
		{
			name:    "null_values",
			json:    `{"id": null, "name": null}`,
			target:  &Account{},
			wantErr: false, // null values should be handled gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := json.Unmarshal([]byte(tt.json), tt.target)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestComplexNestedStructures tests complex nested JSON structures
func TestComplexNestedStructures(t *testing.T) {
	complexResponse := GetAccountsResponse{
		Errors: []string{"warning: partial data"},
		Accounts: []Account{
			{
				Org: Organization{
					Domain:  "bank1.com",
					Name:    "Bank One",
					SfinURL: "https://api.simplefin.org/bank1",
					URL:     "https://bank1.com",
					ID:      "org_bank1",
				},
				ID:               "acc_complex_1",
				Name:             "Primary Checking",
				Currency:         "USD",
				Balance:          "10000.00",
				AvailableBalance: "9500.00",
				BalanceDate:      1704067200,
				Transactions: []Transaction{
					{
						ID:           "txn_c1",
						Posted:       1704067200,
						Amount:       "-100.00",
						Description:  "ATM Withdrawal",
						Payee:        "ATM",
						Memo:         "Cash",
						TransactedAt: 1704067200,
					},
					{
						ID:           "txn_c2",
						Posted:       1704153600,
						Amount:       "2500.00",
						Description:  "Direct Deposit",
						Payee:        "Employer Inc",
						Memo:         "Salary",
						TransactedAt: 1704153600,
					},
				},
				Holdings: []interface{}{},
			},
			{
				Org: Organization{
					Domain:  "bank2.com",
					Name:    "Bank Two",
					SfinURL: "https://api.simplefin.org/bank2",
					URL:     "https://bank2.com",
					ID:      "org_bank2",
				},
				ID:               "acc_complex_2",
				Name:             "Savings Account",
				Currency:         "USD",
				Balance:          "25000.00",
				AvailableBalance: "25000.00",
				BalanceDate:      1704067200,
				Transactions:     []Transaction{},
				Holdings:         []interface{}{},
			},
		},
		XAPIMessage: []string{
			"Rate limit: 95/100",
			"Next refresh: 3600s",
		},
	}

	// Marshal the complex structure
	data, err := json.Marshal(complexResponse)
	require.NoError(t, err)

	// Unmarshal it back
	var result GetAccountsResponse
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify the structure is preserved
	assert.Equal(t, complexResponse, result)
	assert.Len(t, result.Accounts, 2)
	assert.Len(t, result.Accounts[0].Transactions, 2)
	assert.Len(t, result.Errors, 1)
	assert.Len(t, result.XAPIMessage, 2)
}

// Benchmark tests
func BenchmarkAccountJSONMarshal(b *testing.B) {
	account := validAccount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(account)
	}
}

func BenchmarkAccountJSONUnmarshal(b *testing.B) {
	data, _ := json.Marshal(validAccount)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var account Account
		_ = json.Unmarshal(data, &account)
	}
}

func BenchmarkGetAccountsResponseMarshal(b *testing.B) {
	response := GetAccountsResponse{
		Errors: []string{},
		Accounts: []Account{
			validAccount,
			validAccount,
			validAccount,
		},
		XAPIMessage: []string{"msg1", "msg2"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(response)
	}
}

func BenchmarkTransactionPostedTime(b *testing.B) {
	txn := validTransaction
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = txn.PostedTime()
	}
}