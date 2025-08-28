package util

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/spf13/viper"
)

// DifyRequest 定义请求结构体
type DifyRequest struct {
	Inputs       map[string]interface{} `json:"inputs"`
	Query        string                 `json:"query"`
	ResponseMode string                 `json:"response_mode"`
	User         string                 `json:"user"`
}

// DifyResponse 定义响应结构体
type DifyResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CallDifyAPI 调用Dify API，使用blocking模式
func CallDifyAPI(apiKey, apiURL, query, user string) (*DifyResponse, error) {
	// 创建请求体，使用blocking响应模式
	requestBody := DifyRequest{
		Inputs:       map[string]interface{}{},
		Query:        query,
		ResponseMode: "blocking",
		User:         user,
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

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("发送请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		// 读取错误响应内容
		errorBody, _ := io.ReadAll(resp.Body)
		log.Printf("API返回非成功状态码: %d, 响应内容: %s", resp.StatusCode, string(errorBody))
		return nil, &apiError{statusCode: resp.StatusCode, message: string(errorBody)}
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应体失败: %v", err)
		return nil, err
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

	// 调用API
	response, err := CallDifyAPI(apiKey, apiURL, query, user)
	if err != nil {
		log.Printf("调用Dify API失败: %v", err)
		return "", err
	}

	// 处理响应
	if len(response.Choices) == 0 {
		log.Println("未收到有效响应")
		return "", &responseError{message: "未收到有效响应"}
	}

	content := response.Choices[0].Message.Content
	if content == "" {
		log.Println("响应内容为空")
		return "", &responseError{message: "响应内容为空"}
	}

	return content, nil
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
    