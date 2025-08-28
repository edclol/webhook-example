package util

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

// DifyRequest 定义请求结构体
type DifyRequest struct {
	Inputs         map[string]interface{} `json:"inputs"`
	Query          string                 `json:"query"`
	ResponseMode   string                 `json:"response_mode"`
	ConversationID string                 `json:"conversation_id"`
	User           string                 `json:"user"`
	Files          []File                 `json:"files,omitempty"`
}

// File 定义文件结构体
type File struct {
	Type           string `json:"type"`
	TransferMethod string `json:"transfer_method"`
	URL            string `json:"url,omitempty"`
}

// DifyResponse 定义响应结构体
type DifyResponse struct {
	Event          string                 `json:"event"`
	TaskID         string                 `json:"task_id"`
	ID             string                 `json:"id"`
	MessageID      string                 `json:"message_id"`
	ConversationID string                 `json:"conversation_id"`
	Mode           string                 `json:"mode"`
	Answer         string                 `json:"answer"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      int64                  `json:"created_at"`
}

// CallDifyAPI 调用Dify API，使用blocking模式
func CallDifyAPI(apiKey, apiURL, query, user string, conversationID string, files []File) (*DifyResponse, error) {
	// 创建请求体，使用blocking响应模式
	requestBody := DifyRequest{
		Inputs:         map[string]interface{}{},
		Query:          query,
		ResponseMode:   "blocking",
		ConversationID: conversationID,
		User:           user,
		Files:          files,
	}

	// 序列化请求体为JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("序列化请求体失败: %v", err)
		return nil, err
	}

	// 创建HTTP请求
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("创建请求失败: %v", err)
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求，设置超时
	timeoutSeconds := viper.GetInt("DIFY_API_TIMEOUT")
	// 如果配置中没有设置，默认30秒
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("发送请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体一次
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应体失败: %v", err)
		// 打印响应
		log.Printf("响应内容: %s", string(body))
		return nil, err
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("API返回非成功状态码: %d, 响应内容: %s", resp.StatusCode, string(body))
		return nil, &apiError{statusCode: resp.StatusCode, message: string(body)}
	}

	// 解析响应JSON
	var difyResp DifyResponse
	if err := json.Unmarshal(body, &difyResp); err != nil {
		log.Printf("解析响应失败: %v, 响应内容: %s", err, string(body))
		return nil, err
	}

	return &difyResp, nil
}

// GetVisitStage 获取访视阶段
// 参数: query - 要发送的查询字符串
// 返回: 响应内容字符串和可能的错误
func GetVisitStage(query string) (string, error) {
	// 从配置文件获取参数
	apiKey := viper.GetString("DIFY_API_KEY")
	apiURL := viper.GetString("DIFY_API_URL")
	user := viper.GetString("DIFY_API_USER")

	// 验证配置参数
	if apiKey == "" || apiURL == "" {
		log.Println("API密钥或API地址未配置")
		return "", &configError{message: "API密钥或API地址未配置"}
	}

	// 验证查询参数
	if query == "" {
		log.Println("查询字符串不能为空")
		return "", &paramError{message: "查询字符串不能为空"}
	}

	// 添加重试机制，最多重试n次，默认3次
	maxRetries := viper.GetInt("DIFY_API_MAX_RETRY")
	if maxRetries <= 0 {
		maxRetries = 3
	}
	var lastErr error
	var response *DifyResponse

	for i := 0; i < maxRetries; i++ {
		// 调用API
		var err error
		response, err = CallDifyAPI(apiKey, apiURL, query, user, "", nil)
		if err == nil {
			// 成功获取响应，退出重试
			lastErr = nil
			break
		}

		lastErr = err
		log.Printf("调用Dify API失败 (尝试 %d/%d): %v", i+1, maxRetries, err)

		// 如果是配置错误或参数错误，无需重试
		if _, ok := err.(*configError); ok {
			break
		}
		if _, ok := err.(*paramError); ok {
			break
		}

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
	if response == nil {
		log.Println("API响应为空")
		return "", &responseError{message: "API响应为空"}
	}

	// 检查是否收到有效响应
	if response.Event != "message" || response.Answer == "" {
		log.Println("未收到有效响应")
		// 提供一个默认值作为备用
		return "未知", nil
	}

	// 移除可能包含的思考过程标记
	answer := response.Answer
	if len(answer) > 8 && answer[:4] == "</think>" && answer[len(answer)-4:] == "</think>" {
		answer = answer[4:len(answer)-4]
	}

	// 提取响应中的数字
	var result string
	for _, char := range answer {
		if char >= '0' && char <= '9' {
			result += string(char)
		}
	}

	// 如果没有找到数字，返回默认值
	if result == "" {
		return "未知", nil
	}

	return result, nil
}

// 自定义错误类型
type apiError struct {
	statusCode int
	message    string
}

func (e *apiError) Error() string {
	return e.message
}

type configError struct {
	message string
}

func (e *configError) Error() string {
	return e.message
}

type paramError struct {
	message string
}

func (e *paramError) Error() string {
	return e.message
}

type responseError struct {
	message string
}

func (e *responseError) Error() string {
	return e.message
}
