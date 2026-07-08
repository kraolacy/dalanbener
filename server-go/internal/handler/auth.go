package handler

import (
	"strings"
	"time"
	"unicode/utf8"

	"dalanshu/internal/middleware"
	"dalanshu/internal/model"
	"dalanshu/internal/resp"
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
	c.JSON(resp.Codes.OK, gin.H{"ok": true, "name": "dalanshu", "version": 1})
}

type authReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Avatar   string `json:"avatar"`
}

func (h *Handlers) Register(c *gin.Context) {
	var r authReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.Fail(c, resp.Codes.BadRequest, resp.ErrBind)
		return
	}
	name := strings.TrimSpace(r.Username)
	if utf8.RuneCountInString(name) < 2 {
		resp.Fail(c, resp.Codes.BadRequest, resp.ErrUsernameLen)
		return
	}
	if utf8.RuneCountInString(r.Password) < 4 {
		resp.Fail(c, resp.Codes.BadRequest, resp.ErrPasswordLen)
		return
	}
	u, err := h.user.Register(name, r.Password, r.Avatar)
	if err != nil {
		if err == service.ErrDuplicate {
			resp.Fail(c, resp.Codes.Conflict, resp.ErrDuplicateUser)
			return
		}
		resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		return
	}
	resp.OK(c, gin.H{
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
			resp.Fail(c, resp.Codes.NotFound, resp.ErrUserNotFound)
		case service.ErrPassword:
			resp.Fail(c, resp.Codes.Unauthorized, resp.ErrPasswordWrong)
		default:
			resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		}
		return
	}
	resp.OK(c, gin.H{
		"token": h.signToken(u),
		"user":  model.UserOut{Name: u.Username, Avatar: u.Avatar, Bio: u.Bio},
	})
}

func (h *Handlers) Me(c *gin.Context) {
	u := h.user.Get(middleware.UserID(c))
	if u == nil {
		resp.Fail(c, resp.Codes.Unauthorized, resp.ErrAccountGone)
		return
	}
	resp.OK(c, model.UserOut{Name: u.Username, Avatar: u.Avatar, Bio: u.Bio})
}
