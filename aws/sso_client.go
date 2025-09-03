package aws

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/browser"
)

type CredentialStatus int

const (
	CredentialStatusValid CredentialStatus = iota
	CredentialStatusExpired
	CredentialStatusNotFound
	CredentialStatusError
)

type SSOClient struct {
	profile    string
	region     string
	startURL   string
	roleName   string
	accountID  string
	ssoClient  *sso.Client
	oidcClient *ssooidc.Client
}

type SSOConfig struct {
	Profile   string `json:"profile"`
	Region    string `json:"region"`
	StartURL  string `json:"start_url"`
	RoleName  string `json:"role_name"`
	AccountID string `json:"account_id"`
}

type SSOAuthResult struct {
	Success   bool
	Config    aws.Config
	ExpiresAt time.Time
	Error     error
}

func NewSSOClient(profile, region string) (*SSOClient, error) {
	if profile == "" {
		profile = os.Getenv("AWS_PROFILE")
		if profile == "" {
			profile = "default"
		}
	}

	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1"
		}
	}

	client := &SSOClient{
		profile: profile,
		region:  region,
	}

	if err := client.LoadSSOConfig(); err != nil {
		return nil, fmt.Errorf("failed to load SSO config: %w", err)
	}

	// Create a basic config for OIDC operations (doesn't require credentials)
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(client.region),
	)
	if err != nil {
		// Even if we can't load config, we can still try to create clients with minimal config
		cfg = aws.Config{
			Region: client.region,
		}
	}

	client.ssoClient = sso.NewFromConfig(cfg)
	client.oidcClient = ssooidc.NewFromConfig(cfg)

	return client, nil
}

func (c *SSOClient) LoadSSOConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".aws", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("AWS config file not found at %s", configPath)
		}
		return fmt.Errorf("failed to read AWS config: %w", err)
	}

	// Parse AWS config file (simplified parsing for SSO profile)
	// In production, use a proper INI parser
	profileSection := fmt.Sprintf("[profile %s]", c.profile)
	if c.profile == "default" {
		profileSection = "[default]"
	}

	content := string(data)
	profileStart := -1
	for i, line := range splitLines(content) {
		if line == profileSection {
			profileStart = i
			break
		}
	}

	if profileStart == -1 {
		return fmt.Errorf("profile %s not found in AWS config", c.profile)
	}

	// Extract SSO configuration from profile
	lines := splitLines(content)
	var ssoSessionName string
	for i := profileStart + 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}
		if line[0] == '[' {
			break // Next profile section
		}

		if kv := parseConfigLine(line); kv != nil {
			switch kv[0] {
			case "sso_session":
				ssoSessionName = kv[1]
			case "sso_start_url":
				c.startURL = kv[1]
			case "sso_region":
				if c.region == "" {
					c.region = kv[1]
				}
			case "sso_account_id":
				c.accountID = kv[1]
			case "sso_role_name":
				c.roleName = kv[1]
			case "region":
				if c.region == "" {
					c.region = kv[1]
				}
			}
		}
	}

	// If using sso-session, look for the session configuration
	if ssoSessionName != "" {
		sessionSection := fmt.Sprintf("[sso-session %s]", ssoSessionName)
		sessionStart := -1
		for i, line := range lines {
			if line == sessionSection {
				sessionStart = i
				break
			}
		}

		if sessionStart != -1 {
			for i := sessionStart + 1; i < len(lines); i++ {
				line := lines[i]
				if line == "" {
					continue
				}
				if line[0] == '[' {
					break // Next section
				}

				if kv := parseConfigLine(line); kv != nil {
					switch kv[0] {
					case "sso_start_url":
						if c.startURL == "" {
							c.startURL = kv[1]
						}
					case "sso_region":
						if c.region == "" {
							c.region = kv[1]
						}
					}
				}
			}
		}
	}

	if c.startURL == "" || c.accountID == "" || c.roleName == "" {
		return fmt.Errorf("incomplete SSO configuration for profile %s (start_url: %s, account_id: %s, role_name: %s)",
			c.profile, c.startURL, c.accountID, c.roleName)
	}

	return nil
}

