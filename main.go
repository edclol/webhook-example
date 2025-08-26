package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"go-sms/routes" // 导入routes包
	"go-sms/util"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"          // 添加prometheus包导入
	"github.com/prometheus/client_golang/prometheus/promhttp" // 添加promhttp包导入
	"github.com/robfig/cron/v3"                               // 引入cron库
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
	// 初始化viper
	viper.SetConfigName("config") // 配置文件名
	viper.SetConfigType("yaml")   // 配置文件类型
	// 获取当前工作目录并设置为配置文件路径
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting working directory, %s", err)
	}
	viper.AddConfigPath(wd)

	log.Println("Working directory:", wd)
	// log.Println("Config file path:", wd+"/config.yaml")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	log.Println("Config file loaded successfully")

	// 强制覆盖配置
	viper.Set("SERVER_PORT", "8080")
	viper.Set("SMS_PLATFORM_URL", "http://172.16.99.6/csp/hsb/DHC.Published.PUB0010.BS.PUB0010.cls")
	viper.Set("SOAP_ACTION", "http://www.dhcc.com.cn/DHC.Published.PUB0010.BS.PUB0010.HIPMessageServer")
	viper.Set("PHONE_NUMBERS", []string{"18061651276", "13951073551", "18061796602"})
	viper.Set("SMS_SEND_INTERVAL", "5")
	viper.Set("SEND_REAL_SMS", true) // 设置为 false 以避免发送真实短信
	viper.Set("SEND_REAL_SMS_END_DAY", "2025-11-31") // 设置真实短信发送结束时间
	// DB 配置
	viper.Set("DB.HOST", "172.16.97.109")
	viper.Set("DB.USER", "dolphin")
	viper.Set("DB.PASSWORD", "dolphin")
	viper.Set("DB.DATABASE", "dolphinscheduler2")

	// mysql_src 配置
	viper.Set("mysql_src.HOST", "192.168.23.18")
	viper.Set("mysql_src.PORT", 3306)
	viper.Set("mysql_src.USER", "root")
	viper.Set("mysql_src.PASSWORD", "knt123KNT123")
	viper.Set("mysql_src.DATABASE", "ds320")

	// postgres_tgt 配置
	viper.Set("postgres_tgt.HOST", "192.168.23.24")
	viper.Set("postgres_tgt.PORT", 5432)
	viper.Set("postgres_tgt.USER", "postgres")
	viper.Set("postgres_tgt.PASSWORD", "Knt@123456")
	viper.Set("postgres_tgt.DATABASE", "ds320")

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

	// 启动定时任务
	c := cron.New()
	c.AddFunc("@every 1d", func() {
		log.Println("定时任务执行开始...")
		util.DelMysql()
		log.Println("定时任务执行结束...")
	})
	c.Start()

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
