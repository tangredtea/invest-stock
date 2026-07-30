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
	tmplSub, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		fmt.Println("模板目录加载失败:", err)
		os.Exit(1)
	}
	renderer, err := NewRenderer(tmplSub)
	if err != nil {
		fmt.Println("模板解析失败:", err)
		os.Exit(1)
	}

	mux, err := newAppMux(appDeps{jwt: jwtMgr, users: users, limiter: limiter, renderer: renderer})
	if err != nil {
		fmt.Println("路由初始化失败:", err)
		os.Exit(1)
	}
	srv := newHTTPServer(cfg.Addr, mux)

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
