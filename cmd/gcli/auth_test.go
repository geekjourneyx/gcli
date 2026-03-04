package gcli

import "testing"

func TestExtractAuthCode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{name: "raw code", input: "abc123", want: "abc123", ok: true},
		{name: "redirect URL", input: "http://127.0.0.1:8787/callback?code=abc123&state=x", want: "abc123", ok: true},
		{name: "url encoded code", input: "http://127.0.0.1:8787/callback?code=4%2F0AX4XfWh&state=x", want: "4/0AX4XfWh", ok: true},
		{name: "query fragment", input: "code=abc123&scope=gmail", want: "abc123", ok: true},
		{name: "empty", input: "   ", want: "", ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractAuthCode(tc.input)
			if tc.ok && err != nil {
				t.Fatalf("extractAuthCode() unexpected error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("extractAuthCode() expected error, got nil")
			}
			if tc.ok && got != tc.want {
				t.Fatalf("extractAuthCode()=%q want=%q", got, tc.want)
			}
		})
	}
}
