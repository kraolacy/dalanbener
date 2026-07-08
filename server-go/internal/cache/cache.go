package cache

import (
	"context"
	"encoding/json"
	"time"

	"dalanshu/internal/model"
	"github.com/redis/go-redis/v9"
)

const (
	keyPosts = "feed:posts"
	keyHelps = "feed:helps"
)

// Cache 封装 Redis 读多写少场景。Redis 不可达时调用方应降级为直查数据库。
type Cache struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(addr, pass string, ttl time.Duration) *Cache {
	return &Cache{
		rdb: redis.NewClient(&redis.Options{Addr: addr, Password: pass}),
		ttl: ttl,
	}
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// ===== posts feed =====

func (c *Cache) GetPosts(ctx context.Context) ([]model.PostOut, bool) {
	b, err := c.rdb.Get(ctx, keyPosts).Bytes()
	if err != nil {
		return nil, false
	}
	var posts []model.PostOut
	if err := json.Unmarshal(b, &posts); err != nil {
		return nil, false
	}
	return posts, true
}

func (c *Cache) SetPosts(ctx context.Context, posts []model.PostOut) {
	b, err := json.Marshal(posts)
	if err != nil {
		return
	}
	_ = c.rdb.Set(ctx, keyPosts, b, c.ttl).Err()
}

func (c *Cache) DelPosts(ctx context.Context) {
	_ = c.rdb.Del(ctx, keyPosts).Err()
}

// ===== helps feed =====

func (c *Cache) GetHelps(ctx context.Context) ([]model.HelpOut, bool) {
	b, err := c.rdb.Get(ctx, keyHelps).Bytes()
	if err != nil {
		return nil, false
	}
	var helps []model.HelpOut
	if err := json.Unmarshal(b, &helps); err != nil {
		return nil, false
	}
	return helps, true
}

func (c *Cache) SetHelps(ctx context.Context, helps []model.HelpOut) {
	b, err := json.Marshal(helps)
	if err != nil {
		return
	}
	_ = c.rdb.Set(ctx, keyHelps, b, c.ttl).Err()
}

func (c *Cache) DelHelps(ctx context.Context) {
	_ = c.rdb.Del(ctx, keyHelps).Err()
}
