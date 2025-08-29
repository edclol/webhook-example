package util

import (
	"context"
	"encoding/json"
	"log"
	"strings"
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

	// 解析传入的指标JSON字符串，以便后续比较
	var inputIndicators []Indicator
	if err := json.Unmarshal([]byte(indicators), &inputIndicators); err != nil {
		log.Printf("解析传入指标字符串失败: %v", err)
		// 虽然解析失败，但仍然尝试继续处理，因为这不是致命错误
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
				// 清理字符串，移除可能存在的额外引号或空白字符
				str = strings.TrimSpace(str)
				// 移除首尾可能存在的额外引号
				if len(str) > 1 && str[0] == '"' && str[len(str)-1] == '"' {
					str = str[1 : len(str)-1]
				}
				
				// log.Printf("尝试解析的result字符串: %s", str)
				if err := json.Unmarshal([]byte(str), &result); err != nil {
					log.Printf("解析result字符串失败: %v", err)
					return nil, &paramError{message: "解析result字符串失败"}
				}
			} else {
				log.Printf("result字段不是字符串类型，实际类型: %T, 值: %v", resultStr, resultStr)
			}
		}
	}

	// 比较结果指标和传入指标的code和name是否一一对应
	if len(inputIndicators) > 0 && len(result) > 0 {
		// 创建传入指标的映射，用于快速查找
		inputMap := make(map[string]string)
		for _, ind := range inputIndicators {
			inputMap[ind.Code] = ind.Name
		}

		// 检查结果指标是否与传入指标一一对应
		for _, res := range result {
			// 检查结果指标的code是否存在于传入指标中
			if inputName, exists := inputMap[res.Code]; !exists {
				log.Printf("结果指标中存在传入指标没有的code: %s", res.Code)
				return nil, &paramError{message: "结果指标与传入指标不匹配: 存在未知的code"}
			} else if res.Name != inputName {
				// 检查结果指标的name是否与传入指标一致
				log.Printf("结果指标与传入指标的name不匹配: code=%s, 传入name=%s, 结果name=%s", 
					res.Code, inputName, res.Name)
				return nil, &paramError{message: "结果指标与传入指标不匹配: name不一致"}
			}
		}

		// 检查传入指标是否都在结果中出现
		resultMap := make(map[string]bool)
		for _, res := range result {
			resultMap[res.Code] = true
		}

		for _, ind := range inputIndicators {
			if !resultMap[ind.Code] {
				log.Printf("传入指标中有结果指标没有的code: %s", ind.Code)
				return nil, &paramError{message: "结果指标与传入指标不匹配: 缺少某些code"}
			}
		}

		log.Printf("指标验证成功: 结果指标与传入指标一一对应，共%d个指标", len(result))
	}

	return result, nil
}


