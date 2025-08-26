package cmd

import (
	"fmt"

	"github.com/criswit/chi-chi-moni/api"
	"github.com/spf13/cobra"
)

// storeCmd represents the store command
var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Store access token in AWS Secrets Manager",
	Long: `Store an access token in AWS Secrets Manager for secure, reusable access.

This command takes a setup token, resolves it to get the access credentials,
and then stores those credentials securely in AWS Secrets Manager. Once stored,
you can use the 'fetch --use-secrets' command to retrieve accounts without
needing to provide the setup token again.`,
	Example: `  # Store token with a custom secret name
  monies store --setup-token "aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3JnL3NpbXBsZWZpbi9jbGFpbS8uLi4=" --secret-name "my-simplefin-token"
  
  # Store with default secret name
  monies store --setup-token "aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3JnL3NpbXBsZWZpbi9jbGFpbS8uLi4="`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if setupToken == "" {
			return fmt.Errorf("setup token is required for storing")
		}

		if secretName == "" {
			secretName = "chi-chi-moni-access-token"
		}

		// Resolve the setup token to get access token
		resolver := api.NewAccessTokenResolver(setupToken)
		accessToken, err := resolver.Resolve()
		if err != nil {
			return fmt.Errorf("failed to resolve setup token: %w", err)
		}

		// Create Secrets Manager client
		sm, err := api.NewSecretsManagerClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create Secrets Manager client: %w", err)
		}

		// Store the access token
		err = sm.StoreAccessToken(ctx, secretName, accessToken)
		if err != nil {
			return fmt.Errorf("failed to store access token: %w", err)
		}

		fmt.Printf("✅ Successfully stored access token in AWS Secrets Manager\n")
		fmt.Printf("   Secret name: %s\n", secretName)
		fmt.Printf("   You can now use: monies fetch --use-secrets --secret-name \"%s\"\n", secretName)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(storeCmd)
}
