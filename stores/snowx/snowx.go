package snowx

import (
	"context"
	"errors"
	"fmt"
	"github.com/og-game/glib/stores/redisx"
	"github.com/sony/sonyflake"
	"os"
	"time"
)

var (
	flake           *sonyflake.Sonyflake
	flakeMachineKey = "snowflake_machine_id"
)

const (
	MaxMachineID   = 1023
	LockExpireTime = 120 * time.Second
)

func Init() {
	flake = sonyflake.NewSonyflake(sonyflake.Settings{
		StartTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		MachineID: getMachineID,
	})
}

// GenId 生成一个唯一的雪花ID
func GenId() (id uint64, err error) {
	id, err = flake.NextID()
	return
}

func getMachineID() (uint16, error) {
	rdb := redisx.Engine
	ctx := context.Background()

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = fmt.Sprintf("pod-%d", time.Now().UnixNano())
	}

	for i := 1; i <= MaxMachineID; i++ {
		key := fmt.Sprintf("%s:%d", flakeMachineKey, i)

		ok, err := rdb.SetNX(ctx, key, hostname, LockExpireTime).Result()
		if err != nil {
			return 0, fmt.Errorf("redis SETNX error: %w", err)
		}

		if ok {
			// 抢占成功，启动续租 goroutine
			go keepAlive(key, hostname)
			return uint16(i), nil
		}
	}

	return 0, errors.New("no available machineID in 1~1023")
}

func keepAlive(key, hostname string) {
	rdb := redisx.Engine
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	ctx := context.Background()

	for range ticker.C {
		val, err := rdb.Get(ctx, key).Result()
		if err == nil && val == hostname {
			rdb.Expire(ctx, key, LockExpireTime)
		}
	}
}
