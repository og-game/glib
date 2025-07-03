package gormx

import (
	"context"
	"gorm.io/gorm"
)

type BaseDAO struct {
	ctx    context.Context
	engine *gorm.DB
}

func (dao *BaseDAO) TX() *gorm.DB {
	if dao.engine == nil { // default mysql engine
		dao.engine = Engine.Mysql
	}
	return dao.engine.WithContext(dao.Context())
}

func (dao *BaseDAO) WithContext(ctx context.Context) *BaseDAO {
	dao.ctx = ctx
	return dao
}

func (dao *BaseDAO) GetContext() context.Context {
	if dao.ctx == nil {
		dao.ctx = context.Background()
	}
	return dao.ctx
}

func (dao *BaseDAO) Context() context.Context {
	if dao.ctx == nil {
		dao.ctx = context.Background()
	}
	return dao.ctx
}

func (dao *BaseDAO) WithEngine(engine *gorm.DB) *BaseDAO {
	dao.engine = engine
	return dao
}
