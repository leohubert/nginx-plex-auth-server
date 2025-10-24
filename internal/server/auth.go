package server

import (
	"net/http"

	"github.com/leohubert/nginx-plex-auth-server/internal/cache"
)

func (s *Server) AuthHandler(res http.ResponseWriter, req *http.Request) {

	authToken := s.getSessionCookie(req)
	if authToken == "" {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	if entry, found := s.CacheClient.Get(authToken); found {
		if entry.Valid && entry.HasAccess {
			res.WriteHeader(http.StatusOK)
			return
		}

		res.WriteHeader(http.StatusForbidden)
		return
	}

	authorized := false
	defer func() {
		if !authorized {
			s.CacheClient.Set(authToken, &cache.TokenCacheEntry{
				Valid:     false,
				HasAccess: false,
			})
		}
	}()

	user, err := s.PlexClient.GetUserInfo(authToken)
	if user == nil || err != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	access, err := s.PlexClient.CheckServerAccess(authToken)
	if !access || err != nil {
		res.WriteHeader(http.StatusForbidden)
		return
	}

	s.CacheClient.Set(authToken, &cache.TokenCacheEntry{
		Valid:     true,
		HasAccess: true,
	})
	authorized = true

	res.WriteHeader(http.StatusOK)
}
