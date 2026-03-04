package gcli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/your-org/gcli/pkg/auth"
	"github.com/your-org/gcli/pkg/config"
	"github.com/your-org/gcli/pkg/errorsx"
	"github.com/your-org/gcli/pkg/model"
	"github.com/your-org/gcli/pkg/output"
)

const defaultRedirectURI = "http://127.0.0.1:8787/callback"

func newAuthCommand(state *State, streams IOStreams) *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication helpers",
	}
	authCmd.AddCommand(newAuthLoginCommand(state, streams))
	return authCmd
}

func newAuthLoginCommand(state *State, streams IOStreams) *cobra.Command {
	var (
		clientID     string
		clientSecret string
		scope        string
		authURL      string
		tokenURL     string
		redirectURI  string
		listen       bool
		authCode     string
		authTimeout  time.Duration
		printEnv     bool
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Run OAuth Authorization Code + PKCE and return refresh token",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := commandContext(cmd, authTimeout)
			defer cancel()

			resolvedClientID, resolvedClientSecret, err := config.ResolveClientCredentials(clientID, clientSecret)
			if err != nil {
				return err
			}
			resolvedScope := strings.TrimSpace(scope)
			if resolvedScope == "" {
				resolvedScope = config.DefaultScope
			}
			resolvedAuthURL := config.ResolveAuthURL(authURL)
			resolvedTokenURL := config.ResolveTokenURL(tokenURL)
			resolvedRedirectURI := strings.TrimSpace(redirectURI)
			if resolvedRedirectURI == "" {
				resolvedRedirectURI = defaultRedirectURI
			}

			flowParams := auth.AuthCodeFlowParams{
				ClientID:     resolvedClientID,
				ClientSecret: resolvedClientSecret,
				RedirectURL:  resolvedRedirectURI,
				Scope:        resolvedScope,
				AuthURL:      resolvedAuthURL,
				TokenURL:     resolvedTokenURL,
			}

			sess, err := auth.NewPKCESession()
			if err != nil {
				return err
			}

			authorizeURL, err := auth.BuildAuthURL(flowParams, sess)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(streams.ErrOut, "在浏览器打开以下授权链接:\n%s\n\n", authorizeURL)
			_, _ = fmt.Fprintf(streams.ErrOut, "如果你在本地浏览器操作云服务器 CLI，请先建立 SSH 隧道:\n")
			_, _ = fmt.Fprintf(streams.ErrOut, "ssh -L 8787:127.0.0.1:8787 <user>@<server>\n\n")

			finalCode := strings.TrimSpace(authCode)
			switch {
			case finalCode != "":
				// Use pre-provided code.
			case listen:
				_, _ = fmt.Fprintf(streams.ErrOut, "等待 OAuth 回调: %s\n", resolvedRedirectURI)
				callbackCode, callbackErr := waitForAuthCode(ctx, streams, resolvedRedirectURI, sess.State)
				if callbackErr != nil {
					return callbackErr
				}
				finalCode = callbackCode
			default:
				manualCode, manualErr := readAuthCodeFromInput(streams)
				if manualErr != nil {
					return manualErr
				}
				finalCode = manualCode
			}

			tok, err := auth.ExchangeCode(ctx, flowParams, sess, finalCode, http.DefaultClient)
			if err != nil {
				return err
			}

			data := model.AuthLoginData{
				RefreshToken: tok.RefreshToken,
				AccessToken:  tok.AccessToken,
				TokenType:    tok.TokenType,
				Scope:        tokenScope(tok, resolvedScope),
			}
			if !tok.Expiry.IsZero() {
				data.ExpiresAt = tok.Expiry.UTC().Format(time.RFC3339)
			}
			if printEnv {
				data.Env = map[string]string{
					config.EnvClientID:     resolvedClientID,
					config.EnvClientSecret: resolvedClientSecret,
					config.EnvRefreshToken: tok.RefreshToken,
				}
			}

			return output.RenderSuccess(data, output.Options{Format: state.OutputFormat, Writer: streams.Out})
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID (falls back to env)")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth client secret (falls back to env)")
	cmd.Flags().StringVar(&scope, "scope", config.DefaultScope, "OAuth scope")
	cmd.Flags().StringVar(&authURL, "auth-url", "", "OAuth authorization endpoint")
	cmd.Flags().StringVar(&tokenURL, "token-url", "", "OAuth token endpoint")
	cmd.Flags().StringVar(&redirectURI, "redirect-uri", defaultRedirectURI, "OAuth redirect URI (must match Google console config)")
	cmd.Flags().BoolVar(&listen, "listen", true, "Listen on redirect URI and auto-capture callback code")
	cmd.Flags().StringVar(&authCode, "code", "", "Authorization code if already captured")
	cmd.Flags().DurationVar(&authTimeout, "auth-timeout", 10*time.Minute, "Timeout for interactive OAuth login")
	cmd.Flags().BoolVar(&printEnv, "print-env", false, "Include env var mapping in output")

	return cmd
}

