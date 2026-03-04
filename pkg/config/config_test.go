package config

import "testing"

func TestLoadRuntimeConfigFromEnv(t *testing.T) {
	t.Setenv(EnvClientID, "id")
	t.Setenv(EnvClientSecret, "secret")
	t.Setenv(EnvRefreshToken, "refresh")
	t.Setenv(EnvTokenURL, "https://example.com/token")

	cfg, err := LoadRuntimeConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadRuntimeConfigFromEnv error: %v", err)
	}
	if cfg.ClientID != "id" || cfg.ClientSecret != "secret" || cfg.RefreshToken != "refresh" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.TokenURL != "https://example.com/token" {
		t.Fatalf("unexpected token url: %s", cfg.TokenURL)
	}
}

func TestLoadRuntimeConfigMissing(t *testing.T) {
	t.Setenv(EnvClientID, "")
	t.Setenv(EnvClientSecret, "")
	t.Setenv(EnvRefreshToken, "")

	if _, err := LoadRuntimeConfigFromEnv(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveAuthURL(t *testing.T) {
	t.Setenv(EnvAuthURL, "")
	if got := ResolveAuthURL(""); got != DefaultAuthURL {
		t.Fatalf("ResolveAuthURL()=%q want=%q", got, DefaultAuthURL)
	}

	t.Setenv(EnvAuthURL, "https://env.example/auth")
	if got := ResolveAuthURL(""); got != "https://env.example/auth" {
		t.Fatalf("ResolveAuthURL() from env=%q", got)
	}

	if got := ResolveAuthURL("https://flag.example/auth"); got != "https://flag.example/auth" {
		t.Fatalf("ResolveAuthURL() from flag=%q", got)
	}
}
