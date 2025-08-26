package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock RoundTripper for testing
type mockRoundTripper struct {
	lastRequest *http.Request
	response    *http.Response
	err         error
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.lastRequest = req
	return m.response, m.err
}

func TestSimpleFinRoundTripper_RoundTrip_Success(t *testing.T) {
	// Create a mock response
	mockResponse := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       http.NoBody,
	}

	// Create mock base transport
	mockTransport := &mockRoundTripper{
		response: mockResponse,
		err:      nil,
	}

	// Create SimpleFinRoundTripper
	rt := &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     mockTransport,
	}

	// Create a test request
	req := httptest.NewRequest("GET", "https://api.example.com/accounts", nil)

	// Execute RoundTrip
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp != mockResponse {
		t.Error("Expected to receive the mock response")
	}

	// Verify that basic auth was set on the cloned request
	if mockTransport.lastRequest == nil {
		t.Fatal("Expected mock transport to receive a request")
	}

	username, password, ok := mockTransport.lastRequest.BasicAuth()
	if !ok {
		t.Fatal("Expected basic auth to be set")
	}

	if username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", username)
	}

	if password != "testpass" {
		t.Errorf("Expected password 'testpass', got %s", password)
	}

	// Verify that the original request was not modified
	_, _, originalHasAuth := req.BasicAuth()
	if originalHasAuth {
		t.Error("Original request should not have basic auth set")
	}
}

func TestSimpleFinRoundTripper_RoundTrip_WithError(t *testing.T) {
	// Create mock base transport that returns an error
	expectedError := http.ErrUseLastResponse
	mockTransport := &mockRoundTripper{
		response: nil,
		err:      expectedError,
	}

	rt := &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     mockTransport,
	}

	req := httptest.NewRequest("GET", "https://api.example.com/accounts", nil)

	resp, err := rt.RoundTrip(req)
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}

	if resp != nil {
		t.Error("Expected nil response when error occurs")
	}

	// Verify that basic auth was still set despite the error
	if mockTransport.lastRequest == nil {
		t.Fatal("Expected mock transport to receive a request")
	}

	username, password, ok := mockTransport.lastRequest.BasicAuth()
	if !ok {
		t.Fatal("Expected basic auth to be set even when error occurs")
	}

	if username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", username)
	}

	if password != "testpass" {
		t.Errorf("Expected password 'testpass', got %s", password)
	}
}

func TestSimpleFinRoundTripper_RoundTrip_RequestCloning(t *testing.T) {
	mockTransport := &mockRoundTripper{
		response: &http.Response{StatusCode: 200, Body: http.NoBody},
		err:      nil,
	}

	rt := &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     mockTransport,
	}

	// Create a request with headers
	req := httptest.NewRequest("POST", "https://api.example.com/accounts", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "test-value")

	_, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify that the cloned request has all original headers
	clonedReq := mockTransport.lastRequest
	if clonedReq.Header.Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type header to be preserved")
	}

	if clonedReq.Header.Get("X-Custom-Header") != "test-value" {
		t.Error("Expected custom header to be preserved")
	}

	// Verify that the cloned request has the same method and URL
	if clonedReq.Method != req.Method {
		t.Errorf("Expected method %s, got %s", req.Method, clonedReq.Method)
	}

	if clonedReq.URL.String() != req.URL.String() {
		t.Errorf("Expected URL %s, got %s", req.URL.String(), clonedReq.URL.String())
	}

	// Verify that the requests are different objects (cloned, not the same)
	if clonedReq == req {
		t.Error("Expected cloned request to be a different object")
	}
}

func TestSimpleFinRoundTripper_base_WithCustomBase(t *testing.T) {
	mockTransport := &mockRoundTripper{}

	rt := &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     mockTransport,
	}

	base := rt.base()
	if base != mockTransport {
		t.Error("Expected base() to return the custom Base transport")
	}
}

