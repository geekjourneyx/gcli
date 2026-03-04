package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/your-org/gcli/pkg/errorsx"
)

const deviceGrantType = "urn:ietf:params:oauth:grant-type:device_code"

type DeviceAuthClient struct {
	HTTPClient *http.Client
}

type DeviceAuthParams struct {
	ClientID      string
	Scope         string
	DeviceCodeURL string
}

type DeviceCode struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURL         string `json:"verification_url"`
	VerificationURLComplete string `json:"verification_url_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type PollParams struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	DeviceCode   DeviceCode
}

type DeviceToken struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scope        string
	ExpiresIn    int64
}

type oauthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (c DeviceAuthClient) Start(ctx context.Context, params DeviceAuthParams) (DeviceCode, error) {
	if strings.TrimSpace(params.ClientID) == "" {
		return DeviceCode{}, errorsx.New(errorsx.CodeInputInvalid, "client id is required", false)
	}
	if strings.TrimSpace(params.Scope) == "" {
		return DeviceCode{}, errorsx.New(errorsx.CodeInputInvalid, "scope is required", false)
	}
	if strings.TrimSpace(params.DeviceCodeURL) == "" {
		return DeviceCode{}, errorsx.New(errorsx.CodeInputInvalid, "device code URL is required", false)
	}

	client := c.httpClient()
	payload := url.Values{}
	payload.Set("client_id", params.ClientID)
	payload.Set("scope", params.Scope)
	payload.Set("access_type", "offline")
	payload.Set("prompt", "consent")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, params.DeviceCodeURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return DeviceCode{}, errorsx.Wrap(errorsx.CodeAuthDeviceFlowFailed, "build device code request failed", false, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return DeviceCode{}, errorsx.Wrap(errorsx.CodeAuthDeviceFlowFailed, "device code request failed", false, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var oauthErr oauthErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&oauthErr)
		if oauthErr.ErrorDescription == "" {
			oauthErr.ErrorDescription = oauthErr.Error
		}
		return DeviceCode{}, errorsx.New(
			errorsx.CodeAuthDeviceFlowFailed,
			fmt.Sprintf("device authorization failed: %s", strings.TrimSpace(oauthErr.ErrorDescription)),
			false,
		)
	}

	var out DeviceCode
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return DeviceCode{}, errorsx.Wrap(errorsx.CodeAuthDeviceFlowFailed, "decode device code response failed", false, err)
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	if out.ExpiresIn <= 0 {
		out.ExpiresIn = 300
	}
	return out, nil
}

func (c DeviceAuthClient) Poll(ctx context.Context, params PollParams) (DeviceToken, error) {
	if strings.TrimSpace(params.TokenURL) == "" {
		return DeviceToken{}, errorsx.New(errorsx.CodeInputInvalid, "token URL is required", false)
	}
	if strings.TrimSpace(params.DeviceCode.DeviceCode) == "" {
		return DeviceToken{}, errorsx.New(errorsx.CodeInputInvalid, "device code is required", false)
	}

	client := c.httpClient()
	deadline := time.Now().Add(time.Duration(params.DeviceCode.ExpiresIn) * time.Second)
	interval := time.Duration(params.DeviceCode.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}

	for {
		if time.Now().After(deadline) {
			return DeviceToken{}, errorsx.New(errorsx.CodeAuthDeviceFlowFailed, "device code expired before authorization", false)
		}

		payload := url.Values{}
		payload.Set("client_id", params.ClientID)
		payload.Set("client_secret", params.ClientSecret)
		payload.Set("device_code", params.DeviceCode.DeviceCode)
		payload.Set("grant_type", deviceGrantType)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, params.TokenURL, strings.NewReader(payload.Encode()))
		if err != nil {
			return DeviceToken{}, errorsx.Wrap(errorsx.CodeAuthDeviceFlowFailed, "build token polling request failed", false, err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		if err != nil {
			return DeviceToken{}, errorsx.Wrap(errorsx.CodeAuthDeviceFlowFailed, "token polling request failed", false, err)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var tok struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				TokenType    string `json:"token_type"`
				Scope        string `json:"scope"`
				ExpiresIn    int64  `json:"expires_in"`
			}
			decodeErr := json.NewDecoder(resp.Body).Decode(&tok)
			_ = resp.Body.Close()
			if decodeErr != nil {
				return DeviceToken{}, errorsx.Wrap(errorsx.CodeAuthDeviceFlowFailed, "decode token response failed", false, decodeErr)
			}
			if strings.TrimSpace(tok.RefreshToken) == "" {
				return DeviceToken{}, errorsx.New(errorsx.CodeAuthNoRefreshToken, "token endpoint did not return refresh_token", false)
			}
			return DeviceToken{
				AccessToken:  tok.AccessToken,
				RefreshToken: tok.RefreshToken,
				TokenType:    tok.TokenType,
				Scope:        tok.Scope,
				ExpiresIn:    tok.ExpiresIn,
			}, nil
		}

		var oauthErr oauthErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&oauthErr)
		_ = resp.Body.Close()

		switch oauthErr.Error {
		case "authorization_pending":
			if err := wait(ctx, interval); err != nil {
				return DeviceToken{}, err
			}
			continue
		case "slow_down":
			interval += 5 * time.Second
			if err := wait(ctx, interval); err != nil {
				return DeviceToken{}, err
			}
			continue
		case "expired_token":
			return DeviceToken{}, errorsx.New(errorsx.CodeAuthDeviceFlowFailed, "device token expired, re-run auth login", false)
		case "access_denied":
			return DeviceToken{}, errorsx.New(errorsx.CodeAuthDeviceFlowFailed, "authorization denied by user", false)
		default:
			msg := strings.TrimSpace(oauthErr.ErrorDescription)
			if msg == "" {
				msg = strings.TrimSpace(oauthErr.Error)
			}
			if msg == "" {
				msg = "unknown oauth token polling error"
			}
			return DeviceToken{}, errorsx.New(errorsx.CodeAuthDeviceFlowFailed, "token polling failed: "+msg, false)
		}
	}
}

func wait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return errorsx.From(ctx.Err())
	case <-t.C:
		return nil
	}
}

func (c DeviceAuthClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}
