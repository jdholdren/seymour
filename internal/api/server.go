package api

import (
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

func NewServer(config ServerConfig, repo seymour.Repository, temporalCli client.Client) *Server {
	var (
		r        = errRouter{Router: mux.NewRouter()}
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

	authed := errRouter{Router: r.NewRoute().Subrouter()}
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

	// Accout management
	authed.HandleFuncE("/api/account/prompt:precheck", srvr.postPromptPrecheck).Methods(http.MethodPost)
	authed.HandleFuncE("/api/account/prompt", srvr.postPrompt).Methods(http.MethodPost)

	slog.Debug("configured citadel server", "port", config.Port)

	return &srvr
}
