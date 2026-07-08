package handler

import (
	"strings"

	"dalanshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) Helps(c *gin.Context) {
	cursor := c.Query("cursor")
	limit := parseLimit(c.Query("limit"))
	items, next, err := h.help.List(c.Request.Context(), cursor, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	if cursor != "" || c.Query("limit") != "" {
		c.JSON(200, gin.H{"items": items, "next": next})
		return
	}
	c.JSON(200, items)
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
	hp, err := h.help.Create(c.Request.Context(), user.Username, user.Avatar, title, body, typ, city, reward)
	if err != nil {
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	c.JSON(200, h.help.ShapeSingle(hp))
}
