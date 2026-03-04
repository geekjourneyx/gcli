package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/geekjourneyx/gcli/pkg/errorsx"
)

const (
	EnvClientID      = "GCLI_GMAIL_CLIENT_ID"
	EnvClientSecret  = "GCLI_GMAIL_CLIENT_SECRET"
	EnvRefreshToken  = "GCLI_GMAIL_REFRESH_TOKEN"
	EnvTokenURL      = "GCLI_GMAIL_TOKEN_URL"
	EnvAuthURL       = "GCLI_GMAIL_AUTH_URL"
	EnvAPIEndpoint   = "GCLI_GMAIL_API_ENDPOINT"
	EnvDeviceCodeURL = "GCLI_GMAIL_DEVICE_CODE_URL"

	DefaultTokenURL      = "https://oauth2.googleapis.com/token"
	DefaultAuthURL       = "https://accounts.google.com/o/oauth2/v2/auth"
	DefaultDeviceCodeURL = "https://oauth2.googleapis.com/device/code"
	DefaultScope         = "https://www.googleapis.com/auth/gmail.readonly"
)

// RuntimeConfig is used by mail commands.
type RuntimeConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	TokenURL     string
	APIEndpoint  string
}

func LoadRuntimeConfigFromEnv() (RuntimeConfig, error) {
	cfg := RuntimeConfig{
		ClientID:     strings.TrimSpace(os.Getenv(EnvClientID)),
		ClientSecret: strings.TrimSpace(os.Getenv(EnvClientSecret)),
		RefreshToken: strings.TrimSpace(os.Getenv(EnvRefreshToken)),
		TokenURL:     strings.TrimSpace(os.Getenv(EnvTokenURL)),
		APIEndpoint:  strings.TrimSpace(os.Getenv(EnvAPIEndpoint)),
	}
	if cfg.TokenURL == "" {
		cfg.TokenURL = DefaultTokenURL
	}

	missing := make([]string, 0, 3)
	if cfg.ClientID == "" {
		missing = append(missing, EnvClientID)
	}
	if cfg.ClientSecret == "" {
		missing = append(missing, EnvClientSecret)
	}
	if cfg.RefreshToken == "" {
		missing = append(missing, EnvRefreshToken)
	}
	if len(missing) > 0 {
		return RuntimeConfig{}, errorsx.New(
			errorsx.CodeAuthMissingCreds,
			fmt.Sprintf("missing required env vars: %s", strings.Join(missing, ", ")),
			false,
		)
	}

	return cfg, nil
}

func ResolveClientCredentials(clientIDFlag, clientSecretFlag string) (string, string, error) {
	clientID := strings.TrimSpace(clientIDFlag)
	if clientID == "" {
		clientID = strings.TrimSpace(os.Getenv(EnvClientID))
	}
	clientSecret := strings.TrimSpace(clientSecretFlag)
	if clientSecret == "" {
		clientSecret = strings.TrimSpace(os.Getenv(EnvClientSecret))
	}

	missing := make([]string, 0, 2)
	if clientID == "" {
		missing = append(missing, EnvClientID)
	}
	if clientSecret == "" {
		missing = append(missing, EnvClientSecret)
	}
	if len(missing) > 0 {
		return "", "", errorsx.New(
			errorsx.CodeAuthMissingCreds,
			fmt.Sprintf("missing OAuth client credentials: %s", strings.Join(missing, ", ")),
			false,
		)
	}

	return clientID, clientSecret, nil
}

func ResolveTokenURL(tokenURLFlag string) string {
	tokenURL := strings.TrimSpace(tokenURLFlag)
	if tokenURL != "" {
		return tokenURL
	}
	envTokenURL := strings.TrimSpace(os.Getenv(EnvTokenURL))
	if envTokenURL != "" {
		return envTokenURL
	}
	return DefaultTokenURL
}

func ResolveAuthURL(authURLFlag string) string {
	authURL := strings.TrimSpace(authURLFlag)
	if authURL != "" {
		return authURL
	}
	envAuthURL := strings.TrimSpace(os.Getenv(EnvAuthURL))
	if envAuthURL != "" {
		return envAuthURL
	}
	return DefaultAuthURL
}

func ResolveDeviceCodeURL(deviceCodeURLFlag string) string {
	deviceCodeURL := strings.TrimSpace(deviceCodeURLFlag)
	if deviceCodeURL != "" {
		return deviceCodeURL
	}
	envDeviceCodeURL := strings.TrimSpace(os.Getenv(EnvDeviceCodeURL))
	if envDeviceCodeURL != "" {
		return envDeviceCodeURL
	}
	return DefaultDeviceCodeURL
}
