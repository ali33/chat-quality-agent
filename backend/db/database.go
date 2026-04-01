package db

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vietbui/chat-quality-agent/config"
	"github.com/vietbui/chat-quality-agent/db/models"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Driver is the active SQL backend (sqlite, mysql, postgres, sqlserver), set after Connect.
var Driver string

// DateSQL returns a dialect-specific expression that yields a calendar date from a timestamp column.
func DateSQL(column string) string {
	switch Driver {
	case "mysql":
		return "DATE(" + column + ")"
	case "postgres":
		return "CAST(" + column + " AS DATE)"
	case "sqlserver":
		return "CAST(" + column + " AS DATE)"
	default:
		return "date(" + column + ")"
	}
}

// Connect opens the database selected by cfg.DBDriver.
func Connect(cfg *config.Config) error {
	logLevel := logger.Info
	if cfg.IsProduction() {
		logLevel = logger.Warn
	}

	gcfg := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	var dialector gorm.Dialector
	switch cfg.DBDriver {
	case "sqlite":
		path := cfg.SQLitePath
		if err := ensureSQLiteDir(path); err != nil {
			return fmt.Errorf("sqlite data directory: %w", err)
		}
		dialector = sqlite.Open(path)
		Driver = "sqlite"

	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
		dialector = mysql.Open(dsn)
		Driver = "mysql"

	case "postgres":
		dsn := postgresDSN(cfg)
		dialector = postgres.Open(dsn)
		Driver = "postgres"

	case "sqlserver":
		dsn := sqlServerDSN(cfg)
		dialector = sqlserver.Open(dsn)
		Driver = "sqlserver"

	default:
		return fmt.Errorf("unsupported DB driver: %s", cfg.DBDriver)
	}

	var err error
	DB, err = gorm.Open(dialector, gcfg)
	if err != nil {
		return fmt.Errorf("open database (%s): %w", cfg.DBDriver, err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	if cfg.DBDriver == "sqlite" {
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetMaxOpenConns(1)
	} else {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	log.Printf("Database connected (driver=%s)", cfg.DBDriver)
	return nil
}

func postgresDSN(cfg *config.Config) string {
	ssl := cfg.DBSSLMode
	if ssl == "" {
		ssl = "disable"
	}
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort),
		Path:   "/" + cfg.DBName,
	}
	q := u.Query()
	q.Set("sslmode", ssl)
	u.RawQuery = q.Encode()
	return u.String()
}

func sqlServerDSN(cfg *config.Config) string {
	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort),
	}
	q := u.Query()
	q.Set("database", cfg.DBName)
	u.RawQuery = q.Encode()
	return u.String()
}

func ensureSQLiteDir(path string) error {
	if path == "" || path == ":memory:" || strings.HasPrefix(path, "file::memory:") {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

func AutoMigrate() error {
	err := DB.AutoMigrate(
		&models.User{},
		&models.Tenant{},
		&models.UserTenant{},
		&models.Channel{},
		&models.Conversation{},
		&models.Message{},
		&models.Job{},
		&models.JobRun{},
		&models.JobResult{},
		&models.AppSetting{},
		&models.NotificationLog{},
		&models.AIUsageLog{},
		&models.OAuthClient{},
		&models.OAuthAuthorizationCode{},
		&models.OAuthToken{},
		&models.ActivityLog{},
	)
	if err != nil {
		return fmt.Errorf("auto-migrate: %w", err)
	}

	addUniqueConstraints()

	log.Println("Database migration completed")
	return nil
}

func addUniqueConstraints() {
	constraints := []struct {
		name    string
		table   string
		columns string
	}{
		{"uq_channel_tenant_type_ext", "channels", "tenant_id, channel_type, external_id"},
		{"uq_conv_tenant_channel_ext", "conversations", "tenant_id, channel_id, external_conversation_id"},
		{"uq_msg_tenant_conv_ext", "messages", "tenant_id, conversation_id, external_message_id"},
	}

	for _, c := range constraints {
		switch Driver {
		case "mysql":
			sql := fmt.Sprintf(
				"ALTER TABLE `%s` ADD UNIQUE INDEX `%s` (%s)",
				c.table, c.name, c.columns,
			)
			DB.Exec(sql)
		case "sqlserver":
			sql := fmt.Sprintf(`
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = '%s' AND object_id = OBJECT_ID('%s'))
CREATE UNIQUE INDEX [%s] ON [%s] (%s)`,
				c.name, c.table, c.name, c.table, c.columns)
			DB.Exec(sql)
		default:
			sql := fmt.Sprintf(
				"CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s)",
				c.name, c.table, c.columns,
			)
			DB.Exec(sql)
		}
	}
}

func Close() {
	if DB != nil {
		sqlDB, _ := DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}
