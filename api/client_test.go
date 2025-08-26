package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/criswit/chi-chi-moni/model"
)

func TestNewSimpleFinClient(t *testing.T) {
	accessToken := AccessToken{
		Username: "testuser",
		Password: "testpass",
		Url:      "api.example.com",
	}

	client, err := NewSimpleFinClient(accessToken)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	if client.baseUrl != accessToken.Url {
		t.Errorf("Expected baseUrl %s, got %s", accessToken.Url, client.baseUrl)
	}

	if client.client == nil {
		t.Fatal("Expected http client to be non-nil")
	}

	// Verify that the transport is set correctly
	transport, ok := client.client.Transport.(*SimpleFinRoundTripper)
	if !ok {
		t.Fatal("Expected transport to be SimpleFinRoundTripper")
	}

	if transport.username != accessToken.Username {
		t.Errorf("Expected username %s, got %s", accessToken.Username, transport.username)
	}

	if transport.password != accessToken.Password {
		t.Errorf("Expected password %s, got %s", accessToken.Password, transport.password)
	}
}

func TestSimpleFinClient_GetAccounts_Success(t *testing.T) {
	// Create mock financial response
	mockResponse := &model.GetAccountsResponse{
		Errors: []string{},
		Accounts: []model.Account{
			{
				ID:               "acc1",
				Name:             "Checking Account",
				Currency:         "USD",
				Balance:          "1000.00",
				AvailableBalance: "950.00",
				BalanceDate:      1640995200, // Unix timestamp
				Org: model.Organization{
					ID:     "org1",
					Name:   "Test Bank",
					Domain: "testbank.com",
					URL:    "https://testbank.com",
				},
				Transactions: []model.Transaction{
					{
						ID:           "txn1",
						Posted:       1640995200,
						Amount:       "-50.00",
						Description:  "Coffee Shop",
						Payee:        "Starbucks",
						Memo:         "Morning coffee",
						TransactedAt: 1640995200,
					},
				},
				Holdings: []interface{}{},
			},
		},
		XAPIMessage: []string{"Success"},
	}

	// Create mock server with TLS
	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		expectedPath := "/accounts"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Verify basic auth is set (this will be handled by the RoundTripper)
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("Expected basic auth to be set")
		} else {
			if username != "testuser" {
				t.Errorf("Expected username testuser, got %s", username)
			}
			if password != "testpass" {
				t.Errorf("Expected password testpass, got %s", password)
			}
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer mockServer.Close()

	// Extract host from mock server URL (remove https://)
	serverURL := strings.TrimPrefix(mockServer.URL, "https://")

	accessToken := AccessToken{
		Username: "testuser",
		Password: "testpass",
		Url:      serverURL,
	}

	client, err := NewSimpleFinClient(accessToken)
	if err != nil {
		t.Fatalf("Expected no error creating client, got %v", err)
	}

	// Use the server's client but preserve our RoundTripper for auth
	serverClient := mockServer.Client()
	client.client.Transport = &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     serverClient.Transport,
	}

	// Test GetAccounts
	response, err := client.GetAccounts()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response == nil {
		t.Fatal("Expected response to be non-nil")
	}

	// Verify response structure
	if len(response.Accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(response.Accounts))
	}

	account := response.Accounts[0]
	if account.ID != "acc1" {
		t.Errorf("Expected account ID acc1, got %s", account.ID)
	}

	if account.Name != "Checking Account" {
		t.Errorf("Expected account name 'Checking Account', got %s", account.Name)
	}

	if account.Balance != "1000.00" {
		t.Errorf("Expected balance 1000.00, got %s", account.Balance)
	}

	if len(account.Transactions) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(account.Transactions))
	}

	transaction := account.Transactions[0]
	if transaction.ID != "txn1" {
		t.Errorf("Expected transaction ID txn1, got %s", transaction.ID)
	}

	if transaction.Amount != "-50.00" {
		t.Errorf("Expected transaction amount -50.00, got %s", transaction.Amount)
	}
}

func TestSimpleFinClient_GetAccounts_HTTPError(t *testing.T) {
	// Create mock server that returns an error
	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer mockServer.Close()

	serverURL := strings.TrimPrefix(mockServer.URL, "https://")

	accessToken := AccessToken{
		Username: "testuser",
		Password: "testpass",
		Url:      serverURL,
	}

	client, err := NewSimpleFinClient(accessToken)
	if err != nil {
		t.Fatalf("Expected no error creating client, got %v", err)
	}

	// Use the server's client but preserve our RoundTripper for auth
	serverClient := mockServer.Client()
	client.client.Transport = &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     serverClient.Transport,
	}

	_, err = client.GetAccounts()
	if err == nil {
		t.Error("Expected error for HTTP error response, got nil")
	}
}

