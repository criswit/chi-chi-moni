package model

import "time"

// Organization represents a financial institution
type Organization struct {
	Domain  string `json:"domain"`
	Name    string `json:"name"`
	SfinURL string `json:"sfin-url"`
	URL     string `json:"url"`
	ID      string `json:"id"`
}

// Transaction represents a financial transaction
type Transaction struct {
	ID           string `json:"id"`
	Posted       int64  `json:"posted"`
	Amount       string `json:"amount"`
	Description  string `json:"description"`
	Payee        string `json:"payee"`
	Memo         string `json:"memo"`
	TransactedAt int64  `json:"transacted_at"`
}

// Account represents a financial account
type Account struct {
	Org              Organization  `json:"org"`
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Currency         string        `json:"currency"`
	Balance          string        `json:"balance"`
	AvailableBalance string        `json:"available-balance"`
	BalanceDate      int64         `json:"balance-date"`
	Transactions     []Transaction `json:"transactions"`
	Holdings         []interface{} `json:"holdings"` // Empty array in the data, using interface{} for flexibility
}

// GetAccountsResponse represents the complete API response
type GetAccountsResponse struct {
	Errors      []string  `json:"errors"`
	Accounts    []Account `json:"accounts"`
	XAPIMessage []string  `json:"x-api-message"`
}

// Helper methods for working with timestamps
func (t *Transaction) PostedTime() time.Time {
	return time.Unix(t.Posted, 0)
}

func (t *Transaction) TransactedTime() time.Time {
	return time.Unix(t.TransactedAt, 0)
}

func (a *Account) BalanceTime() time.Time {
	return time.Unix(a.BalanceDate, 0)
}
