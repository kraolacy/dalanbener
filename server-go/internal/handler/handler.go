package handler

import (
	"context"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"dalanshu/internal/cache"
	"dalanshu/internal/middleware"
	"dalanshu/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// feedGroup 用于缓存未命中时合并并发请求，防止缓存击穿（cache stampede）。
var feedGroup singleflight.Group

// Handlers 聚合所有 HTTP 处理逻辑与依赖。
type Handlers struct {
	db     *gorm.DB
	cache  *cache.Cache
	secret string
}

func New(database *gorm.DB, c *cache.Cache, secret string) *Handlers {
	return &Handlers{db: database, cache: c, secret: secret}
}

// ===== 鉴权辅助 =====

func (h *Handlers) signToken(user *model.User) string {
	claims := jwt.MapClaims{
		"id":   user.ID,
		"name": user.Username,
		"exp":  time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	t, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.secret))
	return t
}

// ===== 健康检查 =====

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(200, gin.H{"ok": true, "name": "dalanshu", "version": 1})
}

// ===== 账号 =====

type authReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Avatar   string `json:"avatar"`
}

func (h *Handlers) Register(c *gin.Context) {
	var r authReq
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(400, gin.H{"error": "请求格式错误"})
		return
	}
	name := strings.TrimSpace(r.Username)
	if utf8.RuneCountInString(name) < 2 {
		c.JSON(400, gin.H{"error": "用户名至少 2 个字符"})
		return
	}
	if utf8.RuneCountInString(r.Password) < 4 {
		c.JSON(400, gin.H{"error": "密码至少 4 位"})
		return
	}
	var exist model.User
	if err := h.db.Where("username = ?", name).First(&exist).Error; err == nil {
		c.JSON(409, gin.H{"error": "这个用户名已被注册"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(r.Password), 10)
	if err != nil {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	avatar := r.Avatar
	if avatar == "" {
		avatar = "😎"
	}
	user := model.User{
		Username:     name,
		PasswordHash: string(hash),
		Avatar:       avatar,
		Bio:          "新来的散帅，请多关照 🌞",
		CreatedAt:    time.Now().UnixMilli(),
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	c.JSON(200, gin.H{
		"token": h.signToken(&user),
		"user":  model.UserOut{Name: user.Username, Avatar: user.Avatar, Bio: user.Bio},
	})
}

func (h *Handlers) Login(c *gin.Context) {
	var r authReq
	_ = c.ShouldBindJSON(&r)
	name := strings.TrimSpace(r.Username)
	var user model.User
	if err := h.db.Where("username = ?", name).First(&user).Error; err != nil {
		c.JSON(404, gin.H{"error": "用户不存在，去注册一个吧"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(r.Password)) != nil {
		c.JSON(401, gin.H{"error": "密码不对"})
		return
	}
	c.JSON(200, gin.H{
		"token": h.signToken(&user),
		"user":  model.UserOut{Name: user.Username, Avatar: user.Avatar, Bio: user.Bio},
	})
}

func (h *Handlers) Me(c *gin.Context) {
	uid := middleware.UserID(c)
	var user model.User
	if err := h.db.First(&user, uid).Error; err != nil {
		c.JSON(401, gin.H{"error": "账号不存在"})
		return
	}
	c.JSON(200, model.UserOut{Name: user.Username, Avatar: user.Avatar, Bio: user.Bio})
}

// ===== 帖子 =====

func (h *Handlers) Posts(c *gin.Context) {
	ctx := c.Request.Context()
	uid := middleware.UserID(c)

	if h.cache != nil {
		if base, ok := h.cache.GetPosts(ctx); ok {
			c.JSON(200, h.overlayLikes(base, uid))
			return
		}
		// 缓存未命中：singleflight 合并并发请求，仅放行一次 DB 查询（防缓存击穿）。
		v, err, _ := feedGroup.Do("posts", func() (interface{}, error) {
			return h.loadPosts(ctx)
		})
		if err == nil {
			base := v.([]model.PostOut)
			h.cache.SetPosts(ctx, base)
			c.JSON(200, h.overlayLikes(base, uid))
			return
		}
	}

	base, err := h.loadPosts(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	c.JSON(200, h.overlayLikes(base, uid))
}

// loadPosts 从数据库拉取全量帖子并批量组装（不含 per-user 状态，供缓存复用）。
func (h *Handlers) loadPosts(ctx context.Context) ([]model.PostOut, error) {
	var posts []model.Post
	if err := h.db.WithContext(ctx).Order("created_at DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return h.shapePosts(posts), nil
}

// shapePosts 批量组装帖子列表：用 3 条聚合查询取代每帖 4 条查询，消除 N+1。
func (h *Handlers) shapePosts(posts []model.Post) []model.PostOut {
	ids := make([]string, 0, len(posts))
	for i := range posts {
		ids = append(ids, posts[i].ID)
	}
	likeCounts := h.countByPost(&model.Like{}, ids)
	collectCounts := h.countByPost(&model.Collect{}, ids)
	commentsByPost := h.commentsByPost(ids)

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
			Tags:         model.ParseTags(p.Tags),
			Festival:     p.Festival,
			Tall:         p.Tall,
			LikeCount:    p.BaseLikes + likeCounts[p.ID],
			CollectCount: p.BaseCollects + collectCounts[p.ID],
			Comments:     commentsByPost[p.ID],
		})
	}
	return out
}

func (h *Handlers) countByPost(m any, ids []string) map[string]int {
	res := make(map[string]int)
	if len(ids) == 0 {
		return res
	}
	type row struct {
		PostID string
		C      int64
	}
	var rows []row
	h.db.Model(m).Select("post_id, COUNT(*) AS c").Where("post_id IN ?", ids).Group("post_id").Scan(&rows)
	for _, r := range rows {
		res[r.PostID] = int(r.C)
	}
	return res
}

func (h *Handlers) commentsByPost(ids []string) map[string][]model.CommentOut {
	res := make(map[string][]model.CommentOut)
	if len(ids) == 0 {
		return res
	}
	var comments []model.Comment
	h.db.Where("post_id IN ?", ids).Order("created_at ASC").Find(&comments)
	for _, c := range comments {
		res[c.PostID] = append(res[c.PostID], model.CommentOut{Author: c.Author, Avatar: c.Avatar, Text: c.Text})
	}
	return res
}

type postReq struct {
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Cat      string   `json:"cat"`
	Cover    *string  `json:"cover"`
	Tags     []string `json:"tags"`
	Festival bool     `json:"festival"`
}

func (h *Handlers) CreatePost(c *gin.Context) {
	uid := middleware.UserID(c)
	var user model.User
	if h.db.First(&user, uid).Error != nil {
		c.JSON(401, gin.H{"error": "账号不存在"})
		return
	}
	var r postReq
	_ = c.ShouldBindJSON(&r)
	title := strings.TrimSpace(r.Title)
	body := strings.TrimSpace(r.Body)
	if title == "" || body == "" {
		c.JSON(400, gin.H{"error": "标题和正文不能为空"})
		return
	}
	cat := r.Cat
	if cat == "" {
		cat = "rec"
	}
	tags := r.Tags
	if len(tags) > 5 {
		tags = tags[:5]
	}
	cover := r.Cover
	if cover != nil && *cover == "" {
		cover = nil
	}
	post := model.Post{
		ID:           "u" + strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.Itoa(rand.Intn(1000)),
		Cat:          cat,
		Author:       user.Username,
		Avatar:       user.Avatar,
		Title:        title,
		Body:         body,
		Cover:        cover,
		Tags:         model.TagsToJSON(tags),
		Festival:     r.Festival,
		Tall:         utf8.RuneCountInString(body) > 60,
		BaseLikes:    0,
		BaseCollects: 0,
		CreatedAt:    time.Now().UnixMilli(),
	}
	if err := h.db.Create(&post).Error; err != nil {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	if h.cache != nil {
		h.cache.DelPosts(c.Request.Context())
	}
	c.JSON(200, h.shapePost(&post, uid))
}

type commentReq struct {
	Text string `json:"text"`
}

func (h *Handlers) AddComment(c *gin.Context) {
	uid := middleware.UserID(c)
	user := h.currentUser(uid)
	if user == nil {
		c.JSON(401, gin.H{"error": "账号不存在"})
		return
	}
	post := h.postByID(c.Param("id"))
	if post == nil {
		c.JSON(404, gin.H{"error": "帖子不存在"})
		return
	}
	var r commentReq
	_ = c.ShouldBindJSON(&r)
	text := strings.TrimSpace(r.Text)
	if text == "" {
		c.JSON(400, gin.H{"error": "评论不能为空"})
		return
	}
	comment := model.Comment{
		PostID:    post.ID,
		Author:    user.Username,
		Avatar:    user.Avatar,
		Text:      text,
		CreatedAt: time.Now().UnixMilli(),
	}
	if err := h.db.Create(&comment).Error; err != nil {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	if h.cache != nil {
		h.cache.DelPosts(c.Request.Context())
	}
	c.JSON(200, h.shapePost(post, uid))
}

func (h *Handlers) ToggleLike(c *gin.Context)    { h.toggle(c, "like") }
func (h *Handlers) ToggleCollect(c *gin.Context) { h.toggle(c, "collect") }

func (h *Handlers) toggle(c *gin.Context, kind string) {
	uid := middleware.UserID(c)
	user := h.currentUser(uid)
	if user == nil {
		c.JSON(401, gin.H{"error": "账号不存在"})
		return
	}
	post := h.postByID(c.Param("id"))
	if post == nil {
		c.JSON(404, gin.H{"error": "帖子不存在"})
		return
	}
	switch kind {
	case "like":
		h.flip(&model.Like{UserID: uid, PostID: post.ID}, uid, post.ID)
	case "collect":
		h.flip(&model.Collect{UserID: uid, PostID: post.ID}, uid, post.ID)
	}
	if h.cache != nil {
		h.cache.DelPosts(c.Request.Context())
	}
	c.JSON(200, h.shapePost(post, uid))
}

// flip 实现「存在则删、不存在则插」的 toggle 语义（复合主键 + ON CONFLICT DO NOTHING）。
func (h *Handlers) flip(m any, uid int64, postID string) {
	var n int64
	h.db.Model(m).Where("user_id = ? AND post_id = ?", uid, postID).Count(&n)
	if n > 0 {
		h.db.Where("user_id = ? AND post_id = ?", uid, postID).Delete(m)
		return
	}
	h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(m)
}

// ===== 互助 =====

func (h *Handlers) Helps(c *gin.Context) {
	ctx := c.Request.Context()
	if h.cache != nil {
		if hs, ok := h.cache.GetHelps(ctx); ok {
			c.JSON(200, hs)
			return
		}
		v, err, _ := feedGroup.Do("helps", func() (interface{}, error) {
			return h.loadHelps(ctx)
		})
		if err == nil {
			hs := v.([]model.HelpOut)
			h.cache.SetHelps(ctx, hs)
			c.JSON(200, hs)
			return
		}
	}
	hs, err := h.loadHelps(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	c.JSON(200, hs)
}

func (h *Handlers) loadHelps(ctx context.Context) ([]model.HelpOut, error) {
	var helps []model.Help
	if err := h.db.WithContext(ctx).Order("created_at DESC").Find(&helps).Error; err != nil {
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

type helpReq struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Body  string `json:"body"`
	City  string `json:"city"`
}

func (h *Handlers) CreateHelp(c *gin.Context) {
	uid := middleware.UserID(c)
	user := h.currentUser(uid)
	if user == nil {
		c.JSON(401, gin.H{"error": "账号不存在"})
		return
	}
	var r helpReq
	_ = c.ShouldBindJSON(&r)
	title := strings.TrimSpace(r.Title)
	body := strings.TrimSpace(r.Body)
	if title == "" || body == "" {
		c.JSON(400, gin.H{"error": "标题和说明不能为空"})
		return
	}
	typ := "need"
	if r.Type == "offer" {
		typ = "offer"
	}
	city := strings.TrimSpace(r.City)
	if city == "" {
		city = "同城"
	}
	reward := "交个朋友"
	if typ == "need" {
		reward = "当面感谢"
	}
	help := model.Help{
		ID:        "uh" + strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.Itoa(rand.Intn(1000)),
		Type:      typ,
		Author:    user.Username,
		Avatar:    user.Avatar,
		Title:     title,
		Body:      body,
		City:      city,
		Reward:    reward,
		Ts:        "刚刚",
		CreatedAt: time.Now().UnixMilli(),
	}
	if err := h.db.Create(&help).Error; err != nil {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	if h.cache != nil {
		h.cache.DelHelps(c.Request.Context())
	}
	c.JSON(200, model.HelpOut{
		ID: help.ID, Type: help.Type, Author: help.Author, Avatar: help.Avatar,
		Title: help.Title, Body: help.Body, City: help.City, Reward: help.Reward,
		Ts: help.Ts, CreatedAt: help.CreatedAt,
	})
}

// ===== 内部辅助 =====

func (h *Handlers) currentUser(uid int64) *model.User {
	if uid == 0 {
		return nil
	}
	var u model.User
	if err := h.db.First(&u, uid).Error; err != nil {
		return nil
	}
	return &u
}

func (h *Handlers) postByID(id string) *model.Post {
	var p model.Post
	if err := h.db.First(&p, "id = ?", id).Error; err != nil {
		return nil
	}
	return &p
}

// shapePost 单帖组装（含 per-user 点赞/收藏状态），用于写后/详情直返。
func (h *Handlers) shapePost(p *model.Post, uid int64) model.PostOut {
	var likeCount, collectCount, n int64
	h.db.Model(&model.Like{}).Where("post_id = ?", p.ID).Count(&likeCount)
	h.db.Model(&model.Collect{}).Where("post_id = ?", p.ID).Count(&collectCount)

	out := model.PostOut{
		ID:           p.ID,
		Cat:          p.Cat,
		Author:       p.Author,
		Avatar:       p.Avatar,
		Title:        p.Title,
		Body:         p.Body,
		Cover:        p.Cover,
		Tags:         model.ParseTags(p.Tags),
		Festival:     p.Festival,
		Tall:         p.Tall,
		LikeCount:    p.BaseLikes + int(likeCount),
		CollectCount: p.BaseCollects + int(collectCount),
	}
	if uid != 0 {
		h.db.Model(&model.Like{}).Where("user_id = ? AND post_id = ?", uid, p.ID).Count(&n)
		out.Liked = n > 0
		h.db.Model(&model.Collect{}).Where("user_id = ? AND post_id = ?", uid, p.ID).Count(&n)
		out.Collected = n > 0
	}
	var comments []model.Comment
	h.db.Where("post_id = ?", p.ID).Order("created_at ASC").Find(&comments)
	out.Comments = make([]model.CommentOut, 0, len(comments))
	for _, c := range comments {
		out.Comments = append(out.Comments, model.CommentOut{Author: c.Author, Avatar: c.Avatar, Text: c.Text})
	}
	return out
}

// overlayLikes 在基础帖子流上叠加当前用户的点赞/收藏状态。
// 关键：返回新切片，绝不修改入参（缓存中的基础数据），避免并发下共享切片的数据竞争。
func (h *Handlers) overlayLikes(posts []model.PostOut, uid int64) []model.PostOut {
	if uid == 0 {
		return posts
	}
	ids := make([]string, 0, len(posts))
	for _, p := range posts {
		ids = append(ids, p.ID)
	}
	var likedIDs, collectedIDs []string
	h.db.Model(&model.Like{}).Where("user_id = ? AND post_id IN ?", uid, ids).Pluck("post_id", &likedIDs)
	h.db.Model(&model.Collect{}).Where("user_id = ? AND post_id IN ?", uid, ids).Pluck("post_id", &collectedIDs)
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
