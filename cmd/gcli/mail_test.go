package gcli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/geekjourneyx/gcli/pkg/config"
	"github.com/geekjourneyx/gcli/pkg/errorsx"
)

func TestMailSearchAcceptsPositionalQuery(t *testing.T) {
	err := runRootCommandForTest(t, []string{"mail", "search", "is:unread", "--max", "5", "--page", "tok-1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := errorsx.From(err)
	if appErr.Code != errorsx.CodeAuthMissingCreds {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func TestMailSearchRequiresQuery(t *testing.T) {
	err := runRootCommandForTest(t, []string{"mail", "search"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := errorsx.From(err)
	if appErr.Code != errorsx.CodeInputInvalid {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func TestMailSearchRejectsMixedQueryInputs(t *testing.T) {
	err := runRootCommandForTest(t, []string{"mail", "search", "is:unread", "--q", "from:boss@example.com"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := errorsx.From(err)
	if appErr.Code != errorsx.CodeInputInvalid {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func runRootCommandForTest(t *testing.T, args []string) error {
	t.Helper()
	t.Setenv(config.EnvConfigFile, filepath.Join(t.TempDir(), "missing.env"))
	t.Setenv(config.EnvClientID, "")
	t.Setenv(config.EnvClientSecret, "")
	t.Setenv(config.EnvRefreshToken, "")

	var out bytes.Buffer
	var errOut bytes.Buffer
	root, _ := NewRootCommand(IOStreams{
		In:     bytes.NewReader(nil),
		Out:    &out,
		ErrOut: &errOut,
	})
	root.SetArgs(args)
	return root.Execute()
}
