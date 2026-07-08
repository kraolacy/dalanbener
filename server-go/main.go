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
	"dalanshu/internal/middleware"
	"dalanshu/internal/seed"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// 子命令：数据迁移（SQLite -> MySQL 平滑升级）
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrate(cfg)
		return
	}

	gin.SetMode(cfg.GinMode)

	// 数据库驱动可插拔：sqlite（默认，零配置）或 mysql（生产 / 高并发）
	dsn, err := resolveDSN(cfg)
	if err != nil {
		log.Fatalf("数据库配置错误: %v", err)
	}
	database, err := db.Connect(cfg.DBDriver, dsn)
	if err != nil {
		log.Fatalf("连接数据库(%s)失败: %v", cfg.DBDriver, err)
	}
	if err := db.Migrate(database); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	seed.Seed(database)

	// Redis 可选：未配置或不可达时自动降级为直查数据库。
	var c *cache.Cache
	if cfg.RedisAddr != "" {
		c = cache.New(cfg.RedisAddr, cfg.RedisPass, cfg.CacheTTL)
		if err := c.Ping(context.Background()); err != nil {
			log.Printf("[warn] Redis 不可用，已降级为直查数据库: %v", err)
			c = nil
		}
	}

	h := handler.New(database, c, cfg.JWTSecret)
	r := gin.New()
	r.Use(gin.Recovery())
	api := r.Group("/api")
	{
		api.GET("/health", h.Health)
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)
		api.GET("/me", middleware.RequireAuth(cfg.JWTSecret), h.Me)
		api.GET("/posts", middleware.OptionalUser(cfg.JWTSecret), h.Posts)
		api.POST("/posts", middleware.RequireAuth(cfg.JWTSecret), h.CreatePost)
		api.POST("/posts/:id/comments", middleware.RequireAuth(cfg.JWTSecret), h.AddComment)
		api.POST("/posts/:id/like", middleware.RequireAuth(cfg.JWTSecret), h.ToggleLike)
		api.POST("/posts/:id/collect", middleware.RequireAuth(cfg.JWTSecret), h.ToggleCollect)
		api.GET("/helps", h.Helps)
		api.POST("/helps", middleware.RequireAuth(cfg.JWTSecret), h.CreateHelp)
	}

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

// runMigrate 执行 SQLite -> MySQL（或反之）的平滑数据迁移。
func runMigrate(cfg *config.Config) {
	from := cfg.MigrateFrom
	if from == "" {
		from = "sqlite"
	}
	to := cfg.MigrateTo
	if to == "" {
		to = "mysql"
	}
	if from == to {
		log.Fatal("迁移源与目标驱动不能相同")
	}
	srcDSN, err := dsnForDriver(cfg, from)
	if err != nil {
		log.Fatalf("源 DSN 解析失败: %v", err)
	}
	dstDSN, err := dsnForDriver(cfg, to)
	if err != nil {
		log.Fatalf("目标 DSN 解析失败: %v", err)
	}
	src, err := db.Connect(from, srcDSN)
	if err != nil {
		log.Fatalf("连接源库(%s)失败: %v", from, err)
	}
	dst, err := db.Connect(to, dstDSN)
	if err != nil {
		log.Fatalf("连接目标库(%s)失败: %v", to, err)
	}
	if err := db.Migrate(dst); err != nil {
		log.Fatalf("目标库迁移失败: %v", err)
	}
	if err := db.MigrateData(src, dst); err != nil {
		log.Fatalf("数据迁移失败: %v", err)
	}
	log.Printf("[migrate] 完成：%s(%s) -> %s(%s)", from, srcDSN, to, dstDSN)
}

func dsnForDriver(cfg *config.Config, driver string) (string, error) {
	if driver == "mysql" {
		if cfg.MySQLDSN == "" {
			return "", errors.New("MYSQL_DSN 不能为空")
		}
		return cfg.MySQLDSN, nil
	}
	return cfg.SQLitePath, nil
}
