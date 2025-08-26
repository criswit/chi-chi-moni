package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// SecretsManagerClient wraps AWS Secrets Manager operations
type SecretsManagerClient struct {
	client *secretsmanager.Client
}

// NewSecretsManagerClient creates a new Secrets Manager client
func NewSecretsManagerClient(ctx context.Context) (*SecretsManagerClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &SecretsManagerClient{
		client: secretsmanager.NewFromConfig(cfg),
	}, nil
}

// NewSecretsManagerClientWithConfig creates a new Secrets Manager client with custom config
func NewSecretsManagerClientWithConfig(cfg aws.Config) *SecretsManagerClient {
	return &SecretsManagerClient{
		client: secretsmanager.NewFromConfig(cfg),
	}
}

// StoreAccessToken stores an AccessToken in AWS Secrets Manager
func (sm *SecretsManagerClient) StoreAccessToken(ctx context.Context, secretName string, token AccessToken) error {
	// Convert AccessToken to JSON
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal access token: %w", err)
	}

	input := &secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		SecretString: aws.String(string(tokenJSON)),
		Description:  aws.String("SimpleFIN Access Token for chi-chi-moni"),
	}

	_, err = sm.client.CreateSecret(ctx, input)
	if err != nil {
		// If secret already exists, try to update it
		updateInput := &secretsmanager.UpdateSecretInput{
			SecretId:     aws.String(secretName),
			SecretString: aws.String(string(tokenJSON)),
		}
		_, updateErr := sm.client.UpdateSecret(ctx, updateInput)
		if updateErr != nil {
			return fmt.Errorf("failed to create or update secret: create error: %w, update error: %v", err, updateErr)
		}
	}

	return nil
}

// RetrieveAccessToken retrieves an AccessToken from AWS Secrets Manager
func (sm *SecretsManagerClient) RetrieveAccessToken(ctx context.Context, secretName string) (AccessToken, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := sm.client.GetSecretValue(ctx, input)
	if err != nil {
		return AccessToken{}, fmt.Errorf("failed to get secret value: %w", err)
	}

	if result.SecretString == nil {
		return AccessToken{}, fmt.Errorf("secret string is nil")
	}

	var token AccessToken
	err = json.Unmarshal([]byte(*result.SecretString), &token)
	if err != nil {
		return AccessToken{}, fmt.Errorf("failed to unmarshal access token: %w", err)
	}

	return token, nil
}

// DeleteAccessToken deletes an AccessToken from AWS Secrets Manager
func (sm *SecretsManagerClient) DeleteAccessToken(ctx context.Context, secretName string) error {
	input := &secretsmanager.DeleteSecretInput{
		SecretId:                   aws.String(secretName),
		ForceDeleteWithoutRecovery: aws.Bool(true),
	}

	_, err := sm.client.DeleteSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}

// ListSecrets lists all secrets with a specific prefix
func (sm *SecretsManagerClient) ListSecrets(ctx context.Context, prefix string) ([]string, error) {
	input := &secretsmanager.ListSecretsInput{}
	if prefix != "" {
		input.Filters = []types.Filter{
			{
				Key:    types.FilterNameStringTypeName,
				Values: []string{prefix},
			},
		}
	}

	var secretNames []string
	paginator := secretsmanager.NewListSecretsPaginator(sm.client, input)

	for paginator.HasMorePages() {
		result, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, secret := range result.SecretList {
			if secret.Name != nil {
				if prefix == "" || strings.Contains(*secret.Name, prefix) {
					secretNames = append(secretNames, *secret.Name)
				}
			}
		}
	}

	return secretNames, nil
}
