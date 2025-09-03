package aws

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSOClient(t *testing.T) {
	tests := []struct {
		name        string
		profile     string
		region      string
		setupEnv    func()
		cleanupEnv  func()
		setupConfig func() string
		wantErr     bool
	}{
		{
			name:    "Default profile and region",
			profile: "",
			region:  "",
			setupEnv: func() {
				os.Unsetenv("AWS_PROFILE")
				os.Unsetenv("AWS_REGION")
			},
			cleanupEnv: func() {},
			setupConfig: func() string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".aws", "config")
				os.MkdirAll(filepath.Dir(configPath), 0755)
				content := `[default]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = MyRole
region = us-east-1`
				os.WriteFile(configPath, []byte(content), 0644)
				os.Setenv("HOME", tmpDir)
				return tmpDir
			},
			wantErr: false,
		},
		{
			name:    "Custom profile from parameter",
			profile: "custom",
			region:  "us-west-2",
			setupEnv: func() {
				os.Unsetenv("AWS_PROFILE")
				os.Unsetenv("AWS_REGION")
			},
			cleanupEnv: func() {},
			setupConfig: func() string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".aws", "config")
				os.MkdirAll(filepath.Dir(configPath), 0755)
				content := `[profile custom]
sso_start_url = https://custom.awsapps.com/start
sso_region = us-west-2
sso_account_id = 987654321098
sso_role_name = CustomRole
region = us-west-2`
				os.WriteFile(configPath, []byte(content), 0644)
				os.Setenv("HOME", tmpDir)
				return tmpDir
			},
			wantErr: false,
		},
		{
			name:    "Profile from environment",
			profile: "",
			region:  "",
			setupEnv: func() {
				os.Setenv("AWS_PROFILE", "env-profile")
				os.Setenv("AWS_REGION", "eu-west-1")
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_PROFILE")
				os.Unsetenv("AWS_REGION")
			},
			setupConfig: func() string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".aws", "config")
				os.MkdirAll(filepath.Dir(configPath), 0755)
				content := `[profile env-profile]
sso_start_url = https://env.awsapps.com/start
sso_region = eu-west-1
sso_account_id = 111222333444
sso_role_name = EnvRole
region = eu-west-1`
				os.WriteFile(configPath, []byte(content), 0644)
				os.Setenv("HOME", tmpDir)
				return tmpDir
			},
			wantErr: false,
		},
		{
			name:    "Missing SSO configuration",
			profile: "incomplete",
			region:  "us-east-1",
			setupEnv: func() {
				os.Unsetenv("AWS_PROFILE")
				os.Unsetenv("AWS_REGION")
			},
			cleanupEnv: func() {},
			setupConfig: func() string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".aws", "config")
				os.MkdirAll(filepath.Dir(configPath), 0755)
				content := `[profile incomplete]
region = us-east-1`
				os.WriteFile(configPath, []byte(content), 0644)
				os.Setenv("HOME", tmpDir)
				return tmpDir
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalHome := os.Getenv("HOME")
			defer os.Setenv("HOME", originalHome)

			tt.setupEnv()
			defer tt.cleanupEnv()

			tmpDir := tt.setupConfig()
			defer os.RemoveAll(tmpDir)

			client, err := NewSSOClient(tt.profile, tt.region)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotEmpty(t, client.startURL)
				assert.NotEmpty(t, client.accountID)
				assert.NotEmpty(t, client.roleName)
			}
		})
	}
}

