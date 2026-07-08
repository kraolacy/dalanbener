package handler

import (
	"strings"

	"dalanshu/internal/middleware"
	"dalanshu/internal/model"
	"dalanshu/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) Posts(c *gin.Context) {
	items, err := h.post.ListPosts(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	c.JSON(200, items)
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
	user := h.user.Get(uid)
	if user == nil {
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
	cover := r.Cover
	if cover != nil && *cover == "" {
		cover = nil
	}
	p, err := h.post.Create(c.Request.Context(), user.Username, user.Avatar, title, body, cat, r.Tags, r.Festival)
	if err != nil {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	c.JSON(200, h.post.ShapeSingle(p, uid))
}

type commentReq struct {
	Text string `json:"text"`
}

func (h *Handlers) AddComment(c *gin.Context) {
	uid := middleware.UserID(c)
	user := h.user.Get(uid)
	if user == nil {
		c.JSON(401, gin.H{"error": "账号不存在"})
		return
	}
	var r commentReq
	_ = c.ShouldBindJSON(&r)
	text := strings.TrimSpace(r.Text)
	if text == "" {
		c.JSON(400, gin.H{"error": "评论不能为空"})
		return
	}
	p, err := h.post.AddComment(c.Request.Context(), c.Param("id"), user.Username, user.Avatar, text)
	if err != nil {
		if err == service.ErrNotFound {
			c.JSON(404, gin.H{"error": "帖子不存在"})
			return
		}
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	c.JSON(200, h.post.ShapeSingle(p, uid))
}

func (h *Handlers) ToggleLike(c *gin.Context)    { h.togglePost(c, "like") }
func (h *Handlers) ToggleCollect(c *gin.Context) { h.togglePost(c, "collect") }

func (h *Handlers) togglePost(c *gin.Context, kind string) {
	uid := middleware.UserID(c)
	user := h.user.Get(uid)
	if user == nil {
		c.JSON(401, gin.H{"error": "账号不存在"})
		return
	}
	var (
		p   *model.Post
		err error
	)
	switch kind {
	case "like":
		p, err = h.post.ToggleLike(c.Request.Context(), uid, c.Param("id"))
	case "collect":
		p, err = h.post.ToggleCollect(c.Request.Context(), uid, c.Param("id"))
	}
	if err != nil {
		if err == service.ErrNotFound {
			c.JSON(404, gin.H{"error": "帖子不存在"})
			return
		}
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	c.JSON(200, h.post.ShapeSingle(p, uid))
}
