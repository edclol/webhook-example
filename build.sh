#!/bin/bash
set -e
echo "Building webhook"

# 获取脚本所在目录并切换到该目录
SCRIPT_DIR="$(dirname "$0")"
cd "$SCRIPT_DIR" || {
    echo "Failed to change directory to $SCRIPT_DIR"
    exit 1
}

# 设置环境变量
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

build_webhook() {
    echo "Building webhook"
    go build -trimpath \
   	-o webhook
}
# 构建二进制文件
if ! build_webhook; then
    echo "Build failed!"
    exit 1
fi

# 输出构建成功信息
echo "Build succeeded!"
echo "停止远程服务器上的webhook服务"
ssh root@schedulemaster "systemctl stop webhook"
echo "上传新的webhook二进制文件到远程服务器"
scp webhook root@schedulemaster:/data/webhook/
echo "上传新的配置文件到远程服务器"
scp config.yaml root@schedulemaster:/data/webhook/
echo "上传新的服务文件到远程服务器"
scp webhook.service root@schedulemaster:/etc/systemd/system/
echo "更新文件权限，确保webhook服务可以被正确执行"
ssh root@schedulemaster "chmod -R 777 /data/webhook/ && chmod 777 /etc/systemd/system/webhook.service"
echo "重新加载系统服务配置，并重启webhook服务，然后检查其状态"
ssh root@schedulemaster "systemctl daemon-reload && systemctl restart webhook && systemctl status webhook"
# sudo systemctl enable webhook
# sudo systemctl daemon-reload
# sudo systemctl start webhook
# sudo systemctl status webhook

# curl -X POST -H "Content-Type: application/json" -d '{"receiver":"webhook-receiver","status":"firing","alerts":[{"status":"firing","labels":{"alertname":"Watchdog","environment":"schedulemaster","severity":"warning"},"annotations":{"description":"This is an alert meant to ensure that the entire alerting pipeline is functional.\nThis alert is always firing, therefore it should always be firing in Alertmanager\nand always fire against a receiver. There are integrations with various notification\nmechanisms that send a notification when this alert is not firing. For example the\n\"DeadMansSnitch\" integration in PagerDuty.","summary":"Ensure entire alerting pipeline is functional"},"startsAt":"2024-12-18T10:35:30.61Z","endsAt":"0001-01-01T00:00:00Z","generatorURL":"http://schedulemaster:9090/graph?g0.expr=vector%281%29\u0026g0.tab=1","fingerprint":"0c7b31e25484f81a"}],"groupLabels":{"alertname":"Watchdog"},"commonLabels":{"alertname":"Watchdog","environment":"schedulemaster","severity":"warning"},"commonAnnotations":{"description":"This is an alert meant to ensure that the entire alerting pipeline is functional.\nThis alert is always firing, therefore it should always be firing in Alertmanager\nand always fire against a receiver. There are integrations with various notification\nmechanisms that send a notification when this alert is not firing. For example the\n\"DeadMansSnitch\" integration in PagerDuty.","summary":"Ensure entire alerting pipeline is functional"},"externalURL":"http://172.16.97.110:9093","version":"4","groupKey":"{}:{alertname=\"Watchdog\"}","truncatedAlerts":0}' http://172.16.97.110:8080/webhook
# curl -X POST -H "Content-Type: application/json" -d '{"receiver":"webhook-receiver","status":"firing","alerts":[{"status":"firing","labels":{"alertname":"Watchdog","environment":"schedulemaster","severity":"warning"},"annotations":{"description":"This is an alert meant to ensure that the entire alerting pipeline is functional.\nThis alert is always firing, therefore it should always be firing in Alertmanager\nand always fire against a receiver. There are integrations with various notification\nmechanisms that send a notification when this alert is not firing. For example the\n\"DeadMansSnitch\" integration in PagerDuty.","summary":"Ensure entire alerting pipeline is functional"},"startsAt":"2024-12-18T10:35:30.61Z","endsAt":"0001-01-01T00:00:00Z","generatorURL":"http://schedulemaster:9090/graph?g0.expr=vector%281%29\u0026g0.tab=1","fingerprint":"0c7b31e25484f81a"}],"groupLabels":{"alertname":"Watchdog"},"commonLabels":{"alertname":"Watchdog","environment":"schedulemaster","severity":"warning"},"commonAnnotations":{"description":"This is an alert meant to ensure that the entire alerting pipeline is functional.\nThis alert is always firing, therefore it should always be firing in Alertmanager\nand always fire against a receiver. There are integrations with various notification\nmechanisms that send a notification when this alert is not firing. For example the\n\"DeadMansSnitch\" integration in PagerDuty.","summary":"Ensure entire alerting pipeline is functional"},"externalURL":"http://172.16.97.110:9093","version":"4","groupKey":"{}:{alertname=\"Watchdog\"}","truncatedAlerts":0}' http://172.16.97.110:8080/test/webhook
# curl -X POST -H "Content-Type: application/json" http://172.16.97.110:8080/health
# curl -X POST -H "Content-Type: application/json" http://172.16.97.110:8080/del/mysql
# curl -X GET  http://172.16.97.110:8080/
