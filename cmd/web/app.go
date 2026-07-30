package main

import (
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"invest/internal/auth"
	"invest/internal/model"

	"golang.org/x/net/websocket"
)

const (
	httpReadTimeout       = 15 * time.Second
	httpWriteTimeout      = 15 * time.Second
	httpIdleTimeout       = 60 * time.Second
	httpReadHeaderTimeout = 5 * time.Second
)

type appDeps struct {
	jwt      *auth.JWTManager
	users    *model.UserStore
	limiter  *LoginRateLimiter
	renderer *Renderer
}

func newAppMux(deps appDeps) (http.Handler, error) {
	authAPI := &authApp{jwt: deps.jwt, users: deps.users, limiter: deps.limiter}
	pages := &pageApp{renderer: deps.renderer}

	mux := http.NewServeMux()

	staticSub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, fmt.Errorf("load static files: %w", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/login", pages.handleLogin)
	mux.HandleFunc("/register", pages.handleRegister)
	mux.HandleFunc("/dashboard", pages.handleDashboard)
	mux.HandleFunc("/users", pages.handleUsers)

	mux.HandleFunc("/api/auth/login", authAPI.handleLogin)
	mux.HandleFunc("/api/auth/register", authAPI.handleRegister)
	mux.HandleFunc("/api/auth/me", requireAuth(deps.jwt, authAPI.handleMe))

	mux.HandleFunc("/api/admin/users", requireAdmin(deps.jwt, authAPI.handleListUsers))
	mux.HandleFunc("/api/admin/users/", requireAdmin(deps.jwt, func(w http.ResponseWriter, r *http.Request) {
		if !requireAnyMethod(w, r, http.MethodPut, http.MethodDelete) {
			return
		}
		switch r.Method {
		case http.MethodPut:
			authAPI.handleUpdateUser(w, r)
		case http.MethodDelete:
			authAPI.handleDeleteUser(w, r)
		}
	}))

	mux.HandleFunc("/api/analyze", requireAuth(deps.jwt, handleAnalyze))
	mux.HandleFunc("/api/quote", requireAuth(deps.jwt, handleQuote))
	mux.HandleFunc("/api/backtest", requireAuth(deps.jwt, handleBacktest))
	mux.HandleFunc("/api/backtest/compare", requireAuth(deps.jwt, handleBacktestCompare))
	mux.HandleFunc("/api/strategies", requireAuth(deps.jwt, handleListStrategies))
	mux.HandleFunc("/api/portfolio/backtest", requireAuth(deps.jwt, handlePortfolioBacktest))

	wsServer := &websocket.Server{
		Handshake: func(config *websocket.Config, r *http.Request) error {
			config.Protocol = []string{wsTokenProtoPrefix}
			return nil
		},
		Handler: makeMonitorHandler(deps.jwt),
	}
	mux.Handle("/ws/monitor", wsServer)

	return mux, nil
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       httpReadTimeout,
		WriteTimeout:      httpWriteTimeout,
		IdleTimeout:       httpIdleTimeout,
		ReadHeaderTimeout: httpReadHeaderTimeout,
	}
}
