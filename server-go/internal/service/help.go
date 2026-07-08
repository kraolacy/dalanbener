package service

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"dalanshu/internal/cache"
	"dalanshu/internal/db"
	"dalanshu/internal/model"
)

// HelpService 互助域业务：列表聚合、发布、缓存编排。
type HelpService struct {
	db    *db.DBSet
	cache *FeedCache
}

func NewHelpService(d *db.DBSet, c *cache.Cache) *HelpService {
	return &HelpService{db: d, cache: newFeedCache(c)}
}

// List 互助流。
//   - cursor=="" 且 limit<=0：全量兼容模式，返回纯数组。
//   - 否则：keyset 游标分页，返回 (items, next, err)。
func (s *HelpService) List(ctx context.Context, cursor string, limit int) ([]model.HelpOut, string, error) {
	if cursor == "" && limit <= 0 {
		return s.listAll(ctx)
	}
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	ca, cid, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	raw, hasNext, err := s.pageHelps(ctx, ca, cid, limit)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if hasNext && len(raw) > 0 {
		last := raw[len(raw)-1]
		next = encodeCursor(last.CreatedAt, last.ID)
	}
	return s.shapeHelps(raw), next, nil
}

func (s *HelpService) listAll(ctx context.Context) ([]model.HelpOut, string, error) {
	if base, ok := s.cache.getHelps(ctx); ok {
		return base, "", nil
	}
	if s.cache.c != nil {
		v, err, _ := feedGroup.Do("helps", func() (any, error) {
			return s.allHelps(ctx)
		})
		if err == nil {
			base := v.([]model.HelpOut)
			s.cache.setHelps(ctx, base)
			return base, "", nil
		}
	}
	out, err := s.allHelps(ctx)
	return out, "", err
}

// pageHelps keyset 分页。
func (s *HelpService) pageHelps(ctx context.Context, ca int64, cid string, limit int) ([]model.Help, bool, error) {
	q := s.db.R().WithContext(ctx).Model(&model.Help{})
	if ca > 0 {
		q = q.Where("(created_at < ?) OR (created_at = ? AND id < ?)", ca, ca, cid)
	}
	var helps []model.Help
	if err := q.Order("created_at DESC, id DESC").Limit(limit + 1).Find(&helps).Error; err != nil {
		return nil, false, err
	}
	hasNext := len(helps) > limit
	if hasNext {
		helps = helps[:limit]
	}
	return helps, hasNext, nil
}

func (s *HelpService) shapeHelps(helps []model.Help) []model.HelpOut {
	out := make([]model.HelpOut, 0, len(helps))
	for _, hp := range helps {
		out = append(out, model.HelpOut{
			ID: hp.ID, Type: hp.Type, Author: hp.Author, Avatar: hp.Avatar,
			Title: hp.Title, Body: hp.Body, City: hp.City, Reward: hp.Reward,
			Ts: hp.Ts, CreatedAt: hp.CreatedAt,
		})
	}
	return out
}

func (s *HelpService) allHelps(ctx context.Context) ([]model.HelpOut, error) {
	var helps []model.Help
	if err := s.db.R().WithContext(ctx).Order("created_at DESC").Find(&helps).Error; err != nil {
		return nil, err
	}
	out := make([]model.HelpOut, 0, len(helps))
	for _, hp := range helps {
		out = append(out, model.HelpOut{
			ID: hp.ID, Type: hp.Type, Author: hp.Author, Avatar: hp.Avatar,
			Title: hp.Title, Body: hp.Body, City: hp.City, Reward: hp.Reward,
			Ts: hp.Ts, CreatedAt: hp.CreatedAt,
		})
	}
	return out, nil
}

// Create 发布互助（写主库）。
func (s *HelpService) Create(ctx context.Context, author, avatar, title, body, typ, city, reward string) (*model.Help, error) {
	help := model.Help{
		ID:        "uh" + strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.Itoa(rand.Intn(1000)),
		Type:      typ,
		Author:    author,
		Avatar:    avatar,
		Title:     title,
		Body:      body,
		City:      city,
		Reward:    reward,
		Ts:        model.DefaultTs,
		CreatedAt: time.Now().UnixMilli(),
	}
	if err := s.db.W().WithContext(ctx).Create(&help).Error; err != nil {
		return nil, err
	}
	s.cache.delHelps(ctx)
	return &help, nil
}

// ShapeSingle 单条互助组装。
func (s *HelpService) ShapeSingle(h *model.Help) model.HelpOut {
	return model.HelpOut{
		ID: h.ID, Type: h.Type, Author: h.Author, Avatar: h.Avatar,
		Title: h.Title, Body: h.Body, City: h.City, Reward: h.Reward,
		Ts: h.Ts, CreatedAt: h.CreatedAt,
	}
}
