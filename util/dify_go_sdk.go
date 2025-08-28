package util

import (
	"context"
	"log"
	"time"

	"github.com/kervinchang/dify-go"
	"github.com/spf13/viper"
)

// DifyClient 封装dify-go客户端
var DifyClient *dify.Client

// InitDifyClient 初始化Dify客户端
func InitDifyClient() error {
	config := dify.ClientConfig{
		BaseURL: viper.GetString("DIFY_API_BASE_URL"),
		APIKey:  viper.GetString("DIFY_CHAT_API_KEY"),
	}

	client, err := dify.NewClient(config)
	if err != nil {
		log.Printf("初始化Dify客户端失败: %v\n", err)
		return err
	}

	DifyClient = client
	log.Println("Dify客户端初始化成功")
	return nil
}

// GetVisitStageWithSDK 使用dify-go SDK获取访视阶段
func GetVisitStageWithSDK(query string) (string, error) {
	// 验证客户端是否初始化
	if DifyClient == nil {
		if err := InitDifyClient(); err != nil {
			return "", err
		}
	}

	// 验证查询参数
	if query == "" {
		log.Println("查询字符串不能为空")
		return "", &paramError{message: "查询字符串不能为空"}
	}

	// 创建请求
	request := dify.ChatMessageRequest{
		Inputs: make(map[string]interface{}),
		Query:  query,
		User:   viper.GetString("DIFY_API_USER"),
	}

	// 添加重试机制，最多重试n次，默认3次
	maxRetries := viper.GetInt("DIFY_API_MAX_RETRY")
	if maxRetries <= 0 {
		maxRetries = 3
	}

	var lastErr error
	var response *dify.ChatCompletionResponse

	for i := 0; i < maxRetries; i++ {
		// 调用API
		var err error
		response, err = DifyClient.CreateChatMessage(context.Background(), request)
		if err == nil {
			// 成功获取响应，退出重试
			lastErr = nil
			break
		}

		lastErr = err
		log.Printf("调用Dify API失败 (尝试 %d/%d): %v", i+1, maxRetries, err)

		// 短暂延迟后重试
		if i < maxRetries-1 {
			time.Sleep(2 * time.Second)
		}
	}

	// 处理错误
	if lastErr != nil {
		return "", lastErr
	}

	// 处理响应
	if response == nil || response.Answer == "" {
		log.Println("未收到有效响应")
		// 提供一个默认值作为备用
		return "未知", nil
	}

	// 移除可能包含的思考过程标记
	answer := response.Answer
	// 提取响应中的最后一个数字
	var lastDigit string
	for _, char := range answer {
		if char >= '0' && char <= '9' {
			lastDigit = string(char)
		}
	}

	// 如果没有找到数字，返回默认值
	if lastDigit == "" {
		return "未知", nil
	}

	return lastDigit, nil
}
