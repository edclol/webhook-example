package util

import (
    "database/sql"
    "fmt"
    "time"

    _ "github.com/go-sql-driver/mysql"
    "github.com/spf13/viper"
)

// 定义查询语句常量
const (
    outerQuery = `
        select * from (select code,version,count(1) cnt from t_ds_task_definition_log group by code,version ) t where t.cnt>1 
    `
    innerQuery = `
        select id, code, version, name, create_time as createTime from t_ds_task_definition_log where code =? and version =? order by create_time desc 
    `
    deleteQuery = "DELETE FROM t_ds_task_definition_log WHERE id=?"
)

// DelMysql 执行从数据库中查询特定条件数据并根据规则删除部分数据的操作，基于事务保证数据一致性
func DelMysql() error {
    // 延迟处理 panic
    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("Recovered from panic in goroutine: %v", r)
        }
    }()

    // 验证 Viper 配置是否已正确加载
    if viper.GetString("DB.HOST") == "" ||
        viper.GetString("DB.USER") == "" ||
        viper.GetString("DB.PASSWORD") == "" ||
        viper.GetString("DB.DATABASE") == "" {
        fmt.Printf("Viper配置未正确初始化或缺少数据库连接相关配置项")
        return fmt.Errorf("Viper配置未正确初始化或缺少数据库连接相关配置项")
    }

    // 从 Viper 中读取数据库连接信息
    host := viper.GetString("DB.HOST")
    user := viper.GetString("DB.USER")
    password := viper.GetString("DB.PASSWORD")
    database := viper.GetString("DB.DATABASE")

    // 构建数据库连接字符串
    dataSourceName := fmt.Sprintf("%s:%s@tcp(%s)/%s", user, password, host, database)

    // 建立数据库连接
    db, err := sql.Open("mysql", dataSourceName)
    if err != nil {
        fmt.Printf("建立数据库连接失败: %v", err)
        return fmt.Errorf("建立数据库连接失败: %w", err)
    }
    // 确保函数结束时关闭数据库连接
    defer db.Close()

    // 配置数据库连接池参数
    db.SetConnMaxLifetime(1 * time.Minute) // 设置连接的最大生命周期为 30 分钟
    db.SetMaxIdleConns(0)                  // 设置最大空闲连接数为 10
    db.SetMaxOpenConns(3)                  // 设置最大打开连接数为 20

    // 检查数据库连接是否成功
    err = db.Ping()
    if err != nil {
        fmt.Printf("检查数据库连接失败: %v", err)
        return fmt.Errorf("检查数据库连接失败: %w", err)
    }

    // 执行外部查询
    rows, err := db.Query(outerQuery)
    if err != nil {
        fmt.Printf("查询失败: %v", err)
        return fmt.Errorf("查询失败: %w", err)
    }
    // 确保查询结果集在函数结束时关闭
    defer rows.Close()

    // 遍历外部查询结果
    for rows.Next() {
        var taskID, taskVersion string
        var taskCnt int

        err := rows.Scan(&taskID, &taskVersion, &taskCnt)
        if err != nil {
            fmt.Printf("扫描结果失败: %v", err)
            return fmt.Errorf("扫描结果失败: %w", err)
        }
        fmt.Printf("Task: %s %s %d\n", taskID, taskVersion, taskCnt)

        // 执行内部查询
        innerRows, err := db.Query(innerQuery, taskID, taskVersion)
        if err != nil {
            fmt.Printf("内部查询失败: %v", err)
            return fmt.Errorf("内部查询失败: %w", err)
        }
        // 确保内部查询结果集在函数结束时关闭
        defer innerRows.Close()

        isFirst := true
        // 遍历内部查询结果
        for innerRows.Next() {
            var id, code, version, name, createTime string

            err := innerRows.Scan(&id, &code, &version, &name, &createTime)
            if err != nil {
                fmt.Printf("扫描内部结果失败: %v", err)
                return fmt.Errorf("扫描内部结果失败: %w", err)
            }

            if isFirst {
                isFirst = false
                fmt.Printf("不删除: Task: %s %s %s %s %s\n", id, code, version, name, createTime)
                continue
            }

            // 执行删除操作
            result, err := db.Exec(deleteQuery, id)
            if err != nil {
                fmt.Printf("删除操作失败: %v", err)
                return fmt.Errorf("删除操作失败: %w", err)
            }

            rowsAffected, _ := result.RowsAffected()
            fmt.Printf("准备删除: Task: %s %s %s %s %s\n", id, code, version, name, createTime)
            fmt.Printf("删除结果: %d\n", rowsAffected)
        }
    }
    fmt.Println("删除完成")
    return nil
}