func TestSSOClient_LoadSSOConfig(t *testing.T) {
	tests := []struct {
		name           string
		profile        string
		configContent  string
		expectedConfig SSOConfig
		wantErr        bool
	}{
		{
			name:    "Valid SSO configuration",
			profile: "test",
			configContent: `[profile test]
sso_start_url = https://test.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = TestRole
region = us-east-1`,
			expectedConfig: SSOConfig{
				Profile:   "test",
				Region:    "us-east-1",
				StartURL:  "https://test.awsapps.com/start",
				AccountID: "123456789012",
				RoleName:  "TestRole",
			},
			wantErr: false,
		},
		{
			name:    "Default profile configuration",
			profile: "default",
			configContent: `[default]
sso_start_url = https://default.awsapps.com/start
sso_region = us-west-2
sso_account_id = 987654321098
sso_role_name = DefaultRole`,
			expectedConfig: SSOConfig{
				Profile:   "default",
				Region:    "us-west-2",
				StartURL:  "https://default.awsapps.com/start",
				AccountID: "987654321098",
				RoleName:  "DefaultRole",
			},
			wantErr: false,
		},
		{
			name:    "Missing required SSO fields",
			profile: "incomplete",
			configContent: `[profile incomplete]
sso_start_url = https://incomplete.awsapps.com/start
region = us-east-1`,
			expectedConfig: SSOConfig{},
			wantErr:        true,
		},
		{
			name:    "Profile not found",
			profile: "nonexistent",
			configContent: `[profile other]
sso_start_url = https://other.awsapps.com/start`,
			expectedConfig: SSOConfig{},
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, ".aws", "config")
			os.MkdirAll(filepath.Dir(configPath), 0755)
			os.WriteFile(configPath, []byte(tt.configContent), 0644)

			originalHome := os.Getenv("HOME")
			os.Setenv("HOME", tmpDir)
			defer os.Setenv("HOME", originalHome)

			client := &SSOClient{
				profile: tt.profile,
				region:  tt.expectedConfig.Region,
			}

			err := client.LoadSSOConfig()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedConfig.StartURL, client.startURL)
				assert.Equal(t, tt.expectedConfig.AccountID, client.accountID)
				assert.Equal(t, tt.expectedConfig.RoleName, client.roleName)
			}
		})
	}
}

func TestSSOClient_CheckCredentialStatus(t *testing.T) {
	// Note: This test requires mocking AWS STS calls
	// In a real implementation, you would use AWS SDK mocks or a testing framework

	t.Run("Valid credentials", func(t *testing.T) {
		// This test would require AWS credentials to be configured
		// Skipping in unit tests, would be covered in integration tests
		t.Skip("Requires AWS credentials")
	})

	t.Run("Expired credentials", func(t *testing.T) {
		// This test would require mocking expired credentials
		t.Skip("Requires AWS SDK mocking")
	})

	t.Run("No credentials", func(t *testing.T) {
		// This test would require removing all credential providers
		t.Skip("Requires AWS SDK mocking")
	})
}

func TestSSOClient_storeCachedCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	client := &SSOClient{
		profile:   "test-profile",
		region:    "us-east-1",
		startURL:  "https://test.awsapps.com/start",
		accountID: "123456789012",
		roleName:  "TestRole",
	}

	expiration := time.Now().Add(1 * time.Hour).UnixMilli()
	creds := &sso.GetRoleCredentialsOutput{
		RoleCredentials: &types.RoleCredentials{
			AccessKeyId:     aws.String("AKIATEST123456789012"),
			SecretAccessKey: aws.String("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
			SessionToken:    aws.String("FQoGZXIvYXdzEBYaDP...EXAMPLETOKEN"),
			Expiration:      expiration,
		},
	}

	err := client.storeCachedCredentials(creds)
	require.NoError(t, err)

	// Verify cache file was created
	cacheFile := filepath.Join(tmpDir, ".aws", "cli", "cache", "sso-test-profile.json")
	assert.FileExists(t, cacheFile)

	// Verify file permissions
	info, err := os.Stat(cacheFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestHelperFunctions(t *testing.T) {
	t.Run("splitLines", func(t *testing.T) {
		input := "line1\nline2\nline3"
		expected := []string{"line1", "line2", "line3"}
		result := splitLines(input)
		assert.Equal(t, expected, result)
	})

	t.Run("parseConfigLine", func(t *testing.T) {
		tests := []struct {
			input    string
			expected []string
		}{
			{"key = value", []string{"key", "value"}},
			{"key=value", []string{"key", "value"}},
			{"  key  =  value  ", []string{"key", "value"}},
			{"no_equals_sign", nil},
		}

		for _, tt := range tests {
			result := parseConfigLine(tt.input)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("trim", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"  hello  ", "hello"},
			{"\thello\t", "hello"},
			{"hello", "hello"},
			{"  ", ""},
		}

		for _, tt := range tests {
			result := trim(tt.input)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("contains", func(t *testing.T) {
		assert.True(t, contains("hello world", "world"))
		assert.True(t, contains("hello world", "hello"))
		assert.False(t, contains("hello world", "foo"))
		assert.False(t, contains("hello", "hello world"))
	})
}
