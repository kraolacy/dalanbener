package handler

import (
	"strings"
	"time"
	"unicode/utf8"

	"dalanshu/internal/middleware"
	"dalanshu/internal/model"
	"dalanshu/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func (h *Handlers) signToken(user *model.User) string {
	claims := jwt.MapClaims{
		"id":   user.ID,
		"name": user.Username,
		"exp":  time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	t, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.secret))
	return t
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(200, gin.H{"ok": true, "name": "dalanshu", "version": 1})
}

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
	u, err := h.user.Register(name, r.Password, r.Avatar)
	if err != nil {
		if err == service.ErrDuplicate {
			c.JSON(409, gin.H{"error": "这个用户名已被注册"})
			return
		}
		c.JSON(500, gin.H{"error": "服务器开小差了"})
		return
	}
	c.JSON(200, gin.H{
		"token": h.signToken(u),
		"user":  model.UserOut{Name: u.Username, Avatar: u.Avatar, Bio: u.Bio},
	})
}

func (h *Handlers) Login(c *gin.Context) {
	var r authReq
	_ = c.ShouldBindJSON(&r)
	name := strings.TrimSpace(r.Username)
	u, err := h.user.Login(name, r.Password)
	if err != nil {
		switch err {
		case service.ErrNotFound:
			c.JSON(404, gin.H{"error": "用户不存在，去注册一个吧"})
		case service.ErrPassword:
			c.JSON(401, gin.H{"error": "密码不对"})
		default:
			c.JSON(500, gin.H{"error": "服务器开小差了"})
		}
		return
	}
	c.JSON(200, gin.H{
		"token": h.signToken(u),
		"user":  model.UserOut{Name: u.Username, Avatar: u.Avatar, Bio: u.Bio},
	})
}

func (h *Handlers) Me(c *gin.Context) {
	u := h.user.Get(middleware.UserID(c))
	if u == nil {
		c.JSON(401, gin.H{"error": "账号不存在"})
		return
	}
	c.JSON(200, model.UserOut{Name: u.Username, Avatar: u.Avatar, Bio: u.Bio})
}
