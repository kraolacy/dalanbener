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

// List 互助流（C1 全量兼容模式；C2 扩展游标分页）。
func (s *HelpService) List(ctx context.Context) ([]model.HelpOut, error) {
	if base, ok := s.cache.getHelps(ctx); ok {
		return base, nil
	}
	if s.cache.c != nil {
		v, err, _ := feedGroup.Do("helps", func() (any, error) {
			return s.allHelps(ctx)
		})
		if err == nil {
			base := v.([]model.HelpOut)
			s.cache.setHelps(ctx, base)
			return base, nil
		}
	}
	return s.allHelps(ctx)
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
		Ts:        "刚刚",
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
