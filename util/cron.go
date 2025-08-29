package util

import (
	"github.com/robfig/cron/v3"
	"log"
	"time"
)


func InitCron() error {
	// 启动定时任务
	c := cron.New()
	
	// 每天清理MySQL数据
	c.AddFunc("@every 1d", func() {
		log.Println("开始执行DelMysql任务")
		startTime := time.Now()
		
		if err := DelMysql(); err != nil {
			log.Printf("DelMysql任务执行失败: %v", err)
		} else {
			duration := time.Since(startTime)
			log.Printf("DelMysql任务执行完成，耗时: %v", duration)
		}
	})
	
	// 每5分钟清理历史数据（原每2秒频率过高）
	c.AddFunc("@every 5m", func() {
		log.Println("开始执行DelHistory任务")
		startTime := time.Now()
		
		if err := DelHistory(); err != nil {
			log.Printf("DelHistory任务执行失败: %v", err)
		} else {
			duration := time.Since(startTime)
			log.Printf("DelHistory任务执行完成，耗时: %v", duration)
		}
	})
	
	// 每1小时处理患者访视记录
	c.AddFunc("@every 1h", func() {
		log.Println("开始执行ProcessVisits任务")
		startTime := time.Now()
		
		if err := ProcessVisits(); err != nil {
			log.Printf("ProcessVisits任务执行失败: %v", err)
		} else {
			duration := time.Since(startTime)
			log.Printf("ProcessVisits任务执行完成，耗时: %v", duration)
		}
	})
	
	// 每1小时校验患者数据（独立定时任务）
	c.AddFunc("@every 1h", func() {
		log.Println("开始执行ValidatePatientData任务")
		startTime := time.Now()
		
		if err := ValidatePatientData(); err != nil {
			log.Printf("ValidatePatientData任务执行失败: %v", err)
		} else {
			duration := time.Since(startTime)
			log.Printf("ValidatePatientData任务执行完成，耗时: %v", duration)
		}
	})
	
	c.Start()
	return nil
}