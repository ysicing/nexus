package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ysicing/nexus/pkg/models"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	DSN string `json:"dsn"` // 数据库连接字符串

	// 连接池配置
	MaxIdleConns    int           `json:"maxIdleConns"`
	MaxOpenConns    int           `json:"maxOpenConns"`
	ConnMaxLifetime time.Duration `json:"connMaxLifetime"`
}

// Database 数据库管理器
type Database struct {
	config      *DatabaseConfig
	db          *gorm.DB
	clusterRepo models.ClusterRepository
}

// NewDatabase 创建数据库管理器
func NewDatabase(config *DatabaseConfig) (*Database, error) {
	db := &Database{
		config: config,
	}

	if err := db.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return db, nil
}

// initialize 初始化数据库连接
func (d *Database) initialize() error {
	var dialector gorm.Dialector

	// 根据 DSN 前缀判断数据库类型
	dsn := d.config.DSN
	switch {
	case len(dsn) > 7 && dsn[:7] == "sqlite:":
		// SQLite: sqlite:./data/nexus.db
		dbPath := dsn[7:]
		log.Printf("Initializing SQLite database at: %s", dbPath)

		// 确保 SQLite 数据库目录存在
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}

		dialector = sqlite.Open(dbPath)

	case len(dsn) > 8 && dsn[:8] == "mysql://":
		// MySQL: mysql://user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local
		log.Printf("Initializing MySQL database")
		// 转换为 GORM MySQL DSN 格式
		mysqlDSN := dsn[8:] // 去掉 mysql:// 前缀
		dialector = mysql.Open(mysqlDSN)

	case len(dsn) > 11 && dsn[:11] == "postgres://":
		// PostgreSQL: postgres://user:password@localhost:5432/dbname?sslmode=disable
		log.Printf("Initializing PostgreSQL database")
		dialector = postgres.Open(dsn)

	default:
		// 尝试直接解析 DSN
		if len(dsn) == 0 {
			return fmt.Errorf("database DSN is empty")
		}

		// 默认尝试 SQLite
		log.Printf("Trying to parse DSN as SQLite: %s", dsn)

		// 确保 SQLite 数据库目录存在
		if err := os.MkdirAll(filepath.Dir(dsn), 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}

		dialector = sqlite.Open(dsn)
	}

	// 配置 GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// 建立数据库连接
	db, err := gorm.Open(dialector, config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 获取底层的 sql.DB 来配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(d.config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(d.config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(d.config.ConnMaxLifetime)

	d.db = db
	d.clusterRepo = models.NewClusterRepository(db)

	log.Printf("Database initialized successfully")
	return nil
}

// GetClusterRepository 获取集群仓库
func (d *Database) GetClusterRepository() models.ClusterRepository {
	return d.clusterRepo
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	if d.db != nil {
		sqlDB, err := d.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDefaultConfig 获取默认数据库配置
func GetDefaultConfig() *DatabaseConfig {
	dsn := getEnvString("DATABASE_DSN", "sqlite:./data/nexus.db")

	config := &DatabaseConfig{
		DSN:             dsn,
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 100),
		ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME", 3600)) * time.Second,
	}

	return config
}

// getEnvString 获取环境变量字符串值
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取环境变量整数值
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if n, err := fmt.Sscanf(value, "%d", &intValue); err == nil && n == 1 {
			return intValue
		}
	}
	return defaultValue
}

// MigrateDatabase 执行数据库迁移
func (d *Database) MigrateDatabase() error {
	log.Println("Running database migrations...")

	// 自动迁移集群模型
	if err := d.db.AutoMigrate(&models.ClusterModel{}); err != nil {
		return fmt.Errorf("failed to migrate cluster model: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}
