package middleware

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseTokenRoundtrip(t *testing.T) {
	secret := "s3cr3t"
	claims := jwt.MapClaims{"id": float64(42), "name": "bob", "exp": float64(9999999999)}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	id, ok := ParseToken(tok, secret)
	if !ok || id != 42 {
		t.Fatalf("解析失败: id=%d ok=%v", id, ok)
	}
	if _, ok := ParseToken("garbage", secret); ok {
		t.Fatal("非法 token 应失败")
	}
	if _, ok := ParseToken(tok, "wrong-secret"); ok {
		t.Fatal("错误密钥应失败")
	}
}
