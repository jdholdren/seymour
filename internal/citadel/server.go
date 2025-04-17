package citadel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
	"github.com/jmoiron/sqlx"
	"go.uber.org/fx"

	v1 "github.com/jdholdren/seymour/api/citadel/v1"
	"github.com/jdholdren/seymour/internal/server"
)

type (
	// Server is an instance of the aggregation server and handles requests
	// to search feeds or add new ones for ingestion.
	Server struct {
		*server.Server
		userRepo userRepo

		ghID     string
		ghSecret string

		secureCookie *securecookie.SecureCookie
		httpsCookies bool // Whether or not HTTPS should be used for cookies
	}

	Config struct {
		Port               int
		CookieHashKey      []byte
		CookieBlockKey     []byte
		HttpsCookies       bool
		GithubClientID     string
		GithubClientSecret string
	}

	Params struct {
		fx.In

		Config Config
		DB     *sqlx.DB
	}
)

func NewServer(lc fx.Lifecycle, p Params) Server {
	s, r := server.NewServer("citadel", p.Config.Port)
	srvr := Server{
		Server:       s,
		secureCookie: securecookie.New(p.Config.CookieHashKey, p.Config.CookieBlockKey),
		httpsCookies: p.Config.HttpsCookies,
		ghID:         p.Config.GithubClientID,
		ghSecret:     p.Config.GithubClientSecret,
		userRepo:     userRepo{db: p.DB},
	}

	r.Handle("GET /api/viewer", server.HandlerFuncE(srvr.handleViewer))
	r.Handle("GET /api/sso-login", server.HandlerFuncE(srvr.handleSSORedirect))
	r.Handle("GET /api/sso-callback", server.HandlerFuncE(srvr.handleSSOCallback))

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go srvr.ListenAndServe()

			slog.Debug("started citadel server", "port", p.Config.Port)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srvr.Shutdown(ctx)
		},
	})

	return srvr
}

func (s Server) handleViewer(w http.ResponseWriter, r *http.Request) error {
	sess := s.session(r)
	if sess.UserID == "" {
		return server.WriteJSON(w, http.StatusOK, struct{}{})
	}
	usr, err := s.userRepo.user(r.Context(), sess.UserID)
	if err != nil {
		return err
	}

	return server.WriteJSON(w, http.StatusOK, v1.Viewer{
		UserID: usr.ID,
	})
}

// Redirects the user to the SSO login page.
func (s Server) handleSSORedirect(w http.ResponseWriter, r *http.Request) error {
	// Create a state to store as part of the flow
	session := session{
		State: uuid.NewString(),
	}
	s.setSession(w, session)

	u, _ := url.Parse("https://github.com/login/oauth/authorize")
	v := u.Query()
	v.Set("client_id", s.ghID)
	v.Set("state", session.State)
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
	sess := s.session(r)
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
	usr, err := s.userRepo.ensureUser(r.Context(), user{GithubID: details.Login})
	if err != nil {
		return err
	}

	// Issue an update to their session so they're logged in
	s.setSession(w, session{UserID: usr.ID})

	http.Redirect(w, r, "/welcome?error="+url.QueryEscape(errMsg), http.StatusFound)
	return nil
}
