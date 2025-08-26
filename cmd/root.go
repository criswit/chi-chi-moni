package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/criswit/chi-chi-moni/api"
	"github.com/criswit/chi-chi-moni/model"
	"github.com/spf13/cobra"
)

var (
	secretName   string
	useSecrets   bool
	setupToken   string
	outputFormat string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "monies",
	Short: "A CLI tool for fetching financial account information using SimpleFIN API",
	Long: `Chi-Chi-Moni is a command-line tool for fetching financial account information 
using the SimpleFIN API. It supports secure token-based authentication and can store 
access tokens in AWS Secrets Manager for enhanced security.

The tool resolves access tokens and retrieves account data including balances, 
transactions, and organization details.`,
	Example: `  # Fetch accounts using a setup token
  monies fetch --setup-token "aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3JnL3NpbXBsZWZpbi9jbGFpbS8uLi4="
  
  # Store access token in AWS Secrets Manager
  monies store --setup-token "aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3JnL3NpbXBsZWZpbi9jbGFpbS8uLi4=" --secret-name "my-simplefin-token"
  
  # Fetch accounts using stored token from Secrets Manager
  monies fetch --use-secrets --secret-name "my-simplefin-token"
  
  # List stored secrets
  monies secrets list`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	ctx := context.Background()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&secretName, "secret-name", "", "Name of the secret in AWS Secrets Manager")
	rootCmd.PersistentFlags().BoolVar(&useSecrets, "use-secrets", false, "Use AWS Secrets Manager to retrieve access token")
	rootCmd.PersistentFlags().StringVar(&setupToken, "setup-token", "", "Base64-encoded setup token from SimpleFIN")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")
}

// getAccessToken retrieves an access token either from setup token or AWS Secrets Manager
func getAccessToken(ctx context.Context) (api.AccessToken, error) {
	if useSecrets {
		if secretName == "" {
			return api.AccessToken{}, fmt.Errorf("secret name is required when using AWS Secrets Manager")
		}

		sm, err := api.NewSecretsManagerClient(ctx)
		if err != nil {
			return api.AccessToken{}, fmt.Errorf("failed to create Secrets Manager client: %w", err)
		}

		return sm.RetrieveAccessToken(ctx, secretName)
	}

	if setupToken == "" {
		return api.AccessToken{}, fmt.Errorf("setup token is required when not using AWS Secrets Manager")
	}

	resolver := api.NewAccessTokenResolver(setupToken)
	return resolver.Resolve()
}

// displayAccounts formats and displays account information
func displayAccounts(accounts *model.GetAccountsResponse, format string) error {
	switch format {
	case "json":
		return displayAccountsJSON(accounts)
	case "table":
		return displayAccountsTable(accounts)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func displayAccountsTable(accounts *model.GetAccountsResponse) error {
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
	return nil
}

func displayAccountsJSON(accounts *model.GetAccountsResponse) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(accounts)
}
