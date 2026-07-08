package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dalanshu/internal/cache"
	"dalanshu/internal/config"
	"dalanshu/internal/db"
	"dalanshu/internal/handler"
	"dalanshu/internal/seed"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// 子命令：数据迁移（SQLite <-> MySQL 平滑升级）。
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		db.RunMigrate(cfg)
		return
	}

	gin.SetMode(cfg.GinMode)

	// 数据库驱动可插拔：sqlite（默认，零配置）或 mysql（生产 / 高并发）。
	dsn, err := resolveDSN(cfg)
	if err != nil {
		log.Fatalf("数据库配置错误: %v", err)
	}
	set, err := db.Connect(cfg.DBDriver, dsn, cfg.MySQLReadDSN)
	if err != nil {
		log.Fatalf("连接数据库(%s)失败: %v", cfg.DBDriver, err)
	}
	if err := db.Migrate(set); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	seed.Seed(set.W())

	// Redis 可选：未配置或不可达时自动降级为直查数据库。
	var c *cache.Cache
	if cfg.RedisAddr != "" {
		c = cache.New(cfg.RedisAddr, cfg.RedisPass, cfg.CacheTTL)
		if err := c.Ping(context.Background()); err != nil {
			log.Printf("[warn] Redis 不可用，已降级为直查数据库: %v", err)
			c = nil
		}
	}

	r := handler.NewRouter(handler.Deps{DB: set, Cache: c, Secret: cfg.JWTSecret, RateLimit: cfg.RateLimit})

	if cfg.StaticDir != "" {
		if _, err := os.Stat(cfg.StaticDir); err == nil {
			r.NoRoute(func(ctx *gin.Context) {
				ctx.File(cfg.StaticDir + "/index.html")
			})
			log.Printf("[static] 托管前端目录: %s", cfg.StaticDir)
		} else {
			log.Printf("[warn] STATIC_DIR=%s 不存在，仅提供 API", cfg.StaticDir)
		}
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		log.Printf("[dalanshu] 服务已启动 http://0.0.0.0:%s (db=%s)", cfg.Port, cfg.DBDriver)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("服务异常退出: %v", err)
		}
	}()

	// 优雅关闭：收到信号后停止接收新请求，并在超时内排空在途请求。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[dalanshu] 正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[warn] 关闭超时: %v", err)
	}
	log.Println("[dalanshu] 已停止")
}

// resolveDSN 根据驱动返回对应的数据源名称。
func resolveDSN(cfg *config.Config) (string, error) {
	if cfg.DBDriver == "mysql" {
		if cfg.MySQLDSN == "" {
			return "", errors.New("DB_DRIVER=mysql 时 MYSQL_DSN 不能为空")
		}
		return cfg.MySQLDSN, nil
	}
	return cfg.SQLitePath, nil
}
