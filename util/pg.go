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
	EncounterId string `json:"encounter_id"`
	PersonId    string `json:"person_id"`
	PatientId   string `json:"patient_id"`
	Content     string `json:"content"`
}

// ProcessVisits 处理访视记录的主函数
func ProcessVisits() error {
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
	) AS "content" FROM public.dc_mr_document_index_outpat where deleted_flag is null;`)
	if err != nil {
		log.Printf("查询失败: %v", err)
		return err
	}
	defer rows.Close()

	var visits []PatientVisit
	for rows.Next() {
		var visit PatientVisit
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
					
					log.Printf("工作线程 %d 处理记录 encounter_id=%s", workerID, visit.EncounterId)
					
					// 调用Dify API
					resultData, err := RunWorkflowWithSDK(fmt.Sprintf("%s", visit.Content))
					if err != nil {
						log.Printf("工作线程 %d 获取阶段失败 (encounter_id=%s): %v", workerID, visit.EncounterId, err)
						continue
					}
					
					// 更新数据库
					result, err := workerDB.ExecContext(ctx, `
						UPDATE public.dc_mr_document_index_outpat SET deleted_flag = $1,patient_external = $2 WHERE encounter_id = $3 and person_id = $4 and patient_id = $5;`, resultData.VisitNumber,resultData.GestationalWeeks, visit.EncounterId, visit.PersonId, visit.PatientId)
					
					if err != nil {
						log.Printf("工作线程 %d 更新失败 (encounter_id=%s): %v", workerID, visit.EncounterId, err)
						continue
					}
					
					if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
						log.Printf("工作线程 %d 成功更新记录 encounter_id=%s", workerID, visit.EncounterId)
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

// ValidatePatientData 校验t_patient_data表中的数据格式
func ValidatePatientData() error {
	// 从配置加载数据库连接信息
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		viper.GetString("pg_struct.HOST"),
		viper.GetString("pg_struct.PORT"),
		viper.GetString("pg_struct.USER"),
		viper.GetString("pg_struct.PASSWORD"),
		viper.GetString("pg_struct.DATABASE"),
	)

	// 建立数据库连接
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
	log.Println("数据库连接成功，开始校验t_patient_data数据")

	// 查询需要校验的数据
	rows, err := db.Query(`
		SELECT 
			t.p_id,
			t.field_id,
			t.value,
			tmv.value_type,
			tmv.is_multi_choice
		FROM 
			public.t_patient_data t 
		LEFT JOIN public.t_model_view tmv ON 
			t.field_id = tmv.id;`)
	if err != nil {
		log.Printf("查询失败: %v", err)
		return err
	}
	defer rows.Close()

	// 存储需要标记为删除的记录主键
	var invalidRecords []struct {
		pID     string
		fieldID string
	}

	// 遍历结果进行校验
	for rows.Next() {
		var pID, fieldID, value string
		var valueType int
		var isMultiChoice sql.NullString

		if err := rows.Scan(&pID, &fieldID, &value, &valueType, &isMultiChoice); err != nil {
			log.Printf("扫描记录失败: %v", err)
			continue
		}

		// 标记为无效的标志
		isInvalid := false

		// 根据value_type和is_multi_choice进行校验
		// 校验1: value_type=4且is_multi_choice='单选'，value必须是int
		if valueType == 4 && isMultiChoice.Valid && isMultiChoice.String == "单选" {
			// 检查value是否为整数
			if !isInteger(value) {
				log.Printf("校验失败: p_id=%s, field_id=%s, value=%s (value_type=4, is_multi_choice='单选'，要求整数)", pID, fieldID, value)
				isInvalid = true
			}
		// 校验2: value_type=3，value格式必须是2001/01/01
		} else if valueType == 3 {
			// 检查value是否为YYYY/MM/DD格式
			if !isValidDateFormat(value) {
				log.Printf("校验失败: p_id=%s, field_id=%s, value=%s (value_type=3，要求YYYY/MM/DD格式)", pID, fieldID, value)
				isInvalid = true
			}
		// 校验3: value_type=1，value必须是int
		} else if valueType == 1 {
			// 检查value是否为整数
			if !isInteger(value) {
				log.Printf("校验失败: p_id=%s, field_id=%s, value=%s (value_type=1，要求整数)", pID, fieldID, value)
				isInvalid = true
			}
		}

		// 如果校验失败，添加到无效记录列表
		if isInvalid {
			invalidRecords = append(invalidRecords, struct {
				pID     string
				fieldID string
			}{pID: pID, fieldID: fieldID})
		}
	}

	if err = rows.Err(); err != nil {
		log.Printf("行迭代错误: %v", err)
		return err
	}

	log.Printf("共发现 %d 条无效记录需要标记为删除", len(invalidRecords))

	// 批量更新无效记录的del_flag为1
	if len(invalidRecords) > 0 {
		// 使用事务来提高性能和确保原子性
		tx, err := db.Begin()
		if err != nil {
			log.Printf("创建事务失败: %v", err)
			return err
		}

		// 准备更新语句
		stmt, err := tx.Prepare(`
			UPDATE public.t_patient_data 
			SET del_flag = 1 
			WHERE p_id = $1 AND field_id = $2`)
		if err != nil {
			log.Printf("准备更新语句失败: %v", err)
			tx.Rollback()
			return err
		}
		defer stmt.Close()

		// 批量执行更新
		for _, record := range invalidRecords {
			_, err := stmt.Exec(record.pID, record.fieldID)
			if err != nil {
				log.Printf("更新记录失败 (p_id=%s, field_id=%s): %v", record.pID, record.fieldID, err)
				tx.Rollback()
				return err
			}
		}

		// 提交事务
		if err := tx.Commit(); err != nil {
			log.Printf("提交事务失败: %v", err)
			tx.Rollback()
			return err
		}

		log.Printf("成功将 %d 条无效记录标记为删除", len(invalidRecords))
	}

	log.Println("t_patient_data数据校验完成")
	return nil
}

// isInteger 检查字符串是否为整数
func isInteger(s string) bool {
	if len(s) == 0 {
		return false
	}

	// 处理负数
	i := 0
	if s[0] == '-' {
		if len(s) == 1 {
			return false
		}
		i = 1
	}

	// 检查其余字符是否都是数字
	for ; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isValidDateFormat 检查字符串是否为YYYY/MM/DD格式的日期
func isValidDateFormat(s string) bool {
	// 检查字符串长度是否为10
	if len(s) != 10 {
		return false
	}

	// 检查格式是否为YYYY/MM/DD
	if s[4] != '/' || s[7] != '/' {
		return false
	}

	// 检查年、月、日是否为数字
	for i := 0; i < len(s); i++ {
		if i == 4 || i == 7 {
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}

	// 简单检查日期的有效性（这里可以根据需要增加更复杂的日期验证）
	// 例如检查月份是否在1-12之间，日期是否在对应月份的有效范围内
	// 为简化起见，这里只检查格式
	return true
}
