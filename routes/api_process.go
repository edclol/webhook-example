package routes

import (
	"net/http"
	"go-sms/util"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// handleProcessVisits 处理触发ProcessVisits的请求
func handleProcessVisits(c *gin.Context) {
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "Only POST requests are allowed",
		})
		return
	}

	// 异步处理请求
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in ProcessVisits: %v", r)
			}
		}()

		startTime := time.Now()
		log.Println("开始处理ProcessVisits...")
		
		if err := util.ProcessVisits(); err != nil {
			log.Printf("ProcessVisits执行出错: %v", err)
		} else {
			duration := time.Since(startTime)
			log.Printf("ProcessVisits执行完成，耗时: %v", duration)
		}
	}()

	// 返回一个JSON格式的响应，表示已接收请求
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "ProcessVisits请求已接收，正在处理中",
	})
}

// handleProcessMZMain 处理触发ProcessMZMain的请求
func handleProcessMZMain(c *gin.Context) {
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "Only POST requests are allowed",
		})
		return
	}

	// 异步处理请求
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in ProcessMZMain: %v", r)
			}
		}()

		startTime := time.Now()
		log.Println("开始处理ProcessMZMain...")
		
		if err := util.ProcessMZMain(); err != nil {
			log.Printf("ProcessMZMain执行出错: %v", err)
		} else {
			duration := time.Since(startTime)
			log.Printf("ProcessMZMain执行完成，耗时: %v", duration)
		}
	}()

	// 返回一个JSON格式的响应，表示已接收请求
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "ProcessMZMain请求已接收，正在处理中",
	})
}