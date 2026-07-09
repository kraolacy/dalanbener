package middleware

import (
	"strings"

	"dalanshu/internal/resp"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const userIDKey = "uid"

// ParseToken 校验 JWT 并返回用户 ID（无效返回 (0,false)）。
func ParseToken(tokenString, secret string) (int64, bool) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return 0, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, false
	}
	id, ok := claims["id"].(float64)
	if !ok {
		return 0, false
	}
	return int64(id), true
}

func bearer(raw string) string {
	if strings.HasPrefix(raw, "Bearer ") {
		return raw[7:]
	}
	return ""
}

// RequireAuth 强制鉴权，失败返回 401。
func RequireAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, ok := ParseToken(bearer(c.GetHeader("Authorization")), secret)
		if !ok {
			resp.Fail(c, resp.Codes.Unauthorized, "请先登录")
			return
		}
		c.Set(userIDKey, uid)
		c.Next()
	}
}

// OptionalUser 解析 token（不强制）；用于只读接口按当前用户叠加 liked/collected。
func OptionalUser(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if t := bearer(c.GetHeader("Authorization")); t != "" {
			if uid, ok := ParseToken(t, secret); ok {
				c.Set(userIDKey, uid)
			}
		}
		c.Next()
	}
}

// UserID 取上下文中的用户 ID，未登录返回 0。
func UserID(c *gin.Context) int64 {
	if v, ok := c.Get(userIDKey); ok {
		if uid, ok := v.(int64); ok {
			return uid
		}
	}
	return 0
}
