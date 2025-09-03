package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/criswit/chi-chi-moni/api"
)

// SecretsManagerClient wraps AWS Secrets Manager operations
type SecretsManagerClient struct {
	client    *secretsmanager.Client
	ssoClient *SSOClient
	config    aws.Config
}

// NewSecretsManagerClient creates a new Secrets Manager client
func NewSecretsManagerClient(ctx context.Context) (*SecretsManagerClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &SecretsManagerClient{
		client: secretsmanager.NewFromConfig(cfg),
		config: cfg,
	}, nil
}

// NewSecretsManagerClientWithConfig creates a new Secrets Manager client with custom config
func NewSecretsManagerClientWithConfig(cfg aws.Config) *SecretsManagerClient {
	return &SecretsManagerClient{
		client: secretsmanager.NewFromConfig(cfg),
		config: cfg,
	}
}

// StoreAccessToken stores an AccessToken in AWS Secrets Manager
func (sm *SecretsManagerClient) StoreAccessToken(ctx context.Context, secretName string, token api.AccessToken) error {
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
func (sm *SecretsManagerClient) RetrieveAccessToken(ctx context.Context, secretName string) (api.AccessToken, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := sm.client.GetSecretValue(ctx, input)
	if err != nil {
		return api.AccessToken{}, fmt.Errorf("failed to get secret value: %w", err)
	}

	if result.SecretString == nil {
		return api.AccessToken{}, fmt.Errorf("secret string is nil")
	}

	var token api.AccessToken
	err = json.Unmarshal([]byte(*result.SecretString), &token)
	if err != nil {
		return api.AccessToken{}, fmt.Errorf("failed to unmarshal access token: %w", err)
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

// NewSecretsManagerClientWithSSO creates a new Secrets Manager client with SSO support
func NewSecretsManagerClientWithSSO(ctx context.Context, ssoClient *SSOClient) (*SecretsManagerClient, error) {
	// Check credential status
	status, err := ssoClient.CheckCredentialStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check credential status: %w", err)
	}

	// If credentials are expired or not found, initiate SSO login
	if status == CredentialStatusExpired || status == CredentialStatusNotFound {
		fmt.Println("AWS credentials expired or not found. Initiating SSO login...")
		authResult, err := ssoClient.InitiateLoginFlow(ctx)
		if err != nil {
			return nil, fmt.Errorf("SSO login failed: %w", err)
		}
		if !authResult.Success {
			return nil, fmt.Errorf("SSO authentication failed: %w", authResult.Error)
		}

		return &SecretsManagerClient{
			client:    secretsmanager.NewFromConfig(authResult.Config),
			ssoClient: ssoClient,
			config:    authResult.Config,
		}, nil
	}

	// Credentials are valid, create config with SSO
	cfg, err := ssoClient.CreateConfigWithSSO(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create config with SSO: %w", err)
	}

	return &SecretsManagerClient{
		client:    secretsmanager.NewFromConfig(cfg),
		ssoClient: ssoClient,
		config:    cfg,
	}, nil
}

// ValidateCredentials checks if current AWS credentials are valid
func (sm *SecretsManagerClient) ValidateCredentials(ctx context.Context) error {
	stsClient := sts.NewFromConfig(sm.config)
	_, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		// If SSO client is available, try to refresh credentials
		if sm.ssoClient != nil {
			status, checkErr := sm.ssoClient.CheckCredentialStatus(ctx)
			if checkErr == nil && (status == CredentialStatusExpired || status == CredentialStatusNotFound) {
				authResult, loginErr := sm.ssoClient.InitiateLoginFlow(ctx)
				if loginErr == nil && authResult.Success {
					// Update the client with new credentials
					sm.config = authResult.Config
					sm.client = secretsmanager.NewFromConfig(authResult.Config)
					return nil
				}
			}
		}
		return fmt.Errorf("invalid AWS credentials: %w", err)
	}
	return nil
}
