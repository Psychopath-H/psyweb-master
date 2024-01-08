package token

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type tokenMaker interface {
	// CreateToken 创建一个token
	CreateToken(username string, duration time.Duration) (string, error)
	// VerifyToken 验证token
	VerifyToken(token string) error
}

type CustomClaims struct {
	// 可根据需要自行添加字段
	UserID               int64  `json:"user_id"`
	Username             string `json:"username"`
	jwt.RegisteredClaims        // 内嵌标准的声明
}
