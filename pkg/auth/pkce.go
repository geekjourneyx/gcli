package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"

	"github.com/geekjourneyx/gcli/pkg/errorsx"
)

type AuthCodeFlowParams struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scope        string
	AuthURL      string
	TokenURL     string
}

type PKCESession struct {
	Verifier  string
	Challenge string
	State     string
}

func NewPKCESession() (PKCESession, error) {
	verifier, err := randomBase64URL(64)
	if err != nil {
		return PKCESession{}, errorsx.Wrap(errorsx.CodeInternal, "generate PKCE verifier failed", false, err)
	}
	state, err := randomBase64URL(32)
	if err != nil {
		return PKCESession{}, errorsx.Wrap(errorsx.CodeInternal, "generate OAuth state failed", false, err)
	}
	challengeBytes := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes[:])
	return PKCESession{Verifier: verifier, Challenge: challenge, State: state}, nil
}

func BuildAuthURL(params AuthCodeFlowParams, sess PKCESession) (string, error) {
	if err := validateFlowParams(params); err != nil {
		return "", err
	}
	if err := validatePKCESession(sess); err != nil {
		return "", err
	}

	cfg := oauth2.Config{
		ClientID:    params.ClientID,
		RedirectURL: params.RedirectURL,
		Scopes:      []string{params.Scope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  params.AuthURL,
			TokenURL: params.TokenURL,
		},
	}

	authURL := cfg.AuthCodeURL(
		sess.State,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
		oauth2.SetAuthURLParam("code_challenge", sess.Challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	return authURL, nil
}

func ExchangeCode(ctx context.Context, params AuthCodeFlowParams, sess PKCESession, code string, client *http.Client) (*oauth2.Token, error) {
	if err := validateFlowParams(params); err != nil {
		return nil, err
	}
	if err := validatePKCESession(sess); err != nil {
		return nil, err
	}
	trimmedCode := strings.TrimSpace(code)
	if trimmedCode == "" {
		return nil, errorsx.New(errorsx.CodeInputInvalid, "authorization code is required", false)
	}

	cfg := oauth2.Config{
		ClientID:     params.ClientID,
		ClientSecret: params.ClientSecret,
		RedirectURL:  params.RedirectURL,
		Scopes:       []string{params.Scope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  params.AuthURL,
			TokenURL: params.TokenURL,
		},
	}

	ctxWithClient := ctx
	if client != nil {
		ctxWithClient = context.WithValue(ctx, oauth2.HTTPClient, client)
	}

	tok, err := cfg.Exchange(ctxWithClient, trimmedCode, oauth2.SetAuthURLParam("code_verifier", sess.Verifier))
	if err != nil {
		return nil, errorsx.Wrap(errorsx.CodeAuthCodeFlowFailed, "exchange authorization code failed", false, err)
	}

	if strings.TrimSpace(tok.RefreshToken) == "" {
		return nil, errorsx.New(errorsx.CodeAuthNoRefreshToken, "token endpoint did not return refresh_token; re-consent may be required", false)
	}

	return tok, nil
}

func validateFlowParams(params AuthCodeFlowParams) error {
	missing := make([]string, 0, 6)
	if strings.TrimSpace(params.ClientID) == "" {
		missing = append(missing, "client_id")
	}
	if strings.TrimSpace(params.ClientSecret) == "" {
		missing = append(missing, "client_secret")
	}
	if strings.TrimSpace(params.RedirectURL) == "" {
		missing = append(missing, "redirect_url")
	}
	if strings.TrimSpace(params.Scope) == "" {
		missing = append(missing, "scope")
	}
	if strings.TrimSpace(params.AuthURL) == "" {
		missing = append(missing, "auth_url")
	}
	if strings.TrimSpace(params.TokenURL) == "" {
		missing = append(missing, "token_url")
	}
	if len(missing) > 0 {
		return errorsx.New(errorsx.CodeInputInvalid, "missing auth flow params: "+strings.Join(missing, ", "), false)
	}
	return nil
}

func validatePKCESession(sess PKCESession) error {
	if strings.TrimSpace(sess.Verifier) == "" || strings.TrimSpace(sess.Challenge) == "" || strings.TrimSpace(sess.State) == "" {
		return errorsx.New(errorsx.CodeInternal, "incomplete PKCE session", false)
	}
	return nil
}

func randomBase64URL(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
