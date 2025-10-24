package cmd

import (
	"sync"

	"github.com/leohubert/nginx-plex-auth-server/internal/cache"
	"github.com/leohubert/nginx-plex-auth-server/internal/server"
	"github.com/leohubert/nginx-plex-auth-server/pkg/ostb"
	"go.uber.org/zap"
)

func ApiCmd(env *Env, services *Services) {

	cacheClient := cache.NewCacheClient(cache.Options{
		TTL:     env.CacheTTL,
		MaxSize: int(env.CacheMaxSize),
	})

	httpServer := server.NewServer(server.Options{
		Logger:       services.Logger,
		ListenAddr:   env.ServerAddr,
		PlexClient:   services.PlexClient,
		CacheClient:  cacheClient,
		CookieDomain: env.CookieDomain,
		CookieSecure: env.CookieSecure,
	})

	wg := &sync.WaitGroup{}
	defer wg.Wait()
	start := func(f interface{ Start() error }) {
		wg.Add(1)
		defer wg.Done()
		err := f.Start()
		if err != nil {
			services.Logger.Fatal("failed to start service", zap.Error(err))
		}
	}

	go start(httpServer)
	defer httpServer.Stop()

	// Wait for signal to start graceful shutdown
	ostb.WaitForStopSignal()
	services.Logger.Sugar().Infof("Shutting down server")
}
