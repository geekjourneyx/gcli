package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadStartupEnvFile_WithExplicitPath(t *testing.T) {
	tmp := t.TempDir()
	envFile := filepath.Join(tmp, "gcli.env")
	content := "" +
		"# comments are ignored\n" +
		"GCLI_GMAIL_CLIENT_ID=file-id\n" +
		"export GCLI_GMAIL_CLIENT_SECRET=\"file-secret\"\n" +
		"GCLI_GMAIL_REFRESH_TOKEN='file-refresh'\n" +
		"OTHER_VAR=ignored\n"
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	t.Setenv(EnvConfigFile, envFile)
	t.Setenv(EnvClientID, "from-env")
	t.Setenv(EnvClientSecret, "")
	t.Setenv(EnvRefreshToken, "")

	if err := LoadStartupEnvFile(); err != nil {
		t.Fatalf("LoadStartupEnvFile() error: %v", err)
	}

	if got := os.Getenv(EnvClientID); got != "from-env" {
		t.Fatalf("client id got=%q want=%q", got, "from-env")
	}
	if got := os.Getenv(EnvClientSecret); got != "file-secret" {
		t.Fatalf("client secret got=%q want=%q", got, "file-secret")
	}
	if got := os.Getenv(EnvRefreshToken); got != "file-refresh" {
		t.Fatalf("refresh token got=%q want=%q", got, "file-refresh")
	}
	if got := os.Getenv("OTHER_VAR"); got != "" {
		t.Fatalf("OTHER_VAR should not be loaded, got=%q", got)
	}
}

func TestLoadStartupEnvFile_MissingFileNoError(t *testing.T) {
	t.Setenv(EnvConfigFile, filepath.Join(t.TempDir(), "does-not-exist"))
	if err := LoadStartupEnvFile(); err != nil {
		t.Fatalf("LoadStartupEnvFile() expected nil, got %v", err)
	}
}

func TestLoadStartupEnvFile_InvalidEntry(t *testing.T) {
	tmp := t.TempDir()
	envFile := filepath.Join(tmp, "bad.env")
	if err := os.WriteFile(envFile, []byte("GCLI_GMAIL_CLIENT_ID\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	t.Setenv(EnvConfigFile, envFile)

	if err := LoadStartupEnvFile(); err == nil {
		t.Fatal("LoadStartupEnvFile() expected error, got nil")
	}
}
