package util

import (
	"context"
	"log"
	"time"

	"github.com/kervinchang/dify-go"
	"github.com/spf13/viper"
	"encoding/json"
)

// DifyFlowClient 封装dify-go客户端
var DifyFlowClient *dify.Client

// InitDifyFlowClient 初始化Dify客户端
func InitDifyFlowClient() error {
	config := dify.ClientConfig{
		BaseURL: viper.GetString("DIFY_API_BASE_URL"),
		APIKey:  viper.GetString("DIFY_WORKFLOW_API_KEY"),
	}

	client, err := dify.NewClient(config)
	if err != nil {
		log.Printf("初始化Dify客户端失败: %v\n", err)
		return err
	}

	DifyFlowClient = client
	log.Println("Dify客户端初始化成功")
	return nil
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
	if DifyFlowClient == nil {
		if err := InitDifyFlowClient(); err != nil {
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
		response, err = DifyFlowClient.RunWorkflow(context.Background(), request)
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
		if resultStr, ok := response.Data.Outputs["result"]; ok {
			// 先将interface{}类型断言为字符串，再转换为[]byte
			if str, ok := resultStr.(string); ok {
				if err := json.Unmarshal([]byte(str), result); err != nil {
					log.Printf("解析result字符串失败: %v", err)
					return nil, &paramError{message: "解析result字符串失败"}
				}
			}
			
		}
	}
	// 记录获取的结果
	log.Printf("访视和孕期判断结果: 访视次数=%d, 孕期周数=%d", result.VisitNumber, result.GestationalWeeks)

	return result, nil
}