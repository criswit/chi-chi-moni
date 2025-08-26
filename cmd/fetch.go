package cmd

import (
	"fmt"

	"github.com/criswit/chi-chi-moni/api"
	"github.com/spf13/cobra"
)

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch account information from SimpleFIN API",
	Long: `Fetch account information from SimpleFIN API using either a setup token 
or a stored access token from AWS Secrets Manager.

This command will retrieve all accounts associated with the provided credentials,
including account balances, recent transactions, and organization details.`,
	Example: `  # Fetch using setup token
  monies fetch --setup-token "aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3JnL3NpbXBsZWZpbi9jbGFpbS8uLi4="
  
  # Fetch using stored token from Secrets Manager
  monies fetch --use-secrets --secret-name "my-simplefin-token"
  
  # Fetch with JSON output
  monies fetch --setup-token "..." --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Get access token
		accessToken, err := getAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}

		// Create SimpleFIN client
		client, err := api.NewSimpleFinClient(accessToken)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		// Get accounts
		accounts, err := client.GetAccounts()
		if err != nil {
			return fmt.Errorf("failed to get accounts: %w", err)
		}

		// Display accounts
		return displayAccounts(accounts, outputFormat)
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}
