package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	lru "github.com/hashicorp/golang-lru/v2"
	seyerrs "github.com/jdholdren/seymour/internal/errors"
	"github.com/jdholdren/seymour/internal/seymour"
	"go.temporal.io/sdk/client"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("error encoding json response: %s", err)
	}

	return nil
}

// validator is a surface that can validate itself and return an error
// if something is wrong.
type validator interface {
	Validate() error
}

// decodeValid decodes a request and then validates it.
func decodeValid[V validator](r io.Reader) (V, error) {
	var v V
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return v, fmt.Errorf("error decoding request: %w", err)
	}
	if err := v.Validate(); err != nil {
		return v, fmt.Errorf("error validating request: %w", err)
	}

	return v, nil
}

// HandlerFuncE is a modified type of [http.HandlerFunc] that returns an error.
type HandlerFuncE func(w http.ResponseWriter, r *http.Request) error

func (f HandlerFuncE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := f(w, r)
	if err == nil {
		slog.Info("no error coming back")
		return
	}

	// Either it's already a structured error, or coerce it to one
	sErr := &seyerrs.Error{}
	if !errors.As(err, &sErr) {
		slog.Error("non seyerr", "err", err)
		sErr = seyerrs.E(http.StatusInternalServerError, "internal server error")
	}

	if err := writeJSON(w, sErr.Status, sErr); err != nil {
		slog.Error("error writing response", "error", err)
	}
}

// errRouter is a newtype around a mux router that allows attaching handlers that return errors.
type errRouter struct {
	*mux.Router
}

func (r errRouter) HandleFuncE(path string, f HandlerFuncE) *mux.Route {
	return r.Handle(path, f)
}

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

	r.Use(AccessLogMiddleware) // Log everything
	r.HandleFuncE("/api/viewer", srvr.handleViewer).Methods(http.MethodGet)
	r.HandleFuncE("/api/sso-login", srvr.handleSSORedirect).Methods(http.MethodGet)
	r.HandleFuncE("/api/sso-callback", srvr.handleSSOCallback).Methods(http.MethodGet)
	r.HandleFuncE("/api/logout", srvr.getLogout).Methods(http.MethodGet)

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
