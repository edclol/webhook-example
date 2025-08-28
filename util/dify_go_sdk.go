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

// VisitResult 表示访视和孕期判断结果的结构体
type VisitResult struct {
	VisitNumber     int `json:"visit_number"`
	GestationalWeeks int `json:"gestational_weeks"`
}

// GetVisitAndGestation 判断访视次数和孕期周数
type paramError struct {
	message string
}

func (e *paramError) Error() string {
	return e.message
}

// RunWorkflowWithSDK 调用工作流进行访视和孕期判断（仅支持block模式）
func RunWorkflowWithSDK(queryText string) (*VisitResult, error) {
	// 验证客户端是否初始化
	if DifyClient == nil {
		if err := InitDifyClient(); err != nil {
			return nil, err
		}
	}

	// 验证查询参数
	if queryText == "" {
		log.Println("查询字符串不能为空")
		return nil, &paramError{message: "查询字符串不能为空"}
	}

	// 创建工作流请求参数
	inputs := map[string]interface{}{
		"query_text": queryText,
	}

	// 创建工作流请求
	request := dify.RunWorkflowRequest{
		Inputs: inputs,
		User:   viper.GetString("DIFY_API_USER"),
	}

	// 添加重试机制，最多重试n次，默认3次
	maxRetries := viper.GetInt("DIFY_API_MAX_RETRY")
	if maxRetries <= 0 {
		maxRetries = 3
	}

	var lastErr error
	var response *dify.CompletionResponse

	for i := 0; i < maxRetries; i++ {
		// 调用API
		var err error
		response, err = DifyClient.RunWorkflow(context.Background(), request)
		if err == nil {
			// 成功获取响应，退出重试
			lastErr = nil
			break
		}

		lastErr = err
		log.Printf("调用Dify工作流API失败 (尝试 %d/%d): %v", i+1, maxRetries, err)

		// 短暂延迟后重试
		if i < maxRetries-1 {
			time.Sleep(2 * time.Second)
		}
	}

	// 处理错误
	if lastErr != nil {
		return nil, lastErr
	}

	// 处理响应
	if response == nil {
		log.Println("未收到有效响应")
		return nil, &paramError{message: "未收到有效响应"}
	}

	// 检查响应状态
	if response.Data.Status != "succeeded" {
		log.Printf("工作流执行失败，状态: %s, 错误: %s", response.Data.Status, response.Data.Error)
		return nil, &paramError{message: "工作流执行失败"}
	}

	// 解析响应数据
	result := &VisitResult{}

	// 尝试从响应中提取访视次数和孕期周数
	if response.Data.Outputs != nil {
		// 尝试获取访视次数
		if visitNum, ok := response.Data.Outputs["visit_number"]; ok {
			// 处理不同类型的数值
			switch v := visitNum.(type) {
			case int:
				result.VisitNumber = v
			case float64:
				result.VisitNumber = int(v)
			case string:
				// 如果是字符串，尝试解析成整数
				// 这里可以根据实际情况添加字符串转整数的逻辑
			}
		}

		// 尝试获取孕期周数
		if gestWeeks, ok := response.Data.Outputs["gestational_weeks"]; ok {
			// 处理不同类型的数值
			switch v := gestWeeks.(type) {
			case int:
				result.GestationalWeeks = v
			case float64:
				result.GestationalWeeks = int(v)
			case string:
				// 如果是字符串，尝试解析成整数
				// 这里可以根据实际情况添加字符串转整数的逻辑
			}
		}
	}

	// 记录获取的结果
	log.Printf("访视和孕期判断结果: 访视次数=%d, 孕期周数=%d", result.VisitNumber, result.GestationalWeeks)

	return result, nil
}