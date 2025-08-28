
package util

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

func InitConfig() error {
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

	// 结构化数据库pg
	viper.Set("pg_struct.HOST", "192.168.23.24")
	viper.Set("pg_struct.PORT", 5432)
	viper.Set("pg_struct.USER", "postgres")
	viper.Set("pg_struct.PASSWORD", "Knt@123456")
	viper.Set("pg_struct.DATABASE", "data_structuration")

	//dify
	viper.Set("DIFY_API_KEY", "app-Cutxjfv8WztGuQqd3Z1gNUDO")
	viper.Set("DIFY_API_URL", "http://192.168.23.23:8080/v1/chat-messages")
	viper.Set("DIFY_API_USER", "user1")

	return nil
}