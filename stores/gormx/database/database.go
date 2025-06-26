package database

import (
	"fmt"
	"github.com/og-game/glib/stores/gormx/config"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"time"
)

func NewEngine(c config.Config, dialector gorm.Dialector, opt ...gorm.Option) (*gorm.DB, error) {
	cfg := &gorm.Config{
		PrepareStmt:            c.PrepareStmt,
		SkipDefaultTransaction: c.SkipDefaultTransaction,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	}

	engine, err := gorm.Open(dialector, append(opt, cfg)...)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	if c.Debug {
		engine = engine.Debug()
	}

	if c.Trace {
		registerTraceHook(engine)
	}

	// 设置连接池参数
	sqlDB, err := engine.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %v", err)
	}

	if c.MaxIdleConn > 0 {
		sqlDB.SetMaxIdleConns(c.MaxIdleConn)
	}
	if c.MaxOpenConn > 0 {
		sqlDB.SetMaxOpenConns(c.MaxOpenConn)
	}
	if c.MaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(c.MaxLifetime) * time.Minute)
	}

	return engine, nil
}
