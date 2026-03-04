package config

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/your-org/gcli/pkg/errorsx"
)

const (
	EnvConfigFile    = "GCLI_ENV_FILE"
	defaultEnvSubdir = ".config/gcli"
	defaultEnvFile   = "env"
)

// LoadStartupEnvFile loads defaults from an env file for this process.
// Existing environment variables are kept as-is (higher priority).
func LoadStartupEnvFile() error {
	path := strings.TrimSpace(os.Getenv(EnvConfigFile))
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return nil
		}
		path = filepath.Join(home, defaultEnvSubdir, defaultEnvFile)
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return errorsx.Wrap(errorsx.CodeInputInvalid, "open env file failed", false, err).
			AddDetail("path", path)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return errorsx.New(errorsx.CodeInputInvalid, "invalid env file entry", false).
				AddDetail("path", path).
				AddDetail("line", strconv.Itoa(lineNo))
		}
		key = strings.TrimSpace(key)
		rawValue = strings.TrimSpace(rawValue)
		if key == "" {
			return errorsx.New(errorsx.CodeInputInvalid, "invalid env file entry", false).
				AddDetail("path", path).
				AddDetail("line", strconv.Itoa(lineNo))
		}
		// Keep skill-specific scope narrow and avoid clobbering unrelated vars.
		if !strings.HasPrefix(key, "GCLI_") {
			continue
		}
		if existing, exists := os.LookupEnv(key); exists && strings.TrimSpace(existing) != "" {
			continue
		}

		value, parseErr := parseEnvValue(rawValue)
		if parseErr != nil {
			return errorsx.New(errorsx.CodeInputInvalid, "invalid env file entry value", false).
				AddDetail("path", path).
				AddDetail("line", strconv.Itoa(lineNo)).
				AddDetail("key", key)
		}
		if setErr := os.Setenv(key, value); setErr != nil {
			return errorsx.Wrap(errorsx.CodeInternal, "set env var failed", false, setErr).
				AddDetail("path", path).
				AddDetail("line", strconv.Itoa(lineNo)).
				AddDetail("key", key)
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return errorsx.Wrap(errorsx.CodeInputInvalid, "read env file failed", false, scanErr).
			AddDetail("path", path)
	}

	return nil
}

func parseEnvValue(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") {
		return strconv.Unquote(raw)
	}
	if len(raw) >= 2 && strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'") {
		return raw[1 : len(raw)-1], nil
	}
	return raw, nil
}
