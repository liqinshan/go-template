package middlewares

import (
	"devops-gotemplate/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

// Cors 实现跨域处理
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}

		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "false")
			c.Set("content-type", "application/json")
		}
		c.Next()
	}
}

// Logger 日志中间件，使用zap日志
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

// Authenticate 登陆认证中间件，用于接口登陆认证
func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

	}

}

// Authorize 权限认证中间件，用于接口的权限认证
func Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

	}

}

// ParameterConvert 参数预处理中间件，用于参数前后空格的清除
func ParameterConvert() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		contentType := c.Request.Header.Get("Content-Type")
		contentLength := c.Request.Header.Get("Content-Length")

		var err error
		if method == "POST" || method == "PUT" || method == "DELETE" {
			if contentType == "application/json" && len(contentLength) > 1 {
				err = handlePostJson(c)
			} else if contentType == "application/x-www-form-urlencoded" {
				err = handlePostForm(c)
			} else if strings.Contains(contentType, "multipart/form-data") {
				err = handlePostMultiForm(c)
			}
		}

		if method == "GET" {
			err = handleRequestGet(c)
		}

		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
		}

		c.Next()
	}
}


func handleRequestGet(c *gin.Context) error {
	params := c.Request.URL.Query()
	for key, values := range params {
		for _, item := range values {
			v := strings.TrimSpace(item)
			params.Set(key, v)
		}
	}
	c.Request.URL.RawQuery = params.Encode()
	return nil
}

func handlePostJson(c *gin.Context) error {
	return nil

}

func handlePostForm(c *gin.Context) error {
	return nil

}

func handlePostMultiForm(c *gin.Context) error {
	return nil

}
