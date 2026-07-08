package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// bucket 是单客户端的令牌桶（纯标准库实现，避免额外依赖）。
type bucket struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
	rps    float64
	burst  float64
}

// allow 按令牌桶算法判断是否放行：根据流逝时间补充令牌，足够则扣减并返回 true。
func (b *bucket) allow(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * b.rps
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens -= 1
		return true
	}
	return false
}

// ipLimiter 维护每个客户端 IP 的令牌桶，并定期清理空闲桶避免内存泄漏。
type ipLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rps     float64
	burst   float64
	cleanup time.Duration
}

func newIPLimiter(rps float64, burst int) *ipLimiter {
	l := &ipLimiter{
		buckets: make(map[string]*bucket),
		rps:     rps,
		burst:   float64(burst),
		cleanup: 5 * time.Minute,
	}
	go l.sweep()
	return l
}

func (l *ipLimiter) get(ip string) *bucket {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{tokens: l.burst, last: time.Now(), rps: l.rps, burst: l.burst}
		l.buckets[ip] = b
	}
	return b
}

// sweep 定期清空所有桶（下次访问按需重建），防止长期空闲的客户端占用内存。
func (l *ipLimiter) sweep() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		l.buckets = make(map[string]*bucket)
		l.mu.Unlock()
	}
}

// NewRateLimiter 返回按客户端 IP 限流的 Gin 中间件（令牌桶算法）。
// rps 为每秒允许请求数，burst 为突发容量；超出则返回 429。
// RATE_LIMIT=0 时不启用（handler 层据此不挂载该中间件）。
func NewRateLimiter(rps float64, burst int) gin.HandlerFunc {
	limiter := newIPLimiter(rps, burst)
	return func(c *gin.Context) {
		if !limiter.get(clientIP(c)).allow(time.Now()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "请求太频繁了，歇会儿再来"})
			return
		}
		c.Next()
	}
}

func clientIP(c *gin.Context) string {
	if ip := c.ClientIP(); ip != "" {
		return ip
	}
	return "unknown"
}
