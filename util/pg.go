package util

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"

	"github.com/spf13/viper"
	_ "github.com/lib/pq"
)

// PatientVisit 患者访视信息结构体
type PatientVisit struct {
	ID         int    `json:"id"`
	VisitNotes string `json:"visit_notes"`
	VisitStage string `json:"visit_stage"`
}

// ProcessVisits 处理访视记录的主函数
func ProcessVisits() error {
	// 从配置加载数据库连接信息
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s",
		viper.GetString("pg_struct.HOST"),
		viper.GetString("pg_struct.PORT"),
		viper.GetString("pg_struct.USER"),
		viper.GetString("pg_struct.PASSWORD"),
		viper.GetString("pg_struct.DATABASE"),
	)

	// 建立数据库连接（用于查询）
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("无法打开数据库连接: %v", err)
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("数据库连接失败: %v", err)
		return err
	}
	log.Println("数据库连接成功")

	// 查询需要处理的记录
	rows, err := db.Query(`
		SELECT id, visit_notes, visit_stage 
		FROM patient_visits 
		WHERE visit_stage IS NULL OR visit_stage = ''
		LIMIT 100;`)
	if err != nil {
		log.Printf("查询失败: %v", err)
		return err
	}
	defer rows.Close()

	var visits []PatientVisit
	for rows.Next() {
		var visit PatientVisit
		if err := rows.Scan(&visit.ID, &visit.VisitNotes, &visit.VisitStage); err != nil {
			log.Printf("扫描记录失败: %v", err)
			return err
		}
		visits = append(visits, visit)
	}

	if err = rows.Err(); err != nil {
		log.Printf("行迭代错误: %v", err)
		return err
	}

	log.Printf("查询到 %d 条需要处理的记录", len(visits))
	if len(visits) == 0 {
		return nil
	}

	// 处理每条记录 - 多线程版本
	// 创建上下文，可用于取消操作
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 配置工作池参数
	workerCount := 5 // 工作线程数量，可根据实际情况调整
	visitChan := make(chan PatientVisit, len(visits)) // 任务通道
	var wg sync.WaitGroup // 等待组，用于同步所有工作线程

	// 启动工作线程池
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			// 为每个工作线程创建独立的数据库连接
			workerDB, err := sql.Open("postgres", connStr)
			if err != nil {
				log.Printf("工作线程 %d 无法创建数据库连接: %v", workerID, err)
				return
			}
			defer workerDB.Close()
			
			if err := workerDB.Ping(); err != nil {
				log.Printf("工作线程 %d 数据库连接失败: %v", workerID, err)
				return
			}
			
			// 处理从通道接收的任务
			for {
				select {
				case <-ctx.Done():
					log.Printf("工作线程 %d 收到停止信号", workerID)
					return
				case visit, ok := <-visitChan:
					if !ok {
						// 通道已关闭，无更多任务
						log.Printf("工作线程 %d 完成所有任务", workerID)
						return
					}
					
					log.Printf("工作线程 %d 处理记录 ID=%d", workerID, visit.ID)
					
					// 调用Dify API
					stage, err := GetVisitStage(fmt.Sprintf("分析访视记录确定阶段: %s", visit.VisitNotes))
					if err != nil {
						log.Printf("工作线程 %d 获取阶段失败 (ID=%d): %v", workerID, visit.ID, err)
						continue
					}
					
					// 更新数据库
					result, err := workerDB.ExecContext(ctx, `
						UPDATE patient_visits 
						SET visit_stage = $1, updated_at = NOW()
						WHERE id = $2;`, stage, visit.ID)
					
					if err != nil {
						log.Printf("工作线程 %d 更新失败 (ID=%d): %v", workerID, visit.ID, err)
						continue
					}
					
					if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
						log.Printf("工作线程 %d 成功更新记录 ID=%d", workerID, visit.ID)
					}
				}
			}
		}(i)
	}

	// 将所有记录发送到任务通道
	for _, visit := range visits {
		visitChan <- visit
	}
	close(visitChan) // 关闭通道，表示没有更多任务

	// 等待所有工作线程完成
	wg.Wait()

	log.Println("所有记录处理完毕")
	return nil
}
