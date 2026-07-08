// Package resp 统一管理 HTTP 响应：状态码语义常量、错误文案表、成功/失败写出。
// 所有 handler / middleware 只应通过本包写响应，杜绝散落的魔法字符串与字面量状态码。
package resp

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Codes 是 HTTP 状态码的语义常量，替代散落的字面量（400/401/...）。
var Codes = struct {
	OK               int
	BadRequest       int
	Unauthorized     int
	NotFound         int
	Conflict         int
	TooManyRequests  int
	PayloadTooLarge  int
	Internal         int
}{
	OK:              http.StatusOK,
	BadRequest:      http.StatusBadRequest,
	Unauthorized:    http.StatusUnauthorized,
	NotFound:        http.StatusNotFound,
	Conflict:        http.StatusConflict,
	TooManyRequests: http.StatusTooManyRequests,
	PayloadTooLarge: http.StatusRequestEntityTooLarge,
	Internal:        http.StatusInternalServerError,
}

// 错误文案表：键为 Err* 常量，值为模板（支持 fmt.Sprintf 插值）。
// 业务层只持有哨兵错误（用于逻辑判断），展示文案统一在此集中管理。
const (
	ErrBind          = "请求格式错误"
	ErrUsernameLen   = "用户名至少 2 个字符"
	ErrPasswordLen   = "密码至少 4 位"
	ErrDuplicateUser = "这个用户名已被注册"
	ErrUserNotFound  = "用户不存在，去注册一个吧"
	ErrPasswordWrong = "密码不对"
	ErrAccountGone   = "账号不存在"
	ErrTitleBodyReq  = "标题和正文不能为空"
	ErrCommentEmpty  = "评论不能为空"
	ErrPostNotFound  = "帖子不存在"
	ErrHelpReq       = "标题和说明不能为空"
	ErrInvalidCursor = "非法游标"
	ErrRateLimited   = "请求太频繁了，歇会儿再来"
	ErrServerBusy    = "服务器开小差了"
	// 社交相关
	ErrFollowSelf        = "不能关注自己"
	ErrMsgEmpty          = "消息不能为空"
	ErrMsgSelf           = "不能给自己发私信"
	ErrMsgTargetNotFound = "对方还不是注册用户，暂时无法私信"
	ErrUploadType        = "仅支持 png/jpg/gif/webp 图片"
	ErrUploadTooLarge    = "图片太大（请 ≤ 3MB）"
)

// Fail 写出标准化失败响应：{"error": <文案>} + 对应状态码，并中止后续处理链。
// 文案支持 fmt.Sprintf 风格的参数插值。
func Fail(c *gin.Context, code int, key string, args ...any) {
	msg := key
	if len(args) > 0 {
		msg = fmt.Sprintf(key, args...)
	}
	c.AbortWithStatusJSON(code, gin.H{"error": msg})
}

// OK 写出成功响应（任意数据体）。
func OK(c *gin.Context, data any) {
	c.JSON(Codes.OK, data)
}

// OKEnvelope 写出游标分页信封：{items, next}。
func OKEnvelope(c *gin.Context, items any, next string) {
	c.JSON(Codes.OK, gin.H{"items": items, "next": next})
}