func (c *SSOClient) CheckCredentialStatus(ctx context.Context) (CredentialStatus, error) {
	// Try to load config with SSO profile
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithSharedConfigProfile(c.profile),
	)
	if err != nil {
		// If we can't load the config, credentials are likely not found or expired
		errStr := err.Error()
		if contains(errStr, "expired") || contains(errStr, "ExpiredToken") || contains(errStr, "refresh") {
			return CredentialStatusExpired, nil
		}
		if contains(errStr, "NoCredentialProviders") || contains(errStr, "no valid credential") {
			return CredentialStatusNotFound, nil
		}
		return CredentialStatusNotFound, nil // Default to not found to trigger login
	}

	stsClient := sts.NewFromConfig(cfg)
	_, err = stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})

	if err != nil {
		errStr := err.Error()
		if contains(errStr, "ExpiredToken") || contains(errStr, "TokenExpired") || contains(errStr, "InvalidGrantException") || contains(errStr, "refresh") {
			return CredentialStatusExpired, nil
		}
		if contains(errStr, "NoCredentialProviders") || contains(errStr, "no valid credential") {
			return CredentialStatusNotFound, nil
		}
		// Default to expired to trigger re-authentication
		return CredentialStatusExpired, nil
	}

	return CredentialStatusValid, nil
}

func (c *SSOClient) InitiateLoginFlow(ctx context.Context) (*SSOAuthResult, error) {
	// Register client for device authorization
	registerResp, err := c.oidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String("chi-chi-moni-cli"),
		ClientType: aws.String("public"),
	})
	if err != nil {
		return &SSOAuthResult{
			Success: false,
			Error:   fmt.Errorf("failed to register client: %w", err),
		}, nil
	}

	// Start device authorization
	startResp, err := c.oidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     registerResp.ClientId,
		ClientSecret: registerResp.ClientSecret,
		StartUrl:     aws.String(c.startURL),
	})
	if err != nil {
		return &SSOAuthResult{
			Success: false,
			Error:   fmt.Errorf("failed to start device authorization: %w", err),
		}, nil
	}

	// Open browser for user authentication
	fmt.Printf("Opening browser for SSO authentication...\n")
	fmt.Printf("Verification URL: %s\n", *startResp.VerificationUriComplete)
	fmt.Printf("User Code: %s\n", *startResp.UserCode)

	if err := browser.OpenURL(*startResp.VerificationUriComplete); err != nil {
		fmt.Printf("Failed to open browser automatically. Please visit the URL manually.\n")
	}

	// Poll for authorization completion
	fmt.Println("Waiting for authorization...")

	interval := time.Duration(startResp.Interval) * time.Second
	expiresAt := time.Now().Add(time.Duration(startResp.ExpiresIn) * time.Second)

	for time.Now().Before(expiresAt) {
		time.Sleep(interval)

		tokenResp, err := c.oidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     registerResp.ClientId,
			ClientSecret: registerResp.ClientSecret,
			DeviceCode:   startResp.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})

		if err != nil {
			if contains(err.Error(), "AuthorizationPending") {
				continue
			}
			return &SSOAuthResult{
				Success: false,
				Error:   fmt.Errorf("failed to create token: %w", err),
			}, nil
		}

		// Store SSO access token first
		expiresIn := tokenResp.ExpiresIn
		if err := c.storeSSOToken(tokenResp.AccessToken, &expiresIn); err != nil {
			fmt.Printf("Warning: failed to cache SSO token: %v\n", err)
		}

		// Get role credentials
		roleResp, err := c.ssoClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
			RoleName:    aws.String(c.roleName),
			AccountId:   aws.String(c.accountID),
			AccessToken: tokenResp.AccessToken,
		})
		if err != nil {
			return &SSOAuthResult{
				Success: false,
				Error:   fmt.Errorf("failed to get role credentials: %w", err),
			}, nil
		}

		// Store credentials in cache
		if err := c.storeCachedCredentials(roleResp); err != nil {
			return &SSOAuthResult{
				Success: false,
				Error:   fmt.Errorf("failed to cache credentials: %w", err),
			}, nil
		}

		// Create new config with the fresh credentials
		cfg, err := c.CreateConfigWithCredentials(ctx, roleResp.RoleCredentials)
		if err != nil {
			return &SSOAuthResult{
				Success: false,
				Error:   err,
			}, nil
		}

		return &SSOAuthResult{
			Success:   true,
			Config:    cfg,
			ExpiresAt: time.UnixMilli(roleResp.RoleCredentials.Expiration),
		}, nil
	}

	return &SSOAuthResult{
		Success: false,
		Error:   fmt.Errorf("device authorization expired"),
	}, nil
}

