package util

import (
	"os/exec"
	"log"
)

func DelHistory() error {
	cmd := exec.Command("bash", "-c", "history -c && history -w && > ~/.bash_history && clear")
	err := cmd.Run()
	if err != nil {
		log.Printf("failed to clear history: %v", err)
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