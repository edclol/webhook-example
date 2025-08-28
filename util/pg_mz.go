package util

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"strings"

	"github.com/spf13/viper"
	_ "github.com/lib/pq"
)

// PatientVisitMZ 患者访视信息结构体
type PatientVisitMZ struct {
	EncounterId string `json:"encounter_id"`
	PersonId    string `json:"person_id"`
	PatientId   string `json:"patient_id"`
	Content     string `json:"content"`
}

// Indicator 指标信息结构体
type Indicator struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Value        string `json:"value"`
	ValueExplain string `json:"value_explain"`
}

// ProcessMZMain 批量处理不同deleted_flag的记录
func ProcessMZMain() error {
	for i := 1; i <= 6; i++ {
		if err := ProcessMZ(i); err != nil {
			log.Printf("处理访视记录失败(deleted_flag=%d): %v", i, err)
			return err
		}
	}
	return nil
}

// ProcessMZ 处理访视记录的主函数
func ProcessMZ(deletedFlag int) error {
	// 从配置加载数据库连接信息
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
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
		SELECT encounter_id, person_id, patient_id, CONCAT_WS(
		'; ',
		'患者姓名: ' || patient_name,
		'性别: ' || gender_name,
		'年龄: ' || age,
		'就诊科室: ' || department_name,
		'文档名称: ' || document_name,
		'文档内容: ' || COALESCE(documrnt_content_txt, '无文本内容'),
		'诊断: ' || COALESCE(diag_name, '无诊断信息')
	) AS "content" 
		FROM public.dc_mr_document_index_outpat 
		WHERE deleted_flag = $1;`, deletedFlag)
	if err != nil {
		log.Printf("查询失败: %v", err)
		return err
	}
	defer rows.Close()

	var visits []PatientVisitMZ
	for rows.Next() {
		var visit PatientVisitMZ
		if err := rows.Scan(&visit.EncounterId, &visit.PersonId, &visit.PatientId, &visit.Content); err != nil {
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
	visitChan := make(chan PatientVisitMZ, len(visits)) // 任务通道
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

					log.Printf("工作线程 %d 处理记录 encounter_id=%s", workerID, visit.EncounterId)

					// 查询数据库获取指标数据
					indsRows, err := workerDB.QueryContext(ctx, `
						SELECT id as code, "name", null as value, aliass as value_explain 
						FROM public.t_model_view 
						WHERE fsjz = $1;`, deletedFlag)
					if err != nil {
						log.Printf("工作线程 %d 查询指标失败 (encounter_id=%s): %v", workerID, visit.EncounterId, err)
						continue
					}
					defer indsRows.Close()

					// 遍历结果集，将结果封装成列表
					var indicators []Indicator
					for indsRows.Next() {
						var code, name, value, valueExplain string
						if err := indsRows.Scan(&code, &name, &value, &valueExplain); err != nil {
							log.Printf("工作线程 %d 扫描指标结果失败: %v", workerID, err)
							continue
						}

						// 打印指标结果
						log.Printf("工作线程 %d 指标: code=%s, name=%s, value=%s, valueExplain=%s", 
							workerID, code, name, value, valueExplain)

						indicators = append(indicators, Indicator{
							Code:         code,
							Name:         name,
							Value:        value,
							ValueExplain: valueExplain,
						})
					}

					// 检查指标查询是否有错误
					if err = indsRows.Err(); err != nil {
						log.Printf("工作线程 %d 指标行迭代错误: %v", workerID, err)
						continue
					}

					// 按50个一组处理指标
					for i := 0; i < len(indicators); i += 50 {
						end := i + 50
						if end > len(indicators) {
							end = len(indicators)
						}

						// 将指标数组转换为JSON字符串
						indicatorJSON, err := json.Marshal(indicators[i:end])
						if err != nil {
							log.Printf("工作线程 %d 指标JSON序列化失败: %v", workerID, err)
							continue
						}

						// 调用Dify API
						resultData, err := RunWorkflowWithSDK_MZ(visit.Content, string(indicatorJSON))
						if err != nil {
							log.Printf("工作线程 %d 调用Dify API失败 (encounter_id=%s): %v", workerID, visit.EncounterId, err)
							continue
						}

						// 处理返回结果
						processIndicatorResults(ctx, workerDB, workerID, visit, resultData)
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

// processIndicatorResults 处理指标结果，执行数据库删除和插入操作
func processIndicatorResults(ctx context.Context, db *sql.DB, workerID int, visit PatientVisitMZ, indicators []Indicator) {
	for _, item := range indicators {
		item.Value = strings.TrimSpace(item.Value)
		if item.Value == "" {
			continue
		}

		// 删除已存在的数据
		_, err := db.ExecContext(ctx, `
			DELETE FROM public.t_patient_data 
			WHERE p_id = $1 AND v_id = $2 AND field_id = $3;`, 
			visit.PersonId, visit.PersonId, item.Code)
		if err != nil {
			log.Printf("工作线程 %d 删除数据失败 (encounter_id=%s): %v", workerID, visit.EncounterId, err)
			continue
		}

		// 插入新数据
		_, err = db.ExecContext(ctx, `
			INSERT INTO public.t_patient_data (p_id, v_id, field_id, value, "source")
			VALUES ($1, $2, $3, $4, 'dify');`, 
			visit.PersonId, visit.PersonId, item.Code, item.Value)
		if err != nil {
			log.Printf("工作线程 %d 插入数据失败 (encounter_id=%s): %v", workerID, visit.EncounterId, err)
		}
	}
}

