package routes

import (
	"net/http"

	"encoding/json"
	"fmt"
	"go-sms/util"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/spf13/viper"
)

// handleTest 处理测试请求
func handleTestWebhook(c *gin.Context) {
	handleWebhook2(c)

}

func handleWebhook2(c *gin.Context) {
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "Only POST requests are allowed",
		})
		return
	}

	// 获取请求体中的数据
	body, err := c.GetRawData()
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Request body is empty or invalid",
		})
		return
	}

	// 验证请求来源（示例）
	if !validateRequestSource(c) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid request source",
		})
		return
	}

	// 处理幂等性（示例）
	if !checkIdempotency(c) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Duplicate request detected",
		})
		return
	}

	// 在这里可以对读取到的数据进行处理，比如解析JSON格式数据等，目前只是简单打印
	log.Println(string(body))

	smsPlatform := util.CallSmsPlatform{
		URL:        viper.GetString("SMS_PLATFORM_URL"), // 直接从viper中读取配置
		SOAPAction: viper.GetString("SOAP_ACTION"),      // 直接从viper中读取配置
	}

	// 解析JSON数据到结构体实例
	var alert util.ProAlert
	err = json.Unmarshal(body, &alert)
	if err != nil {
		fmt.Println("解析JSON出错:", err)
		return
	}
	log.Println(alert.CommonAnnotations["summary"])

	// 获取配置文件中的多个手机号
	phoneNumbers := viper.GetStringSlice("PHONE_NUMBERS") // 直接从viper中读取配置

	// 获取等待间隔配置项
	smsSendInterval := viper.GetInt("SMS_SEND_INTERVAL") // 从viper中读取配置

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in goroutine: %v", r)
			}
		}()

		// 异步发送短信给每个手机号
		for _, phoneNumber := range phoneNumbers {
			if err := sendSms2(smsPlatform, phoneNumber, alert.CommonAnnotations["summary"]); err != nil {
				log.Printf("Failed to send SMS to %s: %v", phoneNumber, err)
			} else {
				log.Printf("SMS sent successfully to %s", phoneNumber)
			}
			// 增加等待间隔
			time.Sleep(time.Duration(smsSendInterval) * time.Second)
		}
	}()

	// 返回一个JSON格式的响应，表示成功接收
	c.JSON(http.StatusOK, gin.H{
		"message": "Webhook2 received successfully",
	})
}

func sendSms2(platform util.CallSmsPlatform, phoneNumber, message string) error {
	if true {
		log.Println("发送短信")
		log.Println("手机号:", phoneNumber)
		log.Println("消息:", message)
		return nil
	}
	return fmt.Errorf("failed to send SMS")
}
