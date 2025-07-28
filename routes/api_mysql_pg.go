package routes

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

type DBConfig struct {
	Host, User, Password, Database, Schema string
	Port                                   int
}

// 简单日志，直接用fmt
func logf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] %s\n", now, msg)
}

// 从viper读取数据库配置
func getDBConfig(prefix string) DBConfig {
	return DBConfig{
		Host:     viper.GetString(prefix + ".HOST"),
		User:     viper.GetString(prefix + ".USER"),
		Password: viper.GetString(prefix + ".PASSWORD"),
		Database: viper.GetString(prefix + ".DATABASE"),
		Port:     viper.GetInt(prefix + ".PORT"),
	}
}

// 连接MySQL数据库
func connectMySQL(cfg DBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	return sql.Open("mysql", dsn)
}

// 获取MySQL数据库中的所有表名
func getMySQLTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, nil
}

// 生成Seatunnel配置文件
func generateConfigFile(src, tgt DBConfig, table string) (string, error) {
	content := fmt.Sprintf(`env {
  parallelism = 2
  job.mode = "BATCH"
}

source {
  Jdbc {
    url = "jdbc:mysql://%s:%d/%s"
    driver = "com.mysql.cj.jdbc.Driver"
    user = "%s"
    password = "%s"
    table_path = "%s.%s"
    query = "select * from %s.%s"
  }
}

sink {
  Jdbc {
    url = "jdbc:postgresql://%s:%d/%s"
    driver = org.postgresql.Driver
    user = "%s"
    password = "%s"
    generate_sink_sql = true
    database = "%s"
    table = "%s.%s"
    batch_size = 1000
    batch_interval_ms = 3000
    connection_check_timeout_sec = 100
  }
}`,
		src.Host, src.Port, src.Database, src.User, src.Password, src.Database, table, src.Database, table,
		tgt.Host, tgt.Port, tgt.Database, tgt.User, tgt.Password, tgt.Database, tgt.Schema, table)

	filename := fmt.Sprintf("/data/seatunnel/job/pg_sync_%s_%d.conf", table, time.Now().Unix())
	return filename, os.WriteFile(filename, []byte(content), 0644)
}

// 执行Seatunnel命令
func executeSeatunnel(configPath string) error {
	cmd := exec.Command("/data/seatunnel/bin/seatunnel.sh", "--config", configPath, "-m", "local")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logf("Seatunnel命令输出: %s", string(output))
		return err
	}
	return nil
}

// 主同步流程，便于集成到API或CLI
func SyncMySQLToPG(src, tgt DBConfig) {
	db, err := connectMySQL(src)
	if err != nil {
		logf("无法连接MySQL: %v", err)
		return
	}
	defer db.Close()

	tables, err := getMySQLTables(db)
	if err != nil {
		logf("获取表失败: %v", err)
		return
	}

	logf("共发现 %d 个表", len(tables))

	for _, table := range tables {
		logf("同步表: %s", table)
		configPath, err := generateConfigFile(src, tgt, table)
		if err != nil {
			logf("生成配置失败: %v", err)
			continue
		}
		if err := executeSeatunnel(configPath); err != nil {
			logf("同步失败: %v", err)
		} else {
			logf("同步成功: %s", table)
		}
		os.Remove(configPath)
	}

	logf("同步任务完成")
}

func handleSeatunnelMysqlPg(c *gin.Context) {
	// 异步执行同步任务
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered from panic in goroutine: %v\n", r)
			}
		}()
		mysqlCfg := getDBConfig("mysql_src")
		pgCfg := getDBConfig("postgres_tgt")
		SyncMySQLToPG(mysqlCfg, pgCfg)
	}()

	c.JSON(200, gin.H{
		"message": "handleSeatunnelMysqlPg task started (async)",
	})
}
