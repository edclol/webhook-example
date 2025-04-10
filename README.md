# Webhook

## 概述

Webhook 服务接收来自 Prometheus 的告警，并发送短信通知。

## 功能

- 接收来自 Prometheus 的告警。
- 根据接收到的告警发送短信通知。

## 前提条件

- Go 1.23.4 或更高版本
- SMS 网关 API 凭证（例如，Twilio, Nexmo）

## 安装

1. 克隆仓库：
   ```bash
   git clone https://github.com/your-repo/webhook-example.git
   cd webhook-example
   ```
2. 安装依赖：

   ```bash
   go mod download
   ```

3. 配置服务：

   基于.env.example 创建一个.env 文件，并填写您的 SMS 网关 API 凭证和其他必要配置。

4. 运行服务：

   ```bash
   go run main.go
   ```

## 配置

SMS_API_KEY: 您的 SMS 网关 API 密钥。
SMS_API_SECRET: 您的 SMS 网关 API 密钥。
SMS_FROM_NUMBER: 发送短信的电话号码。
PROMETHEUS_WEBHOOK_URL: Prometheus 发送告警的 URL。

## 使用

确保您的 Prometheus 服务器配置为将告警发送到 webhook 服务的 URL。
服务将自动处理传入的告警并发送短信通知。

### 故障排除

- 问题: 没有收到短信通知。
  解决方法: 检查日志中的任何错误，并确保您的 SMS 网关 API 凭证正确。
- 问题: 服务启动时崩溃。
  解决方法: 验证所有依赖项是否已安装，并且配置文件是否正确设置。

### 贡献

欢迎贡献！请打开一个 issue 或提交一个 pull request。

### 许可证

本项目采用 MIT 许可证。
