package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleRoot 处理根路径的请求
func handleRoot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello, World!",
	})
}
