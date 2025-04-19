package citadel

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/securecookie"
)

const sessionCookieName = "citadel_session"

// Describes a user's sessionState that's persisted to their cookie.
type sessionState struct {
	State  string // For SSO
	UserID string
}

// Fetches the current session tied to the request.
func session(r *http.Request, secureCookie *securecookie.SecureCookie) sessionState {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		slog.Error("error fetching cookie", "err", err)
		return sessionState{}
	}

	value := sessionState{}
	err = secureCookie.Decode(sessionCookieName, cookie.Value, &value)
	if err != nil {
		slog.Error("error decoding cookie", "err", err)
		return sessionState{}
	}

	return value
}

// Sets the session on the request.
func setSession(w http.ResponseWriter, secureCookie *securecookie.SecureCookie, https bool, sess sessionState) {
	encoded, err := secureCookie.Encode(sessionCookieName, sess)
	if err != nil {
		slog.Error("error encoding cookie", "err", err)
		return
	}

	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    encoded,
		Path:     "/",
		Secure:   https,
		HttpOnly: true,
	}
	slog.Debug("setting cookie", "cookie", cookie)
	http.SetCookie(w, cookie)
}

// Requires that the request is authenticated.
type requireSessionMiddleware struct {
	inner        http.Handler
	secureCookie *securecookie.SecureCookie
}

func (m requireSessionMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	state := session(r, m.secureCookie)
	if state.UserID == "" {
		http.Error(w, "Unauthenticated", http.StatusUnauthorized)
		return
	}

	m.inner.ServeHTTP(w, r)
}
