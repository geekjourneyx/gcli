package e2e

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/your-org/gcli/cmd/gcli"
	"github.com/your-org/gcli/pkg/errorsx"
)

func TestVersionCommandJSON(t *testing.T) {
	stdout, stderr, err := runCLI(t, []string{"version"}, "")
	if err != nil {
		t.Fatalf("runCLI error: %v; stderr=%s", err, stderr)
	}

	var payload map[string]any
	if unmarshalErr := json.Unmarshal([]byte(stdout), &payload); unmarshalErr != nil {
		t.Fatalf("invalid JSON output: %v; stdout=%s", unmarshalErr, stdout)
	}
	if payload["version"] == nil {
		t.Fatalf("expected version field, got %v", payload)
	}
}

func TestMailListMissingCredentials(t *testing.T) {
	stdout, stderr, err := runCLI(t, []string{"mail", "list", "--limit", "1"}, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	_ = stdout
	_ = stderr

	appErr := errorsx.From(err)
	if appErr.Code != errorsx.CodeAuthMissingCreds {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func runCLI(t *testing.T, args []string, stdin string) (string, string, error) {
	t.Helper()
	var out bytes.Buffer
	var errOut bytes.Buffer

	root, _ := gcli.NewRootCommand(gcli.IOStreams{
		In:     strings.NewReader(stdin),
		Out:    &out,
		ErrOut: &errOut,
	})
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), errOut.String(), err
}
