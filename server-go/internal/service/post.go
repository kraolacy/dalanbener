package service

import (
	"context"
	"math/rand"
	"strconv"
	"time"
	"unicode/utf8"

	"dalanshu/internal/cache"
	"dalanshu/internal/db"
	"dalanshu/internal/model"

	"gorm.io/gorm/clause"
)

// PostService 帖子域业务：feed 聚合、游标分页（C2）、点赞/收藏 toggle、缓存编排。
type PostService struct {
	db    *db.DBSet
	cache *FeedCache
}

func NewPostService(d *db.DBSet, c *cache.Cache) *PostService {
	return &PostService{db: d, cache: newFeedCache(c)}
}

// ListPosts 帖子流。
//   - cursor=="" 且 limit<=0：全量兼容模式（缓存 + singleflight），返回纯数组，next 为空。
//   - 否则：keyset 游标分页，返回 (items, next, err)；next 为空表示末页。
func (s *PostService) ListPosts(ctx context.Context, uid int64, cursor string, limit int) ([]model.PostOut, string, error) {
	if cursor == "" && limit <= 0 {
		return s.listAll(ctx, uid)
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
	raw, hasNext, err := s.pagePosts(ctx, ca, cid, limit)
	if err != nil {
		return nil, "", err
	}
	out := s.overlayLikes(s.shapePosts(raw), uid)
	next := ""
	if hasNext && len(raw) > 0 {
		last := raw[len(raw)-1]
		next = encodeCursor(last.CreatedAt, last.ID)
	}
	return out, next, nil
}

// listAll 全量兼容模式：缓存 + singleflight 合并并发，返回带 per-user 状态的数组。
func (s *PostService) listAll(ctx context.Context, uid int64) ([]model.PostOut, string, error) {
	if base, ok := s.cache.getPosts(ctx); ok {
		return s.overlayLikes(base, uid), "", nil
	}
	if s.cache.c != nil {
		v, err, _ := feedGroup.Do("posts", func() (any, error) {
			raw, e := s.allPosts(ctx)
			if e != nil {
				return nil, e
			}
			return s.shapePosts(raw), nil
		})
		if err == nil {
			base := v.([]model.PostOut)
			s.cache.setPosts(ctx, base)
			return s.overlayLikes(base, uid), "", nil
		}
	}
	raw, err := s.allPosts(ctx)
	if err != nil {
		return nil, "", err
	}
	return s.overlayLikes(s.shapePosts(raw), uid), "", nil
}

// pagePosts keyset 分页：基于 (created_at,id) 游标向后取一页（多取 1 条判断 hasNext）。
func (s *PostService) pagePosts(ctx context.Context, ca int64, cid string, limit int) ([]model.Post, bool, error) {
	q := s.db.R().WithContext(ctx).Model(&model.Post{})
	if ca > 0 {
		q = q.Where("(created_at < ?) OR (created_at = ? AND id < ?)", ca, ca, cid)
	}
	var posts []model.Post
	if err := q.Order("created_at DESC, id DESC").Limit(limit + 1).Find(&posts).Error; err != nil {
		return nil, false, err
	}
	hasNext := len(posts) > limit
	if hasNext {
		posts = posts[:limit]
	}
	return posts, hasNext, nil
}

func (s *PostService) allPosts(ctx context.Context) ([]model.Post, error) {
	var posts []model.Post
	if err := s.db.R().WithContext(ctx).Order("created_at DESC, id DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

// shapePosts 批量组装帖子列表：用 3 条聚合查询取代每帖 4 条查询，消除 N+1。
func (s *PostService) shapePosts(posts []model.Post) []model.PostOut {
	ids := make([]string, 0, len(posts))
	for i := range posts {
		ids = append(ids, posts[i].ID)
	}
	likeCounts := s.countByPost(&model.Like{}, ids)
	collectCounts := s.countByPost(&model.Collect{}, ids)
	commentsByPost := s.commentsByPost(ids)

	out := make([]model.PostOut, 0, len(posts))
	for i := range posts {
		p := &posts[i]
	out = append(out, model.PostOut{
		ID:           p.ID,
		Cat:          p.Cat,
		Author:       p.Author,
		Avatar:       p.Avatar,
		Title:        p.Title,
		Body:         p.Body,
		Cover:        p.Cover,
		Image:        p.Image,
		Tags:         model.ParseTags(p.Tags),
			Festival:     p.Festival,
			Tall:         p.Tall,
			LikeCount:    p.BaseLikes + likeCounts[p.ID],
			CollectCount: p.BaseCollects + collectCounts[p.ID],
			Comments:     commentsByPost[p.ID],
			CreatedAt:    p.CreatedAt,
		})
	}
	return out
}

func (s *PostService) countByPost(m any, ids []string) map[string]int {
	res := make(map[string]int)
	if len(ids) == 0 {
		return res
	}
	type row struct {
		PostID string
		C      int64
	}
	var rows []row
	s.db.R().Model(m).Select("post_id, COUNT(*) AS c").Where("post_id IN ?", ids).Group("post_id").Scan(&rows)
	for _, r := range rows {
		res[r.PostID] = int(r.C)
	}
	return res
}

func (s *PostService) commentsByPost(ids []string) map[string][]model.CommentOut {
	res := make(map[string][]model.CommentOut)
	if len(ids) == 0 {
		return res
	}
	var comments []model.Comment
	s.db.R().Where("post_id IN ?", ids).Order("created_at ASC").Find(&comments)
	for _, c := range comments {
		res[c.PostID] = append(res[c.PostID], model.CommentOut{Author: c.Author, Avatar: c.Avatar, Text: c.Text})
	}
	return res
}

// overlayLikes 在基础帖子流上叠加当前用户的点赞/收藏状态。
// 关键：返回新切片，绝不修改入参（缓存中的基础数据），避免并发下共享切片的数据竞争。
func (s *PostService) overlayLikes(posts []model.PostOut, uid int64) []model.PostOut {
	if uid == 0 {
		return posts
	}
	ids := make([]string, 0, len(posts))
	for _, p := range posts {
		ids = append(ids, p.ID)
	}
	var likedIDs, collectedIDs []string
	s.db.R().Model(&model.Like{}).Where("user_id = ? AND post_id IN ?", uid, ids).Pluck("post_id", &likedIDs)
	s.db.R().Model(&model.Collect{}).Where("user_id = ? AND post_id IN ?", uid, ids).Pluck("post_id", &collectedIDs)
	liked := make(map[string]bool, len(likedIDs))
	for _, id := range likedIDs {
		liked[id] = true
	}
	collected := make(map[string]bool, len(collectedIDs))
	for _, id := range collectedIDs {
		collected[id] = true
	}
	out := make([]model.PostOut, len(posts))
	for i, p := range posts {
		p.Liked = liked[p.ID]
		p.Collected = collected[p.ID]
		out[i] = p
	}
	return out
}

// Create 发布帖子（写主库）。
func (s *PostService) Create(ctx context.Context, author, avatar, title, body, cat string, image *string, tags []string, festival bool) (*model.Post, error) {
	if len(tags) > 5 {
		tags = tags[:5]
	}
	post := model.Post{
		ID:           "u" + strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.Itoa(rand.Intn(1000)),
		Cat:          cat,
		Author:       author,
		Avatar:       avatar,
		Title:        title,
		Body:         body,
		Image:        image,
		Tags:         model.TagsToJSON(tags),
		Festival:     festival,
		Tall:         utf8.RuneCountInString(body) > 60,
		BaseLikes:    0,
		BaseCollects: 0,
		CreatedAt:    time.Now().UnixMilli(),
	}
	if err := s.db.W().WithContext(ctx).Create(&post).Error; err != nil {
		return nil, err
	}
	s.cache.delPosts(ctx)
	return &post, nil
}

// AddComment 评论（写主库）。
func (s *PostService) AddComment(ctx context.Context, postID, author, avatar, text string) (*model.Post, error) {
	post := s.GetByID(postID)
	if post == nil {
		return nil, ErrNotFound
	}
	comment := model.Comment{
		PostID:    postID,
		Author:    author,
		Avatar:    avatar,
		Text:      text,
		CreatedAt: time.Now().UnixMilli(),
	}
	if err := s.db.W().WithContext(ctx).Create(&comment).Error; err != nil {
		return nil, err
	}
	s.cache.delPosts(ctx)
	return post, nil
}

// ToggleLike / ToggleCollect 点赞/收藏切换（写主库）。
func (s *PostService) ToggleLike(ctx context.Context, uid int64, postID string) (*model.Post, error) {
	post := s.GetByID(postID)
	if post == nil {
		return nil, ErrNotFound
	}
	if err := s.flip(ctx, &model.Like{UserID: uid, PostID: postID}, uid, postID); err != nil {
		return nil, err
	}
	s.cache.delPosts(ctx)
	return post, nil
}

func (s *PostService) ToggleCollect(ctx context.Context, uid int64, postID string) (*model.Post, error) {
	post := s.GetByID(postID)
	if post == nil {
		return nil, ErrNotFound
	}
	if err := s.flip(ctx, &model.Collect{UserID: uid, PostID: postID}, uid, postID); err != nil {
		return nil, err
	}
	s.cache.delPosts(ctx)
	return post, nil
}

// flip 实现「存在则删、不存在则插」的 toggle 语义（复合主键 + ON CONFLICT DO NOTHING）。
func (s *PostService) flip(ctx context.Context, m any, uid int64, postID string) error {
	var n int64
	s.db.W().WithContext(ctx).Model(m).Where("user_id = ? AND post_id = ?", uid, postID).Count(&n)
	if n > 0 {
		return s.db.W().WithContext(ctx).Where("user_id = ? AND post_id = ?", uid, postID).Delete(m).Error
	}
	return s.db.W().WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(m).Error
}

// GetByID 读主/从库取单帖（详情/写前校验）。
func (s *PostService) GetByID(id string) *model.Post {
	var p model.Post
	if err := s.db.R().First(&p, "id = ?", id).Error; err != nil {
		return nil
	}
	return &p
}

// ShapeSingle 单帖组装（含 per-user 状态），用于写后/详情直返。
func (s *PostService) ShapeSingle(p *model.Post, uid int64) model.PostOut {
	var likeCount, collectCount, n int64
	s.db.R().Model(&model.Like{}).Where("post_id = ?", p.ID).Count(&likeCount)
	s.db.R().Model(&model.Collect{}).Where("post_id = ?", p.ID).Count(&collectCount)

	out := model.PostOut{
		ID:           p.ID,
		Cat:          p.Cat,
		Author:       p.Author,
		Avatar:       p.Avatar,
		Title:        p.Title,
		Body:         p.Body,
		Cover:        p.Cover,
		Image:        p.Image,
		Tags:         model.ParseTags(p.Tags),
		Festival:     p.Festival,
		Tall:         p.Tall,
		LikeCount:    p.BaseLikes + int(likeCount),
		CollectCount: p.BaseCollects + int(collectCount),
		CreatedAt:    p.CreatedAt,
	}
	if uid != 0 {
		s.db.R().Model(&model.Like{}).Where("user_id = ? AND post_id = ?", uid, p.ID).Count(&n)
		out.Liked = n > 0
		s.db.R().Model(&model.Collect{}).Where("user_id = ? AND post_id = ?", uid, p.ID).Count(&n)
		out.Collected = n > 0
	}
	var comments []model.Comment
	s.db.R().Where("post_id = ?", p.ID).Order("created_at ASC").Find(&comments)
	out.Comments = make([]model.CommentOut, 0, len(comments))
	for _, c := range comments {
		out.Comments = append(out.Comments, model.CommentOut{Author: c.Author, Avatar: c.Avatar, Text: c.Text})
	}
	return out
}
