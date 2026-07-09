package handler

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"dalanshu/internal/middleware"
	"dalanshu/internal/resp"
	"dalanshu/internal/service"

	"github.com/gin-gonic/gin"
)

// dataURLRe 校验并抽取 base64 图片 dataURL 的扩展名与载荷。
var dataURLRe = regexp.MustCompile(`^data:image/(png|jpe?g|gif|webp);base64,(.+)$`)

const maxUploadBytes = 3 * 1024 * 1024 // 3MB

type uploadReq struct {
	DataURL string `json:"dataUrl"`
}

// Upload 接收 base64 图片，落盘到 UploadDir，返回可公开访问的 /uploads/<file>。
func (h *Handlers) Upload(c *gin.Context) {
	var r uploadReq
	if err := c.ShouldBindJSON(&r); err != nil {
		resp.Fail(c, resp.Codes.BadRequest, resp.ErrBind)
		return
	}
	m := dataURLRe.FindStringSubmatch(strings.TrimSpace(r.DataURL))
	if m == nil {
		resp.Fail(c, resp.Codes.BadRequest, resp.ErrUploadType)
		return
	}
	ext := m[1]
	if ext == "jpeg" {
		ext = "jpg"
	}
	buf, err := base64.StdEncoding.DecodeString(m[2])
	if err != nil {
		resp.Fail(c, resp.Codes.BadRequest, resp.ErrUploadType)
		return
	}
	if len(buf) > maxUploadBytes {
		resp.Fail(c, resp.Codes.PayloadTooLarge, resp.ErrUploadTooLarge)
		return
	}
	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		return
	}
	name := fmt.Sprintf("%d_%d.%s", time.Now().UnixNano(), rand.Int63(), ext)
	if err := os.WriteFile(filepath.Join(h.uploadDir, name), buf, 0o644); err != nil {
		resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		return
	}
	resp.OK(c, gin.H{"url": "/uploads/" + name})
}

// Follow toggle 关注/取关，返回最新用户对象（同 me）。
func (h *Handlers) Follow(c *gin.Context) {
	uid := middleware.UserID(c)
	if err := h.social.Follow(uid, c.Param("name")); err != nil {
		switch err {
		case service.ErrFollowSelf:
			resp.Fail(c, resp.Codes.BadRequest, resp.ErrFollowSelf)
		case service.ErrNotFound:
			resp.Fail(c, resp.Codes.Unauthorized, resp.ErrAccountGone)
		default:
			resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		}
		return
	}
	profile, err := h.social.Profile(uid)
	if err != nil {
		resp.Fail(c, resp.Codes.Unauthorized, resp.ErrAccountGone)
		return
	}
	resp.OK(c, profile)
}

// Conversations 返回当前用户会话列表。
func (h *Handlers) Conversations(c *gin.Context) {
	uid := middleware.UserID(c)
	resp.OK(c, h.social.Conversations(uid))
}

// Thread 返回与某人的私信线程，并标记已读。
func (h *Handlers) Thread(c *gin.Context) {
	uid := middleware.UserID(c)
	t, err := h.social.Thread(uid, c.Param("name"))
	if err != nil {
		resp.Fail(c, resp.Codes.Unauthorized, resp.ErrAccountGone)
		return
	}
	resp.OK(c, t)
}

type sendMsgReq struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

// SendMessage 发送私信。
func (h *Handlers) SendMessage(c *gin.Context) {
	uid := middleware.UserID(c)
	var r sendMsgReq
	_ = c.ShouldBindJSON(&r)
	if err := h.social.SendMessage(uid, r.To, strings.TrimSpace(r.Text)); err != nil {
		switch err {
		case service.ErrMsgEmpty:
			resp.Fail(c, resp.Codes.BadRequest, resp.ErrMsgEmpty)
		case service.ErrMsgSelf:
			resp.Fail(c, resp.Codes.BadRequest, resp.ErrMsgSelf)
		case service.ErrMsgTargetNotFound:
			resp.Fail(c, resp.Codes.NotFound, resp.ErrMsgTargetNotFound)
		case service.ErrNotFound:
			resp.Fail(c, resp.Codes.Unauthorized, resp.ErrAccountGone)
		default:
			resp.Fail(c, resp.Codes.Internal, resp.ErrServerBusy)
		}
		return
	}
	resp.OK(c, gin.H{"ok": true})
}
