package middlewares

import (
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/trinhdaiphuc/etcdkeeper/config"
	"github.com/trinhdaiphuc/etcdkeeper/pkg/etcd"
	"time"
)

// JwtCustomClaims are custom claims extending default ones.
// See https://github.com/golang-jwt/jwt for more examples
type JwtCustomClaims struct {
	User *etcd.UserInfo
	jwt.StandardClaims
}

const (
	UserKey = "user"
)

func NewToken(user *etcd.UserInfo) (string, error) {
	// Set custom claims
	claims := &JwtCustomClaims{
		user,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(config.GetConfig().ExpiredTime).Unix(),
		},
	}
	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Generate encoded token and send it as response.
	t, err := token.SignedString(config.GetConfig().SecretKey)
	if err != nil {
		return "", err
	}
	return t, nil
}

func GetUserInfo(c echo.Context) (*etcd.UserInfo, bool) {
	token, ok := c.Get(UserKey).(*jwt.Token)
	if !ok {
		return nil, ok
	}
	if claims, ok := token.Claims.(*JwtCustomClaims); ok && token.Valid {
		return claims.User, true
	}

	return nil, false
}
