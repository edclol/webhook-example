package util

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

// CallSmsPlatform结构体用于封装短信平台相关信息及操作
type CallSmsPlatform struct {
	URL        string
	SOAPAction string
}

// SendSms方法用于向指定手机号发送短信内容
func (c *CallSmsPlatform) SendSms(phone, content string) bool {
	if !viper.GetBool("SEND_REAL_SMS") {
		// 如果配置文件中SEND_REAL_SMS为false，则不发送真实短信
		log.Println("SMS sending disabled in configuration file.")
		log.Println("SMS to:", phone)
		log.Println("SMS content:", content)
		return true
	}
	// 设置请求头
	headers := map[string]string{
		"SOAPAction":   c.SOAPAction,
		"Content-Type": "text/xml;charset=UTF-8",
	}

	// 构造SOAP请求体
	soapRequest := `<?xml version="1.0" encoding="utf-8"?>
    <soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:dhcc="http://www.dhcc.com.cn">
       <soapenv:Header/>
       <soapenv:Body>
          <dhcc:HIPMessageServer>
             <dhcc:input1>MES0395</dhcc:input1>
             <dhcc:input2><![CDATA[<Request><TelNum>` + phone + `</TelNum><MesgInfo>` + content + `</MesgInfo></Request>]]></dhcc:input2>
          </dhcc:HIPMessageServer>
       </soapenv:Body>
    </soapenv:Envelope>`

	// 创建一个HTTP客户端
	client := &http.Client{}

	log.Println("SOAPAction: ", c.SOAPAction)
	log.Println("URL: ", c.URL)

	log.Println("Sending SMS to", phone)
	log.Println("Content:", content)
	log.Println("SOAP Request:", soapRequest)

	// 创建HTTP POST请求
	req, err := http.NewRequest("POST", c.URL, strings.NewReader(soapRequest))
	if err != nil {
		return false
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 读取响应体
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	// 根据状态码判断请求是否成功
	if resp.StatusCode == 200 {
		return true
	}
	return false
}
