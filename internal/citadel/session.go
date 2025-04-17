package citadel

import (
	"log/slog"
	"net/http"
)

const sessionCookieName = "citadel_session"

// Describes a user's session that's persisted to their cookie.
type session struct {
	State  string // For SSO
	UserID string
}

// Fetches the current session tied to the request.
func (s Server) session(r *http.Request) session {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		slog.Error("error fetching cookie", "err", err)
		return session{}
	}

	value := session{}
	err = s.secureCookie.Decode(sessionCookieName, cookie.Value, &value)
	if err != nil {
		slog.Error("error decoding cookie", "err", err)
		return session{}
	}

	return value
}

// Sets the session on the request.
func (s Server) setSession(w http.ResponseWriter, sess session) {
	encoded, err := s.secureCookie.Encode(sessionCookieName, sess)
	if err != nil {
		slog.Error("error encoding cookie", "err", err)
		return
	}

	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    encoded,
		Path:     "/",
		Secure:   s.httpsCookies,
		HttpOnly: true,
	}
	slog.Debug("setting cookie", "cookie", cookie)
	http.SetCookie(w, cookie)
}
