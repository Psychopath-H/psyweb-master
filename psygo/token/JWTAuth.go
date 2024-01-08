package token

import (
	"errors"
	"github.com/Psychopath-H/psyweb-master/psygo"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
	"time"
)

type JWTAuth struct {
	duration        time.Duration                     //token第一次创建时的存在时间
	refreshTime     time.Duration                     //token刷新续存的时间
	secretKey       []byte                            //加密AB部分用的私钥
	AuthFailHandler func(c *psygo.Context, err error) //认证失败处理函数
}

func (j *JWTAuth) SetDuration(duration time.Duration) {
	j.duration = duration
}

func (j *JWTAuth) SetRefreshTime(refreshTime time.Duration) {
	j.refreshTime = refreshTime
}

func (j *JWTAuth) SetAuthFailHandler(failHandler func(c *psygo.Context, err error)) {
	j.AuthFailHandler = failHandler
}

// CreateToken 创建token并返回给前端
func (j *JWTAuth) CreateToken(c *psygo.Context, username string, userId int64) error {
	var zeroTime time.Duration
	if j.duration != zeroTime {
		return j.CreateTokenWithDuration(c, username, userId, j.duration)
	}
	return j.CreateTokenWithDuration(c, username, userId, time.Minute*30)

}

func (j *JWTAuth) CreateTokenWithDuration(c *psygo.Context, username string, userId int64, duration time.Duration) error {
	claims := CustomClaims{
		userId,
		username, // 自定义字段
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)), // 定义过期时间
			Issuer:    "Psychopath_H",                               // 签发人
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	var tokenString string
	var err error
	if j.secretKey != nil {
		tokenString, err = token.SignedString(j.secretKey)
	} else {
		tokenString, err = token.SignedString([]byte("moyn8y9abng7q4zkq2m73yw8tu9j5ixm"))
	}
	if err != nil {
		return err
	}
	c.Header("jwt_claims", tokenString)
	return nil
}

// VerifyToken 验证token是否有效
func (j *JWTAuth) VerifyToken(c *psygo.Context) (*CustomClaims, *jwt.Token, bool) {
	auth := c.Req.Header.Get("Authorization")
	if auth == "" {
		j.AuthErrorHandler(c, errors.New("token is nil"))
		return nil, nil, false
	}
	parts := strings.SplitN(auth, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		j.AuthErrorHandler(c, errors.New("token is invalid"))
		return nil, nil, false
	}
	//解析token
	token := parts[1]
	claims := &CustomClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) { //用得哪一套签名算法给他解析回去
		if j.secretKey != nil {
			return j.secretKey, nil
		} else {
			return []byte("moyn8y9abng7q4zkq2m73yw8tu9j5ixm"), nil
		}
	})
	if err != nil || !parsedToken.Valid {
		j.AuthErrorHandler(c, errors.New("token is invalid"))
		return nil, nil, false
	}
	return claims, parsedToken, true
}

// AuthInterceptor jwt中间件 判断header里面是否有对应的token
func (j *JWTAuth) AuthInterceptor() psygo.HandlerFunc {
	return func(c *psygo.Context) {
		paths := strings.Split(c.Req.URL.Path, "/") //组中间件的最后一个路径是login代表正在进行登录操作，不需要校验
		if paths[len(paths)-1] == "login" {
			return
		}
		claims, parsedToken, ok := j.VerifyToken(c)
		if !ok {
			c.Abort()
			return
		}
		// 刷新token
		now := time.Now()
		if claims.ExpiresAt.Time.Sub(now) < time.Second*15 { //如果说离过期时间还有15s，那就刷新token
			var zeroRefreshTime time.Duration
			var RefreshTime time.Duration
			if j.refreshTime != zeroRefreshTime {
				RefreshTime = j.refreshTime
			} else {
				RefreshTime = time.Minute * 30
			}

			claims.ExpiresAt = jwt.NewNumericDate(claims.ExpiresAt.Time.Add(RefreshTime))
			var tokenString string
			if j.secretKey != nil {
				tokenString, _ = parsedToken.SignedString(j.secretKey)
			} else {
				tokenString, _ = parsedToken.SignedString([]byte("moyn8y9abng7q4zkq2m73yw8tu9j5ixm"))
			}
			c.Header("jwt_claims", tokenString)
		}

	}
}

// AuthErrorHandler 认证失败处理函数
func (j *JWTAuth) AuthErrorHandler(c *psygo.Context, err error) {
	if j.AuthFailHandler == nil {
		c.Writer.WriteHeader(http.StatusUnauthorized)
	} else {
		j.AuthFailHandler(c, err)
	}
}
