package auth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/geekjourneyx/gcli/pkg/errorsx"
)

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestNewPKCESession(t *testing.T) {
	sess, err := NewPKCESession()
	if err != nil {
		t.Fatalf("NewPKCESession() error = %v", err)
	}
	if sess.Verifier == "" || sess.Challenge == "" || sess.State == "" {
		t.Fatalf("invalid PKCE session: %+v", sess)
	}
}

func TestBuildAuthURL(t *testing.T) {
	sess, err := NewPKCESession()
	if err != nil {
		t.Fatalf("NewPKCESession() error = %v", err)
	}

	url, err := BuildAuthURL(AuthCodeFlowParams{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURL:  "http://127.0.0.1:8787/callback",
		Scope:        "scope-a",
		AuthURL:      "https://auth.example/authorize",
		TokenURL:     "https://auth.example/token",
	}, sess)
	if err != nil {
		t.Fatalf("BuildAuthURL() error = %v", err)
	}

	for _, key := range []string{"code_challenge=", "code_challenge_method=S256", "access_type=offline", "prompt=consent", "state="} {
		if !strings.Contains(url, key) {
			t.Fatalf("auth URL missing %q: %s", key, url)
		}
	}
}

func TestExchangeCode(t *testing.T) {
	sess, err := NewPKCESession()
	if err != nil {
		t.Fatalf("NewPKCESession() error = %v", err)
	}

	var gotBody string
	client := &http.Client{
		Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			gotBody = string(body)
			_ = req.Body.Close()
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(`{
					"access_token":"acc",
					"refresh_token":"ref",
					"token_type":"Bearer",
					"expires_in":3600
				}`)),
			}, nil
		}),
	}

	tok, err := ExchangeCode(context.Background(), AuthCodeFlowParams{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURL:  "http://127.0.0.1:8787/callback",
		Scope:        "scope-a",
		AuthURL:      "https://auth.example/authorize",
		TokenURL:     "https://auth.example/token",
	}, sess, "auth-code", client)
	if err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}
	if tok.RefreshToken != "ref" {
		t.Fatalf("unexpected refresh token: %q", tok.RefreshToken)
	}
	if !strings.Contains(gotBody, "code_verifier=") {
		t.Fatalf("request body missing code_verifier: %s", gotBody)
	}
}

func TestExchangeCodeMissingRefreshToken(t *testing.T) {
	sess, err := NewPKCESession()
	if err != nil {
		t.Fatalf("NewPKCESession() error = %v", err)
	}

	client := &http.Client{
		Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"acc","token_type":"Bearer","expires_in":3600}`)),
			}, nil
		}),
	}

	_, err = ExchangeCode(context.Background(), AuthCodeFlowParams{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURL:  "http://127.0.0.1:8787/callback",
		Scope:        "scope-a",
		AuthURL:      "https://auth.example/authorize",
		TokenURL:     "https://auth.example/token",
	}, sess, "auth-code", client)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errorsx.From(err).Code != errorsx.CodeAuthNoRefreshToken {
		t.Fatalf("unexpected error code: %s", errorsx.From(err).Code)
	}
}