func TestSimpleFinRoundTripper_base_WithDefaultTransport(t *testing.T) {
	rt := &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     nil, // No custom base
	}

	base := rt.base()
	if base != http.DefaultTransport {
		t.Error("Expected base() to return http.DefaultTransport when Base is nil")
	}
}

func TestSimpleFinRoundTripper_RoundTrip_WithDefaultTransport(t *testing.T) {
	// This test uses the default transport, so we'll create a real server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify basic auth is set
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("Expected basic auth to be set")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if username != "testuser" {
			t.Errorf("Expected username 'testuser', got %s", username)
		}

		if password != "testpass" {
			t.Errorf("Expected password 'testpass', got %s", password)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	rt := &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     nil, // Use default transport
	}

	req := httptest.NewRequest("GET", server.URL, nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	defer resp.Body.Close()
}

func TestSimpleFinRoundTripper_RoundTrip_EmptyCredentials(t *testing.T) {
	mockTransport := &mockRoundTripper{
		response: &http.Response{StatusCode: 200, Body: http.NoBody},
		err:      nil,
	}

	rt := &SimpleFinRoundTripper{
		username: "", // Empty username
		password: "", // Empty password
		Base:     mockTransport,
	}

	req := httptest.NewRequest("GET", "https://api.example.com/accounts", nil)

	_, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify that basic auth is still set, even with empty credentials
	username, password, ok := mockTransport.lastRequest.BasicAuth()
	if !ok {
		t.Fatal("Expected basic auth to be set even with empty credentials")
	}

	if username != "" {
		t.Errorf("Expected empty username, got %s", username)
	}

	if password != "" {
		t.Errorf("Expected empty password, got %s", password)
	}
}

func TestSimpleFinRoundTripper_RoundTrip_SpecialCharactersInCredentials(t *testing.T) {
	mockTransport := &mockRoundTripper{
		response: &http.Response{StatusCode: 200, Body: http.NoBody},
		err:      nil,
	}

	// Test with special characters that might need encoding
	specialUsername := "user@domain.com"
	specialPassword := "p@ss$word!"

	rt := &SimpleFinRoundTripper{
		username: specialUsername,
		password: specialPassword,
		Base:     mockTransport,
	}

	req := httptest.NewRequest("GET", "https://api.example.com/accounts", nil)

	_, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify that special characters are handled correctly in basic auth
	username, password, ok := mockTransport.lastRequest.BasicAuth()
	if !ok {
		t.Fatal("Expected basic auth to be set")
	}

	if username != specialUsername {
		t.Errorf("Expected username %s, got %s", specialUsername, username)
	}

	if password != specialPassword {
		t.Errorf("Expected password %s, got %s", specialPassword, password)
	}
}

func TestSimpleFinRoundTripper_Creation(t *testing.T) {
	rt := &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     nil,
	}

	if rt.username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", rt.username)
	}

	if rt.password != "testpass" {
		t.Errorf("Expected password 'testpass', got %s", rt.password)
	}

	if rt.Base != nil {
		t.Error("Expected Base to be nil")
	}
}

func TestSimpleFinRoundTripper_MultipleRequests(t *testing.T) {
	// Test that the RoundTripper can handle multiple requests correctly
	requestCount := 0
	mockTransport := &mockRoundTripper{
		response: &http.Response{StatusCode: 200, Body: http.NoBody},
		err:      nil,
	}

	rt := &SimpleFinRoundTripper{
		username: "testuser",
		password: "testpass",
		Base:     mockTransport,
	}

	// Make multiple requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "https://api.example.com/accounts", nil)

		_, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}

		// Verify auth is set correctly for each request
		username, password, ok := mockTransport.lastRequest.BasicAuth()
		if !ok {
			t.Fatalf("Request %d: Expected basic auth to be set", i)
		}

		if username != "testuser" {
			t.Errorf("Request %d: Expected username 'testuser', got %s", i, username)
		}

		if password != "testpass" {
			t.Errorf("Request %d: Expected password 'testpass', got %s", i, password)
		}

		requestCount++
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 requests to be made, got %d", requestCount)
	}
}
