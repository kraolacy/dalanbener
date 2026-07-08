package handler

import (
	"strconv"

	"dalanshu/internal/cache"
	"dalanshu/internal/db"
	"dalanshu/internal/middleware"
	"dalanshu/internal/service"

	"github.com/gin-gonic/gin"
)

// Deps 汇聚 handler 所需的全部依赖。
type Deps struct {
	DB        *db.DBSet
	Cache     *cache.Cache
	Secret    string
	RateLimit float64
}

// Handlers 聚合所有 HTTP 处理逻辑（薄适配层：绑定参数→调 service→写响应）。
type Handlers struct {
	post   *service.PostService
	help   *service.HelpService
	user   *service.UserService
	secret string
}

// NewRouter 装配路由与中间件（供 main 与测试复用）。
func NewRouter(d Deps) *gin.Engine {
	postSvc := service.NewPostService(d.DB, d.Cache)
	helpSvc := service.NewHelpService(d.DB, d.Cache)
	userSvc := service.NewUserService(d.DB)
	h := &Handlers{post: postSvc, help: helpSvc, user: userSvc, secret: d.Secret}

	r := gin.New()
	r.Use(gin.Recovery())
	// 全局限流（可选）：RATE_LIMIT>0 时按客户端 IP 做 token bucket 限速。
	if d.RateLimit > 0 {
		r.Use(middleware.NewRateLimiter(d.RateLimit, burstFor(d.RateLimit)))
	}

	api := r.Group("/api")
	{
		api.GET("/health", h.Health)
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)
		api.GET("/me", middleware.RequireAuth(d.Secret), h.Me)
		api.GET("/posts", middleware.OptionalUser(d.Secret), h.Posts)
		api.POST("/posts", middleware.RequireAuth(d.Secret), h.CreatePost)
		api.POST("/posts/:id/comments", middleware.RequireAuth(d.Secret), h.AddComment)
		api.POST("/posts/:id/like", middleware.RequireAuth(d.Secret), h.ToggleLike)
		api.POST("/posts/:id/collect", middleware.RequireAuth(d.Secret), h.ToggleCollect)
		api.GET("/helps", h.Helps)
		api.POST("/helps", middleware.RequireAuth(d.Secret), h.CreateHelp)
	}
	return r
}

// burstFor 由 rps 推导突发容量，至少 1。
func burstFor(rps float64) int {
	b := int(rps * 2)
	if b < 1 {
		b = 1
	}
	return b
}

// parseLimit 解析分页大小：空或非法→0（表示不启用分页，走全量兼容模式）。
func parseLimit(raw string) int {
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0
	}
	if n > 50 {
		n = 50
	}
	return n
}