func TestSimpleFinClient_GetAccounts_InvalidJSON(t *testing.T) {
	// Create mock server that returns invalid JSON
	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer mockServer.Close()

	serverURL := strings.TrimPrefix(mockServer.URL, "https://")

	accessToken := AccessToken{
		Username: "testuser",
		Password: "testpass",
		Url:      serverURL,
	}

	client, err := NewSimpleFinClient(accessToken)
	if err != nil {
		t.Fatalf("Expected no error creating client, got %v", err)
	}

	// Use the server's client but preserve our RoundTripper for auth
	serverClient := mockServer.Client()
	client.client.Transport = &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     serverClient.Transport,
	}

	_, err = client.GetAccounts()
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestSimpleFinClient_GetAccounts_EmptyResponse(t *testing.T) {
	// Create mock server that returns empty response
	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer mockServer.Close()

	serverURL := strings.TrimPrefix(mockServer.URL, "https://")

	accessToken := AccessToken{
		Username: "testuser",
		Password: "testpass",
		Url:      serverURL,
	}

	client, err := NewSimpleFinClient(accessToken)
	if err != nil {
		t.Fatalf("Expected no error creating client, got %v", err)
	}

	// Use the server's client but preserve our RoundTripper for auth
	serverClient := mockServer.Client()
	client.client.Transport = &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     serverClient.Transport,
	}

	response, err := client.GetAccounts()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response == nil {
		t.Fatal("Expected response to be non-nil")
	}

	// Empty response should have nil slice (this is how JSON unmarshaling works)
	if response.Accounts == nil {
		t.Log("Accounts is nil as expected for empty JSON response")
	}
}

func TestSimpleFinClient_GetAccounts_NetworkError(t *testing.T) {
	// Use an invalid URL to simulate network error
	accessToken := AccessToken{
		Username: "testuser",
		Password: "testpass",
		Url:      "invalid-host-that-does-not-exist.com",
	}

	client, err := NewSimpleFinClient(accessToken)
	if err != nil {
		t.Fatalf("Expected no error creating client, got %v", err)
	}

	_, err = client.GetAccounts()
	if err == nil {
		t.Error("Expected network error, got nil")
	}
}

func TestSimpleFinClient_GetAccounts_URLConstruction(t *testing.T) {
	// Test that the URL is constructed correctly
	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the URL was constructed properly
		expectedPath := "/accounts"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Check that it's HTTPS (the client should add https://)
		if r.URL.Scheme != "" && r.URL.Scheme != "https" {
			t.Errorf("Expected HTTPS scheme, got %s", r.URL.Scheme)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer mockServer.Close()

	serverURL := strings.TrimPrefix(mockServer.URL, "https://")

	accessToken := AccessToken{
		Username: "testuser",
		Password: "testpass",
		Url:      serverURL,
	}

	client, err := NewSimpleFinClient(accessToken)
	if err != nil {
		t.Fatalf("Expected no error creating client, got %v", err)
	}

	// Use the server's client but preserve our RoundTripper for auth
	serverClient := mockServer.Client()
	client.client.Transport = &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     serverClient.Transport,
	}

	_, err = client.GetAccounts()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestSimpleFinClient_GetAccounts_WithComplexResponse(t *testing.T) {
	// Test with a more complex response including errors and multiple accounts
	mockResponse := &model.GetAccountsResponse{
		Errors: []string{"Warning: Some data may be delayed"},
		Accounts: []model.Account{
			{
				ID:               "acc1",
				Name:             "Checking Account",
				Currency:         "USD",
				Balance:          "1000.00",
				AvailableBalance: "950.00",
				BalanceDate:      1640995200,
				Org: model.Organization{
					ID:     "org1",
					Name:   "Test Bank",
					Domain: "testbank.com",
					URL:    "https://testbank.com",
				},
				Transactions: []model.Transaction{},
				Holdings:     []interface{}{},
			},
			{
				ID:               "acc2",
				Name:             "Savings Account",
				Currency:         "USD",
				Balance:          "5000.00",
				AvailableBalance: "5000.00",
				BalanceDate:      1640995200,
				Org: model.Organization{
					ID:     "org1",
					Name:   "Test Bank",
					Domain: "testbank.com",
					URL:    "https://testbank.com",
				},
				Transactions: []model.Transaction{},
				Holdings:     []interface{}{},
			},
		},
		XAPIMessage: []string{"Success", "Data retrieved"},
	}

	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer mockServer.Close()

	serverURL := strings.TrimPrefix(mockServer.URL, "https://")

	accessToken := AccessToken{
		Username: "testuser",
		Password: "testpass",
		Url:      serverURL,
	}

	client, err := NewSimpleFinClient(accessToken)
	if err != nil {
		t.Fatalf("Expected no error creating client, got %v", err)
	}

	// Use the server's client but preserve our RoundTripper for auth
	serverClient := mockServer.Client()
	client.client.Transport = &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     serverClient.Transport,
	}

	response, err := client.GetAccounts()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify complex response
	if len(response.Errors) != 1 {
		t.Errorf("Expected 1 error message, got %d", len(response.Errors))
	}

	if len(response.Accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(response.Accounts))
	}

	if len(response.XAPIMessage) != 2 {
		t.Errorf("Expected 2 API messages, got %d", len(response.XAPIMessage))
	}

	// Verify specific account details
	checkingAccount := response.Accounts[0]
	if checkingAccount.Name != "Checking Account" {
		t.Errorf("Expected first account to be 'Checking Account', got %s", checkingAccount.Name)
	}

	savingsAccount := response.Accounts[1]
	if savingsAccount.Name != "Savings Account" {
		t.Errorf("Expected second account to be 'Savings Account', got %s", savingsAccount.Name)
	}
}
