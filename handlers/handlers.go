package handlers

import (
	"devops-gotemplate/log"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
)

func HomeHandler(c *gin.Context)  {
	// log使用实例
	log.Error("error", errors.New("error"))
	log.Warn("warn", zap.Error(errors.New("warning")))
	log.Info("hello", zap.String("name", "world"))

	c.JSON(http.StatusOK, gin.H{"status": true, "msg": "Hello World"})
}


