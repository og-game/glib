package database

import (
	"context"
	"github.com/zeromicro/go-zero/core/logc"
	"gorm.io/gorm"
)

func registerTraceHook(tx *gorm.DB) {
	tx.Callback().Create().Before("gorm:created").Register("trace:create", func(db *gorm.DB) {
		traceSql(db.Statement.Context, "gorm:create", db)
	})
	tx.Callback().Create().After("gorm:saved").Register("trace:save", func(db *gorm.DB) {
		traceSql(db.Statement.Context, "gorm:save", db)
	})
	tx.Callback().Query().After("gorm:queried").Register("trace:query", func(db *gorm.DB) {
		traceSql(db.Statement.Context, "gorm:query", db)
	})
	tx.Callback().Delete().After("gorm:deleted").Register("trace:delete", func(db *gorm.DB) {
		traceSql(db.Statement.Context, "gorm:delete", db)
	})
	tx.Callback().Update().After("gorm:updated").Register("trace:update", func(db *gorm.DB) {
		traceSql(db.Statement.Context, "gorm:update", db)
	})
	tx.Callback().Raw().After("*").Register("trace:raw", func(db *gorm.DB) {
		traceSql(db.Statement.Context, "gorm:raw", db)
	})
	tx.Callback().Row().After("*").Register("trace:row", func(db *gorm.DB) {
		traceSql(db.Statement.Context, "gorm:row", db)
	})
}

func traceSql(ctx context.Context, spanName string, db *gorm.DB) {
	sql := db.Dialector.Explain(db.Statement.SQL.String(), db.Statement.Vars...)
	logc.Debugf(ctx, "gorm sql: %s", sql)
}
