package middlewares

import (
	"devops-gotemplate/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"time"
)

// 实现跨域处理
func Cors() gin.HandlerFunc {
	return func(context *gin.Context) {
		method := context.Request.Method
		if method == "OPTIONS" {
			context.AbortWithStatus(http.StatusNoContent)
		}

		origin := context.Request.Header.Get("Origin")
		if origin != "" {
			context.Header("Access-Control-Allow-Origin", origin)
			context.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			context.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			context.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			context.Header("Access-Control-Allow-Credentials", "false")
			context.Set("content-type", "application/json")
		}
		context.Next()
	}
}

// 日志中间件，使用zap日志
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		if raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		cost := time.Since(start)
		log.Info(path,
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			zap.Duration("cost", cost),
		)
	}

}

// Auth中间件，用于接口登陆认证
func Authenticate() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Next()

	}

}

// Auth中间件，用于接口的权限认证
func Authorize() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Next()

	}

}

