package main

import (
	"fmt"
	"log"
	"time"

	"go-sms/routes" // 导入routes包
	"go-sms/util"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"          // 添加prometheus包导入
	"github.com/prometheus/client_golang/prometheus/promhttp" // 添加promhttp包导入
	"github.com/spf13/viper"                                  // 引入viper库
)

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	},
	[]string{"method", "path", "status"},
)

func init() {

	// 初始化配置
    if err := util.InitConfig(); err != nil {
        log.Fatalf("Error initializing config: %s", err)
    }
	// 注册Prometheus指标
	prometheus.MustRegister(httpRequestsTotal)
}

func main() {
	// 设置 Gin 运行为 release 模式
	gin.SetMode(gin.ReleaseMode)
	// 创建一个不带默认中间件的Gin引擎
	r := gin.New()
	// 增加自定义 Logger 和 Recovery 中间件
	r.Use(customLogger(), gin.Recovery())
	// 注册路由
	routes.SetupRoutes(r)
	// 在本地的SERVER_PORT端口启动HTTP服务器
	port := viper.GetString("SERVER_PORT") // 从viper中读取配置
	log.Println("Server is running on port " + port)

	// 初始化定时任务
	if err := util.InitCron(); err != nil {
		log.Fatalf("Error initializing cron: %s", err)
	}


	// 暴露Prometheus指标
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.Run(":" + port)
}

// 自定义 Logger 中间件
func customLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()

		// 处理请求
		c.Next()

		// 结束时间
		duration := time.Since(start)

		// 记录请求指标
		httpRequestsTotal.With(prometheus.Labels{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"status": fmt.Sprintf("%d", c.Writer.Status()),
		}).Inc()

		// 跳过指标记录的s请求
		if c.Request.URL.Path == "/metrics" || c.Request.URL.Path == "/health" {
			return
		}

		// 日志格式
		log.Printf("[webhook] %s %s %d %s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration)

	}
}
