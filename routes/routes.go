package routes

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine) {
	// 注册处理Webhook请求的路由，将"/webhook"路径的请求交给handleWebhook函数处理
	r.POST("/webhook", handleWebhook)
	r.GET("/", handleRoot)          // 修改: 调用handleRoot函数
	r.POST("/health", handleHealth) // 修改: 调用handleHealth函数
	r.POST("/test/webhook", handleTestWebhook)
	r.POST("/del/mysql", handleDelMysql)
	r.POST("/seatunnel/mysql/pg", handleSeatunnelMysqlPg)
	// 注册处理ProcessVisits和ProcessMZMain的API路由
	r.POST("/api/process/visits", handleProcessVisits)
	r.POST("/api/process/mz", handleProcessMZMain)
}
