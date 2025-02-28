package util

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

// DelMysql执行从数据库中查询特定条件数据并根据规则删除部分数据的操作，基于事务保证数据一致性
func DelMysql() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in goroutine: %v", r)
		}
	}()

	// 验证Viper配置是否已正确加载，这里简单检查关键的数据库配置项是否存在
	if viper.GetString("DB.HOST") == "" || viper.GetString("DB.USER") == "" || viper.GetString("DB.PASSWORD") == "" || viper.GetString("DB.DATABASE") == "" {
		log.Printf("Viper配置未正确初始化或缺少数据库连接相关配置项")
		return
	}

	// log.Println("开始执行从数据库中查询特定条件数据并根据规则删除部分数据的操作")
	// var zeros int = 0
	// log.Println(1 / zeros)

	// 从Viper中读取数据库连接信息
	host := viper.GetString("DB.HOST")
	user := viper.GetString("DB.USER")
	password := viper.GetString("DB.PASSWORD")
	database := viper.GetString("DB.DATABASE")

	// 构建数据库连接字符串
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s)/%s", user, password, host, database)
	// 建立数据库连接
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Printf("建立数据库连接失败: %v", err)
		return
	}
	defer db.Close()

	// 连接可能会在后续使用时才真正建立，检查连接是否成功
	err = db.Ping()
	if err != nil {
		log.Printf("检查数据库连接失败: %v", err)
		return
	}
	// 获取连接池中的连接
	conn, err := db.Conn(context.Background())
	if err != nil {
		log.Printf("获取连接失败: %v", err)
		return
	}
	defer conn.Close()

	// 开始事务
	tx, err := conn.BeginTx(context.Background(), nil)
	if err != nil {
		log.Printf("开始事务失败: %v", err)
		return
	}
	defer func() {
		// 无论后续操作是否成功，在函数结束时都确保事务正确回滚（如果还未提交的话）
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("事务回滚失败: %v", err)
		}
	}()

	// 定义查询语句，这里查询tasks表的符合条件的记录
	query := `
        select * from (select code,version,count(1) cnt from t_ds_task_definition_log group by code,version ) t where t.cnt>1 
    `
	rows, err := tx.Query(query)
	if err != nil {
		log.Printf("查询失败: %v", err)
		return
	}
	defer rows.Close()

	// 遍历查询结果
	for rows.Next() {
		var taskID, taskVersion string
		var taskCnt int
		err := rows.Scan(&taskID, &taskVersion, &taskCnt)
		if err != nil {
			log.Printf("扫描结果失败: %v", err)
			return
		}
		log.Printf("Task: %s %s %d\n", taskID, taskVersion, taskCnt)

		queryID := `
            select * from t_ds_task_definition_log where code =? and version =? order by create_time desc 
        `
		if err := conn.PingContext(context.Background()); err != nil {
			log.Printf("内部查询前检查数据库连接失败: %v", err)
			return
		}

		// 在内部查询之前检查数据库连接是否有效
		// err = db.Ping()
		// if err != nil {
		// 	log.Printf("内部查询前检查数据库连接失败: %v", err)
		// 	return
		// }
		innerRows, err := tx.Query(queryID, taskID, taskVersion)
		if err != nil {
			log.Printf("内部查询失败: %v", err)
			return
		}
		defer innerRows.Close()

		isFirst := true
		for innerRows.Next() {
			var id, code, version, name, operateTime, createTime, updateTime string
			err := innerRows.Scan(&id, &code, &version, &name, &operateTime, &createTime, &updateTime)
			if err != nil {
				log.Printf("扫描内部结果失败: %v", err)
				return
			}
			if isFirst {
				isFirst = false
				log.Printf("不删除: Task: %s %s %s %s %s %s %s\n", id, code, version, name, operateTime, createTime, updateTime)
				continue
			}
			// 确认删除操作
			deleteQuery := "DELETE FROM t_ds_task_definition_log WHERE id=?"
			result, err := tx.Exec(deleteQuery, id)
			if err != nil {
				log.Printf("删除操作失败: %v", err)
				return
			}
			rowsAffected, _ := result.RowsAffected()
			log.Printf("准备删除: Task: %s %s %s %s %s %s %s\n", id, code, version, name, operateTime, createTime, updateTime)
			log.Printf("删除结果: %d\n", rowsAffected)
		}
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		log.Printf("提交事务失败: %v", err)
		return
	}
}
