package system

import (
	"fmt"
	"time"

	"github.com/hellodex/HelloSecurity/config"
	log "github.com/hellodex/HelloSecurity/log"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Ewriter struct {
	mlog *logrus.Logger
}

// Implement the Printf function required by gorm's logger
func (m *Ewriter) Printf(format string, v ...interface{}) {
	logstr := fmt.Sprintf(format, v...)
	m.mlog.Info(logstr)
}

func NewWriter() *Ewriter {
	// Use the Logger from the sys package
	return &Ewriter{mlog: log.Logger}
}

var DB *gorm.DB

func init() {
	// 获取配置
	cfg := config.GetConfig()

	// 自定义 GORM 日志记录器
	newLogger := logger.New(
		NewWriter(), // 使用自定义的 logrus 日志记录器
		logger.Config{
			SlowThreshold:             100 * time.Millisecond, // 慢 SQL 阈值
			LogLevel:                  logger.Warn,            // 日志级别
			IgnoreRecordNotFoundError: true,                   // 忽略ErrRecordNotFound（记录未找到）错误
			Colorful:                  false,                  // 禁用彩色打印
		},
	)

	// 构造 MySQL DSN（数据源名称）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// 打开 MySQL 连接
	database, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,   // DSN 数据源名称
		DefaultStringSize:         256,   // 默认字符串长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的版本不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的版本不支持
		SkipInitializeWithVersion: false, // 根据版本自动配置
	}), &gorm.Config{
		Logger: newLogger, // 使用自定义的 GORM 日志记录器
	})

	// 错误处理
	if err != nil {
		log.Logger.Fatal(err) // 使用 sys.Logger 记录致命错误
	}

	// 将数据库实例赋值给全局变量 DB
	DB = database
}

func GetDb() *gorm.DB {
	return DB
}
