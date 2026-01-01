package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.temporal.io/sdk/client"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/jdholdren/seymour/internal/serverutil"
	"github.com/jdholdren/seymour/internal/seymour"
)

type (
	// Server is an instance of the aggregation server and handles requests
	// to search feeds or add new ones for ingestion.
	Server struct {
		*http.Server

		fetchClient    *http.Client
		entryRespCache *lru.Cache[string, FeedEntryResp]

		repo    seymour.Repository
		tempCli client.Client

		ghOauthConfig  oauth2.Config
		secureCookie   *securecookie.SecureCookie
		httpsCookies   bool   // Whether or not HTTPS should be used for cookies
		ssoRedirectURL string // URL to redirect to after successful SSO login
	}

	ServerConfig struct {
		Port               int
		CookieHashKey      []byte
		CookieBlockKey     []byte
		HttpsCookies       bool
		GithubClientID     string
		GithubClientSecret string
		CorsHeader         string
		SSORedirectURL     string

		DebugEndpoints bool
	}
)

func NewServer(ctx context.Context, config ServerConfig, repo seymour.Repository, temporalCli client.Client) *Server {
	var (
		r        = serverutil.ErrRouter{Router: mux.NewRouter()}
		cache, _ = lru.New[string, FeedEntryResp](1024)
	)

	srvr := Server{
		fetchClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		entryRespCache: cache,
		secureCookie:   securecookie.New(config.CookieHashKey, config.CookieBlockKey),
		httpsCookies:   config.HttpsCookies,
		ssoRedirectURL: config.SSORedirectURL,
		ghOauthConfig: oauth2.Config{
			ClientID:     config.GithubClientID,
			ClientSecret: config.GithubClientSecret,
			Scopes:       []string{},
			Endpoint:     github.Endpoint,
		},
		repo:    repo,
		tempCli: temporalCli,
		Server: &http.Server{
			Addr:         fmt.Sprintf(":%d", config.Port),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			Handler: handlers.CORS(
				handlers.AllowedOrigins([]string{config.CorsHeader}),
				handlers.AllowCredentials(),
				handlers.AllowedMethods([]string{http.MethodGet, http.MethodPost, http.MethodOptions}),
				handlers.AllowedHeaders([]string{"content-type"}),
			)(r),
		},
	}

	r.Use(serverutil.AccessLogMiddleware) // Log everything
	r.HandleFuncE("/api/viewer", srvr.handleViewer).Methods(http.MethodGet)
	r.HandleFuncE("/api/sso-login", srvr.handleSSORedirect).Methods(http.MethodGet)
	r.HandleFuncE("/api/sso-callback", srvr.handleSSOCallback).Methods(http.MethodGet)
	r.HandleFuncE("/api/logout", srvr.getLogout).Methods(http.MethodGet)

	if config.DebugEndpoints {
		// For local testing
		r.HandleFuncE("/api/login", srvr.handleDebugLogin).Methods(http.MethodPost)
	}

	authed := serverutil.ErrRouter{Router: r.NewRoute().Subrouter()}
	authed.Use(requireSessionMiddleware(srvr.secureCookie))

	// Subscription management
	//
	// TODO: Make these specific to a user
	authed.HandleFuncE("/api/subscriptions", srvr.postSusbcriptions).Methods(http.MethodPost)
	authed.HandleFuncE("/api/subscriptions", srvr.getSusbcriptions).Methods(http.MethodGet)

	// Timeline view
	authed.HandleFuncE("/api/users/{userID}/timeline", srvr.getUserTimeline).Methods(http.MethodGet)

	// Reader view
	authed.HandleFuncE("/api/feed-entries/{feedEntryID}", srvr.getFeedEntry).Methods(http.MethodGet)

	slog.Debug("configured citadel server", "port", config.Port)

	return &srvr
}

// Viewer is the structured data about the current user in the frontend.
type (
	Viewer struct {
		UserID    string    `json:"user_id"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`

		// Information about the user's nav bar
		PersonalSubscriptions map[string]ViewerSubscription `json:"subscriptions"`
	}

	ViewerSubscription struct {
		Name        string `json:"name"`
		FeedID      string `json:"feed_id"`
		Description string `json:"description"`
	}
)

func (s Server) handleViewer(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	sess := session(r, s.secureCookie)
	if sess.UserID == "" {
		return serverutil.WriteJSON(w, http.StatusOK, struct{}{})
	}
	usr, err := s.repo.User(r.Context(), sess.UserID)
	if errors.Is(err, seymour.ErrNotFound) {
		return serverutil.WriteJSON(w, http.StatusOK, struct{}{})
	}
	if err != nil {
		return err
	}

	// Get the feeds that the user has subscribed to.
	// This will populate the nav bar with the individual filters for their personal timeline.
	subs, err := s.repo.UserSubscriptions(ctx, usr.ID)
	if err != nil {
		return err
	}

	// Get the feed names themselves.
	var feedIDs []string
	for _, sub := range subs {
		feedIDs = append(feedIDs, sub.FeedID)
	}
	feeds, err := s.repo.Feeds(ctx, feedIDs)
	if err != nil {
		return err
	}
	feedsByID := make(map[string]seymour.Feed)
	for _, feed := range feeds {
		feedsByID[feed.ID] = feed
	}

	viewerSubs := make(map[string]ViewerSubscription)
	for _, feed := range feeds {
		feed := feedsByID[feed.ID]
		var title, desc string
		if feed.Title != nil {
			title = *feed.Title
		}
		if feed.Description != nil {
			desc = *feed.Description
		}

		viewerSubs[feed.ID] = ViewerSubscription{
			Name:        title,
			FeedID:      feed.ID,
			Description: desc,
		}
	}

	return serverutil.WriteJSON(w, http.StatusOK, Viewer{
		UserID:                usr.ID,
		Email:                 usr.Email,
		CreatedAt:             usr.CreatedAt,
		PersonalSubscriptions: viewerSubs,
	})
}
