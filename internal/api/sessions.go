package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/jdholdren/seymour/internal/seymour"
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
	if errors.Is(err, http.ErrNoCookie) {
		return sessionState{}
	}
	if err != nil {
		slog.Error("error fetching cookie", "err", err)
		return sessionState{}
	}

	value := sessionState{}
	if err := secureCookie.Decode(sessionCookieName, cookie.Value, &value); err != nil {
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

func requireSessionMiddleware(sc *securecookie.SecureCookie) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			state := session(r, sc)
			if state.UserID == "" {
				http.Error(w, "Unauthenticated", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Redirects the user to the SSO login page.
func (s Server) handleSSORedirect(w http.ResponseWriter, r *http.Request) error {
	// Create a state to store as part of the flow
	state := sessionState{
		State: uuid.NewString(),
	}
	setSession(w, s.secureCookie, s.httpsCookies, state)

	http.Redirect(w, r, s.ghOauthConfig.AuthCodeURL(state.State), http.StatusTemporaryRedirect)
	return nil
}

// Handles the code coming back from github
//
// I think everything here should redirect?
func (s Server) handleSSOCallback(w http.ResponseWriter, r *http.Request) error {
	// Check the state and error
	sess := session(r, s.secureCookie)
	q := r.URL.Query()
	if q.Get("state") != sess.State {
		http.Redirect(w, r, "/welcome?error="+url.QueryEscape("invalid_state"), http.StatusFound)
		return nil
	}
	if q.Get("error") != "" {
		http.Redirect(w, r, "/welcome?error="+url.QueryEscape(q.Get("error")), http.StatusFound)
		return nil
	}

	// Exchange:
	code := r.URL.Query().Get("code")
	tok, err := s.ghOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Redirect(w, r, "/welcome?error="+url.QueryEscape(err.Error()), http.StatusFound)
		return nil
	}

	// Get some details about our person
	client := s.ghOauthConfig.Client(r.Context(), tok)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Redirect(w, r, "/welcome?error="+url.QueryEscape(err.Error()), http.StatusFound)
		return nil
	}
	defer resp.Body.Close()

	type userInfo struct {
		Username          string `json:"login"`
		NotificationEmail string `json:"email"`
	}
	var info userInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		http.Redirect(w, r, "/welcome?error="+url.QueryEscape(err.Error()), http.StatusFound)
		return nil
	}

	// Ensure the user
	usr, err := s.repo.EnsureUser(r.Context(), seymour.User{
		GithubID: info.Username,
		Email:    info.NotificationEmail,
	})
	if err != nil {
		http.Redirect(w, r, "/welcome?error="+url.QueryEscape(err.Error()), http.StatusFound)
		return nil
	}

	// Start a session
	session := sessionState{
		UserID: usr.ID,
	}
	setSession(w, s.secureCookie, s.httpsCookies, session)

	// Use the configured redirect URL, defaulting to "/" if not set
	redirectURL := s.ssoRedirectURL
	if redirectURL == "" {
		redirectURL = "/"
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
	return nil
}

func (s Server) getLogout(w http.ResponseWriter, r *http.Request) error {
	setSession(w, s.secureCookie, s.httpsCookies, sessionState{})

	// Redirect to the welcome page
	http.Redirect(w, r, "/welcome", http.StatusFound)

	return nil
}
