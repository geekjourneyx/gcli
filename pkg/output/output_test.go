package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/geekjourneyx/gcli/pkg/errorsx"
)

func TestRenderSuccessJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderSuccess(map[string]string{"ok": "yes"}, Options{Format: FormatJSON, Writer: &buf}); err != nil {
		t.Fatalf("RenderSuccess() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got["version"] != "v1" {
		t.Fatalf("version=%v", got["version"])
	}
	if got["error"] != nil {
		t.Fatalf("error should be nil, got=%v", got["error"])
	}
}

func TestRenderErrorJSON(t *testing.T) {
	var buf bytes.Buffer
	err := errorsx.New(errorsx.CodeInputInvalid, "bad", false)
	_ = err.AddDetail("field", "limit")
	if renderErr := RenderError(err, Options{Format: FormatJSON, Writer: &buf}); renderErr != nil {
		t.Fatalf("RenderError() error = %v", renderErr)
	}

	var got map[string]any
	if unmarshalErr := json.Unmarshal(buf.Bytes(), &got); unmarshalErr != nil {
		t.Fatalf("unmarshal output: %v", unmarshalErr)
	}
	if got["data"] != nil {
		t.Fatalf("data should be nil on error")
	}
	errObj, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("error envelope type mismatch: %T", got["error"])
	}
	details, ok := errObj["details"].(map[string]any)
	if !ok {
		t.Fatalf("details type mismatch: %T", errObj["details"])
	}
	if details["field"] != "limit" {
		t.Fatalf("unexpected details: %v", details)
	}
}
