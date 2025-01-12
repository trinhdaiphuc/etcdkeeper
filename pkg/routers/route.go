package routers

import (
	"errors"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/trinhdaiphuc/etcdkeeper/config"
	"github.com/trinhdaiphuc/etcdkeeper/pkg/controllers"
	"github.com/trinhdaiphuc/etcdkeeper/pkg/middlewares"
)

func SetRoutes(e *echo.Echo) {
	// Configure middleware with the custom claims type
	config := middleware.JWTConfig{
		Claims:     &middlewares.JwtCustomClaims{},
		SigningKey: config.GetConfig().SecretKey,
		Skipper: func(c echo.Context) bool {
			if c.Path() == "/v2/separator" || c.Path() == "/v3/separator" ||
				c.Path() == "/v2/connect" || c.Path() == "/v3/connect" {
				return true
			}
			return false
		},
		ContextKey: middlewares.UserKey,
		ParseTokenFunc: func(auth string, c echo.Context) (interface{}, error) {
			token, err := jwt.ParseWithClaims(auth, &middlewares.JwtCustomClaims{}, func(tkn *jwt.Token) (interface{}, error) {
				return config.GetConfig().SecretKey, nil
			})

			if err != nil {
				return nil, err
			}
			if !token.Valid {
				return nil, errors.New("invalid token")
			}

			claims, ok := token.Claims.(*middlewares.JwtCustomClaims)
			if !(ok && token.Valid) {
				return nil, errors.New("invalid claims")
			}
			return claims.User, nil
		},
	}

	v2 := e.Group("/v2")
	v2.Use(middleware.JWTWithConfig(config))
	v2.GET("/separator", controllers.GetSeparator)
	v2.POST("/connect", controllers.ConnectV2)
	v2.GET("/get", controllers.GetV2)
	v2.PUT("/put", controllers.PutV2)
	v2.POST("/delete", controllers.DelV2)
	v2.GET("/getpath", controllers.GetPathV2)

	v3 := e.Group("/v3")
	v3.Use(middleware.JWTWithConfig(config))
	v3.GET("/separator", controllers.GetSeparator)
	v3.POST("/connect", controllers.ConnectV3)
	v3.GET("/get", controllers.GetV3)
	v3.PUT("/put", controllers.PutV3)
	v3.POST("/delete", controllers.DelV3)
	v3.GET("/getpath", controllers.GetPathV3)
}
