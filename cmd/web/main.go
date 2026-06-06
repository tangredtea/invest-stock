package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"invest/internal/auth"
	"invest/internal/model"
	"invest/internal/store"
	"invest/pkg/data"

	"golang.org/x/net/websocket"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFiles embed.FS

func main() {
	// --- Configuration (fatal if JWT secret missing/insecure) ---
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Println("配置错误:", err)
		os.Exit(1)
	}

	// --- K-line cache TTL ---
	data.SetKLineCacheTTL(cfg.KLineTTL)

	// --- Database ---
	db, err := store.Open("data/invest.db")
	if err != nil {
		fmt.Println("数据库初始化失败:", err)
		os.Exit(1)
	}
	defer db.Close()

	// --- User store + optional admin seed ---
	users := model.NewUserStore(db.DB)
	if cfg.SeedAdmin {
		if err := users.SeedAdmin(cfg.AdminUsername, cfg.AdminPassword); err != nil {
			fmt.Println("创建默认管理员失败:", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("提示: 未配置初始管理员凭据,跳过默认管理员播种")
	}

	// --- JWT ---
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL)

	// --- Login rate limiter ---
	limiter := NewLoginRateLimiter(defaultLoginThreshold, defaultLoginWindow)

	// --- Template renderer ---
	tmplSub, _ := fs.Sub(templateFiles, "templates")
	renderer, err := NewRenderer(tmplSub)
	if err != nil {
		fmt.Println("模板解析失败:", err)
		os.Exit(1)
	}

	// --- App instances ---
	authAPI := &authApp{jwt: jwtMgr, users: users, limiter: limiter}
	pages := &pageApp{renderer: renderer}

	// --- Explicit mux ---
	mux := http.NewServeMux()

	// --- Static files ---
	staticSub, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// --- Page routes ---
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

	// --- Auth API ---
	mux.HandleFunc("/api/auth/login", authAPI.handleLogin)
	mux.HandleFunc("/api/auth/register", authAPI.handleRegister)
	mux.HandleFunc("/api/auth/me", requireAuth(jwtMgr, authAPI.handleMe))

	// --- Admin API ---
	mux.HandleFunc("/api/admin/users", requireAdmin(jwtMgr, authAPI.handleListUsers))
	mux.HandleFunc("/api/admin/users/", requireAdmin(jwtMgr, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			authAPI.handleUpdateUser(w, r)
		case http.MethodDelete:
			authAPI.handleDeleteUser(w, r)
		default:
			writeJSON(w, 405, jsonResp{Code: 405, Message: "method not allowed"})
		}
	}))

	// --- Protected existing API ---
	mux.HandleFunc("/api/analyze", requireAuth(jwtMgr, handleAnalyze))
	mux.HandleFunc("/api/quote", requireAuth(jwtMgr, handleQuote))

	// --- Backtest API (new engine) ---
	mux.HandleFunc("/api/backtest", requireAuth(jwtMgr, handleBacktest))
	mux.HandleFunc("/api/backtest/compare", requireAuth(jwtMgr, handleBacktestCompare))
	mux.HandleFunc("/api/strategies", requireAuth(jwtMgr, handleListStrategies))

	// --- Portfolio backtest API (multi-symbol + risk management) ---
	mux.HandleFunc("/api/portfolio/backtest", requireAuth(jwtMgr, handlePortfolioBacktest))

	// --- WebSocket monitor (authenticated via subprotocol token) ---
	wsServer := &websocket.Server{
		Handshake: func(config *websocket.Config, r *http.Request) error {
			// Echo back the accepted "bearer" subprotocol so the handshake
			// completes per RFC 6455; the token itself is not echoed.
			config.Protocol = []string{wsTokenProtoPrefix}
			return nil
		},
		Handler: makeMonitorHandler(jwtMgr),
	}
	mux.Handle("/ws/monitor", wsServer)

	// --- HTTP server with timeouts ---
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// --- Start + graceful shutdown ---
	go func() {
		fmt.Printf("启动 Web 服务: http://localhost%s\n", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("启动失败:", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("收到关闭信号,正在优雅关闭...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		// In-flight requests did not finish within the bounded period: force close.
		fmt.Println("优雅关闭超时,强制关闭:", err)
		fctx, fcancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer fcancel()
		_ = srv.Close()
		<-fctx.Done()
	}
	fmt.Println("已关闭")
}