func waitForAuthCode(ctx context.Context, streams IOStreams, redirectURI, expectedState string) (string, error) {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return "", errorsx.Wrap(errorsx.CodeInputInvalid, "invalid redirect URI", false, err)
	}
	if !strings.EqualFold(parsed.Scheme, "http") {
		return "", errorsx.New(errorsx.CodeInputInvalid, "redirect URI scheme must be http for local callback", false)
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", errorsx.New(errorsx.CodeInputInvalid, "redirect URI host is required", false)
	}

	listenAddr := parsed.Host
	if _, _, splitErr := net.SplitHostPort(listenAddr); splitErr != nil {
		listenAddr = net.JoinHostPort(listenAddr, "80")
	}
	callbackPath := parsed.Path
	if callbackPath == "" {
		callbackPath = "/"
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return "", errorsx.Wrap(errorsx.CodeAuthCodeFlowFailed, "start callback listener failed", false, err)
	}

	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)
	serveErrCh := make(chan error, 1)
	sendResult := func(r callbackResult) {
		select {
		case resultCh <- r:
		default:
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if oauthErr := strings.TrimSpace(q.Get("error")); oauthErr != "" {
			desc := strings.TrimSpace(q.Get("error_description"))
			if desc != "" {
				sendResult(callbackResult{err: errorsx.New(errorsx.CodeAuthCodeFlowFailed, "OAuth authorize failed: "+desc, false)})
			} else {
				sendResult(callbackResult{err: errorsx.New(errorsx.CodeAuthCodeFlowFailed, "OAuth authorize failed: "+oauthErr, false)})
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Authorization failed. Return to terminal for details."))
			return
		}

		state := strings.TrimSpace(q.Get("state"))
		if state != expectedState {
			sendResult(callbackResult{err: errorsx.New(errorsx.CodeAuthStateMismatch, "OAuth state mismatch", false)})
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("State mismatch. Return to terminal."))
			return
		}

		code := strings.TrimSpace(q.Get("code"))
		if code == "" {
			sendResult(callbackResult{err: errorsx.New(errorsx.CodeInputInvalid, "callback missing authorization code", false)})
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Missing authorization code."))
			return
		}

		sendResult(callbackResult{code: code})
		_, _ = w.Write([]byte("Authorization succeeded. You can close this page and return to terminal."))
	})

	srv := &http.Server{Handler: mux}
	go func() {
		serveErrCh <- srv.Serve(ln)
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		_ = ln.Close()
	}()

	select {
	case <-ctx.Done():
		return "", errorsx.From(ctx.Err())
	case serveErr := <-serveErrCh:
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return "", errorsx.Wrap(errorsx.CodeAuthCodeFlowFailed, "callback server stopped unexpectedly", false, serveErr)
		}
		return "", errorsx.New(errorsx.CodeAuthCodeFlowFailed, "callback server stopped before receiving code", false)
	case result := <-resultCh:
		return result.code, result.err
	}
}

func readAuthCodeFromInput(streams IOStreams) (string, error) {
	_, _ = fmt.Fprintln(streams.ErrOut, "请粘贴回调 URL（包含 code=...）或直接粘贴授权 code，然后回车:")
	reader := bufio.NewReader(streams.In)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return "", errorsx.Wrap(errorsx.CodeInputInvalid, "read authorization code failed", false, err)
		}
	}
	return extractAuthCode(line)
}

func tokenScope(tok interface{ Extra(string) any }, fallback string) string {
	scope, _ := tok.Extra("scope").(string)
	scope = strings.TrimSpace(scope)
	if scope != "" {
		return scope
	}
	return strings.TrimSpace(fallback)
}

func extractAuthCode(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errorsx.New(errorsx.CodeInputInvalid, "authorization code input is empty", false)
	}

	if strings.Contains(trimmed, "://") {
		if parsed, err := url.Parse(trimmed); err == nil {
			if code := strings.TrimSpace(parsed.Query().Get("code")); code != "" {
				return code, nil
			}
		}
	}

	if strings.Contains(trimmed, "code=") {
		if idx := strings.Index(trimmed, "code="); idx >= 0 {
			candidate := trimmed[idx+len("code="):]
			if amp := strings.Index(candidate, "&"); amp >= 0 {
				candidate = candidate[:amp]
			}
			candidate = strings.TrimSpace(candidate)
			if candidate != "" {
				decoded, decodeErr := url.QueryUnescape(candidate)
				if decodeErr == nil && strings.TrimSpace(decoded) != "" {
					return strings.TrimSpace(decoded), nil
				}
				return candidate, nil
			}
		}
	}

	return trimmed, nil
}
