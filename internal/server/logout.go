package server

import (
	"net/http"
)

// LogoutHandler clears the session cookie and shows deleteSessionCookie page
func (s *Server) LogoutHandler(res http.ResponseWriter, req *http.Request) {
	s.deleteSessionCookie(res, req)

	http.Redirect(res, req, "/", http.StatusSeeOther)
}