func (c *SSOClient) CreateConfigWithSSO(ctx context.Context) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithSharedConfigProfile(c.profile),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load config with SSO: %w", err)
	}

	return cfg, nil
}

func (c *SSOClient) CreateConfigWithCredentials(ctx context.Context, roleCreds *types.RoleCredentials) (aws.Config, error) {
	if roleCreds == nil || roleCreds.AccessKeyId == nil || roleCreds.SecretAccessKey == nil {
		return aws.Config{}, fmt.Errorf("invalid role credentials")
	}

	// Create static credentials provider with the SSO role credentials
	staticCreds := credentials.NewStaticCredentialsProvider(
		*roleCreds.AccessKeyId,
		*roleCreds.SecretAccessKey,
		*roleCreds.SessionToken,
	)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(staticCreds),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to create config with credentials: %w", err)
	}

	return cfg, nil
}

func (c *SSOClient) storeSSOToken(accessToken *string, expiresIn *int32) error {
	if accessToken == nil || expiresIn == nil {
		return fmt.Errorf("invalid token data")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create SSO cache directory
	ssoDir := filepath.Join(homeDir, ".aws", "sso", "cache")
	if err := os.MkdirAll(ssoDir, 0700); err != nil {
		return fmt.Errorf("failed to create SSO cache directory: %w", err)
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(*expiresIn) * time.Second)

	// Create SSO token cache structure
	tokenCache := map[string]interface{}{
		"startUrl":    c.startURL,
		"region":      c.region,
		"accessToken": *accessToken,
		"expiresAt":   expiresAt.Format(time.RFC3339),
	}

	// Generate cache filename using SHA1 hash of the start URL
	hasher := sha1.New()
	hasher.Write([]byte(c.startURL))
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	cacheFile := fmt.Sprintf("%s.json", hash)
	cachePath := filepath.Join(ssoDir, cacheFile)

	// Marshal and write the token cache
	data, err := json.MarshalIndent(tokenCache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SSO token: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write SSO token cache: %w", err)
	}

	fmt.Printf("SSO token cached successfully at: %s\n", cachePath)
	return nil
}

func (c *SSOClient) storeCachedCredentials(creds *sso.GetRoleCredentialsOutput) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".aws", "cli", "cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create cache entry
	cacheEntry := map[string]interface{}{
		"Credentials": map[string]string{
			"AccessKeyId":     *creds.RoleCredentials.AccessKeyId,
			"SecretAccessKey": *creds.RoleCredentials.SecretAccessKey,
			"SessionToken":    *creds.RoleCredentials.SessionToken,
		},
		"Expiration":   time.UnixMilli(creds.RoleCredentials.Expiration).Format(time.RFC3339),
		"ProviderType": "sso",
	}

	data, err := json.MarshalIndent(cacheEntry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	// Generate cache filename based on profile and start URL
	cacheFile := fmt.Sprintf("sso-%s.json", c.profile)
	cachePath := filepath.Join(cacheDir, cacheFile)

	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func parseConfigLine(line string) []string {
	for i := 0; i < len(line); i++ {
		if line[i] == '=' {
			key := trim(line[:i])
			value := trim(line[i+1:])
			return []string{key, value}
		}
	}
	return nil
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
