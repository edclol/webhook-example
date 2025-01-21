package routes

import (
	"go-sms/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleRoot 处理根路径的请求
func handleDelMysql(c *gin.Context) {

	util.DelMysql()
	// 在这里可以对读取到的数据进行处理，比如解析JSON格式数据等，目前只是简单打印
	c.JSON(http.StatusOK, gin.H{
		"message": "DelMysql",
	})
}
