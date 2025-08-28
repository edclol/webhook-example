package util

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/kervinchang/dify-go"
	"github.com/spf13/viper"
)

// DifyFlowMZClient 封装dify-go客户端
var DifyFlowMZClient *dify.Client

// InitDifyFlowMZClient 初始化Dify客户端
func InitDifyFlowMZClient() error {
	config := dify.ClientConfig{
		BaseURL: viper.GetString("DIFY_API_BASE_URL"),
		APIKey:  viper.GetString("DIFY_WORKFLOW_API_KEY_MZ"),
	}

	client, err := dify.NewClient(config)
	if err != nil {
		log.Printf("初始化Dify客户端失败: %v\n", err)
		return err
	}

	DifyFlowMZClient = client
	log.Println("Dify客户端初始化成功")
	return nil
}

// RunWorkflowWithSDK_MZ 调用Dify工作流API处理指标数据
func RunWorkflowWithSDK_MZ(queryText string, indicators string) ([]Indicator, error) {
	// 验证客户端是否初始化
	if DifyFlowMZClient == nil {
		if err := InitDifyFlowMZClient(); err != nil {
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
		"indicators": indicators,
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
		response, err = DifyFlowMZClient.RunWorkflow(context.Background(), request)
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
	var result []Indicator

	// 尝试从响应中提取访视次数和孕期周数
	if response.Data.Outputs != nil {
		// 尝试获取指标结果
		if resultStr, ok := response.Data.Outputs["result"]; ok {
			// 先将interface{}类型断言为字符串，再转换为[]byte
			if str, ok := resultStr.(string); ok {
				if err := json.Unmarshal([]byte(str), &result); err != nil {
					log.Printf("解析result字符串失败: %v", err)
					return nil, &paramError{message: "解析result字符串失败"}
				}
			}
		}
	}

	return result, nil
}


