package citadel

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/jmoiron/sqlx"
	"go.uber.org/fx"

	"github.com/jdholdren/seymour/internal/agg"
	"github.com/jdholdren/seymour/internal/citadel/db"
	"github.com/jdholdren/seymour/internal/server"
)

type (
	// Server is an instance of the aggregation server and handles requests
	// to search feeds or add new ones for ingestion.
	Server struct {
		*http.Server

		repo db.Repo
		agg  agg.Server

		ghID     string
		ghSecret string

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

		Config ServerConfig
		DB     *sqlx.DB
	}
)

func NewServer(lc fx.Lifecycle, p Params) Server {
	r := mux.NewRouter()
	srvr := Server{
		Server: &http.Server{
			Addr:         fmt.Sprintf(":%d", p.Config.Port),
			WriteTimeout: 5 * time.Second,
			ReadTimeout:  5 * time.Second,
			Handler:      r,
		},
		secureCookie: securecookie.New(p.Config.CookieHashKey, p.Config.CookieBlockKey),
		httpsCookies: p.Config.HttpsCookies,
		ghID:         p.Config.GithubClientID,
		ghSecret:     p.Config.GithubClientSecret,
		repo:         db.NewRepo(p.DB),
	}

	r.Use(server.AccessLogMiddleware) // Log everything
	r.Handle("/api/viewer", server.HandlerFuncE(srvr.handleViewer)).Methods(http.MethodGet)
	r.Handle("/api/sso-login", server.HandlerFuncE(srvr.handleSSORedirect)).Methods(http.MethodGet)
	r.Handle("/api/sso-callback", server.HandlerFuncE(srvr.handleSSOCallback)).Methods(http.MethodGet)

	if p.Config.DebugEndpoints {
		// For local testing
		r.Handle("/api/login", server.HandlerFuncE(srvr.handleDebugLogin)).Methods(http.MethodPost)
	}

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
		return server.WriteJSON(w, http.StatusOK, struct{}{})
	}
	usr, err := s.repo.User(r.Context(), sess.UserID)
	if err != nil {
		return err
	}

	return server.WriteJSON(w, http.StatusOK, Viewer{
		UserID:    usr.ID,
		Email:     usr.Email,
		CreatedAt: usr.CreatedAt,
	})
}
