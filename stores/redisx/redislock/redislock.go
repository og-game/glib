package redislock

import (
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

// NewLock 获取锁
func NewLock(rdb redis.UniversalClient) *redsync.Redsync {
	return redsync.New(goredis.NewPool(rdb))
}
