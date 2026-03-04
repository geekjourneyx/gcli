package auth

import (
	"context"

	"golang.org/x/oauth2"
)

type RefreshTokenConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	TokenURL     string
}

func NewRefreshTokenSource(ctx context.Context, cfg RefreshTokenConfig) oauth2.TokenSource {
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: cfg.TokenURL,
		},
	}
	return oauthCfg.TokenSource(ctx, &oauth2.Token{RefreshToken: cfg.RefreshToken})
}
