package server

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/leohubert/nginx-plex-auth-server/internal/cache"
	"github.com/leohubert/nginx-plex-auth-server/pkg/plex"
	"go.uber.org/zap"
)

type Options struct {
	Logger       *zap.Logger
	PlexClient   *plex.Client
	CacheClient  *cache.Client
	ListenAddr   string
	TLSCrt       string
	TLSKey       string
	CookieDomain string
	CookieSecure bool
}

type Server struct {
	Options

	server *http.Server
}

func LoggerMiddleware(logger *zap.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			duration := time.Since(start)

			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Duration("duration", duration),
			)
		})
	}
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewServer(opts Options) *Server {
	router := mux.NewRouter()

	server := &Server{
		Options: opts,
		server: &http.Server{
			ReadHeaderTimeout: 10 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Millisecond,
			Handler:           router,
		},
	}

	router.Use(LoggerMiddleware(opts.Logger))
	router.Use(CORSMiddleware)

	// Authentication and OAuth routes
	router.Path("/").HandlerFunc(server.LoginHandler)
	router.Path("/health").HandlerFunc(server.HealthHandler)
	router.Path("/auth").HandlerFunc(server.AuthHandler)
	router.Path("/auth/generate-pin").HandlerFunc(server.GeneratePinHandler)
	router.Path("/callback").HandlerFunc(server.CallbackHandler)
	router.Path("/deleteSessionCookie").HandlerFunc(server.LogoutHandler)

	return server
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	s.Logger.Sugar().Infof("http server listening on %s", listener.Addr().String())

	if s.TLSCrt != "" && s.TLSKey != "" {
		err = s.server.ServeTLS(listener, s.TLSCrt, s.TLSKey)
	} else {
		err = s.server.Serve(listener)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop() {
	err := s.server.Close()
	if err != nil {
		panic(err)
	}
}
