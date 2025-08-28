package util

import (
	"os/exec"
	"log"
)

func DelHistory() error {
	// 使用更简单的命令清除历史记录，并捕获输出以便更好地调试
	cmd := exec.Command("bash", "-c", "cat /dev/null > ~/.bash_history 2>/dev/null || true")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("failed to clear history: %v, output: %s", err, string(out))
	}
	return nil
}

func LsTest() error {
	cmd := exec.Command("ls", "-l", "/var/log/")
	out, err := cmd.CombinedOutput()
	if err != nil {
        log.Printf("combined out:\n%s\n", string(out))
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	log.Printf("combined out:\n%s\n", string(out))
	return nil
}
	
// 获取本机ip的函数
func GetLocalIP() (string, error) {
	cmd := exec.Command("bash", "-c", "hostname -I | awk '{print $1}'")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("failed to get local IP: %v", err)
		return "", err
	}
	log.Printf("local IP: %s", string(out))
	return string(out), nil
}
