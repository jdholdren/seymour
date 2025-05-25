package citadel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"

	"github.com/jdholdren/seymour/internal/citadel/db"
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

	u, _ := url.Parse("https://github.com/login/oauth/authorize")
	v := u.Query()
	v.Set("client_id", s.ghID)
	v.Set("state", state.State)
	v.Set("scope", "read:org")
	u.RawQuery = v.Encode()

	http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
	return nil
}

type githubUserData struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Login string `json:"login"`
}

// Fetches GitHub details using the provided oauth code.
//
// Returns the user details and if the user is part of the organization.
// Claude-Generated
func (s Server) fetchGithubDetails(ctx context.Context, code string) (githubUserData, bool, error) {
	// First, we need to exchange the code for an access token
	byts, _ := json.Marshal(struct {
		Code         string `json:"code"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}{
		Code:         code,
		ClientID:     s.ghID,
		ClientSecret: s.ghSecret,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", bytes.NewReader(byts))
	if err != nil {
		return githubUserData{}, false, fmt.Errorf("error creating access token request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return githubUserData{}, false, fmt.Errorf("error getting access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return githubUserData{}, false, fmt.Errorf("unexpected status code when getting access token: %d", resp.StatusCode)
	}

	tokenResp := struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return githubUserData{}, false, fmt.Errorf("error parsing access token response: %w", err)
	}

	// Now fetch the user details
	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return githubUserData{}, false, fmt.Errorf("error creating user request: %w", err)
	}
	userReq.Header.Set("Authorization", "token "+tokenResp.AccessToken)
	userReq.Header.Set("Accept", "application/vnd.github.v3+json")
	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		return githubUserData{}, false, fmt.Errorf("error getting user details: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		return githubUserData{}, false, fmt.Errorf("unexpected status code from github user API: %d", userResp.StatusCode)
	}

	var userData githubUserData
	if err := json.NewDecoder(userResp.Body).Decode(&userData); err != nil {
		return githubUserData{}, false, fmt.Errorf("error parsing user response: %w", err)
	}

	// Use the api to check membership of ankored
	membershipURL := fmt.Sprintf("https://api.github.com/orgs/ankored/members/%s", userData.Login)
	membershipReq, err := http.NewRequestWithContext(ctx, http.MethodGet, membershipURL, nil)
	if err != nil {
		return githubUserData{}, false, fmt.Errorf("error creating orgs request: %w", err)
	}
	membershipReq.Header.Set("Authorization", "token "+tokenResp.AccessToken)
	membershipReq.Header.Set("Accept", "application/vnd.github+json")

	orgsResp, err := http.DefaultClient.Do(membershipReq)
	if err != nil {
		return githubUserData{}, false, fmt.Errorf("error getting user orgs: %w", err)
	}
	defer orgsResp.Body.Close()

	if orgsResp.StatusCode != http.StatusNoContent {
		// User not part of the org
		return userData, false, nil
	}

	return userData, true, nil
}

// Handles the code coming back from github
//
// I think everything here should redirect?
func (s Server) handleSSOCallback(w http.ResponseWriter, r *http.Request) error {
	// Check the state and error
	sess := session(r, s.secureCookie)
	if r.URL.Query().Get("state") != sess.State {
		http.Redirect(w, r, "/welcome?error="+url.QueryEscape("invalid_state"), http.StatusFound)
		return nil
	}

	// Fetch user's details
	var errMsg string
	details, inOrg, err := s.fetchGithubDetails(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		slog.Error("err fetching github details", "err", err)
		errMsg = "fetching"
	}
	if err == nil && !inOrg { // Ensure they're in the Ankored org
		errMsg = "not_in_org"
	}
	if errMsg != "" {
		http.Redirect(w, r, "/welcome?error="+url.QueryEscape(errMsg), http.StatusFound)
		return nil
	}

	// Ensure they're created and redirect back
	usr, err := s.repo.EnsureUser(r.Context(), db.User{GithubID: details.Login, Email: details.Email})
	if err != nil {
		return err
	}

	// Issue an update to their session so they're logged in
	setSession(w, s.secureCookie, s.httpsCookies, sessionState{UserID: usr.ID})

	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}

type DebugLogin struct {
	UserID string `json:"user_id"`
}

func (s Server) handleDebugLogin(w http.ResponseWriter, r *http.Request) error {
	var body DebugLogin
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return err
	}

	// Issue an update to their session so they're logged in
	setSession(w, s.secureCookie, s.httpsCookies, sessionState{UserID: body.UserID})
	return nil
}
