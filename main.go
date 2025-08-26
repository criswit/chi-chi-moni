package main

import (
	"fmt"
	"os"

	"github.com/criswit/chi-chi-moni/api"
)

func main() {
	// Check if setup token is provided as command-line argument
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <setup-token>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3JnL3NpbXBsZWZpbi9jbGFpbS9...\n", os.Args[0])
		os.Exit(1)
	}

	setupToken := os.Args[1]

	resolver := api.NewAccessTokenResolver(setupToken)
	accessToken, err := resolver.Resolve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving access token: %v\n", err)
		os.Exit(1)
	}

	client, err := api.NewSimpleFinClient(accessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	accounts, err := client.GetAccounts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting accounts: %v\n", err)
		os.Exit(1)
	}

	// Display account information
	fmt.Printf("Found %d account(s):\n", len(accounts.Accounts))
	for i, account := range accounts.Accounts {
		fmt.Printf("%d. Account: %s\n", i+1, account.Name)
		fmt.Printf("   ID: %s\n", account.ID)
		fmt.Printf("   Balance: %s %s\n", account.Balance, account.Currency)
		fmt.Printf("   Organization: %s\n", account.Org.Name)
		if len(account.Transactions) > 0 {
			fmt.Printf("   Recent transactions: %d\n", len(account.Transactions))
		}
		fmt.Println()
	}
}
