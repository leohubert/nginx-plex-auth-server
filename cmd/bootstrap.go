package cmd

import (
	"context"
	"time"

	"github.com/leohubert/nginx-plex-auth-server/pkg/envtb"
	"github.com/leohubert/nginx-plex-auth-server/pkg/logtb"
	"github.com/leohubert/nginx-plex-auth-server/pkg/plex"
	"go.uber.org/zap"
)

type Services struct {
	PlexClient *plex.Client
	Logger     *zap.Logger
}

func Bootstrap(ctx context.Context) (*Env, *Services, func()) {
	env := loadEnv()

	logger, flushLogger := logtb.NewLogger(logtb.Options{
		Format: env.LogFormat,
	})

	plexClient := plex.NewClient(plex.Options{
		BaseURL:  env.PlexURL,
		ClientID: env.PlexClientID,
		ServerID: env.PlexServerID,
	})

	ctx = logtb.InjectLogger(ctx, logger)

	services := &Services{
		PlexClient: plexClient,
		Logger:     logger,
	}

	cleanup := func() {
		flushLogger()
	}

	return env, services, cleanup
}

type Env struct {
	PlexURL      string
	PlexServerID string
	PlexClientID string
	ServerAddr   string
	LogFormat    logtb.Format
	CookieDomain string
	CookieSecure bool
	CacheTTL     time.Duration
	CacheMaxSize int64
}

func loadEnv() *Env {
	envtb.LoadEnvFile(".env")

	return &Env{
		PlexURL:      envtb.GetString("PLEX_URL", "https://plex.tv"),
		PlexServerID: envtb.GetString("PLEX_SERVER_ID", ""),
		PlexClientID: envtb.GetString("PLEX_CLIENT_ID", "nginx-plex-auth-server"),
		ServerAddr:   envtb.GetString("SERVER_ADDR", "localhost:8080"),
		CookieDomain: envtb.GetString("COOKIE_DOMAIN", ""),
		LogFormat:    envtb.GetLogFormat("LOG_FORMAT", logtb.FormatJSON),
		CookieSecure: envtb.GetBool("COOKIE_SECURE", false),
		CacheTTL:     envtb.GetDuration("CACHE_TTL", "10s"),
		CacheMaxSize: envtb.GetInt("CACHE_MAX_SIZE", 100),
	}

}
