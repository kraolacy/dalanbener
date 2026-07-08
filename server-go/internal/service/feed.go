package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"

	"dalanshu/internal/cache"
	"dalanshu/internal/model"

	"golang.org/x/sync/singleflight"
)

// 业务层统一错误，便于 handler 映射 HTTP 状态。
var (
	ErrNotFound  = errors.New("资源不存在")
	ErrDuplicate = errors.New("资源已存在")
	ErrPassword  = errors.New("密码错误")
	ErrBadCursor = errors.New("非法游标")
)

// feedGroup 合并并发 feed 请求，防止缓存击穿（cache stampede）。
var feedGroup singleflight.Group

// 游标分页的默认页大小与上限（避免单页过大导致深翻页性能塌陷）。
const (
	defaultPageLimit = 20
	maxPageLimit     = 50
)

// encodeCursor 将 (createdAt,id) 编码为 base64 游标（URL 安全）。
func encodeCursor(createdAt int64, id string) string {
	raw := strconv.FormatInt(createdAt, 10) + "," + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeCursor 解析游标，空串表示首页。
func decodeCursor(cur string) (int64, string, error) {
	if cur == "" {
		return 0, "", nil
	}
	b, err := base64.RawURLEncoding.DecodeString(cur)
	if err != nil {
		return 0, "", ErrBadCursor
	}
	parts := strings.SplitN(string(b), ",", 2)
	if len(parts) != 2 {
		return 0, "", ErrBadCursor
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", ErrBadCursor
	}
	return ts, parts[1], nil
}

// FeedCache 封装 feed 列表的缓存读写（posts / helps 复用）。
type FeedCache struct {
	c *cache.Cache
}

func newFeedCache(c *cache.Cache) *FeedCache { return &FeedCache{c: c} }

func (f *FeedCache) getPosts(ctx context.Context) ([]model.PostOut, bool) {
	if f.c == nil {
		return nil, false
	}
	return f.c.GetPosts(ctx)
}

func (f *FeedCache) setPosts(ctx context.Context, p []model.PostOut) {
	if f.c == nil {
		return
	}
	f.c.SetPosts(ctx, p)
}

func (f *FeedCache) delPosts(ctx context.Context) {
	if f.c == nil {
		return
	}
	f.c.DelPosts(ctx)
}

func (f *FeedCache) getHelps(ctx context.Context) ([]model.HelpOut, bool) {
	if f.c == nil {
		return nil, false
	}
	return f.c.GetHelps(ctx)
}

func (f *FeedCache) setHelps(ctx context.Context, h []model.HelpOut) {
	if f.c == nil {
		return
	}
	f.c.SetHelps(ctx, h)
}

func (f *FeedCache) delHelps(ctx context.Context) {
	if f.c == nil {
		return
	}
	f.c.DelHelps(ctx)
}
