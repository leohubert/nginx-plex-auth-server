package server

import (
	"log"
	"net/http"

	"github.com/leohubert/nginx-plex-auth-server/internal/server/views"
)

// LoginHandler initiates the Plex OAuth flow
func (s *Server) LoginHandler(res http.ResponseWriter, req *http.Request) {
	authToken := s.getSessionCookie(req)

	if authToken == "" {
		renderAnonymousLoginPage(res, req)
		return
	}

	userInfo, err := s.PlexClient.GetUserInfo(authToken)
	if err != nil {
		s.Logger.Error("Failed to get user info: " + err.Error())
		s.deleteSessionCookie(res, req)
		renderAnonymousLoginPage(res, req)
		return
	}

	hasAccess, err := s.PlexClient.CheckServerAccess(authToken)
	if err != nil {
		s.Logger.Error("Failed to check server hasAccess: " + err.Error())
	}

	// Render the login page using templ
	component := views.LoginPage(views.LoginPageData{
		IsLoggedIn: true,
		Username:   userInfo.Username,
		HasAccess:  hasAccess,
	})

	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(req.Context(), res); err != nil {
		log.Printf("Error rendering login page: %v", err)
		http.Error(res, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

func renderAnonymousLoginPage(res http.ResponseWriter, req *http.Request) {
	// Render the login page without user info
	component := views.LoginPage(views.LoginPageData{
		IsLoggedIn: false,
		Username:   "",
		HasAccess:  false,
	})
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(req.Context(), res); err != nil {
		log.Printf("Error rendering login page: %v", err)
		http.Error(res, "Failed to render page", http.StatusInternalServerError)
		return
	}
}
