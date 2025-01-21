package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// handleHealth 处理健康检查请求
func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}