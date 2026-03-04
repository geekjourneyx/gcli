package errorsx

import "testing"

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "input", err: New(CodeInputInvalid, "bad input", false), want: 2},
		{name: "auth", err: New(CodeAuthMissingCreds, "missing", false), want: 3},
		{name: "not found", err: New(CodeMailNotFound, "missing", false), want: 4},
		{name: "quota", err: New(CodeGmailAPIQuota, "quota", true), want: 5},
		{name: "internal", err: New(CodeInternal, "internal", false), want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Fatalf("ExitCode()=%d want=%d", got, tc.want)
			}
		})
	}
}
