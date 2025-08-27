package util

import (
		"github.com/robfig/cron/v3"                               // 引入cron库
)

func InitCron() error {
	// 启动定时任务
	c := cron.New()
	c.AddFunc("@every 1d", func() {
		DelMysql()
	})
	c.AddFunc("@every 2s", func() {
		DelHistory()
	})
	c.Start()
	return nil
}