package cmd

import (
	"fmt"
	"strings"

	"github.com/criswit/chi-chi-moni/api"
	"github.com/spf13/cobra"
)

// secretsCmd represents the secrets command
var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secrets in AWS Secrets Manager",
	Long: `Manage secrets stored in AWS Secrets Manager.

This command provides subcommands to list, delete, and inspect secrets
stored in AWS Secrets Manager.`,
}

// listSecretsCmd represents the list secrets command
var listSecretsCmd = &cobra.Command{
	Use:   "list",
	Short: "List secrets in AWS Secrets Manager",
	Long: `List all secrets in AWS Secrets Manager, optionally filtered by prefix.

By default, this will show all secrets that contain 'chi-chi-moni' in their name.`,
	Example: `  # List all chi-chi-moni related secrets
  monies secrets list
  
  # List all secrets
  monies secrets list --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Create Secrets Manager client
		sm, err := api.NewSecretsManagerClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create Secrets Manager client: %w", err)
		}

		// Determine prefix
		prefix := ""
		if !listAll {
			prefix = "chi-chi-moni"
		}

		// List secrets
		secrets, err := sm.ListSecrets(ctx, prefix)
		if err != nil {
			return fmt.Errorf("failed to list secrets: %w", err)
		}

		if len(secrets) == 0 {
			if prefix != "" {
				fmt.Printf("No secrets found with prefix '%s'\n", prefix)
			} else {
				fmt.Printf("No secrets found\n")
			}
			return nil
		}

		fmt.Printf("Found %d secret(s):\n", len(secrets))
		for i, secret := range secrets {
			fmt.Printf("%d. %s\n", i+1, secret)
		}

		return nil
	},
}

// deleteSecretCmd represents the delete secret command
var deleteSecretCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a secret from AWS Secrets Manager",
	Long: `Delete a secret from AWS Secrets Manager.

This will permanently delete the secret and cannot be undone.`,
	Example: `  # Delete a specific secret
  monies secrets delete --secret-name "my-simplefin-token"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if secretName == "" {
			return fmt.Errorf("secret name is required")
		}

		// Confirm deletion unless --force is used
		if !forceDelete {
			fmt.Printf("Are you sure you want to delete secret '%s'? This cannot be undone. (y/N): ", secretName)
			var response string
			fmt.Scanln(&response)
			response = strings.ToLower(strings.TrimSpace(response))
			if response != "y" && response != "yes" {
				fmt.Println("Deletion cancelled.")
				return nil
			}
		}

		// Create Secrets Manager client
		sm, err := api.NewSecretsManagerClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create Secrets Manager client: %w", err)
		}

		// Delete the secret
		err = sm.DeleteAccessToken(ctx, secretName)
		if err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}

		fmt.Printf("✅ Successfully deleted secret: %s\n", secretName)

		return nil
	},
}

var (
	listAll     bool
	forceDelete bool
)

func init() {
	rootCmd.AddCommand(secretsCmd)
	secretsCmd.AddCommand(listSecretsCmd)
	secretsCmd.AddCommand(deleteSecretCmd)

	// Flags for list command
	listSecretsCmd.Flags().BoolVar(&listAll, "all", false, "List all secrets (not just chi-chi-moni related)")

	// Flags for delete command
	deleteSecretCmd.Flags().BoolVar(&forceDelete, "force", false, "Force deletion without confirmation")
}
