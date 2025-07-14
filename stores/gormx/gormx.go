package gormx

import (
	"fmt"
	"github.com/og-game/glib/stores/gormx/config"
	"github.com/og-game/glib/stores/gormx/database"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"sync"
)

var (
	once   sync.Once
	Engine *DBManager
)

// DBManager gorm db manager
type DBManager struct {
	Mysql      *gorm.DB
	Postgres   *gorm.DB
	ClickHouse *gorm.DB
}

func NewDBManager() *DBManager {
	return &DBManager{}
}

func (dm *DBManager) Create(cs ...config.Config) error {
	for _, c := range cs {
		engine, err := dm.newEngine(c)
		if err != nil {
			return err
		}
		switch c.Mode {
		case config.Mysql:
			dm.Mysql = engine
		case config.Postgres:
			dm.Postgres = engine
		case config.ClickHouse:
			dm.ClickHouse = engine
		}
	}
	Engine = dm
	return nil
}

// Must initialize the database
func Must(cs ...config.Config) {
	if len(cs) == 0 {
		panic("failed to initialize databases: config is empty")
	}
	once.Do(func() {
		if err := NewDBManager().Create(cs...); err != nil {
			panic(fmt.Sprintf("failed to initialize databases: %v", err))
		}
	})
}

func (dm *DBManager) newEngine(c config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch c.Mode {
	case config.Mysql:
		dialector = mysql.Open(c.DSN)
	case config.Postgres:
		dialector = postgres.Open(c.DSN)
	case config.ClickHouse:
		dialector = clickhouse.Open(c.DSN)
	default:
		return nil, fmt.Errorf("unsupported database mode: %d", c.Mode)
	}

	engine, err := database.NewEngine(c, dialector)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s database: %v", dialector.Name(), err)
	}

	return engine, nil
}
