package server

import (
	"net/http"

	"github.com/leohubert/nginx-plex-auth-server/internal/cache"
)

func (s *Server) getSessionCookie(req *http.Request) string {
	plexToken, _ := req.Cookie("X-Plex-Token")
	if plexToken == nil {
		return ""
	}
	return plexToken.Value
}

func (s *Server) deleteSessionCookie(res http.ResponseWriter, req *http.Request) {
	authToken := s.getSessionCookie(req)
	if authToken != "" {
		s.CacheClient.Invalidate(authToken)
	}

	clearedCookie := &http.Cookie{
		Name:     "X-Plex-Token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}

	if s.CookieDomain != "" {
		clearedCookie.Domain = s.CookieDomain
	}

	http.SetCookie(res, clearedCookie)
}

func (s *Server) createSessionCookie(res http.ResponseWriter, authToken string) {
	// Create the session cookie
	cookie := &http.Cookie{
		Name:     "X-Plex-Token",
		Value:    authToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days
	}

	if s.CookieDomain != "" {
		cookie.Domain = s.CookieDomain
	}

	http.SetCookie(res, cookie)

	// Cache the token for future auth checks
	s.CacheClient.Set(authToken, &cache.TokenCacheEntry{
		Valid:     true,
		HasAccess: true,
	})

}
