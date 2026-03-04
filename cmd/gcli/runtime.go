package gcli

import (
	"context"

	"github.com/geekjourneyx/gcli/pkg/auth"
	"github.com/geekjourneyx/gcli/pkg/config"
	"github.com/geekjourneyx/gcli/pkg/gmail"
)

func newGmailClient(ctx context.Context) (*gmail.Client, error) {
	cfg, err := config.LoadRuntimeConfigFromEnv()
	if err != nil {
		return nil, err
	}

	tokenSource := auth.NewRefreshTokenSource(ctx, auth.RefreshTokenConfig{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RefreshToken: cfg.RefreshToken,
		TokenURL:     cfg.TokenURL,
	})

	return gmail.NewClient(ctx, cfg, tokenSource)
}
