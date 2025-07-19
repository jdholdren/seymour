package citadel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"

	"github.com/jdholdren/seymour/internal/citadel/db"
	"github.com/jdholdren/seymour/internal/serverutil"
	"github.com/jdholdren/seymour/internal/seymour"
)

type (
	// Server is an instance of the aggregation server and handles requests
	// to search feeds or add new ones for ingestion.
	Server struct {
		*http.Server

		repo     db.Repo
		tempCli  client.Client
		feedRepo seymour.FeedService
		timeline seymour.TimelineService

		ghID         string
		ghSecret     string
		secureCookie *securecookie.SecureCookie
		httpsCookies bool // Whether or not HTTPS should be used for cookies
	}

	ServerConfig struct {
		Port               int
		CookieHashKey      []byte
		CookieBlockKey     []byte
		HttpsCookies       bool
		GithubClientID     string
		GithubClientSecret string

		DebugEndpoints bool
	}

	Params struct {
		fx.In

		Config       ServerConfig
		DB           *sqlx.DB
		TemporalCli  client.Client
		FeedRepo     seymour.FeedService
		TimelineRepo seymour.TimelineService
	}
)

func NewServer(lc fx.Lifecycle, p Params) Server {
	r := serverutil.ErrRouter{Router: mux.NewRouter()}
	srvr := Server{
		Server: &http.Server{
			Addr:         fmt.Sprintf(":%d", p.Config.Port),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			Handler:      r,
		},
		secureCookie: securecookie.New(p.Config.CookieHashKey, p.Config.CookieBlockKey),
		httpsCookies: p.Config.HttpsCookies,
		ghID:         p.Config.GithubClientID,
		ghSecret:     p.Config.GithubClientSecret,
		repo:         db.NewRepo(p.DB),
		tempCli:      p.TemporalCli,
		feedRepo:     p.FeedRepo,
		timeline:     p.TimelineRepo,
	}

	r.Use(serverutil.AccessLogMiddleware) // Log everything
	r.HandleFuncE("/api/viewer", srvr.handleViewer).Methods(http.MethodGet)
	r.HandleFuncE("/api/sso-login", srvr.handleSSORedirect).Methods(http.MethodGet)
	r.HandleFuncE("/api/sso-callback", srvr.handleSSOCallback).Methods(http.MethodGet)
	r.HandleFuncE("/api/logout", srvr.getLogout).Methods(http.MethodGet)

	if p.Config.DebugEndpoints {
		// For local testing
		r.HandleFuncE("/api/login", srvr.handleDebugLogin).Methods(http.MethodPost)
	}

	authed := serverutil.ErrRouter{Router: r.NewRoute().Subrouter()}
	authed.Use(requireSessionMiddleware(srvr.secureCookie))
	authed.HandleFuncE("/api/subscriptions", srvr.postSusbcriptions).Methods(http.MethodPost)
	authed.HandleFuncE("/api/subscriptions", srvr.getSusbcriptions).Methods(http.MethodGet)

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

// Viewer is the structured data about the current user in the frontend.
type Viewer struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (s Server) handleViewer(w http.ResponseWriter, r *http.Request) error {
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

	return serverutil.WriteJSON(w, http.StatusOK, Viewer{
		UserID:    usr.ID,
		Email:     usr.Email,
		CreatedAt: usr.CreatedAt,
	})
}
