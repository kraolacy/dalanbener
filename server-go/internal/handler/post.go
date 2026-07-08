package handler

import (
	"strings"

	"dalanshu/internal/middleware"
	"dalanshu/internal/model"
	"dalanshu/internal/resp"
	"dalanshu/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) Posts(c *gin.Context) {
	uid := middleware.UserID(c)
	cursor := c.Query("cursor")
	limit := parseLimit(c.Query("limit"))
	items, next, err := h.post.ListPosts(c.Request.Context(), uid, cursor, limit)
	if err != nil {
		if err == service.ErrBadCursor {
			resp.Fail(c, resp.Codes.BadRequest, resp.ErrInvalidCursor)
			return
		}
		resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		return
	}
	// 仅当显式分页时才返回 {items,next} 信封，默认仍返回纯数组（兼容前端 api.posts()）。
	if cursor != "" || c.Query("limit") != "" {
		resp.OKEnvelope(c, items, next)
		return
	}
	resp.OK(c, items)
}

type postReq struct {
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Cat      string   `json:"cat"`
	Cover    *string  `json:"cover"`
	Image    *string  `json:"image"`
	Tags     []string `json:"tags"`
	Festival bool     `json:"festival"`
}

func (h *Handlers) CreatePost(c *gin.Context) {
	uid := middleware.UserID(c)
	user := h.user.Get(uid)
	if user == nil {
		resp.Fail(c, resp.Codes.Unauthorized, resp.ErrAccountGone)
		return
	}
	var r postReq
	_ = c.ShouldBindJSON(&r)
	title := strings.TrimSpace(r.Title)
	body := strings.TrimSpace(r.Body)
	if title == "" || body == "" {
		resp.Fail(c, resp.Codes.BadRequest, resp.ErrTitleBodyReq)
		return
	}
	cat := r.Cat
	if cat == "" {
		cat = model.DefaultCat
	}
	cover := r.Cover
	if cover != nil && *cover == "" {
		cover = nil
	}
	p, err := h.post.Create(c.Request.Context(), user.Username, user.Avatar, title, body, cat, r.Image, r.Tags, r.Festival)
	if err != nil {
		resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		return
	}
	resp.OK(c, h.post.ShapeSingle(p, uid))
}

type commentReq struct {
	Text string `json:"text"`
}

func (h *Handlers) AddComment(c *gin.Context) {
	uid := middleware.UserID(c)
	user := h.user.Get(uid)
	if user == nil {
		resp.Fail(c, resp.Codes.Unauthorized, resp.ErrAccountGone)
		return
	}
	var r commentReq
	_ = c.ShouldBindJSON(&r)
	text := strings.TrimSpace(r.Text)
	if text == "" {
		resp.Fail(c, resp.Codes.BadRequest, resp.ErrCommentEmpty)
		return
	}
	p, err := h.post.AddComment(c.Request.Context(), c.Param("id"), user.Username, user.Avatar, text)
	if err != nil {
		if err == service.ErrNotFound {
			resp.Fail(c, resp.Codes.NotFound, resp.ErrPostNotFound)
			return
		}
		resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		return
	}
	resp.OK(c, h.post.ShapeSingle(p, uid))
}

func (h *Handlers) ToggleLike(c *gin.Context)    { h.togglePost(c, "like") }
func (h *Handlers) ToggleCollect(c *gin.Context) { h.togglePost(c, "collect") }

func (h *Handlers) togglePost(c *gin.Context, kind string) {
	uid := middleware.UserID(c)
	user := h.user.Get(uid)
	if user == nil {
		resp.Fail(c, resp.Codes.Unauthorized, resp.ErrAccountGone)
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
			resp.Fail(c, resp.Codes.NotFound, resp.ErrPostNotFound)
			return
		}
		resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		return
	}
	resp.OK(c, h.post.ShapeSingle(p, uid))
}
