package handler

import (
	"strings"

	"dalanshu/internal/middleware"
	"dalanshu/internal/model"
	"dalanshu/internal/resp"
	"dalanshu/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) Helps(c *gin.Context) {
	cursor := c.Query("cursor")
	limit := parseLimit(c.Query("limit"))
	items, next, err := h.help.List(c.Request.Context(), cursor, limit)
	if err != nil {
		if err == service.ErrBadCursor {
			resp.Fail(c, resp.Codes.BadRequest, resp.ErrInvalidCursor)
			return
		}
		resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		return
	}
	if cursor != "" || c.Query("limit") != "" {
		resp.OKEnvelope(c, items, next)
		return
	}
	resp.OK(c, items)
}

type helpReq struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Body  string `json:"body"`
	City  string `json:"city"`
}

func (h *Handlers) CreateHelp(c *gin.Context) {
	uid := middleware.UserID(c)
	user := h.user.Get(uid)
	if user == nil {
		resp.Fail(c, resp.Codes.Unauthorized, resp.ErrAccountGone)
		return
	}
	var r helpReq
	_ = c.ShouldBindJSON(&r)
	title := strings.TrimSpace(r.Title)
	body := strings.TrimSpace(r.Body)
	if title == "" || body == "" {
		resp.Fail(c, resp.Codes.BadRequest, resp.ErrHelpReq)
		return
	}
	typ := "need"
	if r.Type == "offer" {
		typ = "offer"
	}
	city := strings.TrimSpace(r.City)
	if city == "" {
		city = model.DefaultCity
	}
	reward := model.DefaultReward
	if typ == "need" {
		reward = model.NeedReward
	}
	hp, err := h.help.Create(c.Request.Context(), user.Username, user.Avatar, title, body, typ, city, reward)
	if err != nil {
		resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		return
	}
	resp.OK(c, h.help.ShapeSingle(hp))
}